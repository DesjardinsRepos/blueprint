#!/bin/bash
# Performance comparison script for SockShop
# Compares: gRPC vs Thrift, Zipkin on vs off, Microservices vs Monolith

set -e

# Detect OS for sed syntax
if [[ "$OSTYPE" == "darwin"* ]]; then
    SED_INPLACE="sed -i ''"
else
    SED_INPLACE="sed -i"
fi

DURATION=60
RATE=50
RESULTS_DIR="performance_results"
mkdir -p $RESULTS_DIR

SPECS=(
    "cmp_grpc_zipkin_micro"
    "cmp_thrift_zipkin_micro"
    "cmp_grpc_nozipkin_micro"
    "cmp_thrift_nozipkin_micro"
    "cmp_grpc_zipkin_mono"
    "cmp_thrift_zipkin_mono"
    "cmp_grpc_nozipkin_mono"
    "cmp_thrift_nozipkin_mono"
)

echo "=== SockShop Performance Comparison ==="
echo "Duration: ${DURATION}s, Rate: ${RATE} req/s"
echo "Results will be saved to: ${RESULTS_DIR}/"
echo ""

for spec in "${SPECS[@]}"; do
    echo "========================================"
    echo "Testing: $spec"
    echo "========================================"
    
    BUILD_DIR="build_$spec"
    
    # Compile the spec
    echo "[1/5] Compiling..."
    go run wiring/main.go -o $BUILD_DIR -w $spec
    
    # Apply fixes for Go version issues
    echo "[2/5] Applying Go version fixes..."
    cd $BUILD_DIR
    cp .local.env docker/.env
    
    # Fix Dockerfiles - update Go version from 1.23 to 1.24
    find "$(pwd)/docker" -name Dockerfile -type f -exec $SED_INPLACE 's/1\.23/1.24/g' {} \;
    
    # Fix go.work files - update Go version from 1.23.1 to 1.24.0
    find "$(pwd)" -name "go.work" -type f -exec $SED_INPLACE 's/^go 1\.23\.1$/go 1.24.0/' {} \;
    
    cd ..
    
    # Build and start containers
    echo "[3/5] Building containers..."
    cd $BUILD_DIR/docker
    sudo docker-compose build
    
    echo "[4/5] Starting containers..."
    sudo docker-compose up -d
    cd ../..
    
    # Wait for services to be ready
    echo "[5/5] Waiting for services to start..."
    sleep 15
    
    # Build and run workload generator
    echo "Running workload..."
    cd $BUILD_DIR/wlgen/wlgen_proc
    go build -o wlgen ./wlgen_proc
    cd ..
    set -a
    source ../.local.env
    ./wlgen_proc/wlgen --duration $DURATION --rate $RATE > "../../${RESULTS_DIR}/${spec}.txt" 2>&1
    cd ../..
    
    # Stop containers
    echo "Stopping containers..."
    cd $BUILD_DIR/docker
    sudo docker-compose down
    cd ../..
    
    echo "✓ Completed: $spec"
    echo ""
done

echo "=== All tests completed ==="
echo "Results saved in: ${RESULTS_DIR}/"
echo ""
echo "Summary of final results:"
for spec in "${SPECS[@]}"; do
    echo "--- $spec ---"
    grep "Final Results" -A 1 "${RESULTS_DIR}/${spec}.txt" 2>/dev/null || echo "No results found"
    echo ""
done
