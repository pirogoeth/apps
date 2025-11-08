package types

import (
	"os"
	"time"

	"github.com/pirogoeth/apps/pkg/config"
)

// LabelsConfig holds label prefix configuration shared by agent and server
type LabelsConfig struct {
	LabelPrefix string `yaml:"label_prefix" envconfig:"LABEL_PREFIX" default:""`
	Prefix      string `yaml:"prefix" envconfig:"PREFIX" default:"hermes"` // Main label prefix (Traefik-style)
}

// Config represents the main configuration struct
type Config struct {
	config.CommonConfig `yaml:",inline"`

	// Labels configuration shared by agent and server
	Labels LabelsConfig `yaml:"labels"`

	// Redis is the common configuration for both the agent and server
	// to connect and use the same Redis instance.
	Redis  RedisConfig  `yaml:"redis"`
	Agent  AgentConfig  `yaml:"agent"`
	Server ServerConfig `yaml:"server"`
}

type RedisConfig struct {
	URL string `yaml:"url" envconfig:"REDIS_URL" default:"redis://localhost:6379"`
}

type AgentConfig struct {
	DockerSocket      string        `yaml:"docker_socket" envconfig:"DOCKER_SOCKET" default:"unix:///var/run/docker.sock"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval" envconfig:"HEARTBEAT_INTERVAL" default:"30s"`
	Hostname          string        `yaml:"hostname" envconfig:"HOSTNAME"`

	// Target resolution configuration
	TargetResolutionStrategy TargetResolutionStrategy `yaml:"target_resolution_strategy" envconfig:"TARGET_RESOLUTION_STRATEGY" default:"hostname"`
	TargetResolutionHost     string                   `yaml:"target_resolution_host" envconfig:"TARGET_RESOLUTION_HOST" default:""`
}

type ServerConfig struct {
	PrometheusSDPath string `yaml:"prometheus_sd_path" envconfig:"PROMETHEUS_SD_PATH" default:"/prometheus/sd"`

	// Target resolution configuration (for Prometheus SD)
	TargetResolutionStrategy TargetResolutionStrategy `yaml:"target_resolution_strategy" envconfig:"TARGET_RESOLUTION_STRATEGY" default:"hostname"`
	TargetResolutionHost     string                   `yaml:"target_resolution_host" envconfig:"TARGET_RESOLUTION_HOST" default:""`
}

// ApplyDefaults applies default values to config structs
func ApplyDefaults(cfg interface{}) {
	switch c := cfg.(type) {
	case *AgentConfig:
		if c.Hostname == "" {
			if hostname, err := os.Hostname(); err == nil {
				c.Hostname = hostname
			} else {
				c.Hostname = "unknown"
			}
		}
	}
}
