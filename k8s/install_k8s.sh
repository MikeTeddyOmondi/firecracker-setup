#!/bin/bash

# Update the package list
sudo apt-get update

# Install required packages
sudo apt-get install -y apt-transport-https ca-certificates curl

# Add the Kubernetes APT repository
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
cat <<EOF | sudo tee /etc/apt/sources.list.d/kubernetes.list
deb https://apt.kubernetes.io/ kubernetes-xenial main
EOF

# Update the package list again
sudo apt-get update

# Install Kubernetes components
sudo apt-get install -y kubelet kubeadm kubectl

# Mark the packages to hold their versions
sudo apt-mark hold kubelet kubeadm kubectl

# Disable swap (Kubernetes requirement)
sudo swapoff -a
# To make this change permanent, you can comment out the swap line in /etc/fstab
sudo sed -i '/ swap / s/^/#/' /etc/fstab

# Enable and start kubelet
sudo systemctl enable kubelet
sudo systemctl start kubelet

# Print the installed versions
echo "Kubernetes components installed:"
kubeadm version
kubectl version --client
