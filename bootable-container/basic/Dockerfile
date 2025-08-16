FROM ubuntu:22.04

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install systemd and basic packages
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
    && rm -rf /var/lib/apt/lists/*

# Install Docker CLI and daemon
RUN curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null \
    && apt-get update \
    && apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin \
    && rm -rf /var/lib/apt/lists/*

# Configure SSH
RUN mkdir /var/run/sshd \
    && echo 'root:password' | chpasswd \
    && sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config \
    && sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config

# Create a non-root user with sudo privileges
RUN useradd -m -s /bin/bash -G sudo,docker admin \
    && echo 'admin:password' | chpasswd

# Enable systemd services
RUN systemctl enable ssh \
    && systemctl enable docker

# Remove some systemd services that don't work well in containers
RUN systemctl mask \
    dev-hugepages.mount \
    sys-fs-fuse-connections.mount \
    sys-kernel-config.mount \
    display-manager.service \
    getty@.service \
    systemd-logind.service \
    systemd-remount-fs.service \
    getty.target \
    graphical.target

# Create systemd drop-in directory and configuration
RUN mkdir -p /etc/systemd/system/docker.service.d

# Custom systemd service for container initialization
COPY <<EOF /etc/systemd/system/container-init.service
[Unit]
Description=Container initialization service
After=multi-user.target

[Service]
Type=oneshot
ExecStart=/bin/bash -c 'echo "Container started successfully at $(date)" >> /var/log/container-init.log'
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF

RUN systemctl enable container-init.service

# Set up environment
ENV container=docker

# Expose SSH port
EXPOSE 22

# Use systemd as init system
CMD ["/lib/systemd/systemd"]
