# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.24-alpine AS build

WORKDIR /src

# Download dependencies first so this layer is cached across source changes
COPY go.mod go.sum ./
RUN go mod download

# Copy the source and compile static binaries (CGO disabled -> no libc dependency)
COPY . .
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

# ---- Runtime stage ----
FROM alpine:3.20

WORKDIR /app

# ca-certificates for outbound TLS (Pusher, image fetch); curl for container healthchecks
RUN apk add --no-cache ca-certificates curl \
 && adduser -D -u 10001 boltq \
 && mkdir -p /app/bin /app/output/images /app/output/reports \
 && chown -R boltq:boltq /app

COPY --from=build /out/api  /app/bin/api
COPY --from=build /out/worker /app/bin/worker

USER boltq

# Default to the API; docker-compose overrides command per service.
CMD ["./bin/api"]
