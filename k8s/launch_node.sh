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

# Create the Firecracker configuration
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
    }
}
EOF

# Start the Firecracker microVM
firecracker --api-sock "${SOCKET_PATH}" --config-file ./firecracker-config.json &

# Wait for the Firecracker process to start
sleep 1

# Check if the VM is running
if [ "$VM_TYPE" == "control-plane" ]; then
    # Additional setup for control plane (e.g., networking)
    echo "Control plane VM ${VM_ID} started."
else
    echo "Worker VM ${VM_ID} started."
fi

# Clean up
rm ./firecracker-config.json
