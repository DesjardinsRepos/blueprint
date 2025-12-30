### Ubuntu workflow

    git clone https://github.com/Blueprint-uServices/blueprint
    cd blueprint/examples/sockshop/
    sudo apt update && sudo apt upgrade -y
    sudo apt install golang-go protobuf-compiler docker.io docker-compose make micro bison flex libboost-all-dev -y
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    export PATH="$PATH:$HOME/go/bin"

    cd ~
    wget https://archive.apache.org/dist/thrift/0.12.0/thrift-0.12.0.tar.gz
    tar -xzf thrift-0.12.0.tar.gz
    cd thrift-0.12.0
    ./configure --without-java --without-python --without-tests --without-nodejs --without-ruby --without-perl --without-php --without-csharp --without-erlang --without-lua --without-haskell --without-d
    make -j$(nproc)
    sudo cp compiler/cpp/thrift /usr/local/bin/thrift
    thrift --version

    cd ~/blueprint/examples/sockshop/
    git clone https://github.com/DesjardinsRepos/blueprint
    cp blueprint/sockshop/build_and_run.sh build_and_run.sh
    chmod +x build_and_run.sh
    cp blueprint/sockshop/run_comparison.sh run_comparison.sh
    chmod +x run_comparison.sh
    cp blueprint/sockshop/analyze_results.py analyze_results.py
    chmod +x analyze_results.py
    cp blueprint/sockshop/wiring/specs/comparison.go wiring/specs/comparison.go
    cp blueprint/sockshop/wiring/main.go wiring/main.go
    cp blueprint/sockshop/workload/workloadgen/workload.go workload/workloadgen/workload.go

    ./run_comparison.sh
    ./analyze_results.py