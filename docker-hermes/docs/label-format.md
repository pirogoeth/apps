# Docker Hermes Label Format

Docker Hermes uses a Traefik-style hierarchical label format for configuring Prometheus scrape targets and resolution strategies.

## Label Structure

All labels use a configurable prefix (default: `hermes`). The structure follows this pattern:

```
{prefix}.{section}.{target}.{property} = {value}
```

## Configuration

The label prefix is configurable via the `labels.prefix` configuration option:

```yaml
labels:
  prefix: "hermes"  # Default, can be changed to any value
```

## Target Definitions

### Named Targets (Structured)

For multiple targets per container, use the structured format:

```yaml
# Target named "admin"
hermes.targets.admin.prometheus.scrape: "true"
hermes.targets.admin.prometheus.port: "9090"
hermes.targets.admin.prometheus.path: "/admin/metrics"
hermes.targets.admin.prometheus.scheme: "https"

# Hermes-specific overrides for this target
hermes.targets.admin.strategy: "container_ip"
hermes.targets.admin.host: "admin.example.com"
hermes.targets.admin.port: "10000"  # Override port for resolution
```

### Shorthand (Default Target)

For a single target, use the shorthand format:

```yaml
hermes.prometheus.scrape: "true"
hermes.prometheus.port: "8080"
hermes.prometheus.path: "/metrics"
hermes.prometheus.scheme: "http"
```

The target name is automatically derived from container details:
- Format: `{host}-{container-name}` (e.g., `host1-nginx-app`)
- Falls back to container name, image name, or container ID if needed

## Label Properties

### Prometheus Configuration

All properties under `prometheus.*`:

- `prometheus.scrape`: `"true"` to enable scraping (required)
- `prometheus.port`: Port number to scrape (required if no published port)
- `prometheus.path`: Metrics path (default: `/metrics`)
- `prometheus.scheme`: `"http"` or `"https"` (default: `http`)

### Hermes Configuration

Properties at the target level (for named targets only):

- `strategy`: Resolution strategy (`hostname`, `container_ip`, `host_port`, `custom`)
- `host`: Custom hostname/IP override
- `port`: Port override for resolution (different from prometheus.port)

## Examples

### Single Target (Shorthand)

```yaml
labels:
  hermes.prometheus.scrape: "true"
  hermes.prometheus.port: "8080"
  app: "nginx"
  version: "1.0"
```

### Multiple Targets

```yaml
labels:
  # Default target
  hermes.prometheus.scrape: "true"
  hermes.prometheus.port: "8080"
  
  # Admin metrics endpoint
  hermes.targets.admin.prometheus.scrape: "true"
  hermes.targets.admin.prometheus.port: "9090"
  hermes.targets.admin.prometheus.path: "/admin/metrics"
  hermes.targets.admin.strategy: "container_ip"
  
  # Health check endpoint
  hermes.targets.health.prometheus.scrape: "true"
  hermes.targets.health.prometheus.port: "8081"
  hermes.targets.health.prometheus.path: "/health"
```

### Custom Resolution Strategy

```yaml
labels:
  hermes.targets.app.prometheus.scrape: "true"
  hermes.targets.app.prometheus.port: "8080"
  hermes.targets.app.strategy: "host_port"
  hermes.targets.app.host: "external.example.com"
```

## Target Name Derivation

When using shorthand (`hermes.prometheus.*`), the target name is derived from:

1. `{host}-{container-name}` (preferred, e.g., `host1-nginx-app`)
2. Container name (if host unavailable)
3. Image name without tag/registry (if name unavailable)
4. Short container ID (12 chars, as last resort)

This ensures each target has a unique, identifiable name even when using shorthand.

## Resolution Strategies

- **`hostname`**: Use container hostname/name (default, works with overlay networks)
- **`container_ip`**: Use container's internal IP address (bridge networks)
- **`host_port`**: Use agent hostname + published port (external monitoring)
- **`custom`**: Use custom host from configuration

## Filtering

The `label_prefix` configuration option filters which labels are processed:

```yaml
labels:
  label_prefix: "hermes"  # Only process labels starting with "hermes"
```

If empty, all labels are processed (default behavior).

