package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/pirogoeth/apps/docker-hermes/types"
)

type ReporterOpts struct {
	HeartbeatInterval        time.Duration
	Hostname                 string
	LabelsConfig             *types.LabelsConfig
	TargetResolutionStrategy types.TargetResolutionStrategy
	TargetResolutionHost     string
}

// Reporter handles reporting container information to storage
type Reporter struct {
	storageClient types.StorageClient
	opts          ReporterOpts
}

// NewReporter creates a new reporter instance
func NewReporter(storageClient types.StorageClient, opts ReporterOpts) *Reporter {
	return &Reporter{
		storageClient: storageClient,
		opts:          opts,
	}
}

// ReportContainer reports container information to storage
func (r *Reporter) ReportContainer(ctx context.Context, container *types.ContainerInfo) error {
	// Attach labels config to container
	container.LabelsConfig = r.opts.LabelsConfig

	// Filter labels by prefix if specified
	if r.opts.LabelsConfig != nil && r.opts.LabelsConfig.LabelPrefix != "" {
		container.Labels = container.FilterLabelsByPrefix(r.opts.LabelsConfig.LabelPrefix)
	}

	// Pre-resolve targets for validation
	targets := container.ResolveTargets(r.opts.TargetResolutionStrategy, r.opts.Hostname, r.opts.TargetResolutionHost)
	if len(targets) > 0 {
		// Log resolved targets for debugging
		for i, target := range targets {
			logrus.Debugf("Container %s target %d: %s%s%s",
				container.ID, i, target.Scheme, target.Address, target.Path)
		}
	}

	// Set TTL to 2x heartbeat interval (default 60s)
	ttl := 2 * r.opts.HeartbeatInterval

	// Store container data
	if err := r.storageClient.StoreContainer(ctx, container, ttl); err != nil {
		return fmt.Errorf("failed to store container: %w", err)
	}

	// Publish update to stream
	if err := r.storageClient.PublishUpdate(ctx, container); err != nil {
		return fmt.Errorf("failed to publish update: %w", err)
	}

	logrus.Debugf("Reported container %s on host %s", container.ID, container.Host)
	return nil
}

// ReportContainerRemoval reports container removal to storage
func (r *Reporter) ReportContainerRemoval(ctx context.Context, container *types.ContainerInfo) error {
	// For removal, we create a minimal container info with stopped state
	removalInfo := &types.ContainerInfo{
		ID:       container.ID,
		Name:     container.Name,
		Host:     container.Host,
		State:    "stopped",
		LastSeen: time.Now(),
	}

	// Publish removal update to stream
	if err := r.storageClient.PublishUpdate(ctx, removalInfo); err != nil {
		return fmt.Errorf("failed to publish removal update: %w", err)
	}

	logrus.Debugf("Reported container removal %s on host %s", container.ID, container.Host)
	return nil
}

// ReportBatch reports multiple containers in a batch
func (r *Reporter) ReportBatch(ctx context.Context, containers []*types.ContainerInfo) error {
	var lastErr error
	successCount := 0

	for _, container := range containers {
		if err := r.ReportContainer(ctx, container); err != nil {
			logrus.Errorf("Failed to report container %s: %v", container.ID, err)
			lastErr = err
		} else {
			successCount++
		}
	}

	logrus.Debugf("Batch report completed: %d/%d containers reported successfully", successCount, len(containers))

	if lastErr != nil {
		return fmt.Errorf("batch report completed with errors, last error: %w", lastErr)
	}

	return nil
}
