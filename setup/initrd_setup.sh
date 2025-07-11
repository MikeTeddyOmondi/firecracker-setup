# Recreate initrd with better init script
mkdir -p initrd/{bin,sbin,lib,lib64,dev,proc,sys,tmp,usr/bin,usr/sbin}

# Copy essential binaries
cp /bin/sh initrd/bin/
cp /bin/echo initrd/bin/
cp /bin/ls initrd/bin/
cp /bin/cat initrd/bin/
cp /bin/mount initrd/bin/

# Copy libraries
ldd /bin/sh | grep -o '/lib[^ ]*' | xargs -I {} cp {} initrd/lib/ 2>/dev/null || true
ldd /bin/echo | grep -o '/lib[^ ]*' | xargs -I {} cp {} initrd/lib/ 2>/dev/null || true

# Create init script that stays running
cat > initrd/init <<'EOF'
#!/bin/sh

echo "=== Initrd Boot Started ==="

# Mount essential filesystems
mount -t proc proc /proc
mount -t sysfs sysfs /sys
mount -t devtmpfs devtmpfs /dev

echo "=== Filesystems mounted ==="
echo "=== Available devices ==="
ls /dev/

echo "=== Starting interactive shell ==="
echo "Type 'exit' to shutdown the VM"

# Start interactive shell that keeps VM running
exec /bin/sh
EOF

chmod +x initrd/init

# Rebuild initrd
cd initrd && find . | cpio -o -H newc | gzip > ../initrd.img && cd ..

