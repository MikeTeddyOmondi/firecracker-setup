# Default
default:
    just --list

# Build Docker image 
build-image:
    docker build -t k8s-img -f ./setup/Dockerfile.k8s .

# Boot VM with firecracker 
boot_vm:
    export API_SOCKET="/tmp/app.socket"
    sudo ./setup/firecracker --api-sock "${API_SOCKET}" --config-file config.json

# Remove socket
rm_sock:
    sudo rm $API_SOCKET

# Run Go application
go_run:    
    go run main.go \
    --name=clxx \
    --nodes=3 \
    --memory=1024 \
    --vcpu=1 \
    --rootfs=./setup/k8s-img-rootfs.ext4 \
    --persistent=true \
    --subnet=172.16.0.0/24 \
    --gateway=172.16.0.1

# Build Go app binary
go_build:
    go build .

# set roo capabilities for the Go binary
set_cap:
    sudo setcap cap_net_admin+ep ./firecracker-k8s

# Execute Go binary    
run_bin:   
    just go_build
    just set_cap
    
    ./firecracker-k8s \
    --name=clxx \
    --nodes=3 \
    --memory=1024 \
    --vcpu=1 \
    --rootfs=./setup/k8s-img-rootfs.ext4 \
    --persistent=true \
    --subnet=172.16.0.0/24 \
    --gateway=172.16.0.1

