package types

import (
	"context"
	"time"
)

// StorageClient defines the interface for storage backends
// This allows different storage implementations (Redis, PostgreSQL, etc.)
// to be used interchangeably
type StorageClient interface {
	// StoreContainer stores container information with TTL
	StoreContainer(ctx context.Context, container *ContainerInfo, ttl time.Duration) error

	// GetContainer retrieves container information by host and container ID
	// Returns nil if container is not found
	GetContainer(ctx context.Context, host, containerID string) (*ContainerInfo, error)

	// ListContainers retrieves all active containers
	ListContainers(ctx context.Context) ([]*ContainerInfo, error)

	// QueryByLabels finds containers matching label filters
	QueryByLabels(ctx context.Context, query *ContainerQuery) ([]*ContainerInfo, error)

	// PublishUpdate publishes a container update to the event stream
	PublishUpdate(ctx context.Context, container *ContainerInfo) error

	// GetPrometheusTargets retrieves all containers that should be scraped by Prometheus
	GetPrometheusTargets(ctx context.Context) (PrometheusSDResponse, error)

	// GetLabels retrieves all unique label keys and their values
	GetLabels(ctx context.Context) ([]LabelInfo, error)

	// GetLabelValues retrieves all values for a specific label key
	GetLabelValues(ctx context.Context, labelKey string) ([]string, error)

	// GetHosts retrieves all known hosts
	GetHosts(ctx context.Context) ([]string, error)

	// CleanupExpired removes expired container data (called periodically)
	CleanupExpired(ctx context.Context) error

	// Close closes the storage connection
	Close() error

	// Ping tests the storage connection
	Ping(ctx context.Context) error
}
