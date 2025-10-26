package types

import "fmt"

type CacheKey = string

// ContainerHostCacheKey returns the Redis key for a container host
func ContainerHostCacheKey(host string) CacheKey {
	return "hermes:containers:" + host
}

// ContainerCacheKey returns the Redis key for a container
func ContainerCacheKey(host, containerID string) CacheKey {
	return fmt.Sprintf("%s:%s", ContainerHostCacheKey(host), containerID)
}

// StreamCacheKey returns the Redis stream key for updates
func StreamCacheKey() CacheKey {
	return "hermes:stream:updates"
}

// LabelsCacheKey returns the Redis key for label metadata
func LabelsCacheKey() CacheKey {
	return "hermes:labels"
}

// LabelCacheKey returns the Redis key for a label
func LabelCacheKey(labelKey string) CacheKey {
	return fmt.Sprintf("%s:%s", LabelsCacheKey(), labelKey)
}

// HostsCacheKey returns the Redis key for host metadata
func HostsCacheKey() CacheKey {
	return "hermes:hosts"
}
