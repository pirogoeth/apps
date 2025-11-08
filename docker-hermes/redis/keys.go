package redis

import "fmt"

type cacheKey = string

// containerHostCacheKey returns the Redis key for a container host
func containerHostCacheKey(host string) cacheKey {
	return "hermes:containers:" + host
}

// containerCacheKey returns the Redis key for a container
func containerCacheKey(host, containerID string) cacheKey {
	return fmt.Sprintf("%s:%s", containerHostCacheKey(host), containerID)
}

// streamCacheKey returns the Redis stream key for updates
func streamCacheKey() cacheKey {
	return "hermes:stream:updates"
}

// labelsCacheKey returns the Redis key for label metadata
func labelsCacheKey() cacheKey {
	return "hermes:labels"
}

// labelCacheKey returns the Redis key for a label
func labelCacheKey(labelKey string) cacheKey {
	return fmt.Sprintf("%s:%s", labelsCacheKey(), labelKey)
}

// hostsCacheKey returns the Redis key for host metadata
func hostsCacheKey() cacheKey {
	return "hermes:hosts"
}



