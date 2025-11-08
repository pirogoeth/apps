package types

import (
	"time"
)

// NetworkSettings represents network information for a container
type NetworkSettings struct {
	IPAddress string             `json:"ip_address"` // Primary IP
	Networks  map[string]Network `json:"networks"`   // All networks
}

// Network represents a single network configuration
type Network struct {
	IPAddress   string `json:"ip_address"`
	Gateway     string `json:"gateway"`
	NetworkID   string `json:"network_id"`
	NetworkName string `json:"network_name"`
}

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

	// Network information for resolution
	NetworkSettings NetworkSettings `json:"network_settings"`

	// Labels configuration reference (not serialized to JSON)
	LabelsConfig *LabelsConfig `json:"-"`
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

// GetPrimaryIP returns the primary IP address
func (c *ContainerInfo) GetPrimaryIP() string {
	if c.NetworkSettings.IPAddress != "" {
		return c.NetworkSettings.IPAddress
	}

	// Fall back to first network IP
	for _, network := range c.NetworkSettings.Networks {
		if network.IPAddress != "" {
			return network.IPAddress
		}
	}

	return ""
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
	RedisClient StorageClient
}
