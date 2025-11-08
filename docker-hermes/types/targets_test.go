package types

import (
	"strings"
	"testing"
)

// createTestContainer creates a ContainerInfo with common test data
func createTestContainer() *ContainerInfo {
	return &ContainerInfo{
		ID:     "abc123def456",
		Name:   "/test-container",
		Host:   "host1",
		Image:  "nginx:latest",
		Status: "running",
		Labels: make(map[string]string),
		Ports:  []PortInfo{{PrivatePort: 8080, PublicPort: 8080, Type: "tcp"}},
		NetworkSettings: NetworkSettings{
			IPAddress: "172.17.0.2",
			Networks: map[string]Network{
				"bridge": {
					IPAddress:   "172.17.0.2",
					NetworkName: "bridge",
				},
			},
		},
		LabelsConfig: &LabelsConfig{
			Prefix: "hermes",
		},
	}
}

func TestNewTargetExtractor(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		expected string
	}{
		{"default prefix", "", "hermes"},
		{"custom prefix", "custom", "custom"},
		{"explicit hermes", "hermes", "hermes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewTargetExtractor(tt.prefix)
			if extractor.prefix != tt.expected {
				t.Errorf("expected prefix %q, got %q", tt.expected, extractor.prefix)
			}
		})
	}
}

func TestExtractTargets_Shorthand(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected int
		validate func(*testing.T, []TargetConfig)
	}{
		{
			name: "minimal shorthand",
			labels: map[string]string{
				"hermes.prometheus.scrape": "true",
				"hermes.prometheus.port":   "9090",
			},
			expected: 1,
			validate: func(t *testing.T, targets []TargetConfig) {
				if len(targets) != 1 {
					t.Fatalf("expected 1 target, got %d", len(targets))
				}
				target := targets[0]
				if target.Port != 9090 {
					t.Errorf("expected port 9090, got %d", target.Port)
				}
				if target.Scheme != "http" {
					t.Errorf("expected default scheme http, got %q", target.Scheme)
				}
				if target.Path != "/metrics" {
					t.Errorf("expected default path /metrics, got %q", target.Path)
				}
				if !target.Scrape {
					t.Error("expected scrape to be true")
				}
			},
		},
		{
			name: "shorthand with all options",
			labels: map[string]string{
				"hermes.prometheus.scrape": "true",
				"hermes.prometheus.port":   "8080",
				"hermes.prometheus.path":   "/custom/metrics",
				"hermes.prometheus.scheme": "https",
			},
			expected: 1,
			validate: func(t *testing.T, targets []TargetConfig) {
				target := targets[0]
				if target.Port != 8080 {
					t.Errorf("expected port 8080, got %d", target.Port)
				}
				if target.Scheme != "https" {
					t.Errorf("expected scheme https, got %q", target.Scheme)
				}
				if target.Path != "/custom/metrics" {
					t.Errorf("expected path /custom/metrics, got %q", target.Path)
				}
			},
		},
		{
			name: "shorthand with port from container",
			labels: map[string]string{
				"hermes.prometheus.scrape": "true",
				// No port specified, should use first container port
			},
			expected: 1,
			validate: func(t *testing.T, targets []TargetConfig) {
				target := targets[0]
				if target.Port != 8080 {
					t.Errorf("expected port 8080 from container, got %d", target.Port)
				}
			},
		},
		{
			name: "shorthand disabled",
			labels: map[string]string{
				"hermes.prometheus.scrape": "false",
				"hermes.prometheus.port":   "9090",
			},
			expected: 0,
		},
		{
			name:     "no shorthand labels",
			labels:   map[string]string{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := createTestContainer()
			container.Labels = tt.labels

			extractor := NewTargetExtractor("hermes")
			targets := extractor.ExtractTargets(container)

			if len(targets) != tt.expected {
				t.Fatalf("expected %d targets, got %d", tt.expected, len(targets))
			}

			if tt.validate != nil {
				tt.validate(t, targets)
			}
		})
	}
}

