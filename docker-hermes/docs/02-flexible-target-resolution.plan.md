<!-- PLAN_ID_PLACEHOLDER -->
# Flexible Target Resolution for docker-hermes

## Overview

Implement flexible target resolution strategies to support multiple deployment scenarios including overlay networks, bridge networks, and external monitoring systems. This enhancement allows operators to configure how container targets are resolved for Prometheus scraping, with support for per-container overrides and multiple targets per container.

## Problem Statement

The current implementation uses a simple `{hostname}:{port}` format for targets, which doesn't support:
1. **Overlay networks** where container hostnames are resolvable across nodes
2. **Bridge networks** where container IPs are needed for direct access
3. **External monitoring** where published host ports are required
4. **Multi-port services** where a single container exposes multiple scrape targets
5. **Per-container customization** for hybrid deployments

## Architecture Summary

### Resolution Strategies

Four primary strategies with configurable defaults and per-container overrides:

1. **`hostname`** - Use container name/hostname (overlay networks with DNS)
2. **`container_ip`** - Use container's internal IP address (bridge networks)
3. **`host_port`** - Use agent hostname + published port (external monitoring)
4. **`custom`** - Use custom host from configuration (NAT, custom DNS)

### Multi-Target Support

Containers can expose multiple Prometheus targets via indexed labels:
```yaml
prometheus.io/scrape=true
prometheus.io/port=8080
prometheus.io/path=/metrics

prometheus.io/scrape.admin=true
prometheus.io/port.admin=9090
prometheus.io/path.admin=/admin/metrics
prometheus.io/scheme.admin=https
```

### Label-Based Overrides

Per-container resolution control via `hermes.io/*` labels:
```yaml
hermes.io/target-strategy=host_port
hermes.io/target-host=custom-hostname.example.com
hermes.io/target-port=9999
hermes.io/target-port.admin=10000  # Multi-target override
```

## Implementation Plan

### Phase 1: Core Resolution Infrastructure

#### 1.1 Update Type Definitions

**File: `types/resolution.go`** (new)

Define resolution strategy types and constants:

```go
type TargetResolutionStrategy string

const (
    StrategyHostname    TargetResolutionStrategy = "hostname"
    StrategyContainerIP TargetResolutionStrategy = "container_ip"
    StrategyHostPort    TargetResolutionStrategy = "host_port"
    StrategyCustom      TargetResolutionStrategy = "custom"
)

type TargetResolution struct {
    Strategy     TargetResolutionStrategy
    Host         string
    Port         int
    TargetSuffix string // For multi-target (e.g., "admin", "metrics")
}

type ResolvedTarget struct {
    Address string // Final "host:port" format
    Scheme  string
    Path    string
    Labels  map[string]string
}
```

#### 1.2 Extend Configuration

**File: `types/types.go`**

Update `Config` struct and add new LabelsConfig type:
```go
// LabelsConfig holds label prefix configuration shared by agent and server
type LabelsConfig struct {
    LabelPrefix         string `yaml:"label_prefix" envconfig:"LABEL_PREFIX" default:""`
    PrometheusPrefix    string `yaml:"prometheus_prefix" envconfig:"PROMETHEUS_PREFIX" default:"prometheus.io"`
    HermesPrefix        string `yaml:"hermes_prefix" envconfig:"HERMES_PREFIX" default:"hermes.io"`
}

type Config struct {
    config.CommonConfig `yaml:",inline"`
    
    // Shared configuration
    Labels LabelsConfig `yaml:"labels"`
    
    // Agent-specific fields
    DockerSocket      string
    RedisURL          string
    HeartbeatInterval time.Duration
    Hostname          string
    
    // Target resolution configuration
    TargetResolutionStrategy TargetResolutionStrategy `yaml:"target_resolution_strategy" envconfig:"TARGET_RESOLUTION_STRATEGY" default:"hostname"`
    TargetResolutionHost     string                   `yaml:"target_resolution_host" envconfig:"TARGET_RESOLUTION_HOST" default:""`
    
    // Server-specific fields
    HTTPListenAddr   string
    PrometheusSDPath string
}
```

**File: `types/resolution.go`** (new)

