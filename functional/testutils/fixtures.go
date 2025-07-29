package testutils

import (
	"context"
	"database/sql"
	"embed"
	"encoding/base64"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/pirogoeth/apps/functional/compute"
	"github.com/pirogoeth/apps/functional/database"
	"github.com/pirogoeth/apps/functional/types"
)

// Test database setup
func SetupTestDatabase(t *testing.T, migrationsFS embed.FS) *database.DbWrapper {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	ctx := context.Background()
	db, err := database.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	
	if err := db.RunMigrations(migrationsFS); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	
	return db
}

// Sample function fixtures
func CreateSampleFunction(t *testing.T, db *database.DbWrapper) *database.Function {
	ctx := context.Background()
	
	params := database.CreateFunctionParams{
		ID:             uuid.New().String(),
		Name:           "sample-function",
		Description:    sql.NullString{String: "Sample test function", Valid: true},
		CodePath:       "/tmp/sample-function",
		Runtime:        "nodejs",
		Handler:        "index.handler",
		TimeoutSeconds: 30,
		MemoryMb:       128,
		EnvVars:        sql.NullString{String: `{"TEST": "value"}`, Valid: true},
	}
	
	function, err := db.CreateFunction(ctx, params)
	if err != nil {
		t.Fatalf("Failed to create sample function: %v", err)
	}
	
	return &function
}

func CreateSampleFunctionWithParams(t *testing.T, db *database.DbWrapper, params database.CreateFunctionParams) *database.Function {
	ctx := context.Background()
	
	function, err := db.CreateFunction(ctx, params)
	if err != nil {
		t.Fatalf("Failed to create sample function: %v", err)
	}
	
	return &function
}

func CreateSampleDeployment(t *testing.T, db *database.DbWrapper, functionID string) *database.Deployment {
	ctx := context.Background()
	
	params := database.CreateDeploymentParams{
		ID:         uuid.New().String(),
		FunctionID: functionID,
		Provider:   "docker",
		ResourceID: "container-" + uuid.New().String()[:8],
		Status:     "running",
		Replicas:   1,
		ImageTag:   sql.NullString{String: "test-image:latest", Valid: true},
	}
	
	deployment, err := db.CreateDeployment(ctx, params)
	if err != nil {
		t.Fatalf("Failed to create sample deployment: %v", err)
	}
	
	return &deployment
}

func CreateSampleInvocation(t *testing.T, db *database.DbWrapper, functionID string, deploymentID string) *database.Invocation {
	ctx := context.Background()
	
	params := database.CreateInvocationParams{
		ID:           uuid.New().String(),
		FunctionID:   functionID,
		DeploymentID: sql.NullString{String: deploymentID, Valid: true},
		Status:       "completed",
	}
	
	invocation, err := db.CreateInvocation(ctx, params)
	if err != nil {
		t.Fatalf("Failed to create sample invocation: %v", err)
	}
	
	return &invocation
}

// Sample request fixtures
func CreateSampleFunctionRequest() types.CreateFunctionRequest {
	sampleCode := `{
		"index.js": "const express = require('express'); const app = express(); app.all('*', (req, res) => res.json({message: 'Hello World'})); app.listen(8080);",
		"package.json": "{\"name\": \"test-function\", \"main\": \"index.js\", \"dependencies\": {\"express\": \"^4.18.0\"}}"
	}`
	encodedCode := base64.StdEncoding.EncodeToString([]byte(sampleCode))
	
	return types.CreateFunctionRequest{
		Name:           "test-function",
		Description:    "Test function description",
		Runtime:        "nodejs",
		Handler:        "index.handler",
		TimeoutSeconds: 30,
		MemoryMB:       128,
		EnvVars:        map[string]string{"TEST": "value"},
		Code:           encodedCode,
	}
}

func CreateSamplePythonFunctionRequest() types.CreateFunctionRequest {
	sampleCode := `{
		"app.py": "from flask import Flask; app = Flask(__name__); @app.route('/', defaults={'path': ''}, methods=['GET', 'POST']); @app.route('/<path:path>', methods=['GET', 'POST']); def handler(path): return {'message': 'Hello from Python'}; app.run(host='0.0.0.0', port=8080)",
		"requirements.txt": "flask==2.3.0"
	}`
	encodedCode := base64.StdEncoding.EncodeToString([]byte(sampleCode))
	
	return types.CreateFunctionRequest{
		Name:           "python-function",
		Description:    "Python test function",
		Runtime:        "python3",
		Handler:        "app.handler",
		TimeoutSeconds: 60,
		MemoryMB:       256,
		EnvVars:        map[string]string{"PYTHON_ENV": "test"},
		Code:           encodedCode,
	}
}

func CreateSampleGoFunctionRequest() types.CreateFunctionRequest {
	sampleCode := `{
		"main.go": "package main; import (\"net/http\"; \"encoding/json\"); func main() { http.HandleFunc(\"/\", func(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(map[string]string{\"message\": \"Hello from Go\"}) }); http.ListenAndServe(\":8080\", nil) }",
		"go.mod": "module test-function\\ngo 1.21"
	}`
	encodedCode := base64.StdEncoding.EncodeToString([]byte(sampleCode))
	
	return types.CreateFunctionRequest{
		Name:           "go-function",
		Description:    "Go test function",
		Runtime:        "go",
		Handler:        "main",
		TimeoutSeconds: 45,
		MemoryMB:       128,
		EnvVars:        map[string]string{"GO_ENV": "test"},
		Code:           encodedCode,
	}
}