func TestExtractTargets_NamedTargets(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected int
		validate func(*testing.T, []TargetConfig)
	}{
		{
			name: "single named target",
			labels: map[string]string{
				"hermes.targets.admin.prometheus.scrape": "true",
				"hermes.targets.admin.prometheus.port":   "9090",
			},
			expected: 1,
			validate: func(t *testing.T, targets []TargetConfig) {
				target := targets[0]
				// Name should be scoped: host1--test-container-admin
				if !strings.HasSuffix(target.Name, "-admin") {
					t.Errorf("expected name to end with '-admin', got %q", target.Name)
				}
				if !strings.Contains(target.Name, "host1") {
					t.Errorf("expected name to contain 'host1', got %q", target.Name)
				}
				if target.Port != 9090 {
					t.Errorf("expected port 9090, got %d", target.Port)
				}
			},
		},
		{
			name: "multiple named targets",
			labels: map[string]string{
				"hermes.targets.admin.prometheus.scrape":   "true",
				"hermes.targets.admin.prometheus.port":     "9090",
				"hermes.targets.metrics.prometheus.scrape": "true",
				"hermes.targets.metrics.prometheus.port":   "8080",
			},
			expected: 2,
			validate: func(t *testing.T, targets []TargetConfig) {
				hasAdmin := false
				hasMetrics := false
				for _, target := range targets {
					if strings.HasSuffix(target.Name, "-admin") {
						hasAdmin = true
						if target.Port != 9090 {
							t.Errorf("admin target expected port 9090, got %d", target.Port)
						}
					}
					if strings.HasSuffix(target.Name, "-metrics") {
						hasMetrics = true
						if target.Port != 8080 {
							t.Errorf("metrics target expected port 8080, got %d", target.Port)
						}
					}
				}
				if !hasAdmin {
					t.Error("expected 'admin' target (scoped)")
				}
				if !hasMetrics {
					t.Error("expected 'metrics' target (scoped)")
				}
			},
		},
		{
			name: "named target with strategy override",
			labels: map[string]string{
				"hermes.targets.app.prometheus.scrape": "true",
				"hermes.targets.app.prometheus.port":   "8080",
				"hermes.targets.app.strategy":          "container_ip",
			},
			expected: 1,
			validate: func(t *testing.T, targets []TargetConfig) {
				target := targets[0]
				if !strings.HasSuffix(target.Name, "-app") {
					t.Errorf("expected name to end with '-app', got %q", target.Name)
				}
				if target.Strategy != StrategyContainerIP {
					t.Errorf("expected strategy container_ip, got %q", target.Strategy)
				}
			},
		},
		{
			name: "named target with host override",
			labels: map[string]string{
				"hermes.targets.app.prometheus.scrape": "true",
				"hermes.targets.app.prometheus.port":   "8080",
				"hermes.targets.app.host":              "custom.example.com",
			},
			expected: 1,
			validate: func(t *testing.T, targets []TargetConfig) {
				target := targets[0]
				if !strings.HasSuffix(target.Name, "-app") {
					t.Errorf("expected name to end with '-app', got %q", target.Name)
				}
				if target.Host != "custom.example.com" {
					t.Errorf("expected host custom.example.com, got %q", target.Host)
				}
			},
		},
		{
			name: "named target with resolution port override",
			labels: map[string]string{
				"hermes.targets.app.prometheus.scrape": "true",
				"hermes.targets.app.prometheus.port":   "8080",
				"hermes.targets.app.port":              "10000", // Overrides prometheus port
			},
			expected: 1,
			validate: func(t *testing.T, targets []TargetConfig) {
				target := targets[0]
				if !strings.HasSuffix(target.Name, "-app") {
					t.Errorf("expected name to end with '-app', got %q", target.Name)
				}
				if target.Port != 10000 {
					t.Errorf("expected port 10000 (overridden), got %d", target.Port)
				}
			},
		},
		{
			name: "named target disabled",
			labels: map[string]string{
				"hermes.targets.admin.prometheus.scrape": "false",
				"hermes.targets.admin.prometheus.port":   "9090",
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := createTestContainer()
			container.Labels = tt.labels

			extractor := NewTargetExtractor("hermes")
			targets := extractor.ExtractTargets(container)

			if len(targets) != tt.expected {
				t.Fatalf("expected %d targets, got %d", tt.expected, len(targets))
			}

			if tt.validate != nil {
				tt.validate(t, targets)
			}
		})
	}
}

