# Default
default:
    just --list

# Build Docker image 
build-image:
    docker build -t k8s-img -f ./setup/Dockerfile.k8s .

# Boot VM with firecracker 
boot_vm:
    sudo rm $API_SOCKET
    export API_SOCKET="/tmp/app.socket"
    sudo ./setup/firecracker --api-sock "${API_SOCKET}" --config-file config.json

