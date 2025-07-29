package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pirogoeth/apps/functional/compute"
	"github.com/pirogoeth/apps/functional/database"
	"github.com/pirogoeth/apps/functional/types"
)

func setupTestAPI(t *testing.T) (*gin.Engine, *types.ApiContext) {
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
	
	// Setup mock compute provider
	mockProvider := &MockComputeProvider{
		name: "mock",
	}
	
	// Create compute registry and register mock provider  
	registry := compute.NewRegistry()
	registry.Register(mockProvider)
	
	// Create test config
	config := &types.Config{
		Storage: types.StorageConfig{
			FunctionsPath: tempDir + "/functions",
			TempPath:      tempDir + "/temp",
		},
	}
	
	// Create API context
	apiContext := &types.ApiContext{
		Config:  config,
		Querier: db.Queries,
		Compute: registry,
	}
	
	// Setup router
	router := gin.New()
	err = MustRegister(router, apiContext)
	if err != nil {
		t.Fatalf("Failed to register routes: %v", err)
	}
	
	return router, apiContext
}

// Mock compute provider for testing
type MockComputeProvider struct {
	name           string
	deployError    error
	deployResult   *compute.DeployResult
	executeError   error
	executeResult  *compute.InvocationResult
	healthError    error
}

// Ensure MockComputeProvider implements ComputeProvider interface
var _ compute.ComputeProvider = (*MockComputeProvider)(nil)

func (m *MockComputeProvider) Name() string {
	return m.name
}

func (m *MockComputeProvider) Deploy(ctx context.Context, fn interface{}, imageName string) (interface{}, error) {
	if m.deployError != nil {
		return nil, m.deployError
	}
	
	if m.deployResult != nil {
		return m.deployResult, nil
	}
	
	return &compute.DeployResult{
		DeploymentID: uuid.New().String(),
		ResourceID:   "mock-resource-" + uuid.New().String()[:8],
		ImageTag:     "mock-image:latest",
	}, nil
}

func (m *MockComputeProvider) Execute(ctx context.Context, deployment interface{}, req interface{}) (interface{}, error) {
	if m.executeError != nil {
		return nil, m.executeError
	}
	
	if m.executeResult != nil {
		return m.executeResult, nil
	}
	
	return &compute.InvocationResult{
		StatusCode:   200,
		Body:         []byte(`{"message": "mock response"}`),
		Headers:      map[string]string{"Content-Type": "application/json"},
		DurationMS:   100,
		MemoryUsedMB: 64,
		ResponseSize: 25,
	}, nil
}

func (m *MockComputeProvider) Scale(ctx context.Context, deployment interface{}, replicas int) error {
	return nil
}

func (m *MockComputeProvider) Remove(ctx context.Context, deployment interface{}) error {
	return nil
}

func (m *MockComputeProvider) Health(ctx context.Context) error {
	return m.healthError
}

