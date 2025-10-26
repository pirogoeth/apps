#!/bin/bash

# Integration test script for docker-hermes
set -e

echo "🚀 Starting docker-hermes integration test..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to wait for service to be ready
wait_for_service() {
    local service_name=$1
    local url=$2
    local max_attempts=30
    local attempt=1

    print_status "Waiting for $service_name to be ready..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s "$url" > /dev/null 2>&1; then
            print_status "$service_name is ready!"
            return 0
        fi
        
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    print_error "$service_name failed to start after $((max_attempts * 2)) seconds"
    return 1
}

# Function to test API endpoints
test_api_endpoints() {
    local base_url="http://localhost:8080"
    
    print_status "Testing API endpoints..."
    
    # Test health endpoint
    if curl -s "$base_url/health" | grep -q "healthy"; then
        print_status "✅ Health endpoint working"
    else
        print_error "❌ Health endpoint failed"
        return 1
    fi
    
    # Test containers endpoint
    if curl -s "$base_url/api/v1/containers" | grep -q "containers"; then
        print_status "✅ Containers endpoint working"
    else
        print_error "❌ Containers endpoint failed"
        return 1
    fi
    
    # Test labels endpoint
    if curl -s "$base_url/api/v1/labels" | grep -q "labels"; then
        print_status "✅ Labels endpoint working"
    else
        print_error "❌ Labels endpoint failed"
        return 1
    fi
    
    # Test Prometheus SD endpoint
    if curl -s "$base_url/prometheus/sd" | grep -q "targets"; then
        print_status "✅ Prometheus SD endpoint working"
    else
        print_error "❌ Prometheus SD endpoint failed"
        return 1
    fi
}

# Function to test Prometheus targets
test_prometheus_targets() {
    print_status "Testing Prometheus targets..."
    
    # Wait a bit for targets to be discovered
    sleep 10
    
    local targets_url="http://localhost:9090/api/v1/targets"
    local targets_response=$(curl -s "$targets_url")
    
    if echo "$targets_response" | grep -q "docker-containers"; then
        print_status "✅ Prometheus discovered docker-containers job"
    else
        print_warning "⚠️  Prometheus docker-containers job not found"
    fi
    
    # Check if we have active targets
    local active_targets=$(echo "$targets_response" | grep -o '"health":"up"' | wc -l)
    if [ "$active_targets" -gt 0 ]; then
        print_status "✅ Found $active_targets active targets"
    else
        print_warning "⚠️  No active targets found"
    fi
}

# Function to test agent metrics
test_agent_metrics() {
    print_status "Testing agent metrics..."
    
    # Test agent 1 metrics
    if curl -s "http://localhost:9091/metrics" | grep -q "docker_hermes_containers_tracked"; then
        print_status "✅ Agent 1 metrics working"
    else
        print_error "❌ Agent 1 metrics failed"
        return 1
    fi
    
    # Test agent 2 metrics
    if curl -s "http://localhost:9092/metrics" | grep -q "docker_hermes_containers_tracked"; then
        print_status "✅ Agent 2 metrics working"
    else
        print_error "❌ Agent 2 metrics failed"
        return 1
    fi
}

# Function to run CLI tests
test_cli_commands() {
    print_status "Testing CLI commands..."
    
    # Build the binary
    if go build -o docker-hermes .; then
        print_status "✅ Binary built successfully"
    else
        print_error "❌ Failed to build binary"
        return 1
    fi
    
    # Test query commands
    if ./docker-hermes query containers --server http://localhost:8080 > /dev/null 2>&1; then
        print_status "✅ Query containers command working"
    else
        print_error "❌ Query containers command failed"
        return 1
    fi
    
    if ./docker-hermes query labels --server http://localhost:8080 > /dev/null 2>&1; then
        print_status "✅ Query labels command working"
    else
        print_error "❌ Query labels command failed"
        return 1
    fi
    
    if ./docker-hermes query targets --server http://localhost:8080 > /dev/null 2>&1; then
        print_status "✅ Query targets command working"
    else
        print_error "❌ Query targets command failed"
        return 1
    fi
    
    # Clean up
    rm -f docker-hermes
}

# Main test function
run_tests() {
    print_status "Starting integration tests..."
    
    # Wait for services to be ready
    wait_for_service "Redis" "http://localhost:8081" || return 1
    wait_for_service "Hermes Server" "http://localhost:8080/health" || return 1
    wait_for_service "Prometheus" "http://localhost:9090" || return 1
    
    # Run tests
    test_api_endpoints || return 1
    test_agent_metrics || return 1
    test_prometheus_targets || return 1
    test_cli_commands || return 1
    
    print_status "🎉 All integration tests passed!"
    return 0
}

# Function to show service URLs
show_service_urls() {
    echo ""
    print_status "Service URLs:"
    echo "  📊 Grafana:           http://localhost:3000 (admin/admin)"
    echo "  📈 Prometheus:        http://localhost:9090"
    echo "  🔧 Redis Commander:   http://localhost:8081"
    echo "  🌐 Hermes Server:     http://localhost:8080"
    echo "  📋 Hermes API:        http://localhost:8080/api/v1/containers"
    echo "  🎯 Prometheus SD:      http://localhost:8080/prometheus/sd"
    echo "  📊 Agent 1 Metrics:   http://localhost:9091/metrics"
    echo "  📊 Agent 2 Metrics:   http://localhost:9092/metrics"
    echo ""
}

# Function to cleanup
cleanup() {
    print_status "Cleaning up..."
    docker-compose down -v
}

# Handle script arguments
case "${1:-}" in
    "test")
        run_tests
        ;;
    "urls")
        show_service_urls
        ;;
    "cleanup")
        cleanup
        ;;
    *)
        echo "Usage: $0 {test|urls|cleanup}"
        echo ""
        echo "Commands:"
        echo "  test     - Run integration tests"
        echo "  urls     - Show service URLs"
        echo "  cleanup  - Stop and remove all containers"
        echo ""
        echo "To start the stack:"
        echo "  docker-compose up -d"
        echo ""
        echo "To run tests:"
        echo "  $0 test"
        echo ""
        echo "To view services:"
        echo "  $0 urls"
        ;;
esac
