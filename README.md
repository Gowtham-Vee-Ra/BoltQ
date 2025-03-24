# BoltQ: Distributed Task Queue

BoltQ is a scalable, event-driven distributed task queue system built with Go and Redis. It enables asynchronous job processing by decoupling job submission from execution, ensuring efficient resource utilization and fault tolerance.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Getting Started](#getting-started)
- [API Documentation](#api-documentation)
- [Monitoring](#monitoring)
- [Development](#development)
- [Testing](#testing)
- [Contributing](#contributing)
- [License](#license)

## Overview

BoltQ provides a robust solution for handling asynchronous tasks in your applications. Jobs are submitted via a REST API, stored in Redis, and processed by configurable worker pools. The system includes comprehensive monitoring and observability features to track job progress and system health.

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

## Features

- **Asynchronous Job Processing**: Decouple job submission from execution
- **Priority Queues**: Support for high, normal, and low priority jobs
- **Delayed Execution**: Schedule jobs to run at a future time
- **Job Workflows**: Define complex job pipelines with dependencies
- **Error Handling**: Sophisticated error categorization and recovery
- **Automatic Retries**: Exponential backoff retry mechanism
- **Dead Letter Queue**: Failed jobs are stored for analysis
- **Concurrent Processing**: Worker pool architecture for parallel processing
- **Real-time Updates**: WebSocket support for live job status updates
- **Monitoring**: Prometheus metrics and Grafana dashboards
- **Tracing**: Distributed tracing with OpenTelemetry and Jaeger
- **Structured Logging**: JSON logging for better observability
- **Web Playground**: Interactive UI for job submission and monitoring

## Architecture

BoltQ uses a modular architecture with the following components:

### API Service

The API service provides RESTful endpoints for job submission, status checking, and system monitoring. It communicates with Redis to store jobs in appropriate queues based on priority and scheduling requirements.

### Redis Queue

Redis serves as the message broker, storing jobs in various queues. It supports:
- Priority queues (high, normal, low)
- Delayed job scheduling using sorted sets
- Job status tracking
- Dead letter queue for failed jobs

### Worker Service

The worker service pulls jobs from Redis queues and processes them according to their type. Features include:
- Configurable worker pool size
- Job processor registration system
- Automatic retries with exponential backoff
- Error handling and categorization
- Metrics collection

### Playground Frontend

A web-based UI provides easy access to BoltQ's features, allowing users to:
- Submit and monitor jobs
- View system statistics
- Access Grafana dashboards
- Track job workflows

## Project Structure

```
BoltQ/
├── cmd/                         # Application entry points
│   ├── api/                     # API service
│   ├── worker/                  # Worker service
│   └── test/                    # Test utilities
├── internal/                    # Internal packages
│   ├── api/                     # API implementation
│   │   ├── handler.go           # Request handlers
│   │   ├── dashboard.go         # Dashboard endpoints
│   │   └── websocket.go         # WebSocket implementation
│   ├── job/                     # Job models and workflows
│   │   ├── workflow.go          # Workflow implementation
│   │   └── workflow_manager.go  # Workflow management
│   ├── queue/                   # Queue implementations
│   │   ├── queue.go             # Queue interface
│   │   ├── redis_queue.go       # Redis implementation
│   │   ├── redis_adapter.go     # Interface adapter
│   │   └── factory.go           # Queue factory
│   └── worker/                  # Worker implementation
│       ├── pool.go              # Worker pool
│       ├── error_handler.go     # Error handling
│       └── delayed_processor.go # Delayed job processing
├── pkg/                         # Public packages
│   ├── config/                  # Configuration
│   ├── logger/                  # Structured logging
│   ├── metrics/                 # Prometheus metrics
│   │   ├── metrics.go           # Metrics collector
│   │   └── prometheus.go        # Prometheus implementation
│   └── api/                     # OpenAPI specifications
├── playground/                  # Frontend UI
│   ├── src/
│   │   ├── components/          # React components
│   │   ├── pages/               # UI pages
│   │   └── App.jsx              # Main application
│   ├── public/                  # Static assets
│   └── package.json             # Node dependencies
├── grafana/                     # Grafana configuration
│   └── provisioning/
│       ├── dashboards/          # Dashboard definitions
│       └── datasources/         # Data source config
├── docker/                      # Docker configurations
├── prometheus.yml               # Prometheus configuration
├── docker-compose.yml           # Development setup
├── docker-compose.prod.yml      # Production setup
└── README.md                    # Project documentation
```

## Getting Started

### Prerequisites

- Go 1.22+
- Redis 7+
- Node.js 20+ (for playground)
- Docker and Docker Compose (optional)

### Running Locally

#### Start Redis

```bash
# Using Docker
docker run -d -p 6379:6379 redis:7-alpine

# Or use your local Redis installation
redis-server
```

#### Start the API Service

```bash
cd cmd/api
go run main.go
```

#### Start the Worker Service

```bash
cd cmd/worker
go run main.go
```

#### Start the Playground Frontend

```bash
cd playground
npm install
npm run dev
```

### Running with Docker Compose

```bash
# Build and start all services
docker-compose up -d

# View logs
docker-compose logs -f
```

This will start:
- Redis on port 6379
- API service on port 8080
- Worker service (internal)
- Prometheus on port 9092
- Grafana on port 3000
- Playground frontend on port 5173

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `API_PORT` | API server port | 8080 |
| `METRICS_PORT` | Metrics server port | 9090 |
| `REDIS_ADDR` | Redis address | localhost:6379 |
| `NUM_WORKERS` | Number of worker goroutines | 4 |
| `MAX_ATTEMPTS` | Maximum retry attempts | 3 |
| `ENVIRONMENT` | Environment (dev/prod) | development |

## API Documentation

### Job Submission

```bash
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "type": "echo",
    "data": {
      "message": "Hello World"
    },
    "priority": 1,
    "delay_seconds": 0
  }'
```

### Job Status Check

```bash
curl -X GET http://localhost:8080/api/v1/jobs/{job_id}
```

### Queue Stats

```bash
curl -X GET http://localhost:8080/api/v1/queues/stats
```

### Workflow Submission

```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Data Processing Pipeline",
    "steps": [
      {
        "job_type": "fetch_data",
        "params": {
          "url": "https://example.com/data.csv"
        }
      },
      {
        "job_type": "process_data",
        "params": {
          "operation": "transform"
        },
        "depends_on": ["step-1"]
      }
    ]
  }'
```

## Monitoring

### Prometheus Queries

Prometheus is available at http://localhost:9092. Useful queries include:

- `boltq_jobs_submitted_total` - Total jobs submitted by type
- `boltq_jobs_processed_total` - Total jobs processed by status
- `boltq_jobs_in_queue` - Current queue depths
- `boltq_job_processing_seconds` - Job processing time distribution
- `boltq_active_workers` - Number of active workers

### Grafana

Grafana is available at http://localhost:3000 (login: admin/password) and includes:

- **Job Dashboard** - Shows job submission rates, processing times, and queue depths
- **Worker Dashboard** - Displays worker utilization and error rates
- **System Dashboard** - Provides overall system health metrics

## Development

### Adding a New Job Type

1. Register a processor in the worker service:

```go
// In cmd/worker/main.go
workerPool.RegisterProcessor("new_job_type", func(ctx context.Context, task *queue.Task) (map[string]interface{}, error) {
    // Job processing logic here
    return result, nil
})
```

2. Submit jobs of the new type via the API:

```json
{
  "type": "new_job_type",
  "data": {
    "param1": "value1"
  }
}
```

### Testing

#### Unit Tests

```bash
go test ./...
```

#### Integration Tests

```bash
go test -tags=integration ./...
```

#### Performance Testing

```bash
python cmd/test/performance_test.py --jobs=1000 --concurrency=10
```

### PowerShell Testing

#### Submit a Job
```powershell
Invoke-WebRequest -Uri "http://localhost:8080/api/v1/jobs" `
  -Method POST `
  -ContentType "application/json" `
  -Body '{
    "type": "echo",
    "data": {
      "message": "Hello, BoltQ!"
    },
    "priority": 1
  }' | ConvertFrom-Json
```

#### Check Job Status
```powershell
# Replace JOB_ID with actual ID
$jobId = "your-job-id"
Invoke-WebRequest -Uri "http://localhost:8080/api/v1/jobs/$jobId" `
  -Method GET | ConvertFrom-Json
```

#### Get Queue Statistics
```powershell
Invoke-WebRequest -Uri "http://localhost:8080/api/v1/queues/stats" `
  -Method GET | ConvertFrom-Json
```

## License

[MIT License](LICENSE)

## Acknowledgments

- [Go Redis](https://github.com/redis/go-redis) library
- [Gorilla Mux](https://github.com/gorilla/mux) for HTTP routing