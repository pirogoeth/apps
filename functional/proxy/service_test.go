package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pirogoeth/apps/functional/database"
	"github.com/pirogoeth/apps/functional/types"
)

func setupTestProxyService(t *testing.T) (*ProxyService, *database.DbWrapper) {
	// Set gin to test mode
	gin.SetMode(gin.TestMode)
	
	// Setup test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	ctx := context.Background()
	db, err := database.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	
	if err := db.RunMigrations(database.MigrationsFS); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	
	// Create test config
	config := &types.Config{
		Proxy: types.ProxyConfig{
			ListenAddress:             ":8080",
			TraefikAPIURL:            "http://localhost:8080/api",
			MaxContainersPerFunction: 3,
			ContainerIdleTimeout:     5 * time.Minute,
		},
		Storage: types.StorageConfig{
			FunctionsPath: tempDir + "/functions",
			TempPath:      tempDir + "/temp",
		},
	}
	
	proxyService := NewProxyService(config, db)
	return proxyService, db
}

func TestProxyService_SerializeRequest(t *testing.T) {
	proxyService, db := setupTestProxyService(t)
	defer db.Close()
	
	// Create test HTTP request
	reqBody := `{"test": "data"}`
	req := httptest.NewRequest(http.MethodPost, "/test/path?param=value", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token123")
	
	// Create gin context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	
	// Test serialization
	funcReq, err := proxyService.serializeRequest(c, "test-request-id", "/test/path")
	if err != nil {
		t.Fatalf("Failed to serialize request: %v", err)
	}
	
	// Verify serialized request
	if funcReq.Method != "POST" {
		t.Errorf("Expected method POST, got %s", funcReq.Method)
	}
	
	if funcReq.Path != "/test/path" {
		t.Errorf("Expected path '/test/path', got %s", funcReq.Path)
	}
	
	if funcReq.Body != reqBody {
		t.Errorf("Expected body '%s', got %s", reqBody, funcReq.Body)
	}
	
	if funcReq.Headers["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type header to be preserved")
	}
	
	if funcReq.Query["param"] != "value" {
		t.Errorf("Expected query parameter to be preserved")
	}
	
	if funcReq.RequestID != "test-request-id" {
		t.Errorf("Expected request ID to be preserved")
	}
}

func TestProxyService_GenerateRequestID(t *testing.T) {
	proxyService, db := setupTestProxyService(t)
	defer db.Close()
	
	// Generate multiple request IDs
	id1 := proxyService.generateRequestID()
	id2 := proxyService.generateRequestID()
	
	// Verify IDs are unique
	if id1 == id2 {
		t.Errorf("Request IDs should be unique, got %s and %s", id1, id2)
	}
	
	// Verify ID format
	if len(id1) == 0 {
		t.Errorf("Request ID should not be empty")
	}
	
	if id1[:4] != "req_" {
		t.Errorf("Request ID should start with 'req_', got %s", id1)
	}
}

func TestProxyService_InFlightTracking(t *testing.T) {
	proxyService, db := setupTestProxyService(t)
	defer db.Close()
	
	// Create test in-flight request
	req := &InFlightRequest{
		ID:         "test-request",
		FunctionID: "test-function",
		StartTime:  time.Now(),
	}
	
	// Track request
	proxyService.trackInFlightRequest(req)
	
	// Verify tracking
	proxyService.inFlightMutex.RLock()
	tracked, exists := proxyService.inFlight[req.ID]
	proxyService.inFlightMutex.RUnlock()
	
	if !exists {
		t.Errorf("Request should be tracked")
	}
	
	if tracked.ID != req.ID {
		t.Errorf("Expected ID %s, got %s", req.ID, tracked.ID)
	}
	
	// Remove request
	proxyService.removeInFlightRequest(req.ID)
	
	// Verify removal
	proxyService.inFlightMutex.RLock()
	_, exists = proxyService.inFlight[req.ID]
	proxyService.inFlightMutex.RUnlock()
	
	if exists {
		t.Errorf("Request should be removed from tracking")
	}
}

func TestProxyService_ReturnResponse(t *testing.T) {
	proxyService, db := setupTestProxyService(t)
	defer db.Close()
	
	// Create test response
	response := &FunctionResponse{
		StatusCode: 201,
		Headers: map[string]string{
			"Content-Type":   "application/json",
			"X-Custom-Header": "test-value",
		},
		Body: `{"message": "success"}`,
	}
	
	// Create gin context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	
	// Return response
	proxyService.returnResponse(c, response)
	
	// Verify response
	if w.Code != 201 {
		t.Errorf("Expected status code 201, got %d", w.Code)
	}
	
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type header to be set")
	}
	
	if w.Header().Get("X-Custom-Header") != "test-value" {
		t.Errorf("Expected custom header to be set")
	}
	
	if w.Body.String() != `{"message": "success"}` {
		t.Errorf("Expected response body to match")
	}
}

func TestFunctionRequest_JSON(t *testing.T) {
	req := &FunctionRequest{
		Method:    "POST",
		Path:      "/api/test",
		Headers:   map[string]string{"Content-Type": "application/json"},
		Query:     map[string]string{"param": "value"},
		Body:      `{"data": "test"}`,
		RequestID: "req-123",
	}
	
	// Marshal to JSON
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}
	
	// Unmarshal from JSON
	var parsed FunctionRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}
	
	// Verify all fields
	if parsed.Method != req.Method {
		t.Errorf("Method mismatch: expected %s, got %s", req.Method, parsed.Method)
	}
	
	if parsed.Path != req.Path {
		t.Errorf("Path mismatch: expected %s, got %s", req.Path, parsed.Path)
	}
	
	if parsed.RequestID != req.RequestID {
		t.Errorf("RequestID mismatch: expected %s, got %s", req.RequestID, parsed.RequestID)
	}
}

func TestFunctionResponse_JSON(t *testing.T) {
	resp := &FunctionResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       `{"result": "success"}`,
		Error:      "",
	}
	
	// Marshal to JSON
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}
	
	// Unmarshal from JSON
	var parsed FunctionResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	
	// Verify all fields
	if parsed.StatusCode != resp.StatusCode {
		t.Errorf("StatusCode mismatch: expected %d, got %d", resp.StatusCode, parsed.StatusCode)
	}
	
	if parsed.Body != resp.Body {
		t.Errorf("Body mismatch: expected %s, got %s", resp.Body, parsed.Body)
	}
}

// Benchmark tests
func BenchmarkProxyService_SerializeRequest(b *testing.B) {
	proxyService, db := setupTestProxyService(&testing.T{})
	defer db.Close()
	
	reqBody := `{"test": "benchmark"}`
	req := httptest.NewRequest(http.MethodPost, "/test?param=value", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proxyService.serializeRequest(c, "bench-request", "/test")
	}
}

func BenchmarkProxyService_GenerateRequestID(b *testing.B) {
	proxyService, db := setupTestProxyService(&testing.T{})
	defer db.Close()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proxyService.generateRequestID()
	}
}