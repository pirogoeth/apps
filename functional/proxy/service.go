package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/pirogoeth/apps/functional/database"
	"github.com/pirogoeth/apps/functional/types"
)

// ProxyService handles function invocations via container pools
type ProxyService struct {
	config        *types.Config
	db            *database.DbWrapper
	containerPool *ContainerPool
	traefik       *TraefikClient
	
	// In-flight request tracking
	inFlightMutex sync.RWMutex
	inFlight      map[string]*InFlightRequest
}

// InFlightRequest tracks a request being processed
type InFlightRequest struct {
	ID         string
	FunctionID string
	StartTime  time.Time
	Container  *PooledContainer
}

// FunctionRequest represents the serialized request sent to functions
type FunctionRequest struct {
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Headers   map[string]string `json:"headers"`
	Query     map[string]string `json:"query"`
	Body      string            `json:"body"` // Base64 encoded for binary safety
	RequestID string            `json:"request_id"`
}

// FunctionResponse represents the response from functions
type FunctionResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"` // Base64 encoded
	Error      string            `json:"error,omitempty"`
}

// NewProxyService creates a new proxy service
func NewProxyService(config *types.Config, db *database.DbWrapper) *ProxyService {
	containerPool := NewContainerPool(config)
	traefik := NewTraefikClient(config.Proxy.TraefikAPIURL)
	
	return &ProxyService{
		config:        config,
		db:            db,
		containerPool: containerPool,
		traefik:       traefik,
		inFlight:      make(map[string]*InFlightRequest),
	}
}

// Start starts the proxy service
func (ps *ProxyService) Start(ctx context.Context) error {
	// Start container pool cleanup routine
	go ps.containerPool.StartCleanup(ctx)
	
	// Setup HTTP handler
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	
	// Function invocation endpoint - this receives requests from Traefik
	router.Any("/invoke/:functionId/*path", ps.handleInvocation)
	
	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})
	
	// Metrics endpoint
	router.GET("/metrics", ps.handleMetrics)
	
	server := &http.Server{
		Addr:    ps.config.Proxy.ListenAddress,
		Handler: router,
	}
	
	logrus.WithField("address", ps.config.Proxy.ListenAddress).Info("Starting proxy service")
	
	go func() {
		<-ctx.Done()
		logrus.Info("Shutting down proxy service")
		server.Shutdown(context.Background())
	}()
	
	return server.ListenAndServe()
}

// handleInvocation processes function invocations
func (ps *ProxyService) handleInvocation(c *gin.Context) {
	functionID := c.Param("functionId")
	path := c.Param("path")
	
	logrus.WithFields(logrus.Fields{
		"function_id": functionID,
		"path":        path,
		"method":      c.Request.Method,
	}).Info("Handling function invocation")
	
	// Get function from database
	ctx := c.Request.Context()
	function, err := ps.db.GetFunction(ctx, functionID)
	if err != nil {
		logrus.WithError(err).WithField("function_id", functionID).Error("Function not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "Function not found"})
		return
	}
	
	// Create in-flight request tracking
	requestID := ps.generateRequestID()
	inFlightReq := &InFlightRequest{
		ID:         requestID,
		FunctionID: functionID,
		StartTime:  time.Now(),
	}
	
	ps.trackInFlightRequest(inFlightReq)
	defer ps.removeInFlightRequest(requestID)
	
	// Get or create container from pool
	container, err := ps.containerPool.GetContainer(ctx, &function)
	if err != nil {
		logrus.WithError(err).WithField("function_id", functionID).Error("Failed to get container")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get container"})
		return
	}
	
	inFlightReq.Container = container
	defer ps.containerPool.ReturnContainer(container)
	
	// Serialize request
	funcReq, err := ps.serializeRequest(c, requestID, path)
	if err != nil {
		logrus.WithError(err).Error("Failed to serialize request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to serialize request"})
		return
	}
	
	// Execute function
	response, err := ps.executeFunction(ctx, container, funcReq)
	if err != nil {
		logrus.WithError(err).WithField("function_id", functionID).Error("Function execution failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Function execution failed"})
		return
	}
	
	// Return response
	ps.returnResponse(c, response)
}

// serializeRequest converts HTTP request to FunctionRequest
func (ps *ProxyService) serializeRequest(c *gin.Context, requestID, path string) (*FunctionRequest, error) {
	// Read body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	
	// Convert headers
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	
	// Convert query parameters
	query := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			query[key] = values[0]
		}
	}
	
	return &FunctionRequest{
		Method:    c.Request.Method,
		Path:      path,
		Headers:   headers,
		Query:     query,
		Body:      string(body), // For now, assume text body
		RequestID: requestID,
	}, nil
}

// executeFunction sends request to container via pipes and gets response
func (ps *ProxyService) executeFunction(ctx context.Context, container *PooledContainer, req *FunctionRequest) (*FunctionResponse, error) {
	// Serialize request to JSON
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Send to container via stdin pipe
	if _, err := container.Stdin.Write(append(reqJSON, '\n')); err != nil {
		return nil, fmt.Errorf("failed to write to container stdin: %w", err)
	}
	
	// Read response from stdout pipe
	response := &FunctionResponse{}
	decoder := json.NewDecoder(container.Stdout)
	if err := decoder.Decode(response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return response, nil
}

// returnResponse converts FunctionResponse back to HTTP response
func (ps *ProxyService) returnResponse(c *gin.Context, response *FunctionResponse) {
	// Set headers
	for key, value := range response.Headers {
		c.Header(key, value)
	}
	
	// Return response
	c.Data(response.StatusCode, c.GetHeader("Content-Type"), []byte(response.Body))
}

// RegisterFunction registers a function with Traefik
func (ps *ProxyService) RegisterFunction(ctx context.Context, functionID string) error {
	// Create Traefik route for this function
	route := &TraefikRoute{
		Rule:    fmt.Sprintf("PathPrefix(`/functions/%s`)", functionID),
		Service: "proxy",
	}
	
	return ps.traefik.RegisterRoute(ctx, functionID, route)
}

// UnregisterFunction removes a function from Traefik
func (ps *ProxyService) UnregisterFunction(ctx context.Context, functionID string) error {
	return ps.traefik.UnregisterRoute(ctx, functionID)
}

// Helper methods

func (ps *ProxyService) generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

func (ps *ProxyService) trackInFlightRequest(req *InFlightRequest) {
	ps.inFlightMutex.Lock()
	defer ps.inFlightMutex.Unlock()
	ps.inFlight[req.ID] = req
}

func (ps *ProxyService) removeInFlightRequest(requestID string) {
	ps.inFlightMutex.Lock()
	defer ps.inFlightMutex.Unlock()
	delete(ps.inFlight, requestID)
}

// handleMetrics returns proxy metrics
func (ps *ProxyService) handleMetrics(c *gin.Context) {
	ps.inFlightMutex.RLock()
	defer ps.inFlightMutex.RUnlock()
	
	metrics := gin.H{
		"in_flight_requests": len(ps.inFlight),
		"container_pools":    ps.containerPool.GetPoolStats(),
		"timestamp":         time.Now(),
	}
	
	c.JSON(http.StatusOK, metrics)
}