# BoltQ - Distributed Task Queue

BoltQ is a scalable, event-driven job processing system built using Golang and Redis. It enables asynchronous execution of tasks by decoupling job submission from execution, ensuring efficient resource utilization and fault tolerance.

## Features (Phase 1)

- Job Submission API: HTTP endpoints for submitting tasks
- Redis-based Task Queue: Reliable message passing between components  
- Worker Service: Asynchronously processes tasks from the queue
- Support for different job types
- Basic logging and configuration

## Prerequisites

- Go 1.20 or later
- Redis server running on localhost:6379
- Make (optional, for using the Makefile)

## Quick Start

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/BoltQ.git
   cd BoltQ
   ```

2. Initialize the module and install dependencies:
   ```bash
   make init
   make deps
   ```

3. Start Redis (if not already running):
   ```bash
   redis-server
   ```

4. In one terminal, start the API service:
   ```bash
   make run-api
   ```

5. In another terminal, start the Worker service:
   ```bash
   make run-worker
   ```

## Using the API

### Submit a Job

```bash
curl -X POST http://localhost:8080/submit \
  -H "Content-Type: application/json" \
  -d '{
    "type": "email",
    "payload": {
      "recipient": "user@example.com",
      "subject": "Hello from BoltQ",
      "body": "This is a test email from BoltQ"
    }
  }'
```

### Check Job Status (Coming in Phase 2)

```bash
curl http://localhost:8080/status?id=job_123
```

## Project Structure

```
📄 .gitignore
📄 Dockerfile
📄 Makefile
📄 README.md
📁 cmd/
  📁 api/
    📄 main.go
  📁 worker/
    📄 main.go
📄 go.mod
📁 internal/
  📁 api/
    📄 handler.go
  📁 queue/
    📄 redis_queue.go
  📁 worker/
    📄 worker.go
📁 pkg/
  📁 config/
    📄 config.go
  📁 logger/
    📄 logger.go
```

## Development Phases

The project is being developed in multiple phases:

- **Phase 1**: Basic Job Submission and Processing ✅
- **Phase 2**: Job Status Tracking and Error Handling (Coming soon)
- **Phase 3**: Scaling and Optimization
- **Phase 4**: Observability and Monitoring
- **Phase 5**: Advanced Features and Extensions
- **Phase 6**: Frontend Playground for Users

## Environment Variables

- `PORT`: API service port (default: 8080)
- `POLLING_INTERVAL`: Worker polling interval in seconds (default: 1)
- `REDIS_ADDR`: Redis server address (default: localhost:6379)

## Docker

Build and run with Docker:

```bash
docker build -t boltq .
docker run -p 8080:8080 boltq
```

## License

[MIT License](LICENSE)