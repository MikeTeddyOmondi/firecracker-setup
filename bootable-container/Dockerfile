FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

# Install essential packages for Firecracker VM
RUN apt-get update && apt-get install -y \
    systemd \
    systemd-sysv \
    openssh-server \
    sudo \
    curl \
    wget \
    vim \
    htop \
    net-tools \
    iputils-ping \
    ca-certificates \
    gnupg \
    lsb-release \
    iptables \
    kmod \
    && rm -rf /var/lib/apt/lists/*

# Install Docker CLI and daemon
RUN curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null \
    && apt-get update \
    && apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin \
    && rm -rf /var/lib/apt/lists/*

# Configure SSH for Firecracker (serial console friendly)
RUN mkdir -p /var/run/sshd \
    && echo 'root:firecracker' | chpasswd \
    && sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config \
    && sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config \
    && sed -i 's/UsePAM yes/UsePAM no/' /etc/ssh/sshd_config

# Create admin user
RUN useradd -m -s /bin/bash -G sudo,docker admin \
    && echo 'admin:firecracker' | chpasswd \
    && echo 'admin ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers

# Configure Docker daemon for Firecracker
RUN mkdir -p /etc/docker
COPY <<EOF /etc/docker/daemon.json
{
  "storage-driver": "overlay2",
  "hosts": ["unix:///var/run/docker.sock", "tcp://0.0.0.0:2375"],
  "data-root": "/var/lib/docker",
  "log-driver": "journald"
}
EOF

# Enable services
RUN systemctl enable ssh \
    && systemctl enable docker \
    && systemctl enable systemd-networkd \
    && systemctl enable systemd-resolved

# Disable problematic services for Firecracker
RUN systemctl mask \
    dev-hugepages.mount \
    sys-fs-fuse-connections.mount \
    sys-kernel-config.mount \
    display-manager.service \
    systemd-logind.service \
    systemd-remount-fs.service \
    getty.target \
    graphical.target \
    systemd-udev-trigger.service \
    systemd-udevd.service \
    systemd-udev-settle.service

# Configure systemd for Firecracker environment
RUN mkdir -p /etc/systemd/system.conf.d
COPY <<EOF /etc/systemd/system.conf.d/firecracker.conf
[Manager]
DefaultTimeoutStartSec=30s
DefaultTimeoutStopSec=10s
DefaultRestartSec=1s
EOF

# Create Firecracker-specific initialization service
COPY <<EOF /etc/systemd/system/firecracker-init.service
[Unit]
Description=Firecracker VM initialization
DefaultDependencies=no
After=systemd-remount-fs.service
Before=systemd-sysusers.service systemd-tmpfiles-setup.service
Wants=systemd-remount-fs.service
Conflicts=shutdown.target

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/bin/bash -c '
    # Set up networking
    echo "127.0.0.1 localhost" > /etc/hosts
    echo "$(hostname -I | cut -d' ' -f1) $(hostname)" >> /etc/hosts
    
    # Initialize Docker data directory
    mkdir -p /var/lib/docker
    
    # Log successful start
    echo "Firecracker VM initialized at $(date)" | systemd-cat -t firecracker-init
'

[Install]
WantedBy=sysinit.target
EOF

RUN systemctl enable firecracker-init.service

# Set container environment
ENV container=docker

# Create welcome message
COPY <<EOF /etc/motd

=== Firecracker Docker VM ===
Hostname: \$(hostname)
IP Address: \$(hostname -I)

Services:
- SSH: Port 22 (root:firecracker, admin:firecracker)
- Docker: API on port 2375, socket at /var/run/docker.sock

Usage:
- docker version
- docker run hello-world
- systemctl status

EOF

EXPOSE 22 2375
