.PHONY: all build clean server agent run test docker

# Variables
SERVER_BINARY=vigilon-server
AGENT_BINARY=vigilon-agent
VERSION=1.0.0

# Default target
all: build

# Build both server and agent
build: server agent

# Build server
server:
	@echo "Building server..."
	CGO_ENABLED=1 go build -ldflags="-s -w" -o $(SERVER_BINARY) cmd/server/main.go

# Build agent
agent:
	@echo "Building agent..."
	CGO_ENABLED=1 go build -ldflags="-s -w" -o $(AGENT_BINARY) cmd/agent/main.go

# Run server
run: server
	./$(SERVER_BINARY) -config configs/config.yaml

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	/bin/rm -f $(SERVER_BINARY) $(AGENT_BINARY)
	/bin/rm -f vigilon-server-* vigilon-agent-*
	/bin/rm -f *.db

# Build for multiple platforms
build-all: build-linux build-windows build-arm

build-linux:
	@echo "Building for Linux (amd64)..."
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(SERVER_BINARY)-linux-amd64 cmd/server/main.go
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(AGENT_BINARY)-linux-amd64 cmd/agent/main.go

build-windows:
	@echo "Building for Windows (amd64)..."
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(SERVER_BINARY)-windows-amd64.exe cmd/server/main.go
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(AGENT_BINARY)-windows-amd64.exe cmd/agent/main.go

build-arm:
	@echo "Building for ARM (Raspberry Pi)..."
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(AGENT_BINARY)-linux-arm64 cmd/agent/main.go

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run with hot reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	air

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Show help
help:
	@echo "Vigilon Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build       - Build server and agent"
	@echo "  make server      - Build only server"
	@echo "  make agent       - Build only agent"
	@echo "  make run         - Build and run server"
	@echo "  make test        - Run tests"
	@echo "  make clean       - Remove build artifacts"
	@echo "  make build-all   - Build for all platforms"
	@echo "  make deps        - Install dependencies"
	@echo "  make fmt         - Format code"
	@echo "  make lint        - Run linter"
	@echo "  make help        - Show this help"