func TestExtractTargets_ShorthandAndNamed(t *testing.T) {
	container := createTestContainer()
	container.Labels = map[string]string{
		// Shorthand target
		"hermes.prometheus.scrape": "true",
		"hermes.prometheus.port":   "8080",
		// Named target
		"hermes.targets.admin.prometheus.scrape": "true",
		"hermes.targets.admin.prometheus.port":   "9090",
	}

	extractor := NewTargetExtractor("hermes")
	targets := extractor.ExtractTargets(container)

	if len(targets) != 2 {
		t.Fatalf("expected 2 targets (shorthand + named), got %d", len(targets))
	}

	// Check shorthand target name is derived
	hasShorthand := false
	hasAdmin := false
	for _, target := range targets {
		// Container name is "/test-container", so derived name will be "host1--test-container"
		if target.Name == "host1--test-container" || target.Name == "host1-test-container" {
			hasShorthand = true
			if target.Port != 8080 {
				t.Errorf("shorthand target expected port 8080, got %d", target.Port)
			}
		}
		// Named target should be scoped: host1--test-container-admin
		if strings.HasSuffix(target.Name, "-admin") {
			hasAdmin = true
			if target.Port != 9090 {
				t.Errorf("admin target expected port 9090, got %d", target.Port)
			}
		}
	}

	if !hasShorthand {
		t.Errorf("expected shorthand target with derived name, got targets: %v", targets)
	}
	if !hasAdmin {
		t.Error("expected named 'admin' target (scoped)")
	}
}

func TestDeriveTargetName(t *testing.T) {
	extractor := NewTargetExtractor("hermes")

	tests := []struct {
		name      string
		container *ContainerInfo
		expected  string
	}{
		{
			name: "host and name",
			container: &ContainerInfo{
				Host: "host1",
				Name: "/my-container",
			},
			expected: "host1--my-container", // Leading / becomes -
		},
		{
			name: "name only",
			container: &ContainerInfo{
				Name: "/my-container",
			},
			expected: "-my-container", // Leading / becomes -
		},
		{
			name: "name with slashes",
			container: &ContainerInfo{
				Host: "host1",
				Name: "/project/my-container",
			},
			expected: "host1--project-my-container", // Leading / becomes -
		},
		{
			name: "image fallback",
			container: &ContainerInfo{
				Image: "registry.example.com/nginx:latest",
			},
			expected: "nginx",
		},
		{
			name: "image with tag",
			container: &ContainerInfo{
				Image: "nginx:1.21",
			},
			expected: "nginx",
		},
		{
			name: "container ID fallback",
			container: &ContainerInfo{
				ID: "abcdef1234567890",
			},
			expected: "abcdef123456",
		},
		{
			name: "short container ID",
			container: &ContainerInfo{
				ID: "abc123",
			},
			expected: "abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			derived := extractor.deriveTargetName(tt.container)
			if derived != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, derived)
			}
		})
	}
}

func TestDeriveScopedTargetName(t *testing.T) {
	extractor := NewTargetExtractor("hermes")

	tests := []struct {
		name      string
		container *ContainerInfo
		rawName   string
		expected  string
	}{
		{
			name: "host and name with raw target",
			container: &ContainerInfo{
				Host: "host1",
				Name: "/my-container",
			},
			rawName:  "admin",
			expected: "host1--my-container-admin",
		},
		{
			name: "name only with raw target",
			container: &ContainerInfo{
				Name: "/my-container",
			},
			rawName:  "metrics",
			expected: "-my-container-metrics",
		},
		{
			name: "image fallback with raw target",
			container: &ContainerInfo{
				Image: "nginx:latest",
			},
			rawName:  "health",
			expected: "nginx-health",
		},
		{
			name: "container ID fallback with raw target",
			container: &ContainerInfo{
				ID: "abcdef1234567890",
			},
			rawName:  "app",
			expected: "abcdef123456-app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			derived := extractor.deriveScopedTargetName(tt.container, tt.rawName)
			if derived != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, derived)
			}
		})
	}
}

func TestResolveTarget_Defaults(t *testing.T) {
	container := createTestContainer()
	container.Labels = map[string]string{
		"hermes.prometheus.scrape": "true",
		"hermes.prometheus.port":   "8080",
	}

	extractor := NewTargetExtractor("hermes")
	targets := extractor.ExtractTargets(container)

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}

	target := targets[0]
	resolved := target.ResolveTarget(container, StrategyHostname, "agent-host", "custom-host")

	if resolved == nil {
		t.Fatal("expected resolved target, got nil")
	}

	// Check defaults
	if resolved.Scheme != "http" {
		t.Errorf("expected default scheme http, got %q", resolved.Scheme)
	}
	if resolved.Path != "/metrics" {
		t.Errorf("expected default path /metrics, got %q", resolved.Path)
	}
	// Container name includes leading /, so hostname strategy returns "/test-container"
	if resolved.Address != "/test-container:8080" {
		t.Errorf("expected address /test-container:8080, got %q", resolved.Address)
	}

	// Check metadata labels
	if resolved.Labels["__meta_docker_container_id"] != container.ID {
		t.Error("missing container ID in labels")
	}
	if resolved.Labels["__meta_docker_container_name"] != container.Name {
		t.Error("missing container name in labels")
	}
	if resolved.Labels["__meta_target_name"] != target.Name {
		t.Error("missing target name in labels")
	}
}

