package proxy

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	containerTypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/pirogoeth/apps/functional/database"
	functypes "github.com/pirogoeth/apps/functional/types"
	"github.com/sirupsen/logrus"
)

// ContainerPool manages a pool of containers for function execution
type ContainerPool struct {
	config *functypes.Config
	client *client.Client

	// Pool management
	pools     map[string]*FunctionPool // functionID -> pool
	poolMutex sync.RWMutex

	// Container tracking
	containers     map[string]*PooledContainer // containerID -> container
	containerMutex sync.RWMutex
}

// FunctionPool represents a pool of containers for a specific function
type FunctionPool struct {
	FunctionID   string
	Available    []*PooledContainer
	InUse        []*PooledContainer
	MaxSize      int
	IdleTimeout  time.Duration
	CreatedCount int64
	mutex        sync.RWMutex
}

// PooledContainer represents a container in the pool with communication pipes
type PooledContainer struct {
	ID          string
	FunctionID  string
	ContainerID string

	// Communication pipes
	Stdin  io.WriteCloser
	Stdout io.ReadCloser
	Stderr io.ReadCloser

	// Lifecycle
	CreatedAt time.Time
	LastUsed  time.Time
	UseCount  int64
	Status    ContainerStatus
}

type ContainerStatus int

const (
	ContainerStatusStarting ContainerStatus = iota
	ContainerStatusReady
	ContainerStatusInUse
	ContainerStatusStopping
	ContainerStatusStopped
)

// NewContainerPool creates a new container pool
func NewContainerPool(config *functypes.Config) *ContainerPool {
	dockerClient, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create Docker client")
	}

	return &ContainerPool{
		config:     config,
		client:     dockerClient,
		pools:      make(map[string]*FunctionPool),
		containers: make(map[string]*PooledContainer),
	}
}

// GetContainer gets or creates a container for the function
func (cp *ContainerPool) GetContainer(ctx context.Context, function *database.Function) (*PooledContainer, error) {
	pool := cp.getOrCreatePool(function.ID)

	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	// Try to get an available container
	if len(pool.Available) > 0 {
		container := pool.Available[0]
		pool.Available = pool.Available[1:]
		pool.InUse = append(pool.InUse, container)

		container.Status = ContainerStatusInUse
		container.LastUsed = time.Now()
		container.UseCount++

		logrus.WithFields(logrus.Fields{
			"function_id":  function.ID,
			"container_id": container.ContainerID,
			"use_count":    container.UseCount,
		}).Debug("Reusing container from pool")

		return container, nil
	}

	// Create new container if pool isn't at capacity
	if len(pool.InUse) < pool.MaxSize {
		container, err := cp.createContainer(ctx, function)
		if err != nil {
			return nil, fmt.Errorf("failed to create container: %w", err)
		}

		pool.InUse = append(pool.InUse, container)
		pool.CreatedCount++

		logrus.WithFields(logrus.Fields{
			"function_id":  function.ID,
			"container_id": container.ContainerID,
			"pool_size":    len(pool.InUse),
		}).Info("Created new container for pool")

		return container, nil
	}

	return nil, fmt.Errorf("container pool at capacity for function %s", function.ID)
}

// ReturnContainer returns a container to the pool
func (cp *ContainerPool) ReturnContainer(container *PooledContainer) {
	pool := cp.getOrCreatePool(container.FunctionID)

	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	// Remove from in-use
	for i, c := range pool.InUse {
		if c.ID == container.ID {
			pool.InUse = append(pool.InUse[:i], pool.InUse[i+1:]...)
			break
		}
	}

	// Add to available if container is still healthy
	if container.Status == ContainerStatusInUse {
		container.Status = ContainerStatusReady
		pool.Available = append(pool.Available, container)

		logrus.WithFields(logrus.Fields{
			"function_id":  container.FunctionID,
			"container_id": container.ContainerID,
		}).Debug("Returned container to pool")
	} else {
		// Container is unhealthy, clean it up
		cp.removeContainer(container)
	}
}

