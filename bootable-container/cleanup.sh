#!/bin/bash

echo "=== Firecracker Cleanup Script ==="

# Function to check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        echo "This script needs root privileges for network cleanup."
        echo "Run with: sudo $0"
        exit 1
    fi
}

# Function to kill Firecracker processes
kill_firecracker_processes() {
    echo "1. Killing Firecracker processes..."
    
    # Find Firecracker processes
    FIRECRACKER_PIDS=$(pgrep -f firecracker 2>/dev/null || true)
    
    if [[ -n "$FIRECRACKER_PIDS" ]]; then
        echo "   Found Firecracker processes: $FIRECRACKER_PIDS"
        for pid in $FIRECRACKER_PIDS; do
            echo "   Killing PID: $pid"
            kill -TERM "$pid" 2>/dev/null || true
            sleep 2
            # Force kill if still running
            if kill -0 "$pid" 2>/dev/null; then
                echo "   Force killing PID: $pid"
                kill -KILL "$pid" 2>/dev/null || true
            fi
        done
    else
        echo "   No Firecracker processes found"
    fi
}

# Function to clean up TAP interfaces
cleanup_tap_interfaces() {
    echo "2. Cleaning up TAP interfaces..."
    
    # Common TAP interface names
    TAP_INTERFACES=(tap-microvm tap0)
    
    for tap in "${TAP_INTERFACES[@]}"; do
        if ip link show "$tap" &>/dev/null; then
            echo "   Removing TAP interface: $tap"
            ip link set "$tap" down 2>/dev/null || true
            ip link delete "$tap" 2>/dev/null || true
        fi
    done
    
    # Find any remaining tap interfaces
    echo "   Checking for other TAP interfaces..."
    ip link show | grep -E "^[0-9]+: tap" | while read -r line; do
        TAP_NAME=$(echo "$line" | cut -d: -f2 | sed 's/^ *//')
        if [[ "$TAP_NAME" =~ ^tap ]]; then
            echo "   Found additional TAP interface: $TAP_NAME"
            ip link set "$TAP_NAME" down 2>/dev/null || true
            ip link delete "$TAP_NAME" 2>/dev/null || true
        fi
    done
}

# Function to clean up socket files
cleanup_sockets() {
    echo "3. Cleaning up socket files..."
    
    SOCKET_PATTERNS=(/tmp/*microvm.socket)
    
    for pattern in "${SOCKET_PATTERNS[@]}"; do
        for socket in $pattern; do
            if [[ -e "$socket" ]]; then
                echo "   Removing: $socket"
                rm -f "$socket"
            fi
        done
    done
}

# Function to clean up PID files and logs
cleanup_files() {
    echo "4. Cleaning up PID files and logs..."
    
    FILE_PATTERNS=(*.pid microvm*.log)
    
    for pattern in "${FILE_PATTERNS[@]}"; do
        for file in $pattern; do
            if [[ -f "$file" ]]; then
                echo "   Found: $file"
                read -p "   Remove $file? (y/n): " -n 1 -r
                echo
                if [[ $REPLY =~ ^[Yy]$ ]]; then
                    rm -f "$file"
                    echo "   Removed: $file"
                fi
            fi
        done
    done
}

# Function to check for running VMs
check_running_vms() {
    echo "5. Checking for active VMs..."
    
    # Check for any processes using the rootfs files
    ROOTFS_FILES=(*.ext4)
    for rootfs in "${ROOTFS_FILES[@]}"; do
        if [[ -f "$rootfs" ]]; then
            PROCS=$(lsof "$rootfs" 2>/dev/null | tail -n +2 || true)
            if [[ -n "$PROCS" ]]; then
                echo "   Processes using $rootfs:"
                echo "$PROCS"
            fi
        fi
    done
}

# Function to show network status
show_network_status() {
    echo "6. Current network status..."
    
    echo "   TAP interfaces:"
    ip link show | grep tap || echo "   No TAP interfaces found"
    
    echo "   Bridge interfaces:"
    ip link show type bridge || echo "   No bridge interfaces found"
    
    echo "   Routing table:"
    ip route | grep 172.16 || echo "   No 172.16.x.x routes found"
}

# Main execution
case "${1:-all}" in
    processes)
        kill_firecracker_processes
        ;;
    network)
        check_root
        cleanup_tap_interfaces
        ;;
    sockets)
        cleanup_sockets
        ;;
    files)
        cleanup_files
        ;;
    status)
        show_network_status
        check_running_vms
        ;;
    all)
        kill_firecracker_processes
        cleanup_sockets
        if [[ $EUID -eq 0 ]]; then
            cleanup_tap_interfaces
            show_network_status
        else
            echo "Skipping network cleanup (not root)"
            echo "Run 'sudo $0 network' to clean up network interfaces"
        fi
        cleanup_files
        check_running_vms
        ;;
    *)
        echo "Usage: $0 [processes|network|sockets|files|status|all]"
        echo ""
        echo "Commands:"
        echo "  processes - Kill all Firecracker processes"
        echo "  network   - Clean up TAP interfaces (requires root)"
        echo "  sockets   - Remove socket files"
        echo "  files     - Clean up PID files and logs (interactive)"
        echo "  status    - Show current status"
        echo "  all       - Run all cleanup steps"
        echo ""
        echo "For immediate fix:"
        echo "  sudo $0 all"
        ;;
esac

echo ""
echo "=== Cleanup Summary ==="
echo "After cleanup, you should be able to start Firecracker again."
echo "If TAP interface issues persist, try:"
echo "  sudo ip link delete tap-microvm"
echo "  sudo modprobe -r tun && sudo modprobe tun"