Add LabelsConfig to ContainerInfo:
```go
type ContainerInfo struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Host        string            `json:"host"`
    Labels      map[string]string `json:"labels"`
    Ports       []PortInfo        `json:"ports"`
    State       string            `json:"state"`
    LastSeen    time.Time         `json:"last_seen"`
    CreatedAt   time.Time         `json:"created_at"`
    Image       string            `json:"image"`
    Command     string            `json:"command"`
    Status      string            `json:"status"`
    
    // Network information for resolution
    NetworkSettings NetworkSettings `json:"network_settings"`
    
    // NEW: Labels configuration reference
    LabelsConfig *LabelsConfig `json:"-"`
}

#### 1.3 Extend ContainerInfo

**File: `types/types.go`**

Add network information to `ContainerInfo`:
```go
type ContainerInfo struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Host        string            `json:"host"`
    Labels      map[string]string `json:"labels"`
    Ports       []PortInfo        `json:"ports"`
    State       string            `json:"state"`
    LastSeen    time.Time         `json:"last_seen"`
    CreatedAt   time.Time         `json:"created_at"`
    Image       string            `json:"image"`
    Command     string            `json:"command"`
    Status      string            `json:"status"`
    
    // NEW: Network information for resolution
    NetworkSettings NetworkSettings `json:"network_settings"`
}

type NetworkSettings struct {
    IPAddress   string              `json:"ip_address"`    // Primary IP
    Networks    map[string]Network  `json:"networks"`      // All networks
}

type Network struct {
    IPAddress   string `json:"ip_address"`
    Gateway     string `json:"gateway"`
    NetworkID   string `json:"network_id"`
    NetworkName string `json:"network_name"`
}
```

### Phase 2: Target Resolution Logic

#### 2.1 Implement Resolution Functions

**File: `types/resolution.go`**

Core resolution logic:

```go
// ResolveTargets resolves all Prometheus targets for a container
func (c *ContainerInfo) ResolveTargets(defaultStrategy TargetResolutionStrategy, agentHost, customHost string) []ResolvedTarget {
    targets := []ResolvedTarget{}
    
    // Find all prometheus.io/scrape labels (including indexed)
    scrapeTargets := c.FindScrapeTargets()
    
    for _, targetSuffix := range scrapeTargets {
        if target := c.resolveTarget(targetSuffix, defaultStrategy, agentHost, customHost); target != nil {
            targets = append(targets, *target)
        }
    }
    
    return targets
}

// FindScrapeTargets identifies all scrape target suffixes in labels
func (c *ContainerInfo) FindScrapeTargets() []string {
    suffixes := []string{""}  // Default (no suffix)
    
    scrapeKey := c.LabelsConfig.PrometheusPrefix + "/scrape"
    
    for key, value := range c.Labels {
        if strings.HasPrefix(key, scrapeKey+".") && value == "true" {
            suffix := strings.TrimPrefix(key, scrapeKey+".")
            suffixes = append(suffixes, suffix)
        }
    }
    
    return suffixes
}

// resolveTarget resolves a single target
func (c *ContainerInfo) resolveTarget(suffix string, defaultStrategy TargetResolutionStrategy, agentHost, customHost string) *ResolvedTarget {
    // Check if scraping is enabled for this target
    if !c.isScrapeEnabled(suffix) {
        return nil
    }
    
    // Determine resolution strategy
    strategy := c.getTargetStrategy(suffix, defaultStrategy)
    
    // Get port
    port := c.getTargetPort(suffix)
    if port == 0 {
        return nil
    }
    
    // Resolve host based on strategy
    host := c.resolveHost(suffix, strategy, agentHost, customHost)
    if host == "" {
        return nil
    }
    
    // Adjust port for host_port strategy
    if strategy == StrategyHostPort {
        if publishedPort := c.getPublishedPort(port); publishedPort > 0 {
            port = publishedPort
        } else {
            // No published port available
            return nil
        }
    }
    
    return &ResolvedTarget{
        Address: fmt.Sprintf("%s:%d", host, port),
        Scheme:  c.getTargetScheme(suffix),
        Path:    c.getTargetPath(suffix),
        Labels:  c.getTargetLabels(suffix),
    }
}

// isScrapeEnabled checks if scraping is enabled
func (c *ContainerInfo) isScrapeEnabled(suffix string) bool {
    key := c.LabelsConfig.PrometheusPrefix + "/scrape"
    if suffix != "" {
        key = fmt.Sprintf("%s/scrape.%s", c.LabelsConfig.PrometheusPrefix, suffix)
    }
    
    value, exists := c.Labels[key]
    return exists && value == "true"
}

// getTargetStrategy determines the resolution strategy
func (c *ContainerInfo) getTargetStrategy(suffix string, defaultStrategy TargetResolutionStrategy) TargetResolutionStrategy {
    // Check for per-target override
    key := c.LabelsConfig.HermesPrefix + "/target-strategy"
    if suffix != "" {
        key = fmt.Sprintf("%s/target-strategy.%s", c.LabelsConfig.HermesPrefix, suffix)
        if strategy, exists := c.Labels[key]; exists {
            return TargetResolutionStrategy(strategy)
        }
    }
    
    // Check for global hermes override
    if strategy, exists := c.Labels[c.LabelsConfig.HermesPrefix+"/target-strategy"]; exists {
        return TargetResolutionStrategy(strategy)
    }
    
    return defaultStrategy
}

// resolveHost resolves the host based on strategy
func (c *ContainerInfo) resolveHost(suffix string, strategy TargetResolutionStrategy, agentHost, customHost string) string {
    // Check for explicit host override
    key := c.LabelsConfig.HermesPrefix + "/target-host"
    if suffix != "" {
        suffixKey := fmt.Sprintf("%s/target-host.%s", c.LabelsConfig.HermesPrefix, suffix)
        if host, exists := c.Labels[suffixKey]; exists {
            return host
        }
    }
    if host, exists := c.Labels[key]; exists {
        return host
    }
    
    // Resolve based on strategy
    switch strategy {
    case StrategyHostname:
        return c.Name
    case StrategyContainerIP:
        return c.GetPrimaryIP()
    case StrategyHostPort:
        if customHost != "" {
            return customHost
        }
        return agentHost
    case StrategyCustom:
        return customHost
    default:
        return c.Name
    }
}