func TestResolveTarget_HostResolution(t *testing.T) {
	container := createTestContainer()
	container.Labels = map[string]string{
		"hermes.prometheus.scrape": "true",
		"hermes.prometheus.port":   "8080",
	}

	extractor := NewTargetExtractor("hermes")
	targets := extractor.ExtractTargets(container)

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}

	target := targets[0]

	tests := []struct {
		name         string
		strategy     TargetResolutionStrategy
		agentHost    string
		customHost   string
		expectedHost string
	}{
		{
			name:         "hostname strategy",
			strategy:     StrategyHostname,
			agentHost:    "agent1",
			customHost:   "",
			expectedHost: "/test-container", // Container name includes leading /
		},
		{
			name:         "container_ip strategy",
			strategy:     StrategyContainerIP,
			agentHost:    "agent1",
			customHost:   "",
			expectedHost: "172.17.0.2",
		},
		{
			name:         "host_port strategy",
			strategy:     StrategyHostPort,
			agentHost:    "agent1",
			customHost:   "",
			expectedHost: "agent1",
		},
		{
			name:         "host_port with custom host",
			strategy:     StrategyHostPort,
			agentHost:    "agent1",
			customHost:   "external.example.com",
			expectedHost: "external.example.com",
		},
		{
			name:         "custom strategy",
			strategy:     StrategyCustom,
			agentHost:    "agent1",
			customHost:   "custom.example.com",
			expectedHost: "custom.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := target.ResolveTarget(container, tt.strategy, tt.agentHost, tt.customHost)
			if resolved == nil {
				t.Fatal("expected resolved target, got nil")
			}

			expectedAddr := tt.expectedHost + ":8080"
			if resolved.Address != expectedAddr {
				t.Errorf("expected address %q, got %q", expectedAddr, resolved.Address)
			}
		})
	}
}

func TestResolveTarget_HostOverride(t *testing.T) {
	container := createTestContainer()
	container.Labels = map[string]string{
		"hermes.targets.app.prometheus.scrape": "true",
		"hermes.targets.app.prometheus.port":   "8080",
		"hermes.targets.app.host":              "override.example.com",
	}

	extractor := NewTargetExtractor("hermes")
	targets := extractor.ExtractTargets(container)

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}

	target := targets[0]
	resolved := target.ResolveTarget(container, StrategyContainerIP, "agent1", "custom")

	if resolved == nil {
		t.Fatal("expected resolved target, got nil")
	}

	// Host override should take precedence over strategy
	if resolved.Address != "override.example.com:8080" {
		t.Errorf("expected address override.example.com:8080, got %q", resolved.Address)
	}
}

func TestResolveTarget_HostPortStrategy(t *testing.T) {
	container := createTestContainer()
	container.Ports = []PortInfo{
		{PrivatePort: 8080, PublicPort: 18080, Type: "tcp"},
		{PrivatePort: 9090, PublicPort: 19090, Type: "tcp"},
	}
	container.Labels = map[string]string{
		"hermes.targets.app.prometheus.scrape": "true",
		"hermes.targets.app.prometheus.port":   "8080",
		"hermes.targets.app.strategy":          "host_port",
	}

	extractor := NewTargetExtractor("hermes")
	targets := extractor.ExtractTargets(container)

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}

	target := targets[0]
	resolved := target.ResolveTarget(container, StrategyHostname, "agent1", "")

	if resolved == nil {
		t.Fatal("expected resolved target, got nil")
	}

	// Should use published port (18080) instead of private port (8080)
	if resolved.Address != "agent1:18080" {
		t.Errorf("expected address agent1:18080 (published port), got %q", resolved.Address)
	}
}

func TestResolveTarget_NoPortAvailable(t *testing.T) {
	container := createTestContainer()
	container.Ports = []PortInfo{} // No ports
	container.Labels = map[string]string{
		"hermes.prometheus.scrape": "true",
		// No port specified
	}

	extractor := NewTargetExtractor("hermes")
	targets := extractor.ExtractTargets(container)

	// Should return no targets when no port is available
	if len(targets) != 0 {
		t.Fatalf("expected 0 targets (no port), got %d", len(targets))
	}
}

