# BoltQ - Distributed Task Queue

BoltQ is a scalable, event-driven job processing system built with Go and Redis. It enables asynchronous execution of tasks by decoupling job submission from execution, ensuring efficient resource utilization and fault tolerance.

## Features

- **REST API** for job submission and management
- **Distributed task processing** with multiple worker nodes
- **Job prioritization** to handle critical tasks first
- **Automatic retries** with exponential backoff
- **Dead letter queue** for failed jobs
- **Worker pools** for concurrent processing
- **Status tracking** for all jobs
- **Graceful shutdown** handling
- **Comprehensive metrics** for monitoring

## Architecture

BoltQ consists of three main components:

1. **API Service**: Exposes HTTP endpoints for job submission, status checking, and queue management
2. **Redis Queue**: Stores and distributes jobs to worker nodes
3. **Worker Service**: Processes jobs asynchronously with support for concurrent execution

### System Flow

```
 ┌────────────┐       ┌───────────────┐       ┌─────────────┐
 │            │       │               │       │             │
 │  Clients   │──────▶│   API Server  │──────▶│ Redis Queue │
 │            │       │               │       │             │
 └────────────┘       └───────────────┘       └──────┬──────┘
                                                     │
                                                     ▼
                                              ┌─────────────┐
                                              │             │
                                              │ Worker Pool │
                                              │             │
                                              └──────┬──────┘
                                                     │
                           ┌───────────────┬────────┴────────┬───────────────┐
                           │               │                 │               │
                           ▼               ▼                 ▼               ▼
                     ┌───────────┐   ┌───────────┐    ┌───────────┐   ┌───────────┐
                     │ Worker 1  │   │ Worker 2  │    │ Worker 3  │   │ Worker N  │
                     └───────────┘   └───────────┘    └───────────┘   └───────────┘
```

## Quick Start

### Prerequisites

- Go 1.19 or higher
- Redis 6.0 or higher

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/boltq.git
   cd boltq
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build the binaries:
   ```bash
   make build
   ```

### Configuration

BoltQ can be configured using environment variables or a `.env` file:

```
# API configuration
API_PORT=8080

# Redis configuration
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Worker configuration
WORKER_ID=worker-1
NUM_WORKERS=4
MAX_ATTEMPTS=3
JOB_STATUS_TTL=168h

# Logging
LOG_LEVEL=info
```

### Running the Services

1. Start Redis:
   ```bash
   redis-server
   ```

2. Start the API server:
   ```bash
   ./bin/boltq-api
   ```

3. Start the worker service:
   ```bash
   ./bin/boltq-worker
   ```

For development, you can also use:
```bash
make run-api
make run-worker
```

Or with Docker:
```bash
make docker-build
make docker-run
```

## API Reference

### Submit a Job

```
POST /api/jobs
```

Request body:
```json
{
  "type": "email",
  "payload": "Send welcome email to user@example.com",
  "priority": 2,
  "max_attempts": 3,
  "timeout": 60,
  "tags": ["welcome", "email"]
}
```

Response:
```json
{
  "job_id": "job_1742524616651842000",
  "status": "pending"
}
```

### Check Job Status

```
GET /api/jobs/{job_id}
```

Response:
```json
{
  "id": "job_1742524616651842000",
  "type": "email",
  "payload": "Send welcome email to user@example.com",
  "status": "completed",
  "priority": 2,
  "created_at": "2025-03-20T22:36:56.6518420-04:00",
  "started_at": "2025-03-20T22:36:56.7542039-04:00",
  "completed_at": "2025-03-20T22:36:57.1950121-04:00",
  "next_retry_at": "0001-01-01T00:00:00Z",
  "attempts": 0,
  "max_attempts": 3,
  "worker_id": "worker-5b789948-worker-1"
}
```

### Get Queue Statistics

```
GET /api/stats
```

Response:
```json
{
  "dead_letter": 2,
  "failed": 0,
  "pending": 0,
  "processed": 28,
  "published": 30,
  "retry": 0
}
```

### Cancel a Job

```
POST /api/jobs/{job_id}/cancel
```

Response:
```json
{
  "job_id": "job_1742524616651842000",
  "status": "cancelled"
}
```

### Retry a Dead Letter Job

```
POST /api/jobs/{job_id}/retry
```

Response:
```json
{
  "job_id": "job_1742524616651842000",
  "status": "retrying"
}
```

## Job Types and Processors

BoltQ supports different job types, each processed by a specific processor function. By default, the following job types are supported:

- `email`: For sending emails
- `notification`: For sending notifications
- `test`: For testing purposes

You can register custom processors for new job types by extending the `worker` package:

```go
worker.RegisterProcessor("custom-job-type", func(j *job.Job) error {
    // Process the job
    return nil
})
```

## Testing

1. Run unit tests:
   ```bash
   make test
   ```

2. Run integration tests (requires Redis):
   ```bash
   make test-integration
   ```

3. Test with PowerShell (Windows):
   ```powershell
   # Submit an email job
   $emailJob = @{
       type = "email"
       payload = "Send welcome email to user@example.com"
   } | ConvertTo-Json

   Invoke-WebRequest -Uri "http://localhost:8080/api/jobs" -Method Post -Body $emailJob -ContentType "application/json" -UseBasicParsing
   ```

4. Test with curl (Linux/macOS):
   ```bash
   # Submit an email job
   curl -X POST http://localhost:8080/api/jobs \
     -H "Content-Type: application/json" \
     -d '{"type":"email","payload":"Send welcome email to user@example.com"}'
   ```

## Project Structure

```
BoltQ/
  ├── cmd/
  │   ├── api/            # API service entrypoint
  │   └── worker/         # Worker service entrypoint
  ├── internal/
  │   ├── api/            # API handlers
  │   ├── job/            # Job model
  │   ├── queue/          # Queue implementations
  │   └── worker/         # Worker implementation
  ├── pkg/
  │   ├── config/         # Configuration helpers
  │   └── logger/         # Logging utilities
  ├── Dockerfile          # Docker configuration
  ├── docker-compose.yml  # Docker Compose configuration
  ├── Makefile            # Build and run commands
  └── README.md           # This file
```

## Development Roadmap

The BoltQ project is being developed in phases:

- ✅ **Phase 1**: Basic Job Submission and Processing
- ✅ **Phase 2**: Job Status Tracking and Error Handling
- ✅ **Phase 3**: Scaling and Optimization
- 🔄 **Phase 4**: Observability and Monitoring
- 🔜 **Phase 5**: Advanced Features and Extensions
- 🔜 **Phase 6**: Frontend Playground for Users

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

[MIT License](LICENSE)

## Acknowledgments

- [Go Redis](https://github.com/go-redis/redis) library
- [Gorilla Mux](https://github.com/gorilla/mux) for HTTP routing
