package compute

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

)

// Local type definitions to avoid import cycle
type DockerConfig struct {
	Socket   string `json:"socket"`
	Network  string `json:"network"`
	Registry string `json:"registry"`
}

type Function struct {
	ID             string
	Name           string
	Description    string
	CodePath       string
	Runtime        string
	Handler        string
	TimeoutSeconds int32
	MemoryMB       int32
	EnvVars        string
}

type DeployResult struct {
	DeploymentID string `json:"deployment_id"`
	ResourceID   string `json:"resource_id"`
	ImageTag     string `json:"image_tag"`
}

type Deployment struct {
	ID         string
	FunctionID string
	Provider   string
	ResourceID string
	Status     string
	Replicas   int32
	ImageTag   string
}

type InvocationRequest struct {
	FunctionID string
	Body       []byte
	Headers    map[string]string
	Method     string
	Path       string
	QueryArgs  map[string]string
}

type InvocationResult struct {
	StatusCode   int
	Body         []byte
	Headers      map[string]string
	DurationMS   int64
	MemoryUsedMB int32
	ResponseSize int64
	Logs         string
	Error        string
}

type DockerProvider struct {
	client *client.Client
	config *DockerConfig
}

func NewDockerProvider(config interface{}) *DockerProvider {
	dockerConfig, ok := config.(*DockerConfig)
	if !ok {
		logrus.Fatal("invalid docker config type")
	}

	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithHost(dockerConfig.Socket),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		logrus.WithError(err).Fatal("failed to create docker client")
	}

	return &DockerProvider{
		client: cli,
		config: dockerConfig,
	}
}

func (d *DockerProvider) Name() string {
	return "docker"
}

func (d *DockerProvider) Deploy(ctx context.Context, fn interface{}, imageName string) (interface{}, error) {
	function, ok := fn.(*Function)
	if !ok {
		return nil, fmt.Errorf("invalid function type")
	}

	logrus.
		WithField("function_id", function.ID).
		WithField("function_name", function.Name).
		Info("starting function deployment")

	// Build function image
	imageTag, err := d.buildFunctionImage(ctx, function)
	if err != nil {
		return nil, fmt.Errorf("failed to build function image: %w", err)
	}

	// Create container
	containerID, err := d.createContainer(ctx, function, imageTag)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := d.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	logrus.
		WithField("function_id", function.ID).
		WithField("container_id", containerID).
		WithField("image_tag", imageTag).
		Info("function deployed successfully")

	return &DeployResult{
		DeploymentID: uuid.New().String(),
		ResourceID:   containerID,
		ImageTag:     imageTag,
	}, nil
}

func (d *DockerProvider) Execute(ctx context.Context, deployment interface{}, req interface{}) (interface{}, error) {
	dep, ok := deployment.(*Deployment)
	if !ok {
		return nil, fmt.Errorf("invalid deployment type")
	}

	invReq, ok := req.(*InvocationRequest)
	if !ok {
		return nil, fmt.Errorf("invalid invocation request type")
	}

	// Get container port
	containerInfo, err := d.client.ContainerInspect(ctx, dep.ResourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	// Find the exposed port
	var port string
	for containerPort := range containerInfo.Config.ExposedPorts {
		port = containerPort.Port()
		break
	}

	if port == "" {
		return nil, fmt.Errorf("no exposed ports found in container")
	}

	// Get container IP or use localhost if port is bound
	var endpoint string
	if binding, ok := containerInfo.NetworkSettings.Ports[nat.Port(port+"/tcp")]; ok && len(binding) > 0 {
		endpoint = fmt.Sprintf("http://localhost:%s", binding[0].HostPort)
	} else {
		endpoint = fmt.Sprintf("http://%s:%s", containerInfo.NetworkSettings.IPAddress, port)
	}

	// Execute HTTP request to function
	start := time.Now()
	result, err := d.executeFunctionHTTP(ctx, endpoint, invReq)
	duration := time.Since(start)

	if err != nil {
		return &InvocationResult{
			StatusCode:   500,
			Body:         []byte(fmt.Sprintf("Function execution failed: %v", err)),
			DurationMS:   duration.Milliseconds(),
			Error:        err.Error(),
		}, nil
	}

	result.DurationMS = duration.Milliseconds()
	return result, nil
}

func (d *DockerProvider) Scale(ctx context.Context, deployment interface{}, replicas int) error {
	// For now, Docker provider doesn't support scaling (single container per function)
	// This would require implementing load balancing and multiple containers
	logrus.WithField("replicas", replicas).Warn("docker provider scaling not implemented")
	return nil
}

func (d *DockerProvider) Remove(ctx context.Context, deployment interface{}) error {
	dep, ok := deployment.(*Deployment)
	if !ok {
		return fmt.Errorf("invalid deployment type")
	}

	logrus.WithField("container_id", dep.ResourceID).Info("removing container")

	// Stop container
	timeoutSeconds := 10
	if err := d.client.ContainerStop(ctx, dep.ResourceID, container.StopOptions{Timeout: &timeoutSeconds}); err != nil {
		logrus.WithError(err).Warn("failed to stop container gracefully")
	}

	// Remove container
	if err := d.client.ContainerRemove(ctx, dep.ResourceID, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	return nil
}

func (d *DockerProvider) Health(ctx context.Context) error {
	// Check Docker daemon connectivity
	_, err := d.client.Ping(ctx)
	if err != nil {
		return fmt.Errorf("docker daemon not accessible: %w", err)
	}

	// TODO: Check if network exists and registry is accessible
	return nil
}

// Helper methods

func (d *DockerProvider) buildFunctionImage(ctx context.Context, function *Function) (string, error) {
	// Decode function code from base64
	// For now, assume the code is stored somewhere accessible
	// In a real implementation, we'd get the code from CodePath or decode from request

	// Create a simple Dockerfile for the function
	dockerfile := d.generateDockerfile(function)
	
	// Create build context
	buildContext, err := d.createBuildContext(dockerfile, function)
	if err != nil {
		return "", fmt.Errorf("failed to create build context: %w", err)
	}

	imageTag := fmt.Sprintf("function-%s:%s", function.Name, function.ID[:8])

	// Build image
	buildResp, err := d.client.ImageBuild(ctx, buildContext, types.ImageBuildOptions{
		Tags:       []string{imageTag},
		Dockerfile: "Dockerfile",
		Remove:     true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to build image: %w", err)
	}
	defer buildResp.Body.Close()

	// Read build output (for debugging)
	_, err = io.Copy(io.Discard, buildResp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read build response: %w", err)
	}

	return imageTag, nil
}

func (d *DockerProvider) createContainer(ctx context.Context, function *Function, imageTag string) (string, error) {
	// Parse environment variables
	envVars := []string{}
	if function.EnvVars != "" {
		var envMap map[string]string
		if err := json.Unmarshal([]byte(function.EnvVars), &envMap); err == nil {
			for k, v := range envMap {
				envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
			}
		}
	}

	// Create container config
	config := &container.Config{
		Image:        imageTag,
		Env:          envVars,
		ExposedPorts: nat.PortSet{"8080/tcp": struct{}{}},
	}

	// Create host config with port binding
	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"8080/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: "0", // Auto-assign port
				},
			},
		},
	}

	// Create container
	resp, err := d.client.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return resp.ID, nil
}