func TestV1Functions_CreateFunction(t *testing.T) {
	router, _ := setupTestAPI(t)
	
	// Create sample function code (base64 encoded)
	sampleCode := `{
		"index.js": "const express = require('express'); const app = express(); app.all('*', (req, res) => res.json({message: 'Hello World'})); app.listen(8080);"
	}`
	encodedCode := base64.StdEncoding.EncodeToString([]byte(sampleCode))
	
	tests := []struct {
		name           string
		requestBody    types.CreateFunctionRequest
		expectedStatus int
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name: "valid function creation",
			requestBody: types.CreateFunctionRequest{
				Name:           "test-function",
				Description:    "Test function description",
				Runtime:        "nodejs",
				Handler:        "index.handler",
				TimeoutSeconds: 30,
				MemoryMB:       128,
				EnvVars:        map[string]string{"TEST": "value"},
				Code:           encodedCode,
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				if response["name"] != "test-function" {
					t.Errorf("Expected name 'test-function', got %v", response["name"])
				}
				if response["runtime"] != "nodejs" {
					t.Errorf("Expected runtime 'nodejs', got %v", response["runtime"])
				}
				if response["id"] == nil || response["id"] == "" {
					t.Errorf("Expected non-empty ID")
				}
			},
		},
		{
			name: "missing required fields",
			requestBody: types.CreateFunctionRequest{
				Description: "Missing required fields",
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				if response["error"] == nil {
					t.Errorf("Expected error message for missing required fields")
				}
			},
		},
		{
			name: "empty function name",
			requestBody: types.CreateFunctionRequest{
				Name:    "",
				Runtime: "nodejs",
				Handler: "index.handler",
				Code:    encodedCode,
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				if response["error"] == nil {
					t.Errorf("Expected error message for empty function name")
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/v1/functions", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}
			
			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestV1Functions_ListFunctions(t *testing.T) {
	router, apiContext := setupTestAPI(t)
	
	// Create some test functions
	ctx := context.Background()
	functions := []database.CreateFunctionParams{
		{
			ID:             "list-test-1",
			Name:           "list-function-1",
			CodePath:       "/tmp/func1",
			Runtime:        "nodejs",
			Handler:        "index.handler",
			TimeoutSeconds: 30,
			MemoryMb:       128,
		},
		{
			ID:             "list-test-2",
			Name:           "list-function-2",
			CodePath:       "/tmp/func2",
			Runtime:        "python3",
			Handler:        "app.handler",
			TimeoutSeconds: 60,
			MemoryMb:       256,
		},
	}
	
	for _, params := range functions {
		_, err := apiContext.Querier.CreateFunction(ctx, params)
		if err != nil {
			t.Fatalf("Failed to create test function: %v", err)
		}
	}
	
	// Test list functions
	req := httptest.NewRequest(http.MethodGet, "/v1/functions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	
	functions_data, ok := response["functions"].([]interface{})
	if !ok {
		t.Fatalf("Expected functions array in response")
	}
	
	if len(functions_data) < 2 {
		t.Errorf("Expected at least 2 functions, got %d", len(functions_data))
	}
}

func TestV1Functions_GetFunction(t *testing.T) {
	router, apiContext := setupTestAPI(t)
	
	// Create a test function
	ctx := context.Background()
	params := database.CreateFunctionParams{
		ID:             "get-test-function",
		Name:           "get-function",
		Description:    sql.NullString{String: "Test description", Valid: true},
		CodePath:       "/tmp/get-func",
		Runtime:        "nodejs",
		Handler:        "index.handler",
		TimeoutSeconds: 30,
		MemoryMb:       128,
		EnvVars:        sql.NullString{String: `{"TEST": "value"}`, Valid: true},
	}
	
	function, err := apiContext.Querier.CreateFunction(ctx, params)
	if err != nil {
		t.Fatalf("Failed to create test function: %v", err)
	}
	
	tests := []struct {
		name           string
		functionID     string
		expectedStatus int
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name:           "valid function ID",
			functionID:     function.ID,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				if response["id"] != function.ID {
					t.Errorf("Expected ID %s, got %v", function.ID, response["id"])
				}
				if response["name"] != function.Name {
					t.Errorf("Expected name %s, got %v", function.Name, response["name"])
				}
			},
		},
		{
			name:           "non-existent function ID",
			functionID:     "non-existent",
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				if response["error"] == nil {
					t.Errorf("Expected error message for non-existent function")
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/functions/"+tt.functionID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}
			
			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestV1Functions_DeployFunction(t *testing.T) {
	router, apiContext := setupTestAPI(t)
	
	// Create a test function
	ctx := context.Background()
	params := database.CreateFunctionParams{
		ID:             "deploy-test-function",
		Name:           "deploy-function",
		CodePath:       "/tmp/deploy-func",
		Runtime:        "nodejs",
		Handler:        "index.handler",
		TimeoutSeconds: 30,
		MemoryMb:       128,
	}
	
	function, err := apiContext.Querier.CreateFunction(ctx, params)
	if err != nil {
		t.Fatalf("Failed to create test function: %v", err)
	}
	
	tests := []struct {
		name           string
		functionID     string
		expectedStatus int
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name:           "valid function deployment",
			functionID:     function.ID,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				if response["deployment_id"] == nil || response["deployment_id"] == "" {
					t.Errorf("Expected non-empty deployment_id")
				}
				if response["resource_id"] == nil || response["resource_id"] == "" {
					t.Errorf("Expected non-empty resource_id")
				}
				if response["image_tag"] == nil || response["image_tag"] == "" {
					t.Errorf("Expected non-empty image_tag")
				}
			},
		},
		{
			name:           "non-existent function",
			functionID:     "non-existent",
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				if response["error"] == nil {
					t.Errorf("Expected error message for non-existent function")
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/functions/"+tt.functionID+"/deploy", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}
			
			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestV1Functions_DeleteFunction(t *testing.T) {
	router, apiContext := setupTestAPI(t)
	
	// Create a test function
	ctx := context.Background()
	params := database.CreateFunctionParams{
		ID:             "delete-test-function",
		Name:           "delete-function",
		CodePath:       "/tmp/delete-func",
		Runtime:        "nodejs",
		Handler:        "index.handler",
		TimeoutSeconds: 30,
		MemoryMb:       128,
	}
	
	function, err := apiContext.Querier.CreateFunction(ctx, params)
	if err != nil {
		t.Fatalf("Failed to create test function: %v", err)
	}
	
	tests := []struct {
		name           string
		functionID     string
		expectedStatus int
	}{
		{
			name:           "valid function deletion",
			functionID:     function.ID,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "non-existent function",
			functionID:     "non-existent",
			expectedStatus: http.StatusNotFound,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/v1/functions/"+tt.functionID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// Benchmark tests
func BenchmarkV1Functions_CreateFunction(b *testing.B) {
	router, _ := setupTestAPI(&testing.T{})
	
	sampleCode := `{"index.js": "console.log('Hello World')"}`
	encodedCode := base64.StdEncoding.EncodeToString([]byte(sampleCode))
	
	requestBody := types.CreateFunctionRequest{
		Name:           "bench-function",
		Runtime:        "nodejs",
		Handler:        "index.handler",
		TimeoutSeconds: 30,
		MemoryMB:       128,
		Code:           encodedCode,
	}
	
	body, _ := json.Marshal(requestBody)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/functions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkV1Functions_ListFunctions(b *testing.B) {
	router, _ := setupTestAPI(&testing.T{})
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/v1/functions", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}