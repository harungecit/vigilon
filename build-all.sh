#!/bin/bash

# Cross-platform build script for Vigilon
# This script builds binaries for all supported platforms

set -e

VERSION=${VERSION:-"1.0.0"}
OUTPUT_DIR="./dist"

echo "🚀 Building Vigilon v${VERSION} for all platforms..."
echo ""

# Create output directory
mkdir -p ${OUTPUT_DIR}

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Build for Linux AMD64 (Server + Agent)
echo -e "${BLUE}Building for Linux AMD64...${NC}"
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o ${OUTPUT_DIR}/vigilon-server-linux-amd64 \
    cmd/server/main.go
echo -e "${GREEN}✓ vigilon-server-linux-amd64${NC}"

CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o ${OUTPUT_DIR}/vigilon-agent-linux-amd64 \
    cmd/agent/main.go
echo -e "${GREEN}✓ vigilon-agent-linux-amd64${NC}"
echo ""

# Build for Linux ARM64 (Agent only - Raspberry Pi)
echo -e "${BLUE}Building for Linux ARM64 (Raspberry Pi)...${NC}"
if command -v aarch64-linux-gnu-gcc &> /dev/null; then
    CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc go build \
        -ldflags="-s -w -X main.version=${VERSION}" \
        -o ${OUTPUT_DIR}/vigilon-agent-linux-arm64 \
        cmd/agent/main.go
    echo -e "${GREEN}✓ vigilon-agent-linux-arm64${NC}"
else
    echo "⚠️  Skipping ARM64 build (aarch64-linux-gnu-gcc not found)"
    echo "   Install with: sudo apt-get install gcc-aarch64-linux-gnu"
fi
echo ""

# Build for Windows AMD64 (Agent only)
echo -e "${BLUE}Building for Windows AMD64...${NC}"
if command -v x86_64-w64-mingw32-gcc &> /dev/null; then
    CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build \
        -ldflags="-s -w -X main.version=${VERSION}" \
        -o ${OUTPUT_DIR}/vigilon-agent-windows-amd64.exe \
        cmd/agent/main.go
    echo -e "${GREEN}✓ vigilon-agent-windows-amd64.exe${NC}"
else
    echo "⚠️  Skipping Windows build (x86_64-w64-mingw32-gcc not found)"
    echo "   Install with: sudo apt-get install gcc-mingw-w64-x86-64"
fi
echo ""

# Copy binaries to web/static/bin for install scripts
echo -e "${BLUE}Copying agent binaries to web/static/bin/...${NC}"
mkdir -p web/static/bin
cp ${OUTPUT_DIR}/vigilon-agent-linux-amd64 web/static/bin/ 2>/dev/null || true
cp ${OUTPUT_DIR}/vigilon-agent-linux-arm64 web/static/bin/ 2>/dev/null || true
cp ${OUTPUT_DIR}/vigilon-agent-windows-amd64.exe web/static/bin/ 2>/dev/null || true
echo -e "${GREEN}✓ Agent binaries copied${NC}"
echo ""

# Generate checksums
echo -e "${BLUE}Generating SHA256 checksums...${NC}"
cd ${OUTPUT_DIR}
sha256sum vigilon-* > checksums.txt 2>/dev/null || shasum -a 256 vigilon-* > checksums.txt
cd ..
echo -e "${GREEN}✓ checksums.txt generated${NC}"
echo ""

# Show build summary
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📦 Build Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
ls -lh ${OUTPUT_DIR}/ | grep vigilon
echo ""
echo -e "${GREEN}✨ Build complete!${NC}"
echo ""
echo "📂 Binaries: ${OUTPUT_DIR}/"
echo "🔐 Checksums: ${OUTPUT_DIR}/checksums.txt"
echo "🌐 Web binaries: web/static/bin/"
