#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

# Function to build for a specific OS and architecture
build() {
    local os=$1
    local arch=$2
    local output="build/${os}_${arch}/easydns"

    echo "Building for ${os}/${arch}..."
    GOOS=${os} GOARCH=${arch} go build -o ${output}
    echo "Build completed: ${output}"
    chmod +x ${output}
}

# Create build directory
mkdir -p build

# Build for Linux (amd64)
build linux amd64

# Build for macOS (M1/M2)
build darwin arm64

# Build for Windows (amd64)
build windows amd64

echo "All builds completed successfully."