#!/bin/bash
# Build and run a single SockShop configuration
# Usage: ./build_and_run.sh <spec_name> [build_dir]
#
# Example: ./build_and_run.sh cmp_grpc_zipkin_micro build_test

set -e

# Detect OS for sed syntax
if [[ "$OSTYPE" == "darwin"* ]]; then
    SED_INPLACE="sed -i ''"
else
    SED_INPLACE="sed -i"
fi

if [ -z "$1" ]; then
    echo "Usage: $0 <spec_name> [build_dir]"
    echo ""
    echo "Available specs:"
    echo "  cmp_grpc_zipkin_micro"
    echo "  cmp_thrift_zipkin_micro"
    echo "  cmp_grpc_nozipkin_micro"
    echo "  cmp_thrift_nozipkin_micro"
    echo "  cmp_grpc_zipkin_mono"
    echo "  cmp_thrift_zipkin_mono"
    echo "  cmp_grpc_nozipkin_mono"
    echo "  cmp_thrift_nozipkin_mono"
    echo "  docker (original docker spec)"
    exit 1
fi

SPEC=$1
BUILD_DIR=${2:-build}

echo "=== Building SockShop with spec: $SPEC ==="
echo "Output directory: $BUILD_DIR"
echo ""

# Compile the spec
echo "[1/5] Compiling..."
go run wiring/main.go -o $BUILD_DIR -w $SPEC

# Apply fixes for Go version issues
echo "[2/5] Applying Go version fixes..."
cd $BUILD_DIR
cp .local.env docker/.env

# Fix Dockerfiles - update Go version from 1.23 to 1.24
echo "  - Fixing Dockerfiles..."
find "$(pwd)/docker" -name Dockerfile -type f -exec $SED_INPLACE 's/1\.23/1.24/g' {} \;

# Fix go.work files - update Go version from 1.23.1 to 1.24.0
echo "  - Fixing go.work files..."
find "$(pwd)" -name "go.work" -type f -exec $SED_INPLACE 's/^go 1\.23\.1$/go 1.24.0/' {} \;

cd ..

# Build containers
echo "[3/5] Building containers..."
cd $BUILD_DIR/docker

# Clean up any existing containers first to avoid ContainerConfig errors
sudo docker-compose down -v 2>/dev/null || true

sudo docker-compose build

echo "[4/5] Starting containers..."
sudo docker-compose up -d

cd ../..

# Wait for services to be ready
echo "[5/5] Waiting for services to start..."
sleep 15

echo ""
echo "✓ SockShop is ready!"
echo ""
echo "To run the workload generator:"
echo "  cd $BUILD_DIR/wlgen/wlgen_proc"
echo "  go build -o wlgen ./wlgen_proc"
echo "  cd .."
echo "  set -a && source ../.local.env"
echo "  ./wlgen_proc/wlgen --duration 60 --rate 10"
echo ""
echo "To stop the containers:"
echo "  cd $BUILD_DIR/docker"
echo "  sudo docker-compose down"
