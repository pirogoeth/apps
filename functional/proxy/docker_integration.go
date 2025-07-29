package proxy

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/pirogoeth/apps/functional/compute"
	"github.com/pirogoeth/apps/functional/types"
)

// ProxyIntegration interface for Traefik integration
type ProxyIntegration interface {
	RegisterFunction(ctx context.Context, functionID string) error
	UnregisterFunction(ctx context.Context, functionID string) error
}

// ProxyDockerProvider wraps the Docker provider with proxy integration
type ProxyDockerProvider struct {
	*compute.DockerProvider
	proxyIntegration ProxyIntegration
	config           *types.Config
}

// NewProxyDockerProvider creates a new proxy-integrated Docker provider
func NewProxyDockerProvider(config *types.Config, proxyIntegration ProxyIntegration) *ProxyDockerProvider {
	dockerConfig := &compute.DockerConfig{
		Socket:   config.Compute.Docker.Socket,
		Network:  config.Compute.Docker.Network,
		Registry: config.Compute.Docker.Registry,
	}
	
	dockerProvider := compute.NewDockerProvider(dockerConfig)
	
	return &ProxyDockerProvider{
		DockerProvider:   dockerProvider,
		proxyIntegration: proxyIntegration,
		config:           config,
	}
}

// Deploy deploys a function and registers it with Traefik via the proxy
func (pdp *ProxyDockerProvider) Deploy(ctx context.Context, fn interface{}, imageName string) (interface{}, error) {
	function, ok := fn.(*compute.Function)
	if !ok {
		return nil, fmt.Errorf("invalid function type")
	}
	
	logrus.WithFields(logrus.Fields{
		"function_id":   function.ID,
		"function_name": function.Name,
	}).Info("Starting function deployment with proxy integration")
	
	// Deploy using the original Docker provider
	result, err := pdp.DockerProvider.Deploy(ctx, fn, imageName)
	if err != nil {
		return nil, fmt.Errorf("docker deployment failed: %w", err)
	}
	
	// Register function with Traefik via proxy
	if pdp.proxyIntegration != nil {
		if err := pdp.proxyIntegration.RegisterFunction(ctx, function.ID); err != nil {
			logrus.WithError(err).WithField("function_id", function.ID).Error("Failed to register function with Traefik")
			// Continue anyway - the function is deployed, just not routed
		}
	}
	
	logrus.WithFields(logrus.Fields{
		"function_id":   function.ID,
		"function_name": function.Name,
	}).Info("Function deployed and registered with Traefik")
	
	return result, nil
}

// Remove removes a function and unregisters it from Traefik
func (pdp *ProxyDockerProvider) Remove(ctx context.Context, deployment interface{}) error {
	dep, ok := deployment.(*compute.Deployment)
	if !ok {
		return fmt.Errorf("invalid deployment type")
	}
	
	logrus.WithField("function_id", dep.FunctionID).Info("Removing function deployment")
	
	// Unregister from Traefik first
	if pdp.proxyIntegration != nil {
		if err := pdp.proxyIntegration.UnregisterFunction(ctx, dep.FunctionID); err != nil {
			logrus.WithError(err).WithField("function_id", dep.FunctionID).Warn("Failed to unregister function from Traefik")
			// Continue with removal anyway
		}
	}
	
	// Remove using the original Docker provider
	if err := pdp.DockerProvider.Remove(ctx, deployment); err != nil {
		return fmt.Errorf("docker removal failed: %w", err)
	}
	
	logrus.WithField("function_id", dep.FunctionID).Info("Function removed and unregistered from Traefik")
	return nil
}

// Execute executes a function via the proxy (this should rarely be called directly)
func (pdp *ProxyDockerProvider) Execute(ctx context.Context, deployment interface{}, req interface{}) (interface{}, error) {
	// In the proxy architecture, execution typically goes through the proxy service
	// This method is kept for compatibility but logs a warning
	logrus.Warn("Direct function execution called - consider using proxy service instead")
	
	return pdp.DockerProvider.Execute(ctx, deployment, req)
}

// GetProxyIntegration returns the proxy integration interface
func (pdp *ProxyDockerProvider) GetProxyIntegration() ProxyIntegration {
	return pdp.proxyIntegration
}