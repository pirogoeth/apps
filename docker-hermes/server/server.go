package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/pirogoeth/apps/docker-hermes/redis"
	"github.com/pirogoeth/apps/docker-hermes/types"
	api "github.com/pirogoeth/apps/pkg/apitools"
	"github.com/pirogoeth/apps/pkg/system"
)

// Server represents the docker-hermes server
type Server struct {
	config      *types.Config
	redisClient types.StorageClient
	router      *gin.Engine
}

// NewServer creates a new server instance
func NewServer(ctx context.Context, config *types.Config) (*Server, error) {
	redisClient, err := redis.NewClient(config.Redis.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis client: %w", err)
	}

	router, err := system.DefaultRouterWithTracing(ctx, config.Tracing)
	if err != nil {
		return nil, fmt.Errorf("failed to create router: %w", err)
	}

	server := &Server{
		config:      config,
		redisClient: redisClient,
		router:      router,
	}

	// Register routes
	if err := server.registerRoutes(); err != nil {
		redisClient.Close()
		return nil, fmt.Errorf("failed to register routes: %w", err)
	}

	return server, nil
}

// registerRoutes registers all HTTP routes
func (s *Server) registerRoutes() error {
	// Create API context
	apiContext := &types.ApiContext{
		Config:      s.config,
		RedisClient: s.redisClient,
	}

	// Register API routes
	if err := RegisterRoutes(s.router, apiContext); err != nil {
		return fmt.Errorf("failed to register API routes: %w", err)
	}

	// Register Prometheus SD endpoint
	if err := s.registerPrometheusSD(apiContext); err != nil {
		return fmt.Errorf("failed to register Prometheus SD endpoint: %w", err)
	}

	// Health check endpoint
	s.router.GET("/health", s.healthCheck)

	return nil
}

// registerPrometheusSD registers the Prometheus HTTP SD endpoint
func (s *Server) registerPrometheusSD(_ *types.ApiContext) error {
	s.router.GET(s.config.Server.PrometheusSDPath, func(c *gin.Context) {
		// Use server hostname or empty (defaults to container hostname)
		agentHost := s.config.Agent.Hostname
		if agentHost == "" {
			agentHost = "unknown"
		}

		// Pass labels config to Redis operations
		targets, err := s.redisClient.GetPrometheusTargets(
			c.Request.Context(),
			s.config.Server.TargetResolutionStrategy,
			agentHost,
			s.config.Server.TargetResolutionHost,
			&s.config.Labels,
		)
		if err != nil {
			logrus.Errorf("Failed to get Prometheus targets: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get targets"})
			return
		}

		c.JSON(http.StatusOK, targets)
	})

	logrus.Infof("Registered Prometheus SD endpoint at %s", s.config.Server.PrometheusSDPath)
	return nil
}

// healthCheck provides a health check endpoint
func (s *Server) healthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Check Redis connection
	if err := s.redisClient.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "Redis connection failed",
		})
		return
	}

	api.Ok(c, &api.Body{
		"status": "healthy",
		"time":   time.Now().UTC(),
	})
}

// Run starts the HTTP server
func (s *Server) Run(ctx context.Context) error {
	listenAddr := s.config.HTTP.ListenAddress

	logrus.Infof("Starting server on %s", listenAddr)
	go s.router.Run(listenAddr)

	// Wait for context cancellation
	<-ctx.Done()
	logrus.Info("Server stopped")

	return nil
}

// Close closes the server and cleans up resources
func (s *Server) Close() error {
	return s.redisClient.Close()
}
