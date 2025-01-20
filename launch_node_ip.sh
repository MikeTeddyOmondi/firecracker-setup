#!/bin/bash

# Check if the required arguments are provided
if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <vm-type>"
    echo "vm-type: control-plane | worker"
    exit 1
fi

VM_TYPE=$1
VM_ID="k8s-${VM_TYPE}-vm"
SOCKET_PATH="./${VM_ID}.sock"
KERNEL_PATH="./setup/vmlinux-5.10.225"
ROOTFS_PATH="./setup/k8s-img-rootfs.ext4"

# Firecracker machine configuration
VCPU_COUNT=1
MEM_SIZE_MIB=512
SMT=false

# Create a TAP interface
# TAP_NAME="tap-${VM_ID}"
# sudo ip tuntap add dev "${TAP_NAME}" mode tap
# sudo ip link set dev "${TAP_NAME}" up

TAP_NAME="tap0"
TAP_IP="192.168.1.100"
MASK_SHORT="/30"

sudo ip link del "$TAP_NAME" 2>/dev/null || true
sudo ip tuntap add dev "$TAP_NAME" mode tap
sudo ip addr add "${TAP_IP}${MASK_SHORT}" dev "$TAP_NAME"
sudo ip link set dev "$TAP_NAME" up

# # Create a bridge if it doesn't exist
# BRIDGE_NAME="br0"
# if ! ip link show "${BRIDGE_NAME}" > /dev/null 2>&1; then
#     sudo ip link add name "${BRIDGE_NAME}" type bridge
#     sudo ip link set dev "${BRIDGE_NAME}" up
# fi

# # Add the TAP interface to the bridge
# sudo ip link set dev "${TAP_NAME}" master "${BRIDGE_NAME}"

# Assign a static IP address based on the VM type
case "${VM_TYPE}" in
    control-plane)
        STATIC_IP="192.168.1.100"
        ;;
    worker)
        STATIC_IP="192.168.1.101"  # Change this for additional workers
        ;;
    *)
        echo "Unknown VM type: ${VM_TYPE}"
        exit 1
        ;;
esac

# Firecracker machine configuration
cat <<EOF > ./firecracker-config.json
{
    "boot-source": {
        "kernel_image_path": "${KERNEL_PATH}",
        "boot_args": "console=ttyS0 reboot=k panic=1 pci=off"
    },
    "drives": [{
        "drive_id": "rootfs",
        "is_root_device": true,
        "is_read_only": false,
        "path_on_host": "${ROOTFS_PATH}"
    }],
    "machine-config": {
        "vcpu_count": ${VCPU_COUNT},
        "mem_size_mib": ${MEM_SIZE_MIB},
        "smt": ${SMT}
    },
    "network-interfaces": [{
        "iface_id": "eth0",
        "guest_mac": "AA:FC:00:00:00:01",
        "host_dev_name": "${TAP_NAME}"
    }]
}
EOF

# Start the Firecracker microVM
firecracker --api-sock "${SOCKET_PATH}" --config-file ./firecracker-config.json &

# Wait for the Firecracker process to start
sleep 1

# Assign the static IP address to the TAP interface
sudo ip addr add "${STATIC_IP}/24" dev "${TAP_NAME}"

# Bring up the interface inside the microVM
if [ "${VM_TYPE}" == "control-plane" ]; then
    echo "Control plane VM ${VM_ID} started with IP ${STATIC_IP}."
else
    echo "Worker VM ${VM_ID} started with IP ${STATIC_IP}."
fi

# Clean up
rm ./firecracker-config.json
