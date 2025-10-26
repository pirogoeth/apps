package agent

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"

	hermesTypes "github.com/pirogoeth/apps/docker-hermes/types"
)

// DockerClient wraps Docker API client with container monitoring functionality
type DockerClient struct {
	client *client.Client
}

// NewDockerClient creates a new Docker client
func NewDockerClient(dockerSocket string) (*DockerClient, error) {
	opts := []client.Opt{
		client.WithAPIVersionNegotiation(),
	}

	if dockerSocket != "" {
		opts = append(opts, client.WithHost(dockerSocket))
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &DockerClient{client: cli}, nil
}

// Close closes the Docker client
func (d *DockerClient) Close() error {
	return d.client.Close()
}

// ScanContainers performs a full scan of running containers
func (d *DockerClient) ScanContainers(ctx context.Context, hostname string) ([]*hermesTypes.ContainerInfo, error) {
	containers, err := d.client.ContainerList(ctx, container.ListOptions{
		All: false, // Only running containers
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var containerInfos []*hermesTypes.ContainerInfo
	for _, container := range containers {
		info, err := d.extractContainerInfo(ctx, container, hostname)
		if err != nil {
			logrus.Warnf("Failed to extract info for container %s: %v", container.ID, err)
			continue
		}
		containerInfos = append(containerInfos, info)
	}

	logrus.Infof("Scanned %d running containers", len(containerInfos))
	return containerInfos, nil
}

// WatchContainers watches for Docker events and calls the callback
func (d *DockerClient) WatchContainers(ctx context.Context, hostname string, callback func(*hermesTypes.ContainerInfo, events.Action)) error {
	eventFilter := filters.NewArgs()
	eventFilter.Add("type", "container")
	eventFilter.Add("event", "start")
	eventFilter.Add("event", "stop")
	eventFilter.Add("event", "die")

	eventChan, errChan := d.client.Events(ctx, events.ListOptions{
		Filters: eventFilter,
	})

	logrus.Info("Started watching Docker events")

	for {
		select {
		case event := <-eventChan:
			if err := d.handleDockerEvent(ctx, event, hostname, callback); err != nil {
				logrus.Errorf("Failed to handle Docker event: %v", err)
			}
		case err := <-errChan:
			if err != nil {
				return fmt.Errorf("received error from Docker: %w", err)
			}
		case <-ctx.Done():
			logrus.Info("Stopped watching Docker events")
			return nil
		}
	}
}

// handleDockerEvent processes a Docker event and extracts container info
func (d *DockerClient) handleDockerEvent(ctx context.Context, event events.Message, hostname string, callback func(*hermesTypes.ContainerInfo, events.Action)) error {
	containerID := event.Actor.ID
	eventType := event.Action

	logrus.Debugf("Docker event: %s for container %s", eventType, containerID)

	// For stop/die events, we don't need to fetch container details
	if eventType == "stop" || eventType == "die" {
		// Create a minimal container info for removal
		containerInfo := &hermesTypes.ContainerInfo{
			ID:       containerID,
			Host:     hostname,
			State:    "stopped",
			LastSeen: time.Now(),
		}
		callback(containerInfo, eventType)
		return nil
	}

	// For start events, fetch full container details
	if eventType == "start" {
		container, err := d.client.ContainerInspect(ctx, containerID)
		if err != nil {
			return fmt.Errorf("failed to inspect container %s: %w", containerID, err)
		}

		info, err := d.extractContainerInfoFromInspect(container, hostname)
		if err != nil {
			return fmt.Errorf("failed to extract container info: %w", err)
		}

		callback(info, eventType)
	}

	return nil
}

// extractContainerInfo extracts container information from container list response
func (d *DockerClient) extractContainerInfo(ctx context.Context, container container.Summary, hostname string) (*hermesTypes.ContainerInfo, error) {
	// Get detailed container info
	inspect, err := d.client.ContainerInspect(ctx, container.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	return d.extractContainerInfoFromInspect(inspect, hostname)
}

// extractContainerInfoFromInspect extracts container information from inspect response
func (d *DockerClient) extractContainerInfoFromInspect(inspect container.InspectResponse, hostname string) (*hermesTypes.ContainerInfo, error) {
	logrus.WithField("container", inspect.ID).Debugf("Extracting container info from inspect")

	// Extract ports
	var ports []hermesTypes.PortInfo
	for port, bindings := range inspect.NetworkSettings.Ports {
		for _, binding := range bindings {
			publicPort, _ := strconv.Atoi(binding.HostPort)
			privatePort, _ := strconv.Atoi(port.Port())

			ports = append(ports, hermesTypes.PortInfo{
				PrivatePort: privatePort,
				PublicPort:  publicPort,
				Type:        port.Proto(),
				IP:          binding.HostIP,
			})
		}
	}

	logrus.WithField("container", inspect.ID).
		WithField("ports", ports).
		Debugf("Extracted ports from container")

	name := strings.TrimPrefix(inspect.Name, "/")

	// Extract command
	var command string
	if len(inspect.Config.Cmd) > 0 {
		command = strings.Join(inspect.Config.Cmd, " ")
	}

	// Extract created time
	createdAt, err := time.Parse(time.RFC3339Nano, inspect.Created)
	if err != nil {
		return nil, fmt.Errorf("failed to parse container creation timestamp: %w", err)
	}

	containerInfo := &hermesTypes.ContainerInfo{
		ID:        inspect.ID,
		Name:      name,
		Host:      hostname,
		Labels:    inspect.Config.Labels,
		Ports:     ports,
		State:     inspect.State.Status,
		LastSeen:  time.Now(),
		CreatedAt: createdAt,
		Image:     inspect.Config.Image,
		Command:   command,
		Status:    inspect.State.Status,
	}

	logrus.WithField("container", containerInfo).Debugf("Extracted container info")

	return containerInfo, nil
}

// GetContainerByID retrieves a specific container by ID
func (d *DockerClient) GetContainerByID(ctx context.Context, containerID string, hostname string) (*hermesTypes.ContainerInfo, error) {
	inspect, err := d.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	return d.extractContainerInfoFromInspect(inspect, hostname)
}
