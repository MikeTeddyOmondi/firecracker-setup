#!/bin/bash
set -e

# Configuration
DOCKER_IMAGE="microvm"
ROOTFS_SIZE="2G"
ROOTFS_FILE="rootfs.ext4"
KERNEL_VERSION="5.10.186"
KERNEL_URL="https://github.com/firecracker-microvm/firecracker/releases/download/v1.4.1/vmlinux.bin"

echo "=== Docker Image to ext4 image format Conversion ==="

# Step 1: Create a temporary directory
TEMP_DIR=$(mktemp -d)
echo "Working directory: $TEMP_DIR"

# Step 2: Export Docker image filesystem
echo "Exporting Docker image filesystem..."
docker create --name temp-container $DOCKER_IMAGE
docker export temp-container | tar -C $TEMP_DIR -xf -
docker rm temp-container

# Step 3: Create ext4 filesystem image
echo "Creating ext4 filesystem ($ROOTFS_SIZE)..."
dd if=/dev/zero of=$ROOTFS_FILE bs=1M count=0 seek=$(echo $ROOTFS_SIZE | sed 's/G//')000
mkfs.ext4 -F $ROOTFS_FILE

# Step 4: Mount and copy filesystem
echo "Mounting and copying filesystem..."
MOUNT_POINT=$(mktemp -d)
sudo mount -o loop $ROOTFS_FILE $MOUNT_POINT

# Copy all files from container
sudo cp -a $TEMP_DIR/* $MOUNT_POINT/

# Step 5: Configure for Firecracker boot
echo "Configuring for Firecracker..."

# Create init script for Firecracker
sudo tee $MOUNT_POINT/init > /dev/null << 'EOF'
#!/bin/bash

# Mount essential filesystems
mount -t proc proc /proc
mount -t sysfs sysfs /sys
mount -t devtmpfs devtmpfs /dev
mkdir -p /dev/pts
mount -t devpts devpts /dev/pts
mount -t tmpfs tmpfs /tmp
mount -t tmpfs tmpfs /run

# Create necessary device nodes
mknod /dev/null c 1 3
mknod /dev/zero c 1 5
mknod /dev/random c 1 8
mknod /dev/urandom c 1 9
chmod 666 /dev/null /dev/zero /dev/random /dev/urandom

# Set hostname
echo "firecracker-vm" > /proc/sys/kernel/hostname

# Start systemd
exec /lib/systemd/systemd
EOF

sudo chmod +x $MOUNT_POINT/init

# Configure systemd for Firecracker
sudo mkdir -p $MOUNT_POINT/etc/systemd/system/serial-getty@ttyS0.service.d
sudo tee $MOUNT_POINT/etc/systemd/system/serial-getty@ttyS0.service.d/override.conf > /dev/null << 'EOF'
[Service]
ExecStart=
ExecStart=-/sbin/agetty --autologin root -8 --keep-baud 115200,38400,9600 ttyS0 $TERM
EOF

# Enable serial console
sudo systemctl --root=$MOUNT_POINT enable serial-getty@ttyS0.service

# Configure network (Firecracker uses eth0)
sudo tee $MOUNT_POINT/etc/systemd/network/eth0.network > /dev/null << 'EOF'
[Match]
Name=eth0

[Network]
DHCP=yes
EOF

sudo systemctl --root=$MOUNT_POINT enable systemd-networkd
sudo systemctl --root=$MOUNT_POINT enable systemd-resolved

# Fix DNS resolution
sudo ln -sf /run/systemd/resolve/resolv.conf $MOUNT_POINT/etc/resolv.conf

# Create fstab
sudo tee $MOUNT_POINT/etc/fstab > /dev/null << 'EOF'
/dev/vda / ext4 defaults 0 1
proc /proc proc defaults 0 0
sysfs /sys sysfs defaults 0 0
devtmpfs /dev devtmpfs defaults 0 0
devpts /dev/pts devpts defaults 0 0
tmpfs /tmp tmpfs defaults 0 0
tmpfs /run tmpfs defaults 0 0
EOF

# Step 6: Unmount and cleanup
sudo umount $MOUNT_POINT
rmdir $MOUNT_POINT
rm -rf $TEMP_DIR

echo "=== Rootfs created: $ROOTFS_FILE ==="

# # Step 7: Download Firecracker kernel if needed
# if [ ! -f "vmlinux.bin" ]; then
#     echo "Downloading Firecracker kernel..."
#     curl -L -o vmlinux.bin $KERNEL_URL
# fi

echo "=== Kernel ready: vmlinux.bin ==="

# Step 8: Create Firecracker configuration
echo "Creating Firecracker configuration..."
cat > vm-config.json << EOF
{
  "boot-source": {
    "kernel_image_path": "vmlinux.bin",
    "boot_args": "console=ttyS0 reboot=k panic=1 pci=off init=/init"
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
    "vcpu_count": 1,
    "mem_size_mib": 1024
  },
  "network-interfaces": [
    {
      "iface_id": "eth0",
      "guest_mac": "AA:FC:00:00:00:01",
      "host_dev_name": "tap0"
    }
  ]
}
EOF

echo "=== Firecracker config created: vm-config.json ==="
echo ""
echo "To run with Firecracker:"
echo "1. Set up TAP interface:"
echo "   sudo ip tuntap add tap0 mode tap"
echo "   sudo ip addr add 172.16.0.1/24 dev tap0"
echo "   sudo ip link set tap0 up"
echo ""
echo "2. Start Firecracker:"
echo "   firecracker --api-sock /tmp/firecracker.socket --config-file vm-config.json"
echo ""
echo "3. The VM will boot and you can access it via the console"
echo "   SSH will be available once the VM boots (may take a moment)"
echo "=== Done! ==="
echo "You can now run the bootable container in Firecracker."

