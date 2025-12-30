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
    
    BUILD_DIR="$(pwd)/build_$spec"
    
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
    
    # Clean up any existing containers first to avoid ContainerConfig errors
    sudo docker-compose down -v 2>/dev/null || true
    
    sudo docker-compose build
    
    echo "[4/5] Starting containers..."
    sudo docker-compose up -d
    cd ../..
    
    # Wait for services to be ready
    echo "[5/5] Waiting for services to start..."
    echo "Waiting for databases and backend services to initialize..."
    sleep 30
    
    # Check if frontend is responding with 200 OK
    echo "Checking if frontend service is ready..."
    MAX_RETRIES=60
    RETRY=0
    while [ $RETRY -lt $MAX_RETRIES ]; do
        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:12349/LoadCatalogue" 2>/dev/null || echo "000")
        if [ "$HTTP_CODE" = "200" ]; then
            echo "✓ Frontend is ready and returning 200 OK!"
            break
        fi
        RETRY=$((RETRY + 1))
        if [ $RETRY -eq $MAX_RETRIES ]; then
            echo "✗ Frontend failed to return 200 after ${MAX_RETRIES} attempts (last status: $HTTP_CODE)"
            echo "Checking container status..."
            cd $BUILD_DIR/docker
            sudo docker-compose ps
            echo "\n=== Frontend logs ==="
            sudo docker-compose logs frontend_ctr | tail -50
            echo "\n=== Catalogue logs ==="
            sudo docker-compose logs catalogue_ctr | tail -30
            echo "\n=== User logs ==="
            sudo docker-compose logs user_ctr | tail -30
            cd $BUILD_DIR
            echo "Skipping this configuration..."
            cd $BUILD_DIR/docker
            sudo docker-compose down 2>/dev/null || true
            continue 2  # Skip to next spec
        fi
        if [ $((RETRY % 10)) -eq 0 ]; then
            echo "  Waiting for frontend... (attempt $RETRY/$MAX_RETRIES, last status: $HTTP_CODE)"
        fi
        sleep 2
    done
    
    # Build and run workload generator
    echo "Running workload..."
    cd $BUILD_DIR/wlgen/wlgen_proc/wlgen_proc
    go build -o wlgen .
    cd ..
    set -a
    source ../../.local.env
    ./wlgen_proc/wlgen --duration $DURATION --rate $RATE > "../../../${RESULTS_DIR}/${spec}.txt" 2>&1
    cd ../../..
    
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
