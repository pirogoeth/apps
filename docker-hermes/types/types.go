package types

import (
	"strconv"
	"time"
)

// ContainerInfo represents information about a Docker container
type ContainerInfo struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Host      string            `json:"host"`
	Labels    map[string]string `json:"labels"`
	Ports     []PortInfo        `json:"ports"`
	State     string            `json:"state"`
	LastSeen  time.Time         `json:"last_seen"`
	CreatedAt time.Time         `json:"created_at"`
	Image     string            `json:"image"`
	Command   string            `json:"command"`
	Status    string            `json:"status"`
}

// IsPrometheusScrapeTarget checks if a container should be included in Prometheus SD
func (c *ContainerInfo) IsPrometheusScrapeTarget() bool {
	if scrape, exists := c.Labels["prometheus.io/scrape"]; exists {
		return scrape == "true"
	}
	return false
}

// GetPrometheusPort returns the port to scrape for Prometheus
func (c *ContainerInfo) GetPrometheusPort() int {
	if portStr, exists := c.Labels["prometheus.io/port"]; exists {
		// Try to parse the port string
		if port, err := strconv.Atoi(portStr); err == nil {
			return port
		}
	}

	// Default to first exposed port
	if len(c.Ports) > 0 {
		return c.Ports[0].PublicPort
	}
	return 0
}

// GetPrometheusPath returns the metrics path for Prometheus
func (c *ContainerInfo) GetPrometheusPath() string {
	if path, exists := c.Labels["prometheus.io/path"]; exists {
		return path
	}
	return "/metrics"
}

// GetPrometheusScheme returns the scheme for Prometheus scraping
func (c *ContainerInfo) GetPrometheusScheme() string {
	if scheme, exists := c.Labels["prometheus.io/scheme"]; exists {
		return scheme
	}
	return "http"
}

// FilterLabelsByPrefix filters labels by the given prefix
func (c *ContainerInfo) FilterLabelsByPrefix(prefix string) map[string]string {
	if prefix == "" {
		return c.Labels
	}

	filtered := make(map[string]string)
	for key, value := range c.Labels {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			filtered[key] = value
		}
	}
	return filtered
}

// ToPrometheusTarget converts ContainerInfo to PrometheusTarget
func (c *ContainerInfo) ToPrometheusTarget() *PrometheusTarget {
	if !c.IsPrometheusScrapeTarget() {
		return nil
	}

	port := c.GetPrometheusPort()
	if port == 0 {
		return nil
	}

	target := c.Host + ":" + strconv.Itoa(port)

	labels := make(map[string]string)
	labels["__meta_docker_container_id"] = c.ID
	labels["__meta_docker_container_name"] = c.Name
	labels["__meta_docker_container_image"] = c.Image
	labels["__meta_docker_container_status"] = c.Status
	labels["__meta_docker_host"] = c.Host

	// Add all container labels
	for key, value := range c.Labels {
		labels[key] = value
	}

	return &PrometheusTarget{
		Targets: []string{target},
		Labels:  labels,
	}
}

// PortInfo represents port information for a container
type PortInfo struct {
	PrivatePort int    `json:"private_port"`
	PublicPort  int    `json:"public_port"`
	Type        string `json:"type"`
	IP          string `json:"ip"`
}

// PrometheusTarget represents a Prometheus scrape target
type PrometheusTarget struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

// PrometheusSDResponse represents the Prometheus HTTP SD response format
type PrometheusSDResponse []PrometheusTarget

// ContainerQuery represents a query for containers by labels
type ContainerQuery struct {
	Labels map[string]string `json:"labels"`
	Host   string            `json:"host,omitempty"`
}

// LabelInfo represents information about a label key and its values
type LabelInfo struct {
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

// ApiContext represents the API context for handlers
type ApiContext struct {
	Config      *Config
	RedisClient interface{} // Will be *redis.Client in actual usage
}
