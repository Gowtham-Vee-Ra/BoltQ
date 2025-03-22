# Build stage
FROM golang:1.20-alpine AS builder

# Install dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build API service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o boltq-api ./cmd/api

# Build worker service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o boltq-worker ./cmd/worker

# Final stage for API
FROM alpine:3.18 AS api

# Add CA certificates
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/boltq-api .

# Use non-root user
USER appuser

# Expose port
EXPOSE 8080

# Run the API service
CMD ["./boltq-api"]

# Final stage for worker
FROM alpine:3.18 AS worker

# Add CA certificates
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/boltq-worker .

# Use non-root user
USER appuser

# Run the worker service
CMD ["./boltq-worker"]