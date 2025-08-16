#!/bin/bash

# Build the bootable container image
echo "Building bootable container image..."
docker build -t microvm-base .

# Run the bootable container
echo "Starting bootable container..."
docker run -d \
    --name microvm-base \
    --privileged \
    --tmpfs /tmp \
    --tmpfs /run \
    --tmpfs /run/lock \
    --volume /sys/fs/cgroup:/sys/fs/cgroup:ro \
    --publish 2222:22 \
    --publish 2374:2375 \
    microvm-base

# Wait a moment for services to start
echo "Waiting for services to initialize..."
sleep 10

# Check if container is running
echo "Container status:"
docker ps -f name=microvm-base

# Show running services inside the container
echo "Services running inside container:"
docker exec microvm-base systemctl list-units --state=running

# Connect via SSH (password: 'password' for root or 'admin' user)
echo "You can now SSH to the container:"
echo "ssh -p 2222 root@localhost"
echo "ssh -p 2222 admin@localhost"
echo "Password: password"

# Access Docker inside the container
echo "To use Docker inside the container:"
echo "docker exec -it microvm-base docker version"

# Stop and cleanup commands
echo "To stop and remove:"
echo "docker stop microvm-base"
echo "docker rm microvm-base"
