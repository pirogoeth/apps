package types

import (
	"fmt"
	"strconv"
	"strings"
)

// TargetConfig represents a single scrape target configuration
//
// Targets can be defined in two ways:
//
//  1. Named targets (structured): hermes.targets.{name}.prometheus.*
//     Example:
//     hermes.targets.admin.prometheus.scrape=true
//     hermes.targets.admin.prometheus.port=9090
//     hermes.targets.admin.strategy=container_ip
//
//  2. Shorthand (default target): hermes.prometheus.*
//     Example:
//     hermes.prometheus.scrape=true
//     hermes.prometheus.port=8080
//     The target name is automatically derived from container details
//     (format: {host}-{container-name} or similar)
type TargetConfig struct {
	Name     string                   // Target identifier (e.g., "admin", or derived from container)
	Scrape   bool                     // Whether scraping is enabled
	Port     int                      // Port to scrape
	Path     string                   // Metrics path (default: /metrics)
	Scheme   string                   // http or https (default: http)
	Strategy TargetResolutionStrategy // Resolution strategy override (empty = use default)
	Host     string                   // Custom host override (empty = use strategy)
	Labels   map[string]string        // Additional labels for this target
}

// TargetExtractor extracts and validates target configurations from container labels
type TargetExtractor struct {
	prefix string
}

// NewTargetExtractor creates a new target extractor
func NewTargetExtractor(prefix string) *TargetExtractor {
	if prefix == "" {
		prefix = "hermes"
	}
	return &TargetExtractor{prefix: prefix}
}

// ExtractTargets extracts all target configurations from container labels
func (e *TargetExtractor) ExtractTargets(container *ContainerInfo) []TargetConfig {
	if e.prefix == "" {
		return nil
	}

	targets := []TargetConfig{}

	// Discover all named targets: hermes.targets.{name}.prometheus.scrape
	namedTargets := e.discoverNamedTargets(container.Labels)
	for _, name := range namedTargets {
		if config := e.extractTargetConfig(container, name, true); config != nil {
			targets = append(targets, *config)
		}
	}

	// Check for shorthand: hermes.prometheus.scrape (derives target name)
	if e.hasShorthandTarget(container.Labels) {
		derivedName := e.deriveTargetName(container)
		if config := e.extractTargetConfig(container, derivedName, false); config != nil {
			targets = append(targets, *config)
		}
	}

	return targets
}

// discoverNamedTargets finds all target names from labels
func (e *TargetExtractor) discoverNamedTargets(labels map[string]string) []string {
	prefix := e.prefix + ".targets."
	suffix := ".prometheus.scrape"

	targetNames := []string{}
	seen := make(map[string]bool)

	for key, value := range labels {
		if strings.HasPrefix(key, prefix) && strings.HasSuffix(key, suffix) && value == "true" {
			// Extract target name: prefix + name + suffix
			name := key[len(prefix) : len(key)-len(suffix)]
			if name != "" && !seen[name] {
				targetNames = append(targetNames, name)
				seen[name] = true
			}
		}
	}

	return targetNames
}

// hasShorthandTarget checks if shorthand format exists
func (e *TargetExtractor) hasShorthandTarget(labels map[string]string) bool {
	key := e.prefix + ".prometheus.scrape"
	if value, exists := labels[key]; exists && value == "true" {
		return true
	}
	return false
}

// deriveTargetName creates a target identifier from container details
func (e *TargetExtractor) deriveTargetName(container *ContainerInfo) string {
	// Use host-container-name format for uniqueness
	if container.Host != "" && container.Name != "" {
		return fmt.Sprintf("%s-%s", container.Host, strings.ReplaceAll(container.Name, "/", "-"))
	}
	if container.Name != "" {
		return strings.ReplaceAll(container.Name, "/", "-")
	}
	if container.Image != "" {
		// Extract image name (remove tag and registry)
		image := container.Image
		if idx := strings.LastIndex(image, "/"); idx >= 0 {
			image = image[idx+1:]
		}
		if idx := strings.Index(image, ":"); idx >= 0 {
			image = image[:idx]
		}
		return image
	}
	// Fallback to container ID (short)
	if len(container.ID) > 12 {
		return container.ID[:12]
	}
	return container.ID
}

// deriveScopedTargetName creates a scoped target identifier from container details and a raw target name
func (e *TargetExtractor) deriveScopedTargetName(container *ContainerInfo, rawTargetName string) string {
	// Build base identifier from container (same logic as deriveTargetName)
	var base string
	if container.Host != "" && container.Name != "" {
		base = fmt.Sprintf("%s-%s", container.Host, strings.ReplaceAll(container.Name, "/", "-"))
	} else if container.Name != "" {
		base = strings.ReplaceAll(container.Name, "/", "-")
	} else if container.Image != "" {
		// Extract image name (remove tag and registry)
		image := container.Image
		if idx := strings.LastIndex(image, "/"); idx >= 0 {
			image = image[idx+1:]
		}
		if idx := strings.Index(image, ":"); idx >= 0 {
			image = image[:idx]
		}
		base = image
	} else {
		// Fallback to container ID (short)
		if len(container.ID) > 12 {
			base = container.ID[:12]
		} else {
			base = container.ID
		}
	}

	// Append the raw target name: {base}-{rawTargetName}
	return fmt.Sprintf("%s-%s", base, rawTargetName)
}

