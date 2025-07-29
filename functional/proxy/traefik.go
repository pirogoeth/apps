package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// TraefikClient handles communication with Traefik's API
type TraefikClient struct {
	apiURL     string
	httpClient *http.Client
}

// TraefikRoute represents a Traefik route configuration
type TraefikRoute struct {
	Rule       string            `json:"rule"`
	Service    string            `json:"service"`
	Priority   *int              `json:"priority,omitempty"`
	Middlewares []string         `json:"middlewares,omitempty"`
	TLS        *TraefikTLS       `json:"tls,omitempty"`
}

// TraefikService represents a Traefik service configuration
type TraefikService struct {
	LoadBalancer *TraefikLoadBalancer `json:"loadBalancer"`
}

// TraefikLoadBalancer represents load balancer configuration
type TraefikLoadBalancer struct {
	Servers []TraefikServer `json:"servers"`
}

// TraefikServer represents a backend server
type TraefikServer struct {
	URL string `json:"url"`
}

// TraefikTLS represents TLS configuration
type TraefikTLS struct {
	CertResolver string `json:"certResolver,omitempty"`
}

// TraefikConfiguration represents the full Traefik configuration
type TraefikConfiguration struct {
	HTTP *TraefikHTTPConfiguration `json:"http"`
}

// TraefikHTTPConfiguration represents HTTP configuration
type TraefikHTTPConfiguration struct {
	Routers  map[string]*TraefikRoute   `json:"routers"`
	Services map[string]*TraefikService `json:"services"`
}

// NewTraefikClient creates a new Traefik API client
func NewTraefikClient(apiURL string) *TraefikClient {
	return &TraefikClient{
		apiURL: apiURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// RegisterRoute registers a function route with Traefik
func (tc *TraefikClient) RegisterRoute(ctx context.Context, functionID string, route *TraefikRoute) error {
	// Create the service configuration for the proxy
	service := &TraefikService{
		LoadBalancer: &TraefikLoadBalancer{
			Servers: []TraefikServer{
				{
					URL: "http://functional:8080", // Points to the proxy service
				},
			},
		},
	}
	
	// Create full configuration
	config := &TraefikConfiguration{
		HTTP: &TraefikHTTPConfiguration{
			Routers: map[string]*TraefikRoute{
				fmt.Sprintf("function-%s", functionID): route,
			},
			Services: map[string]*TraefikService{
				"proxy": service,
			},
		},
	}
	
	return tc.putConfiguration(ctx, fmt.Sprintf("function-%s", functionID), config)
}

// UnregisterRoute removes a function route from Traefik
func (tc *TraefikClient) UnregisterRoute(ctx context.Context, functionID string) error {
	routerName := fmt.Sprintf("function-%s", functionID)
	
	// Send DELETE request to remove the router
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("%s/http/routers/%s", tc.apiURL, routerName),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}
	
	resp, err := tc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete route: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("failed to delete route, status: %d", resp.StatusCode)
	}
	
	logrus.WithField("function_id", functionID).Info("Unregistered function route from Traefik")
	return nil
}

// putConfiguration sends configuration to Traefik
func (tc *TraefikClient) putConfiguration(ctx context.Context, name string, config *TraefikConfiguration) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}
	
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		fmt.Sprintf("%s/providers/rest", tc.apiURL),
		bytes.NewBuffer(configJSON),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := tc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send configuration: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update configuration, status: %d", resp.StatusCode)
	}
	
	logrus.WithField("name", name).Info("Updated Traefik configuration")
	return nil
}

// GetRoutes retrieves all routes from Traefik
func (tc *TraefikClient) GetRoutes(ctx context.Context) (map[string]*TraefikRoute, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/http/routers", tc.apiURL),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := tc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get routes: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get routes, status: %d", resp.StatusCode)
	}
	
	var routes map[string]*TraefikRoute
	if err := json.NewDecoder(resp.Body).Decode(&routes); err != nil {
		return nil, fmt.Errorf("failed to decode routes: %w", err)
	}
	
	return routes, nil
}

// HealthCheck checks if Traefik API is accessible
func (tc *TraefikClient) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/ping", tc.apiURL),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}
	
	resp, err := tc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Traefik API health check failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Traefik API unhealthy, status: %d", resp.StatusCode)
	}
	
	return nil
}