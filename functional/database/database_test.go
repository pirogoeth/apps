package database

import (
	"context"
	"database/sql"
	"embed"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var testMigrations embed.FS

func setupTestDB(t *testing.T) *DbWrapper {
	// Create temporary database file
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	ctx := context.Background()
	db, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	
	// Run migrations
	if err := db.RunMigrations(testMigrations); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	
	return db
}

func TestDbWrapper_Open(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	ctx := context.Background()
	db, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	
	// Verify database connection
	if err := db.DB().Ping(); err != nil {
		t.Errorf("Database ping failed: %v", err)
	}
}

func TestDbWrapper_Close(t *testing.T) {
	db := setupTestDB(t)
	
	err := db.Close()
	if err != nil {
		t.Errorf("Failed to close database: %v", err)
	}
	
	// Verify database is closed
	if err := db.DB().Ping(); err == nil {
		t.Errorf("Expected database to be closed, but ping succeeded")
	}
}

func TestDbWrapper_RunMigrations(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	ctx := context.Background()
	db, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	
	// Run migrations
	err = db.RunMigrations(testMigrations)
	if err != nil {
		t.Errorf("Failed to run migrations: %v", err)
	}
	
	// Verify tables exist
	tables := []string{"functions", "deployments", "invocations"}
	for _, table := range tables {
		var exists bool
		query := "SELECT EXISTS(SELECT name FROM sqlite_master WHERE type='table' AND name=?)"
		err := db.DB().QueryRow(query, table).Scan(&exists)
		if err != nil {
			t.Errorf("Failed to check if table %s exists: %v", table, err)
		}
		if !exists {
			t.Errorf("Table %s does not exist after migrations", table)
		}
	}
}

func TestQueries_CreateFunction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	ctx := context.Background()
	
	params := CreateFunctionParams{
		ID:             "test-function-id",
		Name:           "test-function",
		Description:    sql.NullString{String: "Test function description", Valid: true},
		CodePath:       "/tmp/test-function",
		Runtime:        "nodejs",
		Handler:        "index.handler",
		TimeoutSeconds: 30,
		MemoryMb:       128,
		EnvVars:        sql.NullString{String: `{"TEST": "value"}`, Valid: true},
	}
	
	function, err := db.CreateFunction(ctx, params)
	if err != nil {
		t.Fatalf("Failed to create function: %v", err)
	}
	
	// Verify function was created correctly
	if function.ID != params.ID {
		t.Errorf("Expected ID %s, got %s", params.ID, function.ID)
	}
	
	if function.Name != params.Name {
		t.Errorf("Expected name %s, got %s", params.Name, function.Name)
	}
	
	if function.Runtime != params.Runtime {
		t.Errorf("Expected runtime %s, got %s", params.Runtime, function.Runtime)
	}
	
	if function.TimeoutSeconds != params.TimeoutSeconds {
		t.Errorf("Expected timeout %d, got %d", params.TimeoutSeconds, function.TimeoutSeconds)
	}
	
	// Verify timestamps are set
	if !function.CreatedAt.Valid || function.CreatedAt.Time.IsZero() {
		t.Errorf("Expected valid created_at timestamp")
	}
}

func TestQueries_GetFunction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	ctx := context.Background()
	
	// Create a function first
	params := CreateFunctionParams{
		ID:             "get-test-function-id",
		Name:           "get-test-function",
		Description:    sql.NullString{String: "Get test function", Valid: true},
		CodePath:       "/tmp/get-test-function",
		Runtime:        "python3",
		Handler:        "app.handler",
		TimeoutSeconds: 60,
		MemoryMb:       256,
		EnvVars:        sql.NullString{String: `{"ENV": "test"}`, Valid: true},
	}
	
	created, err := db.CreateFunction(ctx, params)
	if err != nil {
		t.Fatalf("Failed to create function: %v", err)
	}
	
	// Get the function
	retrieved, err := db.GetFunction(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to get function: %v", err)
	}
	
	// Verify function matches
	if retrieved.ID != created.ID {
		t.Errorf("Expected ID %s, got %s", created.ID, retrieved.ID)
	}
	
	if retrieved.Name != created.Name {
		t.Errorf("Expected name %s, got %s", created.Name, retrieved.Name)
	}
	
	if retrieved.Runtime != created.Runtime {
		t.Errorf("Expected runtime %s, got %s", created.Runtime, retrieved.Runtime)
	}
}

func TestQueries_ListFunctions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	ctx := context.Background()
	
	// Create multiple functions
	functions := []CreateFunctionParams{
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
		_, err := db.CreateFunction(ctx, params)
		if err != nil {
			t.Fatalf("Failed to create function %s: %v", params.ID, err)
		}
	}
	
	// List functions
	listed, err := db.ListFunctions(ctx)
	if err != nil {
		t.Fatalf("Failed to list functions: %v", err)
	}
	
	if len(listed) < 2 {
		t.Errorf("Expected at least 2 functions, got %d", len(listed))
	}
	
	// Verify functions are in the list
	foundIDs := make(map[string]bool)
	for _, f := range listed {
		foundIDs[f.ID] = true
	}
	
	for _, params := range functions {
		if !foundIDs[params.ID] {
			t.Errorf("Function %s not found in list", params.ID)
		}
	}
}

