# BoltQ - Distributed Task Queue

BoltQ is a scalable, event-driven job processing system built with Go and Redis. It enables asynchronous execution of tasks by decoupling job submission from execution, ensuring efficient resource utilization and fault tolerance.

## Features

- **REST API** for job submission and status checking
- **Distributed task processing** with worker pool architecture
- **Job prioritization** (Low, Normal, High, Critical)
- **Automatic retries** with exponential backoff
- **Dead letter queue** for failed jobs
- **Worker pools** for concurrent processing
- **Status tracking** for all jobs
- **Observability and monitoring** with structured logging, metrics, and tracing
- **Graceful shutdown** handling

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

### Monitoring Architecture

```
 ┌────────────┐       ┌───────────────┐       ┌─────────────┐
 │            │       │               │       │             │
 │  API       │──────▶│   Prometheus  │──────▶│  Grafana    │
 │  Worker    │       │               │       │             │
 └────────────┘       └───────────────┘       └─────────────┘
        │                                             ▲
        │                                             │
        ▼                                             │
 ┌────────────┐       ┌───────────────┐               │
 │            │       │               │               │
 │  Jaeger    │──────▶│ OpenTelemetry │───────────────┘
 │  Tracing   │       │               │
 └────────────┘       └───────────────┘
```

## Prerequisites

- Go 1.19 or higher
- Redis 6.0 or higher
- Docker and Docker Compose (for monitoring stack)

## Configuration

BoltQ can be configured using environment variables:

```
# API configuration
API_PORT=8080
METRICS_PORT=9090

# Redis configuration
REDIS_ADDR=localhost:6379

# Worker configuration
NUM_WORKERS=4
MAX_ATTEMPTS=3
METRICS_PORT=9091

# Tracing configuration
OTEL_EXPORTER_OTLP_ENDPOINT=jaeger:4317
ENVIRONMENT=development
```

## API Reference

### Submit a Job

```
POST /jobs/submit
```

Request body:
```json
{
  "type": "email",
  "priority": "normal",
  "data": {
    "to": "user@example.com",
    "subject": "Test Email"
  },
  "tags": ["welcome", "email"],
  "timeout": 300
}
```

Response:
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "type": "email",
  "status": "pending",
  "created_at": "2025-03-21T12:34:56Z"
}
```

### Check Job Status

```
GET /jobs/status?id=123e4567-e89b-12d3-a456-426614174000
```

Response:
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "status": "completed"
}
```

### Get Queue Statistics

```
GET /queue/stats
```

Response:
```json
{
  "critical": 0,
  "high": 0,
  "normal": 0,
  "low": 0,
  "retry": 1,
  "deadLetter": 2
}
```

## Project Structure

```
BoltQ/
  ├── cmd/
  │   ├── api/            # API service entrypoint
  │   ├── worker/         # Worker service entrypoint
  │   └── test/           # Test program entrypoint
  ├── internal/
  │   ├── api/            # API handlers
  │   ├── job/            # Job model
  │   ├── queue/          # Queue implementations
  │   └── worker/         # Worker implementation
  ├── pkg/
  │   ├── config/         # Configuration helpers
  │   ├── logger/         # Structured logging
  │   ├── metrics/        # Prometheus metrics
  │   └── tracing/        # OpenTelemetry tracing
  ├── Dockerfile          # Docker configuration
  ├── docker-compose.yml  # Docker Compose configuration
  └── README.md           # This file
```

## Completed Development Phases

The BoltQ project has been developed in phases:

- ✅ **Phase 1**: Basic Job Submission and Processing
  - Initial API implementation
  - Basic Redis queue integration
  - Simple worker model
  
- ✅ **Phase 2**: Job Status Tracking and Error Handling
  - Job status management in Redis
  - Retry mechanism with exponential backoff
  - Dead letter queue for failed jobs

- ✅ **Phase 3**: Scaling and Optimization
  - Worker pool implementation
  - Job prioritization
  - Connection pooling for Redis
  - Performance optimization

- ✅ **Phase 4**: Observability and Monitoring
  - Structured JSON logging
  - Prometheus metrics integration
  - Grafana dashboards
  - OpenTelemetry tracing
  - Comprehensive monitoring

## Testing

### Using PowerShell

```powershell
# Submit an email job
$body = @{
    type = "email"
    priority = "normal"
    data = @{
        to = "user@example.com"
        subject = "Test Email"
    }
} | ConvertTo-Json

$response = Invoke-WebRequest -Uri "http://localhost:8080/jobs/submit" -Method Post -Body $body -ContentType "application/json"
$jobData = $response.Content | ConvertFrom-Json
$jobId = $jobData.id

# Check job status
$statusResponse = Invoke-WebRequest -Uri "http://localhost:8080/jobs/status?id=$jobId" -Method Get
$statusData = $statusResponse.Content | ConvertFrom-Json
Write-Host "Job status: $($statusData.status)"

# Get queue statistics
$statsResponse = Invoke-WebRequest -Uri "http://localhost:8080/queue/stats" -Method Get
$statsData = $statsResponse.Content | ConvertFrom-Json
```

### Automated Testing

Run the test program to automatically submit and process jobs:

```bash
go run cmd/test/main.go
```

### Monitoring

1. **Prometheus**: Access metrics at http://localhost:9092
   - `boltq_jobs_submitted_total`
   - `boltq_jobs_processed_total`
   - `boltq_jobs_in_queue`
   - `boltq_job_processing_seconds`
   - `boltq_active_workers`

2. **Grafana**: Access dashboards at http://localhost:3000
   - Login with admin/admin
   - BoltQ dashboard provides visualizations for all metrics

3. **Jaeger**: Access distributed traces at http://localhost:16686
   - View end-to-end job processing traces
   - Analyze performance bottlenecks

## Next Steps

- **Phase 5**: Advanced Features and Extensions
  - Priority queues for high-priority jobs
  - Delayed job execution
  - Support for Kafka or NATS as queue backend
  - Dashboard UI for job monitoring

- **Phase 6**: Frontend Playground for Users
  - Web-based interface for job submission
  - Live updates of job status
  - Visual representation of queue statistics

## License

[MIT License](LICENSE)

## Acknowledgments

[Go Redis](https://github.com/redis/go-redis) library
[Gorilla Mux](https://github.com/gorilla/mux) for HTTP routing