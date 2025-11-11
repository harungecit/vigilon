# Multi-stage Dockerfile for Vigilon Server and Agent
# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build server and agent
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o vigilon-server cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o vigilon-agent cmd/agent/main.go

# Server runtime stage
FROM alpine:latest AS server

RUN apk --no-cache add ca-certificates openssh-client tzdata

WORKDIR /app

# Copy server binary and web assets
COPY --from=builder /build/vigilon-server .
COPY --from=builder /build/web ./web
COPY --from=builder /build/configs/config.example.yaml ./configs/config.yaml

# Create data directory
RUN mkdir -p /app/data

# Expose port
EXPOSE 8090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8090/api/health || exit 1

# Run server
CMD ["./vigilon-server", "-config", "configs/config.yaml"]

# Agent runtime stage
FROM alpine:latest AS agent

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy agent binary
COPY --from=builder /build/vigilon-agent .
COPY --from=builder /build/configs/agent-config.example.yaml ./config.yaml

# Run agent
CMD ["./vigilon-agent", "-config", "config.yaml"]
