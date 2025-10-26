# Docker Hermes Integration Testing

This directory contains a complete integration testing setup for the docker-hermes service discovery system using Docker Compose.

## 🏗️ Architecture

The integration test stack includes:

- **Redis**: Central storage and messaging
- **docker-hermes-server**: HTTP API and Prometheus SD endpoint
- **docker-hermes-agent-1/2**: Two agents monitoring different "hosts"
- **test-app-1/2/3**: Test applications with various label configurations
- **Prometheus**: Metrics collection with HTTP SD integration
- **Grafana**: Visualization and monitoring dashboards
- **Redis Commander**: Redis management interface

## 🚀 Quick Start

### 1. Start the Stack

```bash
# Start all services
docker-compose up -d

# Check service status
docker-compose ps
```

### 2. Run Integration Tests

```bash
# Run automated tests
./test-integration.sh test

# Show service URLs
./test-integration.sh urls
```

### 3. Access Services

| Service | URL | Credentials |
|---------|-----|-------------|
| Grafana | http://localhost:3000 | admin/admin |
| Prometheus | http://localhost:9090 | - |
| Redis Commander | http://localhost:8081 | - |
| Hermes API | http://localhost:8080/api/v1/containers | - |
| Prometheus SD | http://localhost:8080/prometheus/sd | - |

## 🧪 Test Scenarios

### Scenario 1: Basic Container Discovery
- **test-app-1** and **test-app-2** have `prometheus.io/scrape=true` labels
- **test-app-3** does not have Prometheus labels
- Agents should discover all containers, but only apps 1&2 should appear in Prometheus SD

### Scenario 2: Multi-Host Simulation
- **hermes-agent-1** reports as "host1"
- **hermes-agent-2** reports as "host2"
- Both agents monitor the same Docker daemon but report different hostnames
- Tests hostname isolation and multi-host scenarios

### Scenario 3: Label Filtering
- All agents use empty `LABEL_PREFIX` (reports all labels)
- Tests label propagation from containers to Prometheus targets

### Scenario 4: TTL and Cleanup
- Containers are automatically removed from Redis after TTL expires
- Tests automatic cleanup when agents stop reporting

## 📊 Monitoring

### Grafana Dashboard
The included dashboard shows:
- Containers tracked per agent
- Update rates
- Error rates
- Container target status in Prometheus

### Prometheus Targets
Check `http://localhost:9090/targets` to see:
- Static targets (Prometheus, agents, server)
- Dynamic targets from HTTP SD (test applications)

### Agent Metrics
Each agent exposes metrics on:
- `http://localhost:9091/metrics` (agent-1)
- `http://localhost:9092/metrics` (agent-2)

## 🔧 Configuration

### Environment Variables

**Server:**
- `REDIS_URL`: Redis connection string
- `HTTP_LISTEN_ADDR`: Server listen address
- `PROMETHEUS_SD_PATH`: Prometheus SD endpoint path

**Agents:**
- `DOCKER_SOCKET`: Docker socket path
- `HOSTNAME`: Agent hostname identifier
- `LABEL_PREFIX`: Label filtering prefix
- `HEARTBEAT_INTERVAL`: Heartbeat frequency

### Test Applications

**test-app-1** (Prometheus enabled):
```yaml
labels:
  - "prometheus.io/scrape=true"
  - "prometheus.io/port=80"
  - "prometheus.io/path=/metrics"
  - "app=nginx"
  - "environment=test"
  - "version=1.0"
```

**test-app-2** (Prometheus enabled):
```yaml
labels:
  - "prometheus.io/scrape=true"
  - "prometheus.io/port=80"
  - "prometheus.io/path=/metrics"
  - "app=nginx"
  - "environment=test"
  - "version=2.0"
```

**test-app-3** (No Prometheus):
```yaml
labels:
  - "app=nginx"
  - "environment=test"
  - "version=3.0"
  - "no-prometheus=true"
```

## 🧪 Manual Testing

### 1. Test API Endpoints

```bash
# List all containers
curl http://localhost:8080/api/v1/containers

# Get specific container
curl http://localhost:8080/api/v1/containers/host1/test-app-1

# Query by labels
curl -X POST http://localhost:8080/api/v1/containers/query \
  -H "Content-Type: application/json" \
  -d '{"labels": {"app": "nginx"}}'

# List all labels
curl http://localhost:8080/api/v1/labels

# Get label values
curl http://localhost:8080/api/v1/labels/app
```

### 2. Test Prometheus SD

```bash
# Get Prometheus SD targets
curl http://localhost:8080/prometheus/sd | jq
```

Expected output:
```json
[
  {
    "targets": ["host1:80"],
    "labels": {
      "__meta_docker_container_id": "...",
      "__meta_docker_container_name": "test-app-1",
      "app": "nginx",
      "environment": "test",
      "version": "1.0"
    }
  },
  {
    "targets": ["host2:80"],
    "labels": {
      "__meta_docker_container_id": "...",
      "__meta_docker_container_name": "test-app-2",
      "app": "nginx",
      "environment": "test",
      "version": "2.0"
    }
  }
]
```

### 3. Test CLI Commands

```bash
# Build binary
go build -o docker-hermes .

# Query containers
./docker-hermes query containers --server http://localhost:8080

# Query labels
./docker-hermes query labels --server http://localhost:8080

# Query Prometheus targets
./docker-hermes query targets --server http://localhost:8080
```

## 🐛 Troubleshooting

### Services Not Starting

```bash
# Check logs
docker-compose logs hermes-server
docker-compose logs hermes-agent-1
docker-compose logs redis

# Check service health
docker-compose ps
```

### No Containers Discovered

1. Check agent logs for Docker connection issues
2. Verify Docker socket is mounted correctly
3. Check Redis connectivity
4. Verify heartbeat intervals

### Prometheus Targets Missing

1. Check Prometheus SD endpoint: `curl http://localhost:8080/prometheus/sd`
2. Verify containers have `prometheus.io/scrape=true` labels
3. Check Prometheus configuration
4. Look at Prometheus logs: `docker-compose logs prometheus`

### Redis Issues

1. Check Redis connectivity: `docker-compose exec redis redis-cli ping`
2. Use Redis Commander: http://localhost:8081
3. Check Redis logs: `docker-compose logs redis`

## 🧹 Cleanup

```bash
# Stop and remove all containers
./test-integration.sh cleanup

# Or manually
docker-compose down -v
docker system prune -f
```

## 📈 Performance Testing

### Scale Testing

To test with more containers:

```bash
# Add more test applications
docker run -d --name test-app-4 \
  --label "prometheus.io/scrape=true" \
  --label "prometheus.io/port=80" \
  --label "app=nginx" \
  --label "environment=test" \
  --label "version=4.0" \
  nginx:alpine

# Check if discovered
curl http://localhost:8080/api/v1/containers | jq '.count'
```

### Load Testing

```bash
# Test API performance
ab -n 1000 -c 10 http://localhost:8080/api/v1/containers

# Test Prometheus SD performance
ab -n 1000 -c 10 http://localhost:8080/prometheus/sd
```

## 🔄 Continuous Integration

The integration test can be run in CI/CD pipelines:

```yaml
# Example GitHub Actions
- name: Run Integration Tests
  run: |
    docker-compose up -d
    ./test-integration.sh test
    docker-compose down -v
```

This setup provides a comprehensive testing environment for validating docker-hermes functionality across all components.
