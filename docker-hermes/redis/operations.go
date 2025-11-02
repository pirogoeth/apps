package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/pirogoeth/apps/docker-hermes/types"
)

// StoreContainer stores container information in Redis with TTL
func (c *Client) StoreContainer(ctx context.Context, container *types.ContainerInfo, ttl time.Duration) error {
	key := containerCacheKey(container.Host, container.ID)

	data, err := json.Marshal(container)
	if err != nil {
		return fmt.Errorf("failed to marshal container data: %w", err)
	}

	pipe := c.rdb.Pipeline()

	// Store container data
	pipe.Set(ctx, key, data, ttl)

	// Add to host set
	hostKey := hostsCacheKey()
	pipe.SAdd(ctx, hostKey, container.Host)
	pipe.Expire(ctx, hostKey, ttl*2) // Keep host set longer

	// Add labels to label sets
	for labelKey, labelValue := range container.Labels {
		labelKeyStr := labelCacheKey(labelKey)
		pipe.SAdd(ctx, labelKeyStr, labelValue)
		pipe.Expire(ctx, labelKeyStr, ttl*2)
	}
	pipe.Expire(ctx, labelsCacheKey(), ttl*2)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to store container data: %w", err)
	}

	logrus.Debugf("Stored container %s on host %s", container.ID, container.Host)
	return nil
}

// GetContainer retrieves container information from Redis
func (c *Client) GetContainer(ctx context.Context, host, containerID string) (*types.ContainerInfo, error) {
	key := containerCacheKey(host, containerID)

	data, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get container data: %w", err)
	}

	var container types.ContainerInfo
	if err := json.Unmarshal([]byte(data), &container); err != nil {
		return nil, fmt.Errorf("failed to unmarshal container data: %w", err)
	}

	return &container, nil
}

// ListContainers retrieves all active containers
func (c *Client) ListContainers(ctx context.Context) ([]*types.ContainerInfo, error) {
	pattern := containerHostCacheKey("*")
	keys, err := c.rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list container keys: %w", err)
	}

	var containers []*types.ContainerInfo
	for _, key := range keys {
		data, err := c.rdb.Get(ctx, key).Result()
		if err != nil {
			logrus.Warnf("Failed to get container data for key %s: %v", key, err)
			continue
		}

		var container types.ContainerInfo
		if err := json.Unmarshal([]byte(data), &container); err != nil {
			logrus.Warnf("Failed to unmarshal container data for key %s: %v", key, err)
			continue
		}

		containers = append(containers, &container)
	}

	return containers, nil
}

// QueryByLabels finds containers matching label filters
func (c *Client) QueryByLabels(ctx context.Context, query *types.ContainerQuery) ([]*types.ContainerInfo, error) {
	containers, err := c.ListContainers(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []*types.ContainerInfo
	for _, container := range containers {
		// Filter by host if specified
		if query.Host != "" && container.Host != query.Host {
			continue
		}

		// Check if all query labels match
		matches := true
		for queryKey, queryValue := range query.Labels {
			if containerValue, exists := container.Labels[queryKey]; !exists || containerValue != queryValue {
				matches = false
				break
			}
		}

		if matches {
			filtered = append(filtered, container)
		}
	}

	return filtered, nil
}

// PublishUpdate publishes a container update to Redis Stream
func (c *Client) PublishUpdate(ctx context.Context, container *types.ContainerInfo) error {
	streamKey := streamCacheKey()

	data, err := json.Marshal(container)
	if err != nil {
		return fmt.Errorf("failed to marshal container data for stream: %w", err)
	}

	args := map[string]interface{}{
		"container": string(data),
		"timestamp": time.Now().Unix(),
		"host":      container.Host,
		"id":        container.ID,
	}

	_, err = c.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey,
		Values: args,
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to publish update to stream: %w", err)
	}

	logrus.Debugf("Published update for container %s on host %s", container.ID, container.Host)
	return nil
}

// GetPrometheusTargets retrieves all containers that should be scraped by Prometheus
func (c *Client) GetPrometheusTargets(ctx context.Context) (types.PrometheusSDResponse, error) {
	containers, err := c.ListContainers(ctx)
	if err != nil {
		return nil, err
	}

	var targets types.PrometheusSDResponse
	for _, container := range containers {
		if target := container.ToPrometheusTarget(); target != nil {
			targets = append(targets, *target)
		}
	}

	return targets, nil
}

// GetLabels retrieves all unique label keys and their values
func (c *Client) GetLabels(ctx context.Context) ([]types.LabelInfo, error) {
	// Get all label key patterns
	pattern := labelCacheKey("*")
	keys, err := c.rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list label keys: %w", err)
	}

	labelsKeyPrefix := labelsCacheKey()
	var labels []types.LabelInfo
	for _, key := range keys {
		// Extract label key from Redis key
		labelKey := key[len(labelsKeyPrefix)+1:] // Remove "hermes:labels:" prefix

		// Get all values for this label
		values, err := c.rdb.SMembers(ctx, key).Result()
		if err != nil {
			logrus.Warnf("Failed to get values for label %s: %v", labelKey, err)
			continue
		}

		labels = append(labels, types.LabelInfo{
			Key:    labelKey,
			Values: values,
		})
	}

	return labels, nil
}

// GetLabelValues retrieves all values for a specific label key
func (c *Client) GetLabelValues(ctx context.Context, labelKey string) ([]string, error) {
	key := labelCacheKey(labelKey)
	values, err := c.rdb.SMembers(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to get label values: %w", err)
	}

	return values, nil
}

// GetHosts retrieves all known hosts
func (c *Client) GetHosts(ctx context.Context) ([]string, error) {
	hostsKey := hostsCacheKey()
	hosts, err := c.rdb.SMembers(ctx, hostsKey).Result()
	if err != nil {
		if err == redis.Nil {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to get hosts: %w", err)
	}

	return hosts, nil
}

// CleanupExpired removes expired container data (called periodically)
func (c *Client) CleanupExpired(ctx context.Context) error {
	pattern := containerHostCacheKey("*")
	keys, err := c.rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to list container keys for cleanup: %w", err)
	}

	var expiredKeys []string
	for _, key := range keys {
		ttl, err := c.rdb.TTL(ctx, key).Result()
		if err != nil {
			logrus.Warnf("Failed to get TTL for key %s: %v", key, err)
			continue
		}

		// If TTL is very low (less than 5 seconds), consider it expired
		if ttl < 5*time.Second {
			expiredKeys = append(expiredKeys, key)
		}
	}

	if len(expiredKeys) > 0 {
		_, err = c.rdb.Del(ctx, expiredKeys...).Result()
		if err != nil {
			return fmt.Errorf("failed to delete expired keys: %w", err)
		}
		logrus.Debugf("Cleaned up %d expired container keys", len(expiredKeys))
	}

	return nil
}
