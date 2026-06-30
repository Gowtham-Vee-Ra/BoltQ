# BoltQ

A distributed task queue built with Go and Redis. Jobs are submitted via a REST API, stored in priority queues, and processed by a configurable worker pool. A React playground ships alongside for submitting and monitoring jobs in the browser.

## Prerequisites

- Go 1.22+
- Redis 7+
- Node.js 20+ (playground only)
- A [Pusher Channels](https://pusher.com) account (notification jobs)
- [Mailpit](https://mailpit.axllent.org) or any SMTP server (email jobs)

## Setup

Copy `.env.example` to `.env` and fill in the values:

```
REDIS_ADDR=localhost:6379
API_PORT=8080
WORKER_METRICS_PORT=9094
ALLOWED_ORIGIN=http://localhost:5173

# Required in production — protects mutating endpoints. Empty = unauthenticated (dev only).
API_KEY=
RATE_LIMIT_RPS=20
RATE_LIMIT_BURST=40

SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_FROM=boltq@localhost

PUSHER_APP_ID=
PUSHER_KEY=
PUSHER_SECRET=
PUSHER_CLUSTER=
```

### Authentication

When `API_KEY` is set, mutating requests (submit job, cancel job, create/delete
workflow) must include the key; read-only endpoints stay open. Requests are also
rate-limited per client IP (`RATE_LIMIT_RPS` / `RATE_LIMIT_BURST`).

```bash
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"type": "echo", "data": {"message": "hello"}}'
```

The playground reads `VITE_API_URL` from `playground/.env` (defaults to `http://localhost:8080`).

## Running

Start each service in a separate terminal.

**Redis** (if not already running):
```bash
redis-server
```

**API server:**
```bash
go run ./cmd/api/
```

**Worker:**
```bash
go run ./cmd/worker/
```

**Playground:**
```bash
cd playground
npm install
npm run dev
```

Open `http://localhost:5173`.

## Job types

| Type | Required fields | What it does |
|---|---|---|
| `echo` | `message` | Returns input after a 1s delay |
| `sleep` | `seconds` | Sleeps for up to 60s |
| `email` | `to`, `subject`, `body` | Sends via SMTP (Mailpit in dev) |
| `notification` | `message`, `recipient` | Fires a Pusher event to the playground |
| `process-image` | `url`, `width`, `height`, `format` | Fetches, resizes, and saves JPEG/PNG |
| `generate-report` | `report_type`, `format` | Writes CSV, HTML, and/or PDF to `output/reports/` |

For `generate-report`, `report_type` accepts `summary`, `monthly`, or `detailed`. `format` accepts `csv`, `html`, or `pdf` — omit to generate all three.

For `process-image`, `format` accepts `jpeg` or `png`. WebP and GIF are accepted as input but always output as JPEG.

## Submitting a job

```bash
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{"type": "echo", "data": {"message": "hello"}, "priority": 1, "delay_seconds": 0}'
```

Priority: `2` = high, `1` = normal, `0` = low. `delay_seconds` schedules the job for future execution.

## API endpoints

```
POST   /api/v1/jobs              submit a job
GET    /api/v1/jobs              list jobs (limit, offset)
GET    /api/v1/jobs/:id          get job status and result
POST   /api/v1/jobs/:id/cancel   cancel a pending job

GET    /api/v1/queues/stats      queue depths and dead letter count

POST   /api/v1/workflows         create a workflow
GET    /api/v1/workflows         list workflows
GET    /api/v1/workflows/:id     get workflow detail
DELETE /api/v1/workflows/:id     delete a workflow

GET    /health                   API health check
GET    /ws/jobs                  WebSocket — live job and workflow updates
```

## Workflows

A workflow is a DAG of jobs. Steps run in dependency order; independent steps run concurrently.

```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "image then report",
    "steps": [
      {
        "id": "step-a",
        "job_type": "process-image",
        "params": {"url": "https://example.com/photo.jpg", "width": 640, "height": 360, "format": "jpeg"}
      },
      {
        "id": "step-b",
        "job_type": "generate-report",
        "params": {"report_type": "summary", "format": "pdf"},
        "depends_on": ["step-a"]
      }
    ]
  }'
```

Client-generated step IDs let you reference dependencies before the workflow is created. If you omit `id`, UUIDs are assigned server-side — but you cannot then express `depends_on` between steps in the same request.

## Adding a job type

Register a processor in `cmd/worker/main.go`:

```go
workerPool.RegisterProcessor("my-job", func(ctx context.Context, task *queue.Task) (map[string]interface{}, error) {
    // task.Data contains the fields from the job payload
    return map[string]interface{}{"done": true}, nil
})
```

Then add it to the dropdown in `playground/src/components/JobForm.jsx`.

## Output directories

Processed images are saved to `./output/images/`. Reports are saved to `./output/reports/`. Both directories are created automatically. Set `IMAGE_OUTPUT_DIR` and `REPORT_OUTPUT_DIR` in `.env` to override.

## Metrics

The worker exposes Prometheus metrics at `:9094/metrics`. The API exposes them at `:9093/metrics`.
