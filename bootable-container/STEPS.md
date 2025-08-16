## Complete Workflow

To transform your Docker image into an ext4 filesystem that can run with Firecracker, you'll need to extract the container filesystem and create a bootable ext4 image with a Linux kernel. 

Here's how to use this setup :

1. **Prepare the environment:**
   ```bash
   # Install Firecracker
   curl -L -o firecracker https://github.com/firecracker-microvm/firecracker/releases/download/v1.4.1/firecracker-v1.4.1-x86_64.tgz
   tar -xzf firecracker-v1.4.1-x86_64.tgz
   sudo mv release-v1.4.1-x86_64/firecracker-v1.4.1-x86_64 /usr/local/bin/firecracker
   sudo chmod +x /usr/local/bin/firecracker
   ```

2. **Build and run:**
   ```bash
   # Make scripts executable
   chmod +x docker_to_ext4.sh microvm_runner.sh
   
   # Build the VM image
   ./microvm_runner.sh build
   
   # Set up networking (requires sudo)
   sudo ./microvm_runner.sh setup-net
   
   # Start the VM
   ./microvm_runner.sh start
   
   # Wait for boot, then SSH in
   ./microvm_runner.sh ssh
   ```

3. **Inside the VM:**
   ```bash
   # Check Docker
   docker version
   docker run hello-world
   
   # Check services
   systemctl status docker
   systemctl status ssh
   ```

## Key Benefits

- **Fast boot**: Firecracker VMs boot in milliseconds
- **Lightweight**: Much lower overhead than traditional VMs  
- **Isolation**: Better security isolation than containers
- **Docker support**: Full Docker daemon and CLI available
- **SSH access**: Standard remote access capabilities
- **Networking**: Proper network stack with TAP interfaces

## Important Notes

- The VM uses about 2GB of RAM and 2 CPU cores by default
- SSH password is set to "firecracker" for both root and admin users
- Docker API is exposed on port 2375 inside the VM
- The rootfs file can be reused and is persistent between runs
- Firecracker requires KVM support on the host system

This approach gives you the container's software stack running in a proper microVM with full kernel support, making it suitable for workloads that need both container capabilities and VM-like isolation.