// createContainer creates a new container for the function
func (cp *ContainerPool) createContainer(ctx context.Context, function *database.Function) (*PooledContainer, error) {
	// Generate container image tag (this would integrate with your existing Docker provider)
	imageTag := fmt.Sprintf("function-%s:latest", function.Name)

	// Create container config for pipe communication
	config := &containerTypes.Config{
		Image:        imageTag,
		Cmd:          []string{"/app/wrapper"}, // We'll need a wrapper script in containers
		Tty:          false,
		OpenStdin:    true,
		StdinOnce:    false,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	}

	hostConfig := &containerTypes.HostConfig{
		// Add resource constraints
		Resources: containerTypes.Resources{
			Memory:    int64(function.MemoryMb) * 1024 * 1024,
			CPUShares: 1024, // Default CPU shares
		},
	}

	// Create container
	resp, err := cp.client.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := cp.client.ContainerStart(ctx, resp.ID, containerTypes.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Attach to container for pipe communication
	attachResp, err := cp.client.ContainerAttach(ctx, resp.ID, containerTypes.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to attach to container: %w", err)
	}

	pooledContainer := &PooledContainer{
		ID:          fmt.Sprintf("pool_%s_%d", function.ID, time.Now().UnixNano()),
		FunctionID:  function.ID,
		ContainerID: resp.ID,
		Stdin:       attachResp.Conn,
		Stdout:      attachResp.Conn, // Use Conn which implements ReadWriteCloser
		Stderr:      attachResp.Conn, // Docker multiplexes stdout/stderr
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
		UseCount:    1,
		Status:      ContainerStatusInUse,
	}

	// Track container
	cp.containerMutex.Lock()
	cp.containers[pooledContainer.ID] = pooledContainer
	cp.containerMutex.Unlock()

	return pooledContainer, nil
}

// getOrCreatePool gets or creates a function pool
func (cp *ContainerPool) getOrCreatePool(functionID string) *FunctionPool {
	cp.poolMutex.RLock()
	pool, exists := cp.pools[functionID]
	cp.poolMutex.RUnlock()

	if exists {
		return pool
	}

	// Create new pool
	cp.poolMutex.Lock()
	defer cp.poolMutex.Unlock()

	// Double-check after acquiring write lock
	if pool, exists := cp.pools[functionID]; exists {
		return pool
	}

	pool = &FunctionPool{
		FunctionID:  functionID,
		Available:   make([]*PooledContainer, 0),
		InUse:       make([]*PooledContainer, 0),
		MaxSize:     cp.config.Proxy.MaxContainersPerFunction,
		IdleTimeout: cp.config.Proxy.ContainerIdleTimeout,
	}

	cp.pools[functionID] = pool
	return pool
}

// StartCleanup starts the background cleanup routine
func (cp *ContainerPool) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cp.cleanupIdleContainers()
		}
	}
}

// cleanupIdleContainers removes idle containers from pools
func (cp *ContainerPool) cleanupIdleContainers() {
	cp.poolMutex.RLock()
	pools := make([]*FunctionPool, 0, len(cp.pools))
	for _, pool := range cp.pools {
		pools = append(pools, pool)
	}
	cp.poolMutex.RUnlock()

	now := time.Now()

	for _, pool := range pools {
		pool.mutex.Lock()

		// Check available containers for idle timeout
		available := make([]*PooledContainer, 0, len(pool.Available))
		for _, container := range pool.Available {
			if now.Sub(container.LastUsed) > pool.IdleTimeout {
				logrus.WithFields(logrus.Fields{
					"function_id":  container.FunctionID,
					"container_id": container.ContainerID,
					"idle_time":    now.Sub(container.LastUsed),
				}).Info("Removing idle container")

				cp.removeContainer(container)
			} else {
				available = append(available, container)
			}
		}
		pool.Available = available

		pool.mutex.Unlock()
	}
}

// removeContainer removes and cleans up a container
func (cp *ContainerPool) removeContainer(container *PooledContainer) {
	// Stop container
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop container with timeout
	timeoutSeconds := 5
	if err := cp.client.ContainerStop(ctx, container.ContainerID, containerTypes.StopOptions{Timeout: &timeoutSeconds}); err != nil {
		logrus.WithError(err).WithField("container_id", container.ContainerID).Warn("Failed to stop container")
		// Continue with forced removal
	}

	// Remove container
	if err := cp.client.ContainerRemove(ctx, container.ContainerID, containerTypes.RemoveOptions{Force: true}); err != nil {
		logrus.WithError(err).WithField("container_id", container.ContainerID).Warn("Failed to remove container")
	}

	// Close pipes
	if container.Stdin != nil {
		container.Stdin.Close()
	}
	if container.Stdout != nil {
		container.Stdout.Close()
	}
	if container.Stderr != nil {
		container.Stderr.Close()
	}

	// Remove from tracking
	cp.containerMutex.Lock()
	delete(cp.containers, container.ID)
	cp.containerMutex.Unlock()

	container.Status = ContainerStatusStopped
}

// GetPoolStats returns statistics about the container pools
func (cp *ContainerPool) GetPoolStats() map[string]interface{} {
	cp.poolMutex.RLock()
	defer cp.poolMutex.RUnlock()

	stats := make(map[string]interface{})

	for functionID, pool := range cp.pools {
		pool.mutex.RLock()
		poolStats := map[string]interface{}{
			"available_containers": len(pool.Available),
			"in_use_containers":    len(pool.InUse),
			"max_size":             pool.MaxSize,
			"created_count":        pool.CreatedCount,
		}
		pool.mutex.RUnlock()

		stats[functionID] = poolStats
	}

	return stats
}