// getTargetPort gets the port for a target
func (c *ContainerInfo) getTargetPort(suffix string) int {
    // Check for hermes override first
    hermesKey := c.LabelsConfig.HermesPrefix + "/target-port"
    if suffix != "" {
        suffixKey := fmt.Sprintf("%s/target-port.%s", c.LabelsConfig.HermesPrefix, suffix)
        if portStr, exists := c.Labels[suffixKey]; exists {
            if port, err := strconv.Atoi(portStr); err == nil {
                return port
            }
        }
    }
    if portStr, exists := c.Labels[hermesKey]; exists {
        if port, err := strconv.Atoi(portStr); err == nil {
            return port
        }
    }
    
    // Check prometheus.io/port
    promKey := c.LabelsConfig.PrometheusPrefix + "/port"
    if suffix != "" {
        promKey = fmt.Sprintf("%s/port.%s", c.LabelsConfig.PrometheusPrefix, suffix)
    }
    if portStr, exists := c.Labels[promKey]; exists {
        if port, err := strconv.Atoi(portStr); err == nil {
            return port
        }
    }
    
    // Default to first exposed port
    if len(c.Ports) > 0 {
        return c.Ports[0].PrivatePort
    }
    
    return 0
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

// getTargetScheme gets the scheme (http/https)
func (c *ContainerInfo) getTargetScheme(suffix string) string {
    key := c.LabelsConfig.PrometheusPrefix + "/scheme"
    if suffix != "" {
        key = fmt.Sprintf("%s/scheme.%s", c.LabelsConfig.PrometheusPrefix, suffix)
    }
    
    if scheme, exists := c.Labels[key]; exists {
        return scheme
    }
    return "http"
}

// getTargetPath gets the metrics path
func (c *ContainerInfo) getTargetPath(suffix string) string {
    key := c.LabelsConfig.PrometheusPrefix + "/path"
    if suffix != "" {
        key = fmt.Sprintf("%s/path.%s", c.LabelsConfig.PrometheusPrefix, suffix)
    }
    
    if path, exists := c.Labels[key]; exists {
        return path
    }
    return "/metrics"
}

// getTargetLabels builds labels for the target
func (c *ContainerInfo) getTargetLabels(suffix string) map[string]string {
    labels := make(map[string]string)
    
    // Add meta labels
    labels["__meta_docker_container_id"] = c.ID
    labels["__meta_docker_container_name"] = c.Name
    labels["__meta_docker_container_image"] = c.Image
    labels["__meta_docker_host"] = c.Host
    
    if suffix != "" {
        labels["__meta_target_suffix"] = suffix
    }
    
    // Copy labels, excluding prometheus and hermes prefixes
    for key, value := range c.Labels {
        if !strings.HasPrefix(key, c.LabelsConfig.PrometheusPrefix+"/") && 
           !strings.HasPrefix(key, c.LabelsConfig.HermesPrefix+"/") {
            labels[key] = value
        }
    }
    
    return labels
}

// GetPrimaryIP returns the primary IP address
func (c *ContainerInfo) GetPrimaryIP() string {
    if c.NetworkSettings.IPAddress != "" {
        return c.NetworkSettings.IPAddress
    }
    
    // Fall back to first network IP
    for _, network := range c.NetworkSettings.Networks {
        if network.IPAddress != "" {
            return network.IPAddress
        }
    }
    
    return ""
}
```

### Phase 3: Agent Updates

#### 3.1 Enhanced Docker Information Extraction

**File: `agent/docker.go`**

Update `extractContainerInfoFromInspect()` to capture network details:

```go
func (d *DockerClient) extractContainerInfoFromInspect(inspect types.ContainerJSON, hostname string) (*hermesTypes.ContainerInfo, error) {
    // ... existing port extraction ...
    
    // NEW: Extract network settings
    networkSettings := hermesTypes.NetworkSettings{
        IPAddress: inspect.NetworkSettings.IPAddress,
        Networks:  make(map[string]hermesTypes.Network),
    }
    
    for netName, netConfig := range inspect.NetworkSettings.Networks {
        networkSettings.Networks[netName] = hermesTypes.Network{
            IPAddress:   netConfig.IPAddress,
            Gateway:     netConfig.Gateway,
            NetworkID:   netConfig.NetworkID,
            NetworkName: netName,
        }
    }
    
    containerInfo := &hermesTypes.ContainerInfo{
        // ... existing fields ...
        NetworkSettings: networkSettings,
    }
    
    return containerInfo, nil
}
```

#### 3.2 Update Reporter

**File: `agent/reporter.go`**

Pass resolution configuration to reporter:

```go
type Reporter struct {
    redisClient  *redis.Client
    
    // Resolution configuration
    resolutionStrategy hermesTypes.TargetResolutionStrategy
    agentHost          string
    customHost         string
    
    // Labels configuration
    labelsConfig *hermesTypes.LabelsConfig
}

func NewReporter(redisClient *redis.Client, config *hermesTypes.Config) *Reporter {
    return &Reporter{
        redisClient:        redisClient,
        resolutionStrategy: config.TargetResolutionStrategy,
        agentHost:          config.Hostname,
        customHost:         config.TargetResolutionHost,
        labelsConfig:       &config.Labels,
    }
}

func (r *Reporter) ReportContainer(ctx context.Context, container *hermesTypes.ContainerInfo) error {
    // Attach labels config to container
    container.LabelsConfig = r.labelsConfig
    
    // Filter labels by prefix if specified
    if r.labelsConfig.LabelPrefix != "" {
        container.Labels = container.FilterLabelsByPrefix(r.labelsConfig.LabelPrefix)
    }
    
    // Pre-resolve targets for validation
    targets := container.ResolveTargets(r.resolutionStrategy, r.agentHost, r.customHost)
    if len(targets) == 0 {
        logrus.Debugf("Container %s has no valid targets, skipping", container.ID)
        return nil
    }
    
    // Log resolved targets
    for i, target := range targets {
        logrus.Debugf("Container %s target %d: %s%s%s", 
            container.ID, i, target.Scheme, target.Address, target.Path)
    }
    
    // ... existing store logic ...
}
```

### Phase 4: Server Updates

#### 4.1 Update Prometheus SD Generation

**File: `server/prometheus.go`** or **`types/prometheus.go`**

Update `ToPrometheusTarget()` to use resolution:

```go
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
```

**File: `redis/operations.go`**

Update `GetPrometheusTargets()` to handle multi-target:

```go
func (c *Client) GetPrometheusTargets(ctx context.Context, strategy types.TargetResolutionStrategy, agentHost, customHost string, labelsConfig *types.LabelsConfig) (types.PrometheusSDResponse, error) {
    containers, err := c.ListContainers(ctx)
    if err != nil {
        return nil, err
    }

    var targets types.PrometheusSDResponse
    for _, container := range containers {
        // Attach labels config to containers retrieved from Redis (they won't have it)
        container.LabelsConfig = labelsConfig
        
        containerTargets := container.ToPrometheusTargets(strategy, agentHost, customHost)
        targets = append(targets, containerTargets...)
    }

    return targets, nil
}
```

**File: `server/server.go`**

Update Prometheus SD endpoint to pass resolution config:

```go
func (s *Server) registerPrometheusSD(apiContext *types.ApiContext) error {
    s.router.GET(s.config.PrometheusSDPath, func(c *gin.Context) {
        // Pass labels config to Redis operations
        targets, err := s.redisClient.GetPrometheusTargets(
            c.Request.Context(),
            s.config.TargetResolutionStrategy,
            s.config.Hostname,
            s.config.TargetResolutionHost,
            &s.config.Labels,
        )
        if err != nil {
            logrus.Errorf("Failed to get Prometheus targets: %v", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get targets"})
            return
        }

        c.JSON(http.StatusOK, targets)
    })

    return nil
}
```

### Phase 5: Documentation and Examples

#### 5.1 Update Configuration Example

**File: `config.yml.example`**

```yaml
# Label configuration (shared by agent and server)
labels:
  # Filter label prefix for agent (empty = all labels)
  label_prefix: ""
  
  # Label prefixes for Prometheus and Hermes labels
  prometheus_prefix: "prometheus.io"  # Default: prometheus.io
  hermes_prefix: "hermes.io"          # Default: hermes.io

# Target resolution strategy
# Options: hostname, container_ip, host_port, custom
target_resolution_strategy: "hostname"

# Custom host for resolution (used with 'custom' or 'host_port' strategies)
# For 'host_port': uses this instead of hostname for external access
# For 'custom': always uses this as the target host
target_resolution_host: ""

# Examples by use case:
# 
# Overlay network (Docker Swarm, Kubernetes):
#   target_resolution_strategy: "hostname"
#
# Bridge network with routable IPs:
#   target_resolution_strategy: "container_ip"
#
# External monitoring via published ports:
#   target_resolution_strategy: "host_port"
#   target_resolution_host: "192.168.1.10"  # optional override
#
# Custom DNS/NAT scenario:
#   target_resolution_strategy: "custom"
#   target_resolution_host: "docker-host.example.com"
```

#### 5.2 Create Resolution Guide

**File: `docs/target-resolution.md`**

Comprehensive guide covering:
- All resolution strategies with examples
- Multi-target configuration
- Per-container label overrides
- Use case scenarios
- Troubleshooting

### Phase 6: Testing

#### 6.1 Unit Tests

**File: `types/resolution_test.go`** (new)

Test cases:
- Each resolution strategy
- Multi-target discovery
- Label override precedence
- Edge cases (no ports, no IPs, etc.)

#### 6.2 Integration Tests

**File: `integration/docker-compose.yml`**

Add test scenarios:
- Container with multiple prometheus.io/scrape targets
- Containers with different resolution strategies
- Containers with label overrides

### Phase 7: Migration and Validation

#### 7.1 Backward Compatibility

Ensure existing deployments continue to work:
- Default strategy is `hostname` (current behavior)
- Existing label handling unchanged
- No breaking changes to Redis data format

#### 7.2 Validation

- Test all strategies in integration environment
- Verify multi-target Prometheus scraping
- Validate label override behavior
- Performance testing with many containers

## Implementation Checklist

### Phase 1: Core Infrastructure
- [ ] Create `types/resolution.go` with strategy types
- [ ] Create `LabelsConfig` struct in `types/types.go` for shared label configuration
- [ ] Update `types/types.go` with nested `Labels` field in `Config`
- [ ] Extend `ContainerInfo` with `NetworkSettings` and `LabelsConfig` reference
- [ ] Update `PortInfo` if needed for published port tracking

### Phase 2: Resolution Logic
- [ ] Implement `ResolveTargets()` function
- [ ] Implement `FindScrapeTargets()` for multi-target discovery
- [ ] Implement `resolveTarget()` with strategy logic
- [ ] Implement helper functions (`isScrapeEnabled`, `getTargetStrategy`, etc.)
- [ ] Implement `GetPrimaryIP()` for container IP resolution

### Phase 3: Agent Updates
- [ ] Update `agent/docker.go` to extract network settings and attach `LabelsConfig`
- [ ] Update `agent/reporter.go` to store `LabelsConfig` and attach to containers
- [ ] Update `agent/agent.go` to pass config to reporter
- [ ] Add logging for resolved targets
- [ ] Ensure `LabelsConfig` is attached to containers before reporting

### Phase 4: Server Updates
- [ ] Update `ToPrometheusTargets()` to support multi-target (no prefix parameters)
- [ ] Update `redis/operations.go` GetPrometheusTargets to accept `LabelsConfig`
- [ ] Update `server/server.go` Prometheus SD endpoint to pass `LabelsConfig`
- [ ] Attach `LabelsConfig` to containers retrieved from Redis
- [ ] Remove prefix parameters from `ContainerInfo` method signatures

### Phase 5: Documentation
- [ ] Update `config.yml.example` with resolution options and label prefixes
- [ ] Create `docs/target-resolution.md` guide
- [ ] Update README with resolution feature overview
- [ ] Document configurable label prefixes and label override behavior

### Phase 6: Testing
- [ ] Write unit tests for resolution logic
- [ ] Write unit tests for multi-target discovery
- [ ] Add integration test scenarios
- [ ] Test each resolution strategy
- [ ] Test label override precedence

### Phase 7: Validation
- [ ] Verify backward compatibility
- [ ] Test with existing deployments
- [ ] Performance testing
- [ ] Update integration test suite

## Label Reference

### Note on Label Prefixes

Both the Prometheus and Hermes label prefixes are configurable via the `prometheus_label_prefix` and `hermes_label_prefix` configuration options. The examples below use the default prefixes (`prometheus.io` and `hermes.io`), but these can be customized in your configuration.

### Standard Prometheus Labels

```yaml
# Single target (default)
# Note: prefix is configurable via prometheus_label_prefix config
prometheus.io/scrape: "true"
prometheus.io/port: "8080"
prometheus.io/path: "/metrics"
prometheus.io/scheme: "http"

# Multiple targets (indexed)
prometheus.io/scrape.admin: "true"
prometheus.io/port.admin: "9090"
prometheus.io/path.admin: "/admin/metrics"
prometheus.io/scheme.admin: "https"
```

### Hermes Resolution Labels

```yaml
# Global strategy override
# Note: prefix is configurable via hermes_label_prefix config
hermes.io/target-strategy: "host_port"  # hostname, container_ip, host_port, custom

# Global host override
hermes.io/target-host: "custom-host.example.com"

# Global port override (takes precedence over prometheus.io/port)
hermes.io/target-port: "8080"

# Per-target overrides
hermes.io/target-strategy.admin: "container_ip"
hermes.io/target-host.admin: "admin.example.com"
hermes.io/target-port.admin: "9090"
```

### Custom Label Prefix Examples

If you set custom prefixes in configuration:

```yaml
# config.yml
prometheus_label_prefix: "custom.monitoring"
hermes_label_prefix: "hermes.service"
```

Then your labels would be:

```yaml
custom.monitoring/scrape: "true"
custom.monitoring/port: "8080"
hermes.service/target-strategy: "hostname"
hermes.service/target-host: "custom-host"
```

## Resolution Strategy Decision Tree

```
For each container target:
  Note: Label prefixes are configurable (default: prometheus.io, hermes.io)
  
  1. Is {prometheus_prefix}/scrape[.suffix] == "true"? → No: Skip
  2. Check {hermes_prefix}/target-strategy[.suffix] → Use if present
  3. Else use agent's default strategy
  4. Check {hermes_prefix}/target-host[.suffix] → Use if present
  5. Else resolve host based on strategy:
     - hostname: Use container name
     - container_ip: Use container's primary IP
     - host_port: Use agent hostname (or custom)
     - custom: Use configured custom host
  6. Check {hermes_prefix}/target-port[.suffix] → Use if present
  7. Else use {prometheus_prefix}/port[.suffix] → Use if present
  8. Else use first exposed port → Use if present
  9. Else skip target (no valid port)
  10. If strategy is host_port: lookup published port → Skip if not found
  11. Return resolved target
```

## Future Enhancements (Not in Scope)

- DNS validation/lookup at registration time
- Automatic fallback chains with testing
- Network-specific resolution strategies
- Dynamic strategy selection based on network topology
- IPv6 support
- Multiple host strategies (primary/fallback hosts)
