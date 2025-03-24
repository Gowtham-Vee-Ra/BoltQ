FROM golang:1.22-alpine

WORKDIR /app

# Install build dependencies and curl for healthcheck
RUN apk add --no-cache gcc musl-dev curl

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Default command (will be overridden in docker-compose)
CMD ["go", "run", "cmd/api/main.go"]