func TestQueries_CreateDeployment(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	ctx := context.Background()
	
	// Create a function first
	functionParams := CreateFunctionParams{
		ID:             "deploy-test-function",
		Name:           "deploy-test",
		CodePath:       "/tmp/deploy-test",
		Runtime:        "nodejs",
		Handler:        "index.handler",
		TimeoutSeconds: 30,
		MemoryMb:       128,
	}
	
	function, err := db.CreateFunction(ctx, functionParams)
	if err != nil {
		t.Fatalf("Failed to create function: %v", err)
	}
	
	// Create deployment
	deployParams := CreateDeploymentParams{
		ID:         "test-deployment-id",
		FunctionID: function.ID,
		Provider:   "docker",
		ResourceID: "container-123",
		Status:     "running",
		Replicas:   1,
		ImageTag:   sql.NullString{String: "test-image:latest", Valid: true},
	}
	
	deployment, err := db.CreateDeployment(ctx, deployParams)
	if err != nil {
		t.Fatalf("Failed to create deployment: %v", err)
	}
	
	// Verify deployment
	if deployment.ID != deployParams.ID {
		t.Errorf("Expected ID %s, got %s", deployParams.ID, deployment.ID)
	}
	
	if deployment.FunctionID != deployParams.FunctionID {
		t.Errorf("Expected function ID %s, got %s", deployParams.FunctionID, deployment.FunctionID)
	}
	
	if deployment.Provider != deployParams.Provider {
		t.Errorf("Expected provider %s, got %s", deployParams.Provider, deployment.Provider)
	}
	
	if deployment.Status != deployParams.Status {
		t.Errorf("Expected status %s, got %s", deployParams.Status, deployment.Status)
	}
}

func TestQueries_CreateInvocation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	ctx := context.Background()
	
	// Create function and deployment first
	functionParams := CreateFunctionParams{
		ID:             "invoke-test-function",
		Name:           "invoke-test",
		CodePath:       "/tmp/invoke-test",
		Runtime:        "nodejs",
		Handler:        "index.handler",
		TimeoutSeconds: 30,
		MemoryMb:       128,
	}
	
	function, err := db.CreateFunction(ctx, functionParams)
	if err != nil {
		t.Fatalf("Failed to create function: %v", err)
	}
	
	deployParams := CreateDeploymentParams{
		ID:         "invoke-test-deployment",
		FunctionID: function.ID,
		Provider:   "docker",
		ResourceID: "container-456",
		Status:     "running",
		Replicas:   1,
	}
	
	deployment, err := db.CreateDeployment(ctx, deployParams)
	if err != nil {
		t.Fatalf("Failed to create deployment: %v", err)
	}
	
	// Create invocation
	invocationParams := CreateInvocationParams{
		ID:           "test-invocation-id",
		FunctionID:   function.ID,
		DeploymentID: sql.NullString{String: deployment.ID, Valid: true},
		Status:       "pending",
	}
	
	invocation, err := db.CreateInvocation(ctx, invocationParams)
	if err != nil {
		t.Fatalf("Failed to create invocation: %v", err)
	}
	
	// Verify invocation
	if invocation.ID != invocationParams.ID {
		t.Errorf("Expected ID %s, got %s", invocationParams.ID, invocation.ID)
	}
	
	if invocation.FunctionID != invocationParams.FunctionID {
		t.Errorf("Expected function ID %s, got %s", invocationParams.FunctionID, invocation.FunctionID)
	}
	
	if invocation.Status != invocationParams.Status {
		t.Errorf("Expected status %s, got %s", invocationParams.Status, invocation.Status)
	}
}

// Test database constraints and edge cases
func TestDatabase_Constraints(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	ctx := context.Background()
	
	t.Run("Duplicate function ID", func(t *testing.T) {
		params := CreateFunctionParams{
			ID:             "duplicate-test",
			Name:           "test-function",
			CodePath:       "/tmp/test",
			Runtime:        "nodejs",
			Handler:        "index.handler",
			TimeoutSeconds: 30,
			MemoryMb:       128,
		}
		
		// Create first function
		_, err := db.CreateFunction(ctx, params)
		if err != nil {
			t.Fatalf("Failed to create first function: %v", err)
		}
		
		// Try to create duplicate
		_, err = db.CreateFunction(ctx, params)
		if err == nil {
			t.Errorf("Expected error when creating duplicate function ID, got nil")
		}
	})
	
	t.Run("Foreign key constraint", func(t *testing.T) {
		// Try to create deployment without valid function
		deployParams := CreateDeploymentParams{
			ID:         "fk-test-deployment",
			FunctionID: "non-existent-function",
			Provider:   "docker",
			ResourceID: "container-789",
			Status:     "running",
			Replicas:   1,
		}
		
		_, err := db.CreateDeployment(ctx, deployParams)
		if err == nil {
			t.Errorf("Expected foreign key constraint error, got nil")
		}
	})
}

// Benchmark tests
func BenchmarkCreateFunction(b *testing.B) {
	tempDir := b.TempDir()
	dbPath := filepath.Join(tempDir, "bench.db")
	
	ctx := context.Background()
	db, err := Open(ctx, dbPath)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	
	if err := db.RunMigrations(testMigrations); err != nil {
		b.Fatalf("Failed to run migrations: %v", err)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		params := CreateFunctionParams{
			ID:             "bench-function-" + string(rune(i)),
			Name:           "bench-function",
			CodePath:       "/tmp/bench",
			Runtime:        "nodejs",
			Handler:        "index.handler",
			TimeoutSeconds: 30,
			MemoryMb:       128,
		}
		
		_, err := db.CreateFunction(ctx, params)
		if err != nil {
			b.Fatalf("Failed to create function: %v", err)
		}
	}
}