package proxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// WrapperService handles communication between proxy and function code
type WrapperService struct {
	Runtime string
	Handler string
}

// FunctionWrapperRequest represents a request from the proxy
type FunctionWrapperRequest struct {
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Headers   map[string]string `json:"headers"`
	Query     map[string]string `json:"query"`
	Body      string            `json:"body"`
	RequestID string            `json:"request_id"`
}

// FunctionWrapperResponse represents a response to the proxy
type FunctionWrapperResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Error      string            `json:"error,omitempty"`
}

// StartWrapper starts the wrapper service for function containers
func StartWrapper(runtime, handler string) error {
	wrapper := &WrapperService{
		Runtime: runtime,
		Handler: handler,
	}
	
	logrus.WithFields(logrus.Fields{
		"runtime": runtime,
		"handler": handler,
	}).Info("Starting function wrapper")
	
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		
		// Parse request
		var request FunctionWrapperRequest
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			logrus.WithError(err).Error("Failed to parse request")
			continue
		}
		
		// Execute function
		response := wrapper.executeFunction(&request)
		
		// Send response
		if err := encoder.Encode(response); err != nil {
			logrus.WithError(err).Error("Failed to encode response")
		}
	}
	
	if err := scanner.Err(); err != nil {
		logrus.WithError(err).Error("Scanner error")
		return err
	}
	
	return nil
}

// executeFunction executes the function based on runtime
func (ws *WrapperService) executeFunction(request *FunctionWrapperRequest) *FunctionWrapperResponse {
	start := time.Now()
	
	logrus.WithFields(logrus.Fields{
		"request_id": request.RequestID,
		"method":     request.Method,
		"path":       request.Path,
	}).Debug("Executing function")
	
	response := &FunctionWrapperResponse{
		Headers: make(map[string]string),
	}
	
	// Execute based on runtime
	var cmd *exec.Cmd
	var err error
	
	switch strings.ToLower(ws.Runtime) {
	case "nodejs", "node", "node18", "node20":
		cmd, err = ws.executeNodeJS(request)
	case "python", "python3", "python3.9", "python3.11":
		cmd, err = ws.executePython(request)
	case "go", "golang":
		cmd, err = ws.executeGo(request)
	default:
		response.StatusCode = 500
		response.Error = fmt.Sprintf("Unsupported runtime: %s", ws.Runtime)
		return response
	}
	
	if err != nil {
		response.StatusCode = 500
		response.Error = fmt.Sprintf("Failed to setup execution: %v", err)
		return response
	}
	
	// Execute command
	output, err := cmd.Output()
	if err != nil {
		response.StatusCode = 500
		response.Error = fmt.Sprintf("Function execution failed: %v", err)
		logrus.WithError(err).WithField("request_id", request.RequestID).Error("Function execution failed")
	} else {
		response.StatusCode = 200
		response.Body = string(output)
		response.Headers["Content-Type"] = "application/json"
	}
	
	duration := time.Since(start)
	response.Headers["X-Execution-Time"] = fmt.Sprintf("%dms", duration.Milliseconds())
	response.Headers["X-Request-ID"] = request.RequestID
	
	logrus.WithFields(logrus.Fields{
		"request_id": request.RequestID,
		"duration":   duration,
		"status":     response.StatusCode,
	}).Debug("Function execution completed")
	
	return response
}

// executeNodeJS executes Node.js functions
func (ws *WrapperService) executeNodeJS(request *FunctionWrapperRequest) (*exec.Cmd, error) {
	// Create a simple Node.js script that processes the request
	script := fmt.Sprintf(`
const request = %s;
try {
	const handler = require('./%s');
	const result = handler(request);
	console.log(JSON.stringify({message: 'Hello from Node.js', request: request, result: result}));
} catch (error) {
	console.error(JSON.stringify({error: error.message}));
	process.exit(1);
}
`, ws.jsonString(request), ws.Handler)
	
	cmd := exec.Command("node", "-e", script)
	cmd.Dir = "/app"
	return cmd, nil
}

// executePython executes Python functions
func (ws *WrapperService) executePython(request *FunctionWrapperRequest) (*exec.Cmd, error) {
	// Create a simple Python script that processes the request
	script := fmt.Sprintf(`
import json
import sys
import importlib.util

request = %s
try:
    # Load the handler module
    spec = importlib.util.spec_from_file_location("handler", "/app/%s")
    handler_module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(handler_module)
    
    # Call the handler
    result = handler_module.handler(request)
    print(json.dumps({"message": "Hello from Python", "request": request, "result": result}))
except Exception as error:
    print(json.dumps({"error": str(error)}), file=sys.stderr)
    sys.exit(1)
`, ws.jsonString(request), ws.Handler)
	
	cmd := exec.Command("python3", "-c", script)
	cmd.Dir = "/app"
	return cmd, nil
}

// executeGo executes Go functions
func (ws *WrapperService) executeGo(request *FunctionWrapperRequest) (*exec.Cmd, error) {
	// For Go, we'll assume the binary is already built and available
	// Pass the request as JSON via environment variable
	requestJSON := ws.jsonString(request)
	
	cmd := exec.Command("./main")
	cmd.Dir = "/app"
	cmd.Env = append(os.Environ(), "FUNCTION_REQUEST="+requestJSON)
	return cmd, nil
}

// Helper function to convert request to JSON string
func (ws *WrapperService) jsonString(request *FunctionWrapperRequest) string {
	data, _ := json.Marshal(request)
	return string(data)
}

// This can be used as a standalone binary for containers
func main() {
	runtime := os.Getenv("FUNCTION_RUNTIME")
	handler := os.Getenv("FUNCTION_HANDLER")
	
	if runtime == "" {
		runtime = "nodejs"
	}
	if handler == "" {
		handler = "index.js"
	}
	
	if err := StartWrapper(runtime, handler); err != nil {
		logrus.WithError(err).Fatal("Wrapper failed")
	}
}