<!-- 0c0b7ac1-5929-4fa8-88e1-16e0f336e37b dbdba457-0f3e-4e4f-a697-a10b13b7d1cb -->
# Implement docker-hermes Service Discovery System

## Architecture Summary

**Components:**

- **Agent**: Monitors Docker containers on each host, reports to server via Redis
- **Server**: Provides Prometheus HTTP SD endpoint and query APIs
- **Storage**: Redis with TTL-based expiry for automatic cleanup
- **Communication**: Redis Streams for agent→server updates, periodic heartbeats

**Key Decisions:**

- Single binary with subcommands (`agent`, `server`, `query`)
- Redis for both storage (with TTL) and messaging (Streams)
- Agents send full container state on heartbeats (every 30s default)
- Filter by label prefix on agent side (configurable, default: all labels)
- Support standard Prometheus labels (`prometheus.io/*`)

## Implementation Steps

### 1. Project Setup & Dependencies

Create `main.go`, `go.mod`, and update workspace:

- Module: `github.com/pirogoeth/apps/docker-hermes`
- Dependencies: Docker SDK, Redis client (`github.com/redis/go-redis/v9`), standard apps/pkg
- Add to `go.work`

### 2. Core Types & Configuration

**File: `types/types.go`**

- `ContainerInfo`: ID, name, host, labels, ports, state, last seen timestamp
- `PrometheusTarget`: Host, port, labels for SD format
- Config structs for Agent and Server with Redis connection info

**File: `types/config.go`**

- `AgentConfig`: Docker socket, Redis connection, label prefix filter, heartbeat interval
- `ServerConfig`: Redis connection, HTTP listen address, Prometheus SD path
- Embed `pkg/config.CommonConfig` for logging/tracing

### 3. Redis Client & Data Layer

**File: `redis/client.go`**

- Redis client wrapper with connection pooling
- Key schema: `hermes:containers:{host}:{containerID}` for container data
- Key schema: `hermes:stream:updates` for container update stream
- TTL management: Set expiry on container keys (2x heartbeat interval)

**File: `redis/operations.go`**

- `StoreContainer()`: Store container info with TTL
- `GetContainer()`: Retrieve container by host+ID
- `ListContainers()`: Get all active containers
- `QueryByLabels()`: Find containers matching label filters
- `PublishUpdate()`: Push update to Redis Stream

### 4. Agent Implementation

**File: `agent/docker.go`**

- Docker client initialization
- `WatchContainers()`: Listen to Docker events (start, stop, die)
- `ScanContainers()`: Full scan of running containers on startup
- `ExtractContainerInfo()`: Parse container data, filter labels by prefix
- Extract network info and published ports

**File: `agent/agent.go`**

- Main agent loop with context support
- Periodic heartbeat ticker (configurable interval)
- On heartbeat: scan all containers, report full state to Redis
- On Docker event: immediate update to Redis Stream
- Prometheus metrics: containers tracked, updates sent, errors

**File: `agent/reporter.go`**

- Report container updates to Redis
- Handle connection failures with retry logic
- Batch updates for efficiency

### 5. Server Implementation

**File: `server/server.go`**

- Main server with Gin router setup
- Register API routes and Prometheus SD endpoint
- Context lifecycle management

**File: `server/prometheus.go`**

- Implement Prometheus HTTP SD endpoint
- Query Redis for containers with `prometheus.io/scrape=true`
- Build target list with host:port, labels, and metadata
- Parse `prometheus.io/port`, `prometheus.io/path`, `prometheus.io/scheme`
- Return JSON in Prometheus SD format

**File: `api/api.go`**

- `GET /api/v1/containers`: List all active containers
- `GET /api/v1/containers/:host/:id`: Get specific container
- `POST /api/v1/containers/query`: Query containers by label filters
- `GET /api/v1/labels`: List all unique labels
- `GET /api/v1/labels/:key`: Get all values for a label key

### 6. CLI Commands

**File: `cmd/root.go`**

- Root command setup with app metadata
- Common initialization: logging, tracing setup
- Register subcommands

**File: `cmd/agent.go`**

- `docker-hermes agent` command
- Load agent config, initialize Docker client
- Start agent with signal handling
- Graceful shutdown on SIGINT/SIGTERM

**File: `cmd/server.go`**

- `docker-hermes server` command
- Load server config, initialize Redis client
- Start HTTP server with Prometheus SD and API endpoints
- Graceful shutdown

**File: `cmd/query.go`** (optional but useful)

- `docker-hermes query containers`: List containers
- `docker-hermes query labels`: List labels
- `docker-hermes query targets`: Show Prometheus targets
- Helper for debugging and CLI interaction

### 7. Testing & Validation

- Unit tests for label filtering logic
- Integration test: Run agent against local Docker daemon
- Integration test: Query server API endpoints
- Manual test: Verify Prometheus SD JSON format
- Test TTL expiry: Stop container, wait for expiry

## Key Files Structure

```
docker-hermes/
├── main.go
├── go.mod
├── cmd/
│   ├── root.go
│   ├── agent.go
│   ├── server.go
│   └── query.go
├── agent/
│   ├── agent.go
│   ├── docker.go
│   └── reporter.go
├── server/
│   ├── server.go
│   └── prometheus.go
├── api/
│   └── api.go
├── redis/
│   ├── client.go
│   └── operations.go
└── types/
    ├── types.go
    └── config.go
```

## Prometheus SD Format Reference

Standard format for HTTP SD:

```json
[
  {
    "targets": ["host:port"],
    "labels": {
      "__meta_docker_container_id": "abc123",
      "__meta_docker_container_name": "myapp",
      "label_key": "label_value"
    }
  }
]
```

Support these prometheus.io labels:

- `prometheus.io/scrape`: "true" to include in SD
- `prometheus.io/port`: Port to scrape (default: first exposed port)
- `prometheus.io/path`: Metrics path (default: `/metrics`)
- `prometheus.io/scheme`: http or https (default: `http`)

### To-dos

- [x] Create main.go, go.mod, and add to go.work
- [x] Implement types/types.go and types/config.go with core data structures
- [x] Create redis/client.go and redis/operations.go for Redis integration
- [x] Implement agent/docker.go for Docker container monitoring
- [x] Implement agent/agent.go and agent/reporter.go for heartbeat and reporting
- [x] Implement server/server.go with HTTP router setup
- [x] Implement server/prometheus.go for Prometheus HTTP SD endpoint
- [x] Implement api/api.go with query endpoints
- [x] Implement cmd/root.go, cmd/agent.go, cmd/server.go, and cmd/query.go