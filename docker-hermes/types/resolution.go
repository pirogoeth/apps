package types

// TargetResolutionStrategy defines how to resolve container targets
type TargetResolutionStrategy string

const (
	StrategyHostname    TargetResolutionStrategy = "hostname"
	StrategyContainerIP TargetResolutionStrategy = "container_ip"
	StrategyHostPort    TargetResolutionStrategy = "host_port"
	StrategyCustom      TargetResolutionStrategy = "custom"
)

// ResolvedTarget represents a resolved Prometheus scrape target
type ResolvedTarget struct {
	Address string            // Final "host:port" format
	Scheme  string            // http or https
	Path    string            // Metrics path
	Labels  map[string]string // Labels for the target
}

// ResolveTargets resolves all Prometheus targets for a container
func (c *ContainerInfo) ResolveTargets(defaultStrategy TargetResolutionStrategy, agentHost, customHost string) []ResolvedTarget {
	if c.LabelsConfig == nil {
		return nil
	}

	// Create extractor with configured prefix
	extractor := NewTargetExtractor(c.LabelsConfig.Prefix)

	// Extract all target configurations
	targetConfigs := extractor.ExtractTargets(c)

	// Resolve each target
	resolved := []ResolvedTarget{}
	for _, targetConfig := range targetConfigs {
		if resolvedTarget := targetConfig.ResolveTarget(c, defaultStrategy, agentHost, customHost); resolvedTarget != nil {
			resolved = append(resolved, *resolvedTarget)
		}
	}

	return resolved
}

// ToPrometheusTargets converts ContainerInfo to Prometheus targets
func (c *ContainerInfo) ToPrometheusTargets(strategy TargetResolutionStrategy, agentHost, customHost string) []PrometheusTarget {
	resolvedTargets := c.ResolveTargets(strategy, agentHost, customHost)

	promTargets := make([]PrometheusTarget, 0, len(resolvedTargets))
	for _, resolved := range resolvedTargets {
		promTargets = append(promTargets, PrometheusTarget{
			Targets: []string{resolved.Address},
			Labels:  resolved.Labels,
		})
	}

	return promTargets
}
