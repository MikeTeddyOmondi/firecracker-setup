#!/bin/bash
# setup-firecracker-docker-bridge.sh

TAP_NAME="tap0"
DOCKER_BRIDGE="docker0"
VM_IP="172.17.0.100"
BRIDGE_IP="172.17.0.1"

# Create TAP interface
if ! ip link show tap0 &>/dev/null; then
    sudo ip tuntap add $TAP_NAME mode tap user $(whoami)
fi
sudo ip link set $TAP_NAME up

# Add TAP to Docker bridge
sudo ip link set $TAP_NAME master $DOCKER_BRIDGE

# Ensure Docker bridge is up with IP
sudo ip addr add $BRIDGE_IP/16 dev $DOCKER_BRIDGE 2>/dev/null || true
sudo ip link set $DOCKER_BRIDGE up

# Enable IP forwarding
sudo sysctl net.ipv4.ip_forward=1

# Add NAT rule if not exists
sudo iptables -t nat -C POSTROUTING -s 172.17.0.0/16 ! -o $DOCKER_BRIDGE -j MASQUERADE 2>/dev/null || \
sudo iptables -t nat -A POSTROUTING -s 172.17.0.0/16 ! -o $DOCKER_BRIDGE -j MASQUERADE

echo "TAP interface $TAP_NAME created and bridged to $DOCKER_BRIDGE"
echo "Start Firecracker with: --tap-device $TAP_NAME/02:00:00:00:00:01"
echo "Configure VM with IP: $VM_IP/16, Gateway: $BRIDGE_IP"