// extractTargetConfig extracts configuration for a specific target
func (e *TargetExtractor) extractTargetConfig(container *ContainerInfo, targetName string, isNamed bool) *TargetConfig {
	var basePath string
	var finalTargetName string

	if isNamed {
		// For named targets, use raw name for label lookup but derive scoped name
		basePath = fmt.Sprintf("%s.targets.%s", e.prefix, targetName)
		finalTargetName = e.deriveScopedTargetName(container, targetName)
	} else {
		// Shorthand: hermes.prometheus.*
		basePath = e.prefix
		finalTargetName = targetName // Already derived in ExtractTargets
	}

	config := &TargetConfig{
		Name:   finalTargetName, // Use scoped name
		Scheme: "http",
		Path:   "/metrics",
		Labels: make(map[string]string),
	}

	// Check if scraping is enabled
	scrapeKey := basePath + ".prometheus.scrape"
	if value, exists := container.Labels[scrapeKey]; !exists || value != "true" {
		return nil // Not enabled
	}
	config.Scrape = true

	// Extract port (same pattern for both named and shorthand)
	portKey := basePath + ".prometheus.port"
	if portStr, exists := container.Labels[portKey]; exists {
		if port, err := strconv.Atoi(portStr); err == nil {
			config.Port = port
		}
	}

	// Extract path
	pathKey := basePath + ".prometheus.path"
	if path, exists := container.Labels[pathKey]; exists {
		config.Path = path
	}

	// Extract scheme
	schemeKey := basePath + ".prometheus.scheme"
	if scheme, exists := container.Labels[schemeKey]; exists {
		config.Scheme = scheme
	}

	// Extract Hermes-specific overrides (only for named targets)
	if isNamed {
		// Strategy override
		strategyKey := basePath + ".strategy"
		if strategy, exists := container.Labels[strategyKey]; exists {
			config.Strategy = TargetResolutionStrategy(strategy)
		}

		// Host override
		hostKey := basePath + ".host"
		if host, exists := container.Labels[hostKey]; exists {
			config.Host = host
		}

		// Port override (for resolution, different from prometheus port)
		resolutionPortKey := basePath + ".port"
		if portStr, exists := container.Labels[resolutionPortKey]; exists {
			if port, err := strconv.Atoi(portStr); err == nil {
				config.Port = port // Override prometheus port
			}
		}
	}

	// If port not set, use first exposed port
	if config.Port == 0 {
		if len(container.Ports) > 0 {
			config.Port = container.Ports[0].PrivatePort
		} else {
			return nil // No port available
		}
	}

	return config
}

// ResolveTarget resolves a target configuration to a ResolvedTarget
func (c *TargetConfig) ResolveTarget(container *ContainerInfo, defaultStrategy TargetResolutionStrategy, agentHost, customHost string) *ResolvedTarget {
	if !c.Scrape {
		return nil
	}

	if c.Port == 0 {
		return nil
	}

	// Determine strategy
	strategy := c.Strategy
	if strategy == "" {
		strategy = defaultStrategy
	}

	// Resolve host
	host := c.resolveHost(container, strategy, agentHost, customHost)
	if host == "" {
		return nil
	}

	// Adjust port for host_port strategy
	port := c.Port
	if strategy == StrategyHostPort {
		if publishedPort := container.getPublishedPort(port); publishedPort > 0 {
			port = publishedPort
		} else {
			return nil // No published port available
		}
	}

	// Build labels
	labels := make(map[string]string)
	labels["__meta_docker_container_id"] = container.ID
	labels["__meta_docker_container_name"] = container.Name
	labels["__meta_docker_container_image"] = container.Image
	labels["__meta_docker_container_status"] = container.Status
	labels["__meta_docker_host"] = container.Host
	labels["__meta_target_name"] = c.Name

	// Add target-specific labels
	for key, value := range c.Labels {
		labels[key] = value
	}

	// Copy non-hermes labels from container
	if container.LabelsConfig != nil {
		prefix := container.LabelsConfig.Prefix + "."
		for key, value := range container.Labels {
			if !strings.HasPrefix(key, prefix) {
				labels[key] = value
			}
		}
	} else {
		// Fallback: copy all labels if no config
		for key, value := range container.Labels {
			labels[key] = value
		}
	}

	return &ResolvedTarget{
		Address: fmt.Sprintf("%s:%d", host, port),
		Scheme:  c.Scheme,
		Path:    c.Path,
		Labels:  labels,
	}
}

// resolveHost resolves the host based on strategy
func (c *TargetConfig) resolveHost(container *ContainerInfo, strategy TargetResolutionStrategy, agentHost, customHost string) string {
	// Check for explicit host override
	if c.Host != "" {
		return c.Host
	}

	// Resolve based on strategy
	switch strategy {
	case StrategyHostname:
		return container.Name
	case StrategyContainerIP:
		return container.GetPrimaryIP()
	case StrategyHostPort:
		if customHost != "" {
			return customHost
		}
		return agentHost
	case StrategyCustom:
		return customHost
	default:
		return container.Name
	}
}

// getPublishedPort finds the published port for a private port
func (c *ContainerInfo) getPublishedPort(privatePort int) int {
	for _, port := range c.Ports {
		if port.PrivatePort == privatePort && port.PublicPort > 0 {
			return port.PublicPort
		}
	}
	return 0
}
