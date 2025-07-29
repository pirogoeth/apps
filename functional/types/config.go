package types

import (
	"time"

	"github.com/pirogoeth/apps/functional/database"
	"github.com/pirogoeth/apps/pkg/config"
)

type Config struct {
	config.CommonConfig

	Database *database.Config `json:"database"`
	Compute  ComputeConfig    `json:"compute"`
	Storage  StorageConfig    `json:"storage"`
	Runtime  RuntimeConfig    `json:"runtime"`
	Proxy    ProxyConfig      `json:"proxy"`
}

type ComputeConfig struct {
	Provider    string             `json:"provider" envconfig:"COMPUTE_PROVIDER"`
	Docker      *DockerConfig      `json:"docker,omitempty"`
	Firecracker *FirecrackerConfig `json:"firecracker,omitempty"`
}

type DockerConfig struct {
	Socket   string `json:"socket" envconfig:"DOCKER_SOCKET"`
	Network  string `json:"network" envconfig:"DOCKER_NETWORK"`
	Registry string `json:"registry" envconfig:"DOCKER_REGISTRY"`
}

type FirecrackerConfig struct {
	KernelImagePath string `json:"kernel_image_path" envconfig:"FIRECRACKER_KERNEL_IMAGE_PATH"`
	RootfsImagePath string `json:"rootfs_image_path" envconfig:"FIRECRACKER_ROOTFS_IMAGE_PATH"`
	WorkDir         string `json:"work_dir" envconfig:"FIRECRACKER_WORK_DIR"`
	NetworkDevice   string `json:"network_device" envconfig:"FIRECRACKER_NETWORK_DEVICE"`
}

type StorageConfig struct {
	FunctionsPath string `json:"functions_path" envconfig:"STORAGE_FUNCTIONS_PATH"`
	TempPath      string `json:"temp_path" envconfig:"STORAGE_TEMP_PATH"`
}

type RuntimeConfig struct {
	MaxConcurrentExecutions int                 `json:"max_concurrent_executions" envconfig:"RUNTIME_MAX_CONCURRENT_EXECUTIONS"`
	DefaultTimeout          config.TimeDuration `json:"default_timeout" envconfig:"RUNTIME_DEFAULT_TIMEOUT"`
	Scaling                 ScalingConfig       `json:"scaling"`
}

type ScalingConfig struct {
	// TODO: Revisit these - this are most useful as configured on individual functions but these could serve as global defaults or limits?
	MinReplicas        int     `json:"min_replicas" envconfig:"SCALING_MIN_REPLICAS"`
	MaxReplicas        int     `json:"max_replicas" envconfig:"SCALING_MAX_REPLICAS"`
	ScaleUpThreshold   float64 `json:"scale_up_threshold" envconfig:"SCALING_SCALE_UP_THRESHOLD"`
	ScaleDownThreshold float64 `json:"scale_down_threshold" envconfig:"SCALING_SCALE_DOWN_THRESHOLD"`
}

type ProxyConfig struct {
	ListenAddress            string        `json:"listen_address" envconfig:"PROXY_LISTEN_ADDRESS"`
	TraefikAPIURL            string        `json:"traefik_api_url" envconfig:"PROXY_TRAEFIK_API_URL"`
	MaxContainersPerFunction int           `json:"max_containers_per_function" envconfig:"PROXY_MAX_CONTAINERS_PER_FUNCTION"`
	ContainerIdleTimeout     time.Duration `json:"container_idle_timeout" envconfig:"PROXY_CONTAINER_IDLE_TIMEOUT"`
}
