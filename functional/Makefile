sqlc_generated := database/deployments.sql.go database/functions.sql.go database/invocations.sql.go database/models.go

functional: test **/*.go $(sqlc_generated)
	go build -v $(buildargs) ./

$(sqlc_generated): sqlc.yml database/queries/*.sql database/migrations/*.sql
	sqlc vet
	sqlc generate

.PHONY: run-debug
run-debug: functional
	CONFIG_TYPE=file CONFIG_FILE=./config.yml LOG_LEVEL=DEBUG DATABASE_PATH=./functional.db ./functional serve

.PHONY: dev
dev: functional
	@echo "Starting development environment..."
	@echo "This will start:"
	@echo "  - Functional server on :8080"
	@echo "  - Traefik proxy (when configured)"
	@echo "  - Docker daemon (assumes running)"
	CONFIG_TYPE=file CONFIG_FILE=./config.yml LOG_LEVEL=DEBUG DATABASE_PATH=./functional.db ./functional serve

.PHONY: test
test:
	go test -v ./...

.PHONY: test-integration
test-integration: functional
	@echo "Running integration tests with Docker..."
	@echo "Ensure Docker daemon is running and accessible"
	go test -v -tags=integration ./...

.PHONY: clean
clean:
	rm -f functional
	rm -f $(sqlc_generated)

.PHONY: clean-data
clean-data:
	rm -f functional.db

.PHONY: clean-all
clean-all: clean clean-data
	docker system prune -f
	@echo "Cleaned up Docker images and containers"

.PHONY: setup-dev
setup-dev:
	@echo "Setting up development environment..."
	@echo "1. Checking Docker..."
	@docker version > /dev/null 2>&1 || (echo "❌ Docker not running" && exit 1)
	@echo "✅ Docker is running"
	@echo "2. Checking sqlc..."
	@which sqlc > /dev/null 2>&1 || (echo "❌ sqlc not found. Install with: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest" && exit 1)
	@echo "✅ sqlc found"
	@echo "3. Installing dependencies..."
	@go mod download
	@echo "✅ Setup complete!"
	@echo
	@echo "Next steps:"
	@echo "  make functional    # Build the binary"
	@echo "  make dev          # Start development server"
	@echo "  make test         # Run tests"

.PHONY: lint
lint:
	@which golangci-lint > /dev/null 2>&1 || (echo "❌ golangci-lint not found. Install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run

.PHONY: fmt
fmt:
	go fmt ./...
	go mod tidy
