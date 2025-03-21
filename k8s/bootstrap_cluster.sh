#!/bin/bash

# Launch microVMs
./launch_node_ip.sh control-plane
./launch_node_ip.sh worker
# ./launch_microvm.sh worker-2

# Wait for VMs to boot and become reachable
sleep 30

# # Install Kubernetes components on each VM
# for vm in control-plane worker; do
#     ssh root@$vm "bash -s" < ./k8s/install_k8s.sh
# done

# # Initialize control plane
# ssh root@control-plane "sudo kubeadm init --control-plane-endpoint '192.168.1.100:6443' --upload-certs --pod-network-cidr=10.244.0.0/16"

# # Join worker nodes
# JOIN_COMMAND=$(ssh root@control-plane "kubeadm token create --print-join-command")
# ssh root@worker "$JOIN_COMMAND"
# # ssh root@worker-2 "$JOIN_COMMAND"

# Define IP addresses
CONTROL_PLANE_IP="192.168.1.100"
WORKER_IP="192.168.1.101"

# Install Kubernetes components on each VM
ssh root@"${CONTROL_PLANE_IP}" "bash -s" < install_k8s.sh
ssh root@"${WORKER_IP}" "bash -s" < install_k8s.sh

# Initialize control plane
ssh root@"${CONTROL_PLANE_IP}" "sudo kubeadm init --control-plane-endpoint '${CONTROL_PLANE_IP}:6443' --upload-certs --pod-network-cidr=10.244.0.0/16"

# Join worker nodes
JOIN_COMMAND=$(ssh root@"${CONTROL_PLANE_IP}" "kubeadm token create --print-join-command")
ssh root@"${WORKER_IP}" "$JOIN_COMMAND"