// Mock compute provider for testing
type MockComputeProvider struct {
	Name_           string
	DeployError     error
	DeployResult    *compute.DeployResult
	ExecuteError    error
	ExecuteResult   *compute.InvocationResult
	ScaleError      error
	RemoveError     error
	HealthError     error
	
	// Call tracking
	DeployCalls   int
	ExecuteCalls  int
	ScaleCalls    int
	RemoveCalls   int
	HealthCalls   int
}

func NewMockComputeProvider() *MockComputeProvider {
	return &MockComputeProvider{
		Name_: "mock",
	}
}

func (m *MockComputeProvider) Name() string {
	return m.Name_
}

func (m *MockComputeProvider) Deploy(ctx context.Context, fn interface{}, imageName string) (interface{}, error) {
	m.DeployCalls++
	
	if m.DeployError != nil {
		return nil, m.DeployError
	}
	
	if m.DeployResult != nil {
		return m.DeployResult, nil
	}
	
	return &compute.DeployResult{
		DeploymentID: "mock-deployment-" + uuid.New().String()[:8],
		ResourceID:   "mock-resource-" + uuid.New().String()[:8],
		ImageTag:     "mock-image:latest",
	}, nil
}

func (m *MockComputeProvider) Execute(ctx context.Context, deployment interface{}, req interface{}) (interface{}, error) {
	m.ExecuteCalls++
	
	if m.ExecuteError != nil {
		return nil, m.ExecuteError
	}
	
	if m.ExecuteResult != nil {
		return m.ExecuteResult, nil
	}
	
	return &compute.InvocationResult{
		StatusCode:   200,
		Body:         []byte(`{"message": "mock response", "timestamp": "2024-01-01T00:00:00Z"}`),
		Headers:      map[string]string{"Content-Type": "application/json", "X-Mock": "true"},
		DurationMS:   100,
		MemoryUsedMB: 64,
		ResponseSize: 65,
		Logs:         "Mock execution logs",
		Error:        "",
	}, nil
}

func (m *MockComputeProvider) Scale(ctx context.Context, deployment interface{}, replicas int) error {
	m.ScaleCalls++
	return m.ScaleError
}

func (m *MockComputeProvider) Remove(ctx context.Context, deployment interface{}) error {
	m.RemoveCalls++
	return m.RemoveError
}

func (m *MockComputeProvider) Health(ctx context.Context) error {
	m.HealthCalls++
	return m.HealthError
}

// Reset call counters
func (m *MockComputeProvider) ResetCalls() {
	m.DeployCalls = 0
	m.ExecuteCalls = 0
	m.ScaleCalls = 0
	m.RemoveCalls = 0
	m.HealthCalls = 0
}

// Test assertion helpers
func AssertStringEquals(t *testing.T, expected, actual, field string) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %s to be '%s', got '%s'", field, expected, actual)
	}
}

func AssertStringNotEmpty(t *testing.T, value, field string) {
	t.Helper()
	if value == "" {
		t.Errorf("Expected %s to be non-empty", field)
	}
}

func AssertIntEquals(t *testing.T, expected, actual int, field string) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %s to be %d, got %d", field, expected, actual)
	}
}

func AssertInt64Equals(t *testing.T, expected, actual int64, field string) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %s to be %d, got %d", field, expected, actual)
	}
}

func AssertNoError(t *testing.T, err error, operation string) {
	t.Helper()
	if err != nil {
		t.Fatalf("Unexpected error in %s: %v", operation, err)
	}
}

func AssertError(t *testing.T, err error, operation string) {
	t.Helper()
	if err == nil {
		t.Fatalf("Expected error in %s, got nil", operation)
	}
}

func AssertNil(t *testing.T, value interface{}, field string) {
	t.Helper()
	if value != nil {
		t.Errorf("Expected %s to be nil, got %v", field, value)
	}
}

func AssertNotNil(t *testing.T, value interface{}, field string) {
	t.Helper()
	if value == nil {
		t.Errorf("Expected %s to be non-nil", field)
	}
}

// Sample runtime configurations for testing different scenarios
var RuntimeConfigs = map[string]database.CreateFunctionParams{
	"nodejs": {
		ID:             "nodejs-test",
		Name:           "nodejs-function",
		Runtime:        "nodejs",
		Handler:        "index.handler",
		TimeoutSeconds: 30,
		MemoryMb:       128,
		CodePath:       "/tmp/nodejs-function",
	},
	"python": {
		ID:             "python-test",
		Name:           "python-function",
		Runtime:        "python3",
		Handler:        "app.handler",
		TimeoutSeconds: 60,
		MemoryMb:       256,
		CodePath:       "/tmp/python-function",
	},
	"go": {
		ID:             "go-test",
		Name:           "go-function",
		Runtime:        "go",
		Handler:        "main",
		TimeoutSeconds: 45,
		MemoryMb:       128,
		CodePath:       "/tmp/go-function",
	},
}