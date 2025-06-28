#!/bin/bash

# Build script for Linux binaries
# Builds Mist for various Linux architectures

# Don't exit on error, we'll handle individual build failures
set +e

# Define the output directory for binaries
OUTPUT_DIR="./bin"

# Ensure the output directory exists
mkdir -p "$OUTPUT_DIR"

# Define version (you can update this or get from git tag)
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
LDFLAGS="-X main.version=$VERSION -s -w"

echo "Building Mist Linux binaries (version: $VERSION)..."
echo

# Build function with error handling
build_target() {
    local os=$1
    local arch=$2
    local output_name="mist-$os-$arch"
    local output_path="$OUTPUT_DIR/$output_name"
    
    echo "Building for $os $arch..."
    if env GOOS=$os GOARCH=$arch go build -ldflags "$LDFLAGS" -o "$output_path" ./cmd/mist 2>/dev/null; then
        echo "✅ Built: $output_path"
        return 0
    else
        echo "❌ Failed to build for $os $arch (likely due to TiDB 32-bit compatibility issues)"
        return 1
    fi
}

# Build for Linux AMD64 (most common - should always work)
build_target linux amd64

# Build for Linux ARM64 (Raspberry Pi, ARM servers - should work)
build_target linux arm64

# Build for Linux ARM (32-bit ARM devices - may fail due to TiDB)
build_target linux arm

# Build for Linux 386 (32-bit x86 - may fail due to TiDB)
build_target linux 386

echo
echo "Build process completed!"
echo

# Count successful builds
if ls "$OUTPUT_DIR"/mist-linux-* 1> /dev/null 2>&1; then
    echo "Successfully built binaries:"
    ls -la "$OUTPUT_DIR"/mist-linux-*
else
    echo "❌ No binaries were built successfully!"
    exit 1
fi
echo
echo "Usage examples:"
echo "  # Run on AMD64 Linux:"
echo "  ./bin/mist-linux-amd64"
echo "  ./bin/mist-linux-amd64 -i"
echo "  ./bin/mist-linux-amd64 -d --port 3306"
echo
echo "  # Run on ARM64 Linux (Raspberry Pi):"
echo "  ./bin/mist-linux-arm64 -i"
echo
echo "  # Run on 32-bit ARM Linux:"
echo "  ./bin/mist-linux-arm -i"