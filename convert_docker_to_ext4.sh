#!/bin/bash

set -e  # Exit immediately if a command exits with a non-zero status

# Variables
IMAGE_NAME="$1"  # Docker image name passed as the first argument
UPLOAD_DIR="./uploads"
EXT4_IMAGE="./output.img"
TARBALL_PATH="${UPLOAD_DIR}/${IMAGE_NAME}.tar"
MOUNT_POINT=$(mktemp -d)

# Create upload directory if it doesn't exist
mkdir -p "$UPLOAD_DIR"

# Step 1: Export the Docker image to a tarball
echo "Exporting Docker image '$IMAGE_NAME' to tarball..."
docker save -o "$TARBALL_PATH" "$IMAGE_NAME"

# Step 2: Create an ext4 filesystem image
echo "Creating ext4 filesystem image..."
dd if=/dev/zero of="$EXT4_IMAGE" bs=1M count=64  # 64MB image
mkfs.ext4 "$EXT4_IMAGE"

# Step 3: Mount the ext4 image
echo "Mounting ext4 image..."
sudo mount "$EXT4_IMAGE" "$MOUNT_POINT"

# Step 4: Extract the tarball contents to the mounted ext4 filesystem
echo "Extracting tarball contents to ext4 filesystem..."
tar -xf "$TARBALL_PATH" -C "$MOUNT_POINT"

# Step 5: Unmount the ext4 image
echo "Unmounting ext4 image..."
sudo umount "$MOUNT_POINT"

# Cleanup
rmdir "$MOUNT_POINT"
echo "Conversion successful! Ext4 image created at: $EXT4_IMAGE"