func (d *DockerProvider) executeFunctionHTTP(ctx context.Context, endpoint string, req *InvocationRequest) (*InvocationResult, error) {
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, endpoint+req.Path, bytes.NewReader(req.Body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Add query parameters
	if len(req.QueryArgs) > 0 {
		q := httpReq.URL.Query()
		for k, v := range req.QueryArgs {
			q.Add(k, v)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// Make request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Convert headers
	headers := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	return &InvocationResult{
		StatusCode:   resp.StatusCode,
		Body:         body,
		Headers:      headers,
		ResponseSize: int64(len(body)),
	}, nil
}

func (d *DockerProvider) generateDockerfile(function *Function) string {
	// Generate a basic Dockerfile based on runtime
	switch strings.ToLower(function.Runtime) {
	case "node", "nodejs", "node18", "node20":
		return `FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install --production
COPY . .
EXPOSE 8080
CMD ["node", "index.js"]`

	case "python", "python3", "python3.9", "python3.11":
		return `FROM python:3.11-alpine
WORKDIR /app
COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
EXPOSE 8080
CMD ["python", "app.py"]`

	case "go", "golang":
		return `FROM golang:1.21-alpine AS builder
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
CMD ["./main"]`

	default:
		// Generic container
		return `FROM alpine:latest
WORKDIR /app
COPY . .
EXPOSE 8080
CMD ["sh", "-c", "echo 'Runtime not supported: ` + function.Runtime + `' && exit 1"]`
	}
}

func (d *DockerProvider) createBuildContext(dockerfile string, function *Function) (io.Reader, error) {
	// Create a tar archive with Dockerfile and function code
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Add Dockerfile
	dockerfileHeader := &tar.Header{
		Name: "Dockerfile",
		Mode: 0644,
		Size: int64(len(dockerfile)),
	}
	if err := tw.WriteHeader(dockerfileHeader); err != nil {
		return nil, err
	}
	if _, err := tw.Write([]byte(dockerfile)); err != nil {
		return nil, err
	}

	// Add function code (simplified - in reality we'd extract from CodePath or decode from request)
	functionCode := d.generateSampleCode(function)
	
	for filename, content := range functionCode {
		header := &tar.Header{
			Name: filename,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(header); err != nil {
			return nil, err
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			return nil, err
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return &buf, nil
}

func (d *DockerProvider) generateSampleCode(function *Function) map[string]string {
	// Generate sample function code based on runtime
	// In a real implementation, this would come from the function's actual code
	
	switch strings.ToLower(function.Runtime) {
	case "node", "nodejs", "node18", "node20":
		return map[string]string{
			"package.json": `{
  "name": "` + function.Name + `",
  "version": "1.0.0",
  "main": "index.js",
  "dependencies": {
    "express": "^4.18.0"
  }
}`,
			"index.js": `const express = require('express');
const app = express();
app.use(express.json());

app.all('*', (req, res) => {
  res.json({
    message: 'Hello from ` + function.Name + `',
    method: req.method,
    path: req.path,
    headers: req.headers,
    body: req.body,
    query: req.query
  });
});

app.listen(8080, () => {
  console.log('Function ` + function.Name + ` listening on port 8080');
});`,
		}

	case "python", "python3", "python3.9", "python3.11":
		return map[string]string{
			"requirements.txt": "flask==2.3.0",
			"app.py": `from flask import Flask, request, jsonify
import json

app = Flask(__name__)

@app.route('/', defaults={'path': ''}, methods=['GET', 'POST', 'PUT', 'DELETE'])
@app.route('/<path:path>', methods=['GET', 'POST', 'PUT', 'DELETE'])
def handler(path):
    return jsonify({
        'message': 'Hello from ` + function.Name + `',
        'method': request.method,
        'path': '/' + path,
        'headers': dict(request.headers),
        'body': request.get_data(as_text=True),
        'args': dict(request.args)
    })

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8080)`,
		}

	default:
		return map[string]string{
			"README.md": "Runtime " + function.Runtime + " not supported yet",
		}
	}
}