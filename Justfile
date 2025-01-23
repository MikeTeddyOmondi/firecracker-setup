# Default
default:
    just --list

# Build Docker image 
build_image:
    # docker build -t k8s-img -f ./setup/Dockerfile.k8s .
    docker build -t k8s-img -f ./setup/Dockerfile.k8s-ssh .

# Boot VM with firecracker 
boot_vm:
    export API_SOCKET="/tmp/app.socket"
    sudo ./setup/bin/firecracker --api-sock "${API_SOCKET}" --config-file config.json

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


# Manual tap network setting 
set_tun:
    sudo ip link add name br0 type bridge
    sudo ip tuntap add tap0 mode tap
    sudo ip link set dev tap0 master br0
    
    sudo ip addr add 172.16.0.10/30 dev tap0
    sudo ip addr add 172.16.0.20/30 dev tap0
    sudo ip addr add 172.16.0.21/30 dev tap0
    
    sudo ip tuntap add dev tap-clxx-wk-0 mode tap
    sudo ip tuntap add dev tap-clxx-wk-1 mode tap
    sudo ip tuntap add dev tap-clxx-ms mode tap
    
    sudo ip link set dev tap-clxx-wk-0 up
    sudo ip link set dev tap-clxx-wk-1 up
    sudo ip link set dev tap-clxx-ms up


# Delete folders
clean_up:
    sudo ip link delete tap-clxx-wk-0 || true
    sudo ip link delete tap-clxx-wk-1 || true
    sudo ip link delete tap-clxx-ms || true
    rm -rf ./firecracker-k8s-cluster
    pkill -f firecracker || true

