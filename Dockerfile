# ----------- Build stage -----------
    FROM golang:1.22 AS builder

    # Install git (needed for go mod download if using private or VCS-based modules)
    RUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*
    
    # Set working directory
    WORKDIR /app
    
    # Copy go.mod and go.sum first (leverages Docker cache)
    COPY go.mod go.sum ./
    
    # Download dependencies
    RUN go mod download
    
    # Copy the rest of the source code
    COPY . .
    
    # Build the API service
    RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o boltq-api ./cmd/api
    
    # Build the worker service
    RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o boltq-worker ./cmd/worker
    
    # ----------- Final image: API -----------
    FROM alpine:3.18 AS api
    
    # Add CA certificates
    RUN apk --no-cache add ca-certificates
    
    # Create non-root user
    RUN addgroup -S appgroup && adduser -S appuser -G appgroup
    
    # Set working directory
    WORKDIR /app
    
    # Copy binary from the builder stage
    COPY --from=builder /app/boltq-api .
    
    # Use non-root user
    USER appuser
    
    # Expose port for API
    EXPOSE 8080
    
    # Run the API service
    CMD ["./boltq-api"]
    
    # ----------- Final image: Worker -----------
    FROM alpine:3.18 AS worker
    
    # Add CA certificates
    RUN apk --no-cache add ca-certificates
    
    # Create non-root user
    RUN addgroup -S appgroup && adduser -S appuser -G appgroup
    
    # Set working directory
    WORKDIR /app
    
    # Copy binary from the builder stage
    COPY --from=builder /app/boltq-worker .
    
    # Use non-root user
    USER appuser
    
    # Run the worker service
    CMD ["./boltq-worker"]
    