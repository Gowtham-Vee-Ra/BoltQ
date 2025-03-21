# BoltQ - Distributed Task Queue

BoltQ is a scalable, event-driven job processing system built using Golang and Redis. It enables asynchronous execution of tasks by decoupling job submission from execution, ensuring efficient resource utilization and fault tolerance.

## Features (Phase 1)
* Job Submission API: HTTP endpoints for submitting tasks
* Redis-based Task Queue: Reliable message passing between components
* Worker Service: Asynchronously processes tasks from the queue
* Support for different job types
* Basic logging and configuration

## Features (Phase 2)
* Job Status Tracking: Monitor job progress from submission to completion
* Automatic Retry Mechanism: Failed jobs are retried with exponential backoff
* Dead Letter Queue: Storage for jobs that fail repeatedly
* Enhanced API Endpoints: Check job status and queue statistics
* Improved Error Handling: Robust handling of various failure scenarios
* Structured JSON Logging: Better observability for production environments

## Prerequisites
* Go 1.20 or later
* Redis server running on localhost:6379
* Make (optional, for using the Makefile)

## Quick Start
1. Clone the repository:
```sh
git clone https://github.com/yourusername/BoltQ.git
cd BoltQ
```
2. Initialize the module and install dependencies:
```sh
make init
make deps
```
3. Start Redis (if not already running):
```sh
redis-server
```
4. In one terminal, start the API service:
```sh
make run-api
```
5. In another terminal, start the Worker service:
```sh
make run-worker
```

## Using the API

### Submit a Job
```sh
curl -X POST http://localhost:8080/api/jobs \
-H "Content-Type: application/json" \
-d '{
  "type": "email",
  "payload": {
    "recipient": "user@example.com",
    "subject": "Hello from BoltQ",
    "body": "This is a test email from BoltQ"
  },
  "max_attempts": 3
}'
```

### Check Job Status
```sh
curl http://localhost:8080/api/jobs/status?id=20250320123456-abcdefgh
```

### Get Queue Statistics
```sh
curl http://localhost:8080/api/stats
```

## Using PowerShell for Testing

### Submit a Job
```powershell
Invoke-WebRequest -Uri http://localhost:8080/api/jobs -Method POST -ContentType "application/json" -Body '{"type":"email","payload":{"recipient":"test@example.com","subject":"Test Email","body":"This is a test email"},"max_attempts":3}'
```

### Check Job Status
```powershell
Invoke-WebRequest -Uri "http://localhost:8080/api/jobs/status?id=YOUR_JOB_ID" -Method GET
```

### Get Queue Statistics
```powershell
(Invoke-WebRequest -Uri http://localhost:8080/api/stats -Method GET).Content
```

## Project Structure
```
📄 .gitignore
📄 Dockerfile
📄 docker-compose.yml
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
  📁 job/
    📄 job.go
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
- **Phase 2**: Job Status Tracking and Error Handling ✅
- **Phase 3**: Scaling and Optimization (Coming soon)
- **Phase 4**: Observability and Monitoring
- **Phase 5**: Advanced Features and Extensions
- **Phase 6**: Frontend Playground for Users

## Environment Variables
- `API_PORT`: API service port (default: 8080)
- `REDIS_ADDR`: Redis server address (default: localhost:6379)
- `WORKER_ID`: Worker identifier (default: auto-generated)

A `.env` file can be used for local development.

## Docker
Build and run with Docker:
```sh
docker-compose up -d
```

## License

[MIT License](LICENSE)