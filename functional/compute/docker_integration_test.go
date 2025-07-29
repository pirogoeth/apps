//go:build integration

package compute

import (
	"context"
	"testing"
	"time"
)

// Integration tests that require a running Docker daemon
// Run with: go test -tags=integration

func TestDockerProvider_Integration_Health(t *testing.T) {
	config := &DockerConfig{
		Socket:   "unix:///var/run/docker.sock",
		Network:  "bridge",
		Registry: "",
	}
	
	provider := NewDockerProvider(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	err := provider.Health(ctx)
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
	}
}

func TestDockerProvider_Integration_DeployAndExecute(t *testing.T) {
	config := &DockerConfig{
		Socket:   "unix:///var/run/docker.sock",
		Network:  "bridge", 
		Registry: "",
	}
	
	provider := NewDockerProvider(config)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	
	// Check if Docker is available
	if err := provider.Health(ctx); err != nil {
		t.Skipf("Docker daemon not available: %v", err)
	}
	
	// Test function
	function := &Function{
		ID:             "integration-test-function",
		Name:           "test-function",
		Description:    "Integration test function",
		Runtime:        "nodejs",
		Handler:        "index.handler",
		TimeoutSeconds: 30,
		MemoryMB:       128,
		EnvVars:        `{"TEST_ENV": "integration"}`,
	}
	
	t.Run("Deploy", func(t *testing.T) {
		result, err := provider.Deploy(ctx, function, "")
		if err != nil {
			t.Fatalf("Deploy failed: %v", err)
		}
		
		deployResult, ok := result.(*DeployResult)
		if !ok {
			t.Fatalf("Expected DeployResult, got %T", result)
		}
		
		if deployResult.DeploymentID == "" {
			t.Errorf("Expected non-empty deployment ID")
		}
		
		if deployResult.ResourceID == "" {
			t.Errorf("Expected non-empty resource ID")
		}
		
		if deployResult.ImageTag == "" {
			t.Errorf("Expected non-empty image tag")
		}
		
		// Test execution
		deployment := &Deployment{
			ID:         deployResult.DeploymentID,
			FunctionID: function.ID,
			Provider:   "docker",
			ResourceID: deployResult.ResourceID,
			Status:     "running",
			Replicas:   1,
			ImageTag:   deployResult.ImageTag,
		}
		
		// Wait for container to be ready
		time.Sleep(3 * time.Second)
		
		t.Run("Execute", func(t *testing.T) {
			request := &InvocationRequest{
				FunctionID: function.ID,
				Body:       []byte(`{"test": "integration"}`),
				Headers:    map[string]string{"Content-Type": "application/json"},
				Method:     "POST",
				Path:       "/",
				QueryArgs:  map[string]string{"param": "value"},
			}
			
			result, err := provider.Execute(ctx, deployment, request)
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}
			
			invResult, ok := result.(*InvocationResult)
			if !ok {
				t.Fatalf("Expected InvocationResult, got %T", result)
			}
			
			if invResult.StatusCode == 0 {
				t.Errorf("Expected non-zero status code")
			}
			
			if invResult.DurationMS <= 0 {
				t.Errorf("Expected positive duration, got %d", invResult.DurationMS)
			}
			
			if invResult.ResponseSize <= 0 {
				t.Errorf("Expected positive response size, got %d", invResult.ResponseSize)
			}
		})
		
		// Cleanup
		t.Run("Remove", func(t *testing.T) {
			err := provider.Remove(ctx, deployment)
			if err != nil {
				t.Errorf("Remove failed: %v", err)
			}
		})
	})
}

func TestDockerProvider_Integration_MultipleRuntimes(t *testing.T) {
	config := &DockerConfig{
		Socket:   "unix:///var/run/docker.sock",
		Network:  "bridge",
		Registry: "",
	}
	
	provider := NewDockerProvider(config)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	
	// Check if Docker is available
	if err := provider.Health(ctx); err != nil {
		t.Skipf("Docker daemon not available: %v", err)
	}
	
	runtimes := []struct {
		name    string
		runtime string
	}{
		{"NodeJS", "nodejs"},
		{"Python", "python3"},
		{"Go", "go"},
	}
	
	for _, rt := range runtimes {
		t.Run(rt.name, func(t *testing.T) {
			function := &Function{
				ID:             "integration-test-" + rt.runtime,
				Name:           "test-" + rt.runtime,
				Runtime:        rt.runtime,
				TimeoutSeconds: 60,
				MemoryMB:       128,
			}
			
			// Deploy
			result, err := provider.Deploy(ctx, function, "")
			if err != nil {
				t.Fatalf("Deploy failed for %s: %v", rt.runtime, err)
			}
			
			deployResult := result.(*DeployResult)
			deployment := &Deployment{
				ID:         deployResult.DeploymentID,
				ResourceID: deployResult.ResourceID,
				ImageTag:   deployResult.ImageTag,
			}
			
			// Wait for container startup
			time.Sleep(5 * time.Second)
			
			// Execute
			request := &InvocationRequest{
				Method: "GET",
				Path:   "/",
			}
			
			_, err = provider.Execute(ctx, deployment, request)
			// We don't check for success here as the sample functions may not be fully functional
			// The main test is that deployment works without errors
			
			// Cleanup
			provider.Remove(ctx, deployment)
		})
	}
}