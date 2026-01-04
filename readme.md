### Ubuntu workflow

    # Clone and install prerequisites
    git clone https://github.com/Blueprint-uServices/blueprint
    cd blueprint/examples/sockshop/
    sudo apt update && sudo apt upgrade -y
    sudo apt install golang-go protobuf-compiler thrift-compiler docker.io docker-compose make micro bison flex libboost-all-dev -y
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    export PATH="$PATH:$HOME/go/bin"

    # Fix the thrift plugin
    cd ~/blueprint/plugins/thrift/thriftcodegen
    cp blueprint/thrift_plugin_fixes/clientgen.go clientgen.go
    cp blueprint/thrift_plugin_fixes/marshallgen.go marshallgen.go
    cp blueprint/thrift_plugin_fixes/servergen.go servergen.go
    cp blueprint/thrift_plugin_fixes/thriftgen.go thriftgen.go

    # Get the evaluation scripts, experiemental setups and custom workload generator
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

    # Execute the experiments
    ./run_comparison.sh
    ./analyze_results.py