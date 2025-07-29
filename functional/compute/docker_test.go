package compute

import (
	"context"
	"testing"
)

func TestDockerProvider_Name(t *testing.T) {
	config := &DockerConfig{
		Socket:   "unix:///var/run/docker.sock",
		Network:  "bridge",
		Registry: "localhost:5000",
	}
	
	provider := NewDockerProvider(config)
	
	if provider.Name() != "docker" {
		t.Errorf("Expected provider name to be 'docker', got %s", provider.Name())
	}
}

func TestDockerProvider_Health(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
	}{
		{
			name:        "healthy docker daemon",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test would require mocking the Docker client
			// For integration tests, we'd test against real Docker
			t.Skip("Requires Docker client mocking - see integration tests")
		})
	}
}

func TestDockerProvider_generateDockerfile(t *testing.T) {
	provider := &DockerProvider{}
	
	tests := []struct {
		name     string
		function *Function
		expected string
	}{
		{
			name: "nodejs runtime",
			function: &Function{
				ID:      "test-id",
				Name:    "test-function",
				Runtime: "nodejs",
			},
			expected: `FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install --production
COPY . .
EXPOSE 8080
CMD ["node", "index.js"]`,
		},
		{
			name: "python runtime",
			function: &Function{
				ID:      "test-id",
				Name:    "test-function",
				Runtime: "python3",
			},
			expected: `FROM python:3.11-alpine
WORKDIR /app
COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
EXPOSE 8080
CMD ["python", "app.py"]`,
		},
		{
			name: "go runtime",
			function: &Function{
				ID:      "test-id",
				Name:    "test-function",
				Runtime: "go",
			},
			expected: `FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]`,
		},
		{
			name: "unsupported runtime",
			function: &Function{
				ID:      "test-id",
				Name:    "test-function",
				Runtime: "unsupported",
			},
			expected: `FROM alpine:latest
WORKDIR /app
COPY . .
EXPOSE 8080
CMD ["sh", "-c", "echo 'Runtime not supported: unsupported' && exit 1"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.generateDockerfile(tt.function)
			if result != tt.expected {
				t.Errorf("generateDockerfile() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDockerProvider_generateSampleCode(t *testing.T) {
	provider := &DockerProvider{}
	
	tests := []struct {
		name     string
		function *Function
		checkKey string
	}{
		{
			name: "nodejs runtime generates package.json",
			function: &Function{
				ID:      "test-id",
				Name:    "test-function",
				Runtime: "nodejs",
			},
			checkKey: "package.json",
		},
		{
			name: "python runtime generates requirements.txt",
			function: &Function{
				ID:      "test-id",
				Name:    "test-function",
				Runtime: "python3",
			},
			checkKey: "requirements.txt",
		},
		{
			name: "unsupported runtime generates README",
			function: &Function{
				ID:      "test-id",
				Name:    "test-function",
				Runtime: "unsupported",
			},
			checkKey: "README.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.generateSampleCode(tt.function)
			if _, exists := result[tt.checkKey]; !exists {
				t.Errorf("generateSampleCode() missing expected key %s", tt.checkKey)
			}
		})
	}
}

func TestNewDockerProvider(t *testing.T) {
	t.Skip("Requires Docker daemon - see integration tests")
}

func TestNewDockerProvider_InvalidConfig(t *testing.T) {
	t.Skip("Causes fatal exit - requires different test approach")
}

// Mock types for testing without Docker dependency
type MockDockerProvider struct {
	name   string
	health error
}

func (m *MockDockerProvider) Name() string {
	return m.name
}

func (m *MockDockerProvider) Deploy(ctx context.Context, fn interface{}, imageName string) (interface{}, error) {
	return &DeployResult{
		DeploymentID: "mock-deployment-id",
		ResourceID:   "mock-container-id",
		ImageTag:     "mock-image:latest",
	}, nil
}

func (m *MockDockerProvider) Execute(ctx context.Context, deployment interface{}, req interface{}) (interface{}, error) {
	return &InvocationResult{
		StatusCode:   200,
		Body:         []byte(`{"message": "mock response"}`),
		Headers:      map[string]string{"Content-Type": "application/json"},
		DurationMS:   100,
		MemoryUsedMB: 64,
		ResponseSize: 25,
	}, nil
}

func (m *MockDockerProvider) Scale(ctx context.Context, deployment interface{}, replicas int) error {
	return nil
}

func (m *MockDockerProvider) Remove(ctx context.Context, deployment interface{}) error {
	return nil
}

func (m *MockDockerProvider) Health(ctx context.Context) error {
	return m.health
}

func TestMockDockerProvider(t *testing.T) {
	mock := &MockDockerProvider{
		name:   "mock-docker",
		health: nil,
	}
	
	ctx := context.Background()
	
	t.Run("Name", func(t *testing.T) {
		if mock.Name() != "mock-docker" {
			t.Errorf("Expected name 'mock-docker', got %s", mock.Name())
		}
	})
	
	t.Run("Health", func(t *testing.T) {
		if err := mock.Health(ctx); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
	
	t.Run("Deploy", func(t *testing.T) {
		function := &Function{
			ID:   "test-id",
			Name: "test-function",
		}
		
		result, err := mock.Deploy(ctx, function, "test-image")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		
		deployResult, ok := result.(*DeployResult)
		if !ok {
			t.Errorf("Expected DeployResult, got %T", result)
		}
		
		if deployResult.DeploymentID == "" {
			t.Errorf("Expected non-empty deployment ID")
		}
	})
	
	t.Run("Execute", func(t *testing.T) {
		deployment := &Deployment{
			ID:         "test-deployment",
			ResourceID: "test-container",
		}
		
		request := &InvocationRequest{
			FunctionID: "test-function",
			Body:       []byte(`{"test": "data"}`),
			Method:     "POST",
			Path:       "/",
		}
		
		result, err := mock.Execute(ctx, deployment, request)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		
		invResult, ok := result.(*InvocationResult)
		if !ok {
			t.Errorf("Expected InvocationResult, got %T", result)
		}
		
		if invResult.StatusCode != 200 {
			t.Errorf("Expected status code 200, got %d", invResult.StatusCode)
		}
	})
}

// Benchmark tests
func BenchmarkDockerProvider_generateDockerfile(b *testing.B) {
	provider := &DockerProvider{}
	function := &Function{
		ID:      "bench-id",
		Name:    "bench-function",
		Runtime: "nodejs",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider.generateDockerfile(function)
	}
}

func BenchmarkDockerProvider_generateSampleCode(b *testing.B) {
	provider := &DockerProvider{}
	function := &Function{
		ID:      "bench-id",
		Name:    "bench-function",
		Runtime: "nodejs",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider.generateSampleCode(function)
	}
}