func TestResolveTarget_ScrapeDisabled(t *testing.T) {
	container := createTestContainer()
	container.Labels = map[string]string{
		"hermes.prometheus.scrape": "true",
		"hermes.prometheus.port":   "8080",
	}

	extractor := NewTargetExtractor("hermes")
	targets := extractor.ExtractTargets(container)

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}

	target := targets[0]
	target.Scrape = false // Disable scraping

	resolved := target.ResolveTarget(container, StrategyHostname, "agent1", "")
	if resolved != nil {
		t.Error("expected nil when scrape is disabled")
	}
}

func TestResolveTarget_LabelFiltering(t *testing.T) {
	container := createTestContainer()
	container.Labels = map[string]string{
		"hermes.prometheus.scrape": "true",
		"hermes.prometheus.port":   "8080",
		"app":                      "myapp",
		"version":                  "1.0",
		"hermes.internal":          "should-be-filtered",
	}

	extractor := NewTargetExtractor("hermes")
	targets := extractor.ExtractTargets(container)

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}

	target := targets[0]
	resolved := target.ResolveTarget(container, StrategyHostname, "agent1", "")

	if resolved == nil {
		t.Fatal("expected resolved target, got nil")
	}

	// Non-hermes labels should be included
	if resolved.Labels["app"] != "myapp" {
		t.Error("expected 'app' label to be included")
	}
	if resolved.Labels["version"] != "1.0" {
		t.Error("expected 'version' label to be included")
	}

	// Hermes labels should be filtered out
	if resolved.Labels["hermes.internal"] != "" {
		t.Error("expected 'hermes.internal' label to be filtered out")
	}
}

func TestResolveTarget_CustomPrefix(t *testing.T) {
	container := createTestContainer()
	container.LabelsConfig.Prefix = "custom"
	container.Labels = map[string]string{
		"custom.prometheus.scrape": "true",
		"custom.prometheus.port":   "8080",
		"app":                      "myapp",
		"custom.internal":          "should-be-filtered",
	}

	extractor := NewTargetExtractor("custom")
	targets := extractor.ExtractTargets(container)

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}

	target := targets[0]
	resolved := target.ResolveTarget(container, StrategyHostname, "agent1", "")

	if resolved == nil {
		t.Fatal("expected resolved target, got nil")
	}

	// Non-custom labels should be included
	if resolved.Labels["app"] != "myapp" {
		t.Error("expected 'app' label to be included")
	}

	// Custom labels should be filtered out
	if resolved.Labels["custom.internal"] != "" {
		t.Error("expected 'custom.internal' label to be filtered out")
	}
}

func TestGetPublishedPort(t *testing.T) {
	container := createTestContainer()
	container.Ports = []PortInfo{
		{PrivatePort: 8080, PublicPort: 18080, Type: "tcp"},
		{PrivatePort: 9090, PublicPort: 0, Type: "tcp"}, // Not published
		{PrivatePort: 3000, PublicPort: 3000, Type: "tcp"},
	}

	tests := []struct {
		name         string
		privatePort  int
		expectedPort int
	}{
		{"published port", 8080, 18080},
		{"unpublished port", 9090, 0},
		{"same port", 3000, 3000},
		{"non-existent port", 5000, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := container.getPublishedPort(tt.privatePort)
			if result != tt.expectedPort {
				t.Errorf("expected published port %d, got %d", tt.expectedPort, result)
			}
		})
	}
}

func TestResolveTarget_NoLabelsConfig(t *testing.T) {
	container := createTestContainer()
	container.LabelsConfig = nil // No labels config
	container.Labels = map[string]string{
		"hermes.prometheus.scrape": "true",
		"hermes.prometheus.port":   "8080",
		"app":                      "myapp",
	}

	extractor := NewTargetExtractor("hermes")
	targets := extractor.ExtractTargets(container)

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}

	target := targets[0]
	resolved := target.ResolveTarget(container, StrategyHostname, "agent1", "")

	if resolved == nil {
		t.Fatal("expected resolved target, got nil")
	}

	// Without labels config, all labels should be included (fallback behavior)
	if resolved.Labels["app"] != "myapp" {
		t.Error("expected 'app' label to be included (fallback)")
	}
	if resolved.Labels["hermes.prometheus.scrape"] != "true" {
		t.Error("expected hermes labels to be included (fallback)")
	}
}
