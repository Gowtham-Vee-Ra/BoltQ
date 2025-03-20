.PHONY: build run-api run-worker clean all

# Set up module path
MODULE_PATH=github.com/yourusername/BoltQ

# Build binaries
build:
	@echo "Building API and Worker services..."
	go build -o bin/api $(MODULE_PATH)/cmd/api
	go build -o bin/worker $(MODULE_PATH)/cmd/worker
	@echo "Build complete"

# Run API service
run-api:
	@echo "Starting API service..."
	go run cmd/api/main.go

# Run Worker service
run-worker:
	@echo "Starting Worker service..."
	go run cmd/worker/main.go

# Clean binaries
clean:
	@echo "Cleaning binaries..."
	rm -rf bin/

# Build and run all services
all: build
	@echo "Running all services..."
	./bin/api & ./bin/worker

# Initialize project with proper module name
init:
	@echo "Initializing module..."
	go mod init $(MODULE_PATH)
	go mod tidy

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go get github.com/go-redis/redis/v8