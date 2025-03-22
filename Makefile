.PHONY: build run-api run-worker docker-build docker-run test clean

# Default Go parameters
GO=go
GOFLAGS=-v
API_BINARY=bin/boltq-api
WORKER_BINARY=bin/boltq-worker

# Build both API and worker binaries
build:
	mkdir -p bin
	$(GO) build $(GOFLAGS) -o $(API_BINARY) ./cmd/api
	$(GO) build $(GOFLAGS) -o $(WORKER_BINARY) ./cmd/worker
	@echo "Build complete"

# Run the API service
run-api:
	$(GO) run $(GOFLAGS) ./cmd/api

# Run the worker service
run-worker:
	$(GO) run $(GOFLAGS) ./cmd/worker

# Build Docker images
docker-build:
	docker-compose build

# Run with Docker Compose
docker-run:
	docker-compose up

# Run tests
test:
	$(GO) test ./...

# Run tests with coverage
test-coverage:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out

# Clean build artifacts
clean:
	rm -rf bin
	rm -f coverage.out

# Start Redis locally (requires Redis to be installed)
redis-start:
	redis-server --daemonize yes
	@echo "Redis started"

# Stop local Redis
redis-stop:
	redis-cli shutdown
	@echo "Redis stopped"

# Scale up workers with Docker Compose
scale-workers:
	docker-compose up -d --scale worker=4

# Check health of services
health-check:
	@echo "Checking API health..."
	@curl -s http://localhost:8080/health | jq || echo "API not running"
	
	@echo "Checking Redis connection..."
	@redis-cli ping || echo "Redis not running"

# Initialize dev environment
init-dev:
	$(GO) mod tidy
	cp .env.example .env
	@echo "Development environment initialized"

# Generate API documentation
gen-docs:
	@echo "Generating API documentation..."
	# Add documentation generation command here
	@echo "Documentation generated"

# Load test the API
load-test:
	@echo "Running load test..."
	# Add load testing command here (e.g., with hey or vegeta)
	@echo "Load test complete"