#!/bin/bash
set -e

# Configuration
VM_NAME="microvm"
ROOTFS_FILE="rootfs.ext4"
KERNEL_FILE="vmlinux.bin"
CONFIG_FILE="vm-config.json"
SOCKET_FILE="./tmp/${VM_NAME}.socket"
TAP_DEVICE="tap-${VM_NAME}"
VM_IP="172.16.0.2"
HOST_IP="172.16.0.1"

function show_help() {
    cat << EOF
Firecracker Docker VM Manager

Usage: $0 [COMMAND]

Commands:
    build       Build Docker image and create ext4 rootfs
    setup-net   Set up TAP network interface  
    start       Start the Firecracker VM
    stop        Stop the Firecracker VM
    ssh         SSH into the VM
    status      Check VM status
    cleanup     Clean up network and files
    logs        Show VM console output
    help        Show this help

Examples:
    $0 build && $0 setup-net && $0 start
    $0 ssh
    $0 stop && $0 cleanup

VM will be accessible at: $VM_IP
SSH: ssh root@$VM_IP (password: firecracker)
Docker API: http://$VM_IP:2375

EOF
}

function build_image() {
    echo "=== Building Docker image and creating rootfs ==="
    
    # Build Docker image
    docker build -t microvm .
    
    # Run conversion script
    bash docker_to_ext4.sh
    
    echo "Build complete: $ROOTFS_FILE"
}

function setup_network() {
    echo "=== Setting up TAP network ==="
    
    # Check if running as root
    if [[ $EUID -ne 0 ]]; then
        echo "Network setup requires root privileges. Run with sudo."
        exit 1
    fi
    
    # Create TAP interface
    ip tuntap add $TAP_DEVICE mode tap || true
    ip addr add $HOST_IP/24 dev $TAP_DEVICE || true
    ip link set $TAP_DEVICE up
    
    # Enable IP forwarding
    echo 1 > /proc/sys/net/ipv4/ip_forward
    
    # Set up NAT (optional, for internet access)
    iptables -t nat -A POSTROUTING -s 172.16.0.0/24 -j MASQUERADE || true
    iptables -A FORWARD -i $TAP_DEVICE -j ACCEPT || true
    iptables -A FORWARD -o $TAP_DEVICE -j ACCEPT || true
    
    echo "Network configured: $TAP_DEVICE ($HOST_IP/24)"
}

function start_vm() {
    echo "=== Starting Firecracker VM ==="
    
    # Check if files exist
    if [[ ! -f $KERNEL_FILE ]]; then
        echo "Kernel file not found: $KERNEL_FILE"
        echo "Run '$0 build' first"
        exit 1
    fi
    
    if [[ ! -f $ROOTFS_FILE ]]; then
        echo "Rootfs file not found: $ROOTFS_FILE" 
        echo "Run '$0 build' first"
        exit 1
    fi
    
    # Create VM configuration
    cat > $CONFIG_FILE << EOF
{
  "boot-source": {
    "kernel_image_path": "$KERNEL_FILE",
    "boot_args": "console=ttyS0 reboot=k panic=1 pci=off rw ip=$VM_IP::$HOST_IP:255.255.255.0::eth0:off"
  },
  "drives": [
    {
      "drive_id": "rootfs",
      "path_on_host": "$ROOTFS_FILE",
      "is_root_device": true,
      "is_read_only": false
    }
  ],
  "machine-config": {
    "vcpu_count": 2,
    "mem_size_mib": 2048,
    "smt": false
  },
  "network-interfaces": [
    {
      "iface_id": "eth0",
      "guest_mac": "AA:FC:00:00:00:01",
      "host_dev_name": "$TAP_DEVICE"
    }
  ],
  "logger": {
    "log_path": "${VM_NAME}.log",
    "level": "Info"
  }
}
EOF

    # Remove existing socket
    rm -f $SOCKET_FILE
    
    # Start Firecracker in background
    echo "Starting Firecracker VM..."
    firecracker --api-sock $SOCKET_FILE --config-file $CONFIG_FILE &
    
    FIRECRACKER_PID=$!
    echo "Firecracker started with PID: $FIRECRACKER_PID"
    echo $FIRECRACKER_PID > ${VM_NAME}.pid
    
    echo "VM booting... This may take 30-60 seconds"
    echo "Console output in: ${VM_NAME}.log"
    echo "Try SSH: ssh root@$VM_IP"
}

function stop_vm() {
    echo "=== Stopping Firecracker VM ==="
    
    if [[ -f firecracker-${VM_NAME}.pid ]]; then
        PID=$(cat firecracker-${VM_NAME}.pid)
        if kill -0 $PID 2>/dev/null; then
            kill $PID
            echo "VM stopped (PID: $PID)"
        fi
        rm -f firecracker-${VM_NAME}.pid
    fi
    
    rm -f $SOCKET_FILE
}

function ssh_vm() {
    echo "=== Connecting to VM via SSH ==="
    echo "Password: firecracker"
    ssh -o StrictHostKeyChecking=no root@$VM_IP
}

function vm_status() {
    echo "=== VM Status ==="
    
    if [[ -f firecracker-${VM_NAME}.pid ]]; then
        PID=$(cat firecracker-${VM_NAME}.pid)
        if kill -0 $PID 2>/dev/null; then
            echo "Firecracker VM running (PID: $PID)"
        else
            echo "Firecracker VM not running (stale PID file)"
        fi
    else
        echo "Firecracker VM not running"
    fi
    
    echo "Network interface: $TAP_DEVICE"
    ip addr show $TAP_DEVICE 2>/dev/null || echo "TAP interface not found"
    
    echo "VM connectivity:"
    ping -c 1 -W 1 $VM_IP && echo "VM is reachable" || echo "VM not reachable"
}

function show_logs() {
    echo "=== VM Console Logs ==="
    tail -f firecracker-${VM_NAME}.log 2>/dev/null || echo "No log file found"
}

function cleanup() {
    echo "=== Cleaning up ==="
    
    # Stop VM first
    stop_vm
    
    # Clean up network (requires root)
    if [[ $EUID -eq 0 ]]; then
        ip link delete $TAP_DEVICE 2>/dev/null || true
        echo "Network cleaned up"
    else
        echo "Run with sudo to clean up network interface"
    fi
    
    # Remove files
    rm -f $CONFIG_FILE ${VM_NAME}.log
    echo "Cleanup complete"
}

# Main command handling
case "${1:-}" in
    build)
        build_image
        ;;
    setup-net)
        setup_network
        ;;
    start)
        start_vm
        ;;
    stop)
        stop_vm
        ;;
    ssh)
        ssh_vm
        ;;
    status)
        vm_status
        ;;
    logs)
        show_logs
        ;;
    cleanup)
        cleanup
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        echo "Unknown command: ${1:-}"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac
