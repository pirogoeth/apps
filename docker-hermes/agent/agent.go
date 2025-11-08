package agent

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"github.com/pirogoeth/apps/docker-hermes/redis"
	"github.com/pirogoeth/apps/docker-hermes/types"
)

// Agent represents the docker-hermes agent
type Agent struct {
	config        *types.Config
	dockerClient  *DockerClient
	storageClient types.StorageClient
	reporter      *Reporter

	// Metrics
	containersTracked prometheus.Gauge
	updatesSent       prometheus.Counter
	errorsTotal       prometheus.Counter

	// State
	containers map[string]*types.ContainerInfo
	mu         sync.RWMutex
}

// NewAgent creates a new agent instance
func NewAgent(config *types.Config) (*Agent, error) {
	dockerClient, err := NewDockerClient(config.Agent.DockerSocket)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// TODO: Resolve storage client type based on config
	storageClient, err := redis.NewClient(config.Redis.URL)
	if err != nil {
		dockerClient.Close()
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	reporter := NewReporter(storageClient, ReporterOpts{
		HeartbeatInterval:        config.Agent.HeartbeatInterval,
		Hostname:                 config.Agent.Hostname,
		LabelsConfig:             &config.Labels,
		TargetResolutionStrategy: config.Agent.TargetResolutionStrategy,
		TargetResolutionHost:     config.Agent.TargetResolutionHost,
	})

	// Initialize metrics
	containersTracked := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "docker_hermes_containers_tracked",
		Help: "Number of containers currently being tracked",
	})

	updatesSent := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "docker_hermes_updates_sent_total",
		Help: "Total number of container updates sent to server",
	})

	errorsTotal := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "docker_hermes_errors_total",
		Help: "Total number of errors encountered",
	})

	// Register metrics
	prometheus.MustRegister(containersTracked, updatesSent, errorsTotal)

	return &Agent{
		config:            config,
		dockerClient:      dockerClient,
		storageClient:     storageClient,
		reporter:          reporter,
		containersTracked: containersTracked,
		updatesSent:       updatesSent,
		errorsTotal:       errorsTotal,
		containers:        make(map[string]*types.ContainerInfo),
	}, nil
}

// Run starts the agent main loop
func (a *Agent) Run(ctx context.Context) error {
	logrus.Info("Starting docker-hermes agent")

	// Initial scan
	if err := a.performHeartbeat(ctx); err != nil {
		logrus.Errorf("Initial heartbeat failed: %v", err)
		a.errorsTotal.Inc()
	}

	// Start event watcher
	go func() {
		if err := a.dockerClient.WatchContainers(ctx, a.config.Agent.Hostname, a.handleDockerEvent); err != nil {
			logrus.Errorf("Docker event watcher failed: %v", err)
			a.errorsTotal.Inc()
		}
	}()

	// Start heartbeat ticker
	ticker := time.NewTicker(a.config.Agent.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := a.performHeartbeat(ctx); err != nil {
				logrus.Errorf("Heartbeat failed: %v", err)
				a.errorsTotal.Inc()
			}
		case <-ctx.Done():
			logrus.Info("Agent shutting down")
			return nil
		}
	}
}

// performHeartbeat performs a full container scan and reports to server
func (a *Agent) performHeartbeat(ctx context.Context) error {
	logrus.Debug("Performing heartbeat")

	// Scan all running containers
	containers, err := a.dockerClient.ScanContainers(ctx, a.config.Agent.Hostname)
	if err != nil {
		return fmt.Errorf("failed to scan containers: %w", err)
	}

	// Update internal state
	a.mu.Lock()
	newContainers := make(map[string]*types.ContainerInfo)
	for _, container := range containers {
		newContainers[container.ID] = container
	}
	a.containers = newContainers
	a.mu.Unlock()

	// Update metrics
	a.containersTracked.Set(float64(len(containers)))

	// Report all containers
	for _, container := range containers {
		if err := a.reporter.ReportContainer(ctx, container); err != nil {
			logrus.Errorf("Failed to report container %s: %v", container.ID, err)
			a.errorsTotal.Inc()
			continue
		}
		a.updatesSent.Inc()
	}

	logrus.Debugf("Heartbeat completed: %d containers reported", len(containers))
	return nil
}

// handleDockerEvent handles Docker events
func (a *Agent) handleDockerEvent(container *types.ContainerInfo, eventType events.Action) {
	ctx := context.Background()

	logrus.Debugf("Handling Docker event: %s for container %s", eventType, container.ID)

	switch eventType {
	case "start":
		// Update internal state
		a.mu.Lock()
		a.containers[container.ID] = container
		a.mu.Unlock()

		// Report to server
		if err := a.reporter.ReportContainer(ctx, container); err != nil {
			logrus.Errorf("Failed to report container start: %v", err)
			a.errorsTotal.Inc()
		} else {
			a.updatesSent.Inc()
		}

	case "stop", "die":
		// Remove from internal state
		a.mu.Lock()
		delete(a.containers, container.ID)
		a.mu.Unlock()

		// Report removal to server
		if err := a.reporter.ReportContainerRemoval(ctx, container); err != nil {
			logrus.Errorf("Failed to report container removal: %v", err)
			a.errorsTotal.Inc()
		} else {
			a.updatesSent.Inc()
		}
	}

	// Update metrics
	a.mu.RLock()
	trackedCount := len(a.containers)
	a.mu.RUnlock()
	a.containersTracked.Set(float64(trackedCount))
}

// Close closes the agent and cleans up resources
func (a *Agent) Close() error {
	var errs []error

	if err := a.dockerClient.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close Docker client: %w", err))
	}

	if err := a.storageClient.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close storage client: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during cleanup: %v", errs)
	}

	return nil
}

// GetMetricsHandler returns the Prometheus metrics handler
func (a *Agent) GetMetricsHandler() http.Handler {
	return promhttp.Handler()
}
