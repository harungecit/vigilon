.PHONY: all build clean server agent run test docker

# Variables
SERVER_BINARY=vigilon-server
AGENT_BINARY=vigilon-agent
VERSION=1.1.2
GO=/usr/local/go/bin/go

# Default target
all: build

# Build both server and agent
build: server agent

# Build server
server:
	@echo "Building server..."
	CGO_ENABLED=1 $(GO) build -ldflags="-s -w" -o $(SERVER_BINARY) cmd/server/main.go

# Build agent
agent:
	@echo "Building agent..."
	CGO_ENABLED=1 $(GO) build -ldflags="-s -w" -o $(AGENT_BINARY) cmd/agent/main.go

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
# Build for multiple platforms
build-all: build-linux build-windows-check build-arm-check copy-to-web

build-linux:
	@echo "Building for Linux (amd64)..."
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GO) build -ldflags="-s -w" -o $(SERVER_BINARY)-linux-amd64 cmd/server/main.go
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GO) build -ldflags="-s -w" -o $(AGENT_BINARY)-linux-amd64 cmd/agent/main.go

build-windows-check:
	@echo "Building for Windows (amd64)..."
	@if command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then \
		CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc $(GO) build -ldflags="-s -w" -o $(AGENT_BINARY)-windows-amd64.exe cmd/agent/main.go && \
		echo "✓ Windows build successful"; \
	else \
		echo "⚠️  Skipping Windows build (x86_64-w64-mingw32-gcc not found)"; \
		echo "   Install with: sudo apt-get install gcc-mingw-w64-x86-64"; \
	fi

build-arm-check:
	@echo "Building for ARM (Raspberry Pi)..."
	@if command -v aarch64-linux-gnu-gcc >/dev/null 2>&1; then \
		CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc $(GO) build -ldflags="-s -w" -o $(AGENT_BINARY)-linux-arm64 cmd/agent/main.go && \
		echo "✓ ARM64 build successful"; \
	else \
		echo "⚠️  Skipping ARM64 build (aarch64-linux-gnu-gcc not found)"; \
		echo "   Install with: sudo apt-get install gcc-aarch64-linux-gnu"; \
	fi

# Copy agent binaries to web static directory for download
copy-to-web:
	@echo "Copying agent binaries to web/static/bin/..."
	@mkdir -p web/static/bin
	@if [ -f $(AGENT_BINARY)-linux-amd64 ]; then \
		cp $(AGENT_BINARY)-linux-amd64 web/static/bin/ && \
		echo "✓ Copied $(AGENT_BINARY)-linux-amd64"; \
	fi
	@if [ -f $(AGENT_BINARY)-linux-arm64 ]; then \
		cp $(AGENT_BINARY)-linux-arm64 web/static/bin/ && \
		echo "✓ Copied $(AGENT_BINARY)-linux-arm64"; \
	fi
	@if [ -f $(AGENT_BINARY)-windows-amd64.exe ]; then \
		cp $(AGENT_BINARY)-windows-amd64.exe web/static/bin/ && \
		echo "✓ Copied $(AGENT_BINARY)-windows-amd64.exe"; \
	fi

build-windows:
	@echo "Building for Windows (amd64)..."
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc $(GO) build -ldflags="-s -w" -o $(SERVER_BINARY)-windows-amd64.exe cmd/server/main.go
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc $(GO) build -ldflags="-s -w" -o $(AGENT_BINARY)-windows-amd64.exe cmd/agent/main.go

build-arm:
	@echo "Building for ARM (Raspberry Pi)..."
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc $(GO) build -ldflags="-s -w" -o $(AGENT_BINARY)-linux-arm64 cmd/agent/main.go

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

# Create git tag and prepare release
tag:
	@echo "Creating git tag v$(VERSION)..."
	@if git rev-parse v$(VERSION) >/dev/null 2>&1; then \
		echo "Tag v$(VERSION) already exists!"; \
		exit 1; \
	fi
	git tag -a v$(VERSION) -m "Release v$(VERSION)"
	@echo "✓ Tag v$(VERSION) created"
	@echo ""
	@echo "To push the tag, run:"
	@echo "  git push origin v$(VERSION)"

# Create GitHub release (requires gh CLI)
release: build-all
	@echo "Creating GitHub release v$(VERSION)..."
	@if ! command -v gh >/dev/null 2>&1; then \
		echo "Error: GitHub CLI (gh) not found"; \
		echo "Install with: sudo apt install gh"; \
		exit 1; \
	fi
	@if ! git rev-parse v$(VERSION) >/dev/null 2>&1; then \
		echo "Creating tag v$(VERSION)..."; \
		git tag -a v$(VERSION) -m "Release v$(VERSION)"; \
	fi
	@echo "Pushing tag..."
	git push origin v$(VERSION) || true
	@echo "Creating release with binaries..."
	gh release create v$(VERSION) \
		--title "Vigilon v$(VERSION)" \
		--notes-file CHANGELOG.md \
		$(SERVER_BINARY)-linux-amd64 \
		$(AGENT_BINARY)-linux-amd64 \
		$(AGENT_BINARY)-linux-arm64 \
		$(AGENT_BINARY)-windows-amd64.exe
	@echo "✓ Release v$(VERSION) created successfully!"

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
	@echo "  make tag         - Create git tag for current version"
	@echo "  make release     - Build and create GitHub release"
	@echo "  make fmt         - Format code"
	@echo "  make lint        - Run linter"
	@echo "  make help        - Show this help"
