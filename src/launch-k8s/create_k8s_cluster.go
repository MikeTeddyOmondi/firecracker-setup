package main

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"golang.org/x/crypto/ssh"
)

func executeSSHCommand(user, host, privateKeyPath, command string) (string, error) {
	key, err := ssh.ParsePrivateKey([]byte(privateKeyPath))
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %v", err)
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{ssh.PublicKeys(key)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", host+":22", config)
	if err != nil {
		return "", fmt.Errorf("failed to dial: %v", err)
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	var output bytes.Buffer
	session.Stdout = &output
	if err := session.Run(command); err != nil {
		return "", fmt.Errorf("failed to run command: %v", err)
	}

	return output.String(), nil
}

func main() {
	ctx := context.Background()

	vmID := "vmID"
	socketPath := "./microvm-k8s-cluster.sock" 
	kernelPath := "./vmlinux-5.10.225"
	rootFSPath := "./rootfs"

	// Firecracker microVM launch 
	launchK8sNode(ctx, vmID, socketPath, kernelPath, rootFSPath)

	// Example VM configuration
	user := "root"
	privateKeyPath := "/path/to/private/key"
	controlPlaneIP := "192.168.1.101"
	workerIPs := []string{"192.168.1.102", "192.168.1.103"}

	// Initialize control plane
	if err := initControlPlane(user, controlPlaneIP, privateKeyPath); err != nil {
		log.Fatalf("Failed to initialize control plane: %v", err)
	}

	// Retrieve join command
	joinCommand, err := getJoinCommand(user, controlPlaneIP, privateKeyPath)
	if err != nil {
		log.Fatalf("Failed to get join command: %v", err)
	}

	// Join worker nodes
	for _, workerIP := range workerIPs {
		if err := joinNode(user, workerIP, privateKeyPath, joinCommand); err != nil {
			log.Printf("Failed to join worker node %s: %v", workerIP, err)
		}
	}

	log.Println("Kubernetes cluster initialized successfully!")
}

func joinNode(user, host, privateKeyPath, joinCommand string) error {
	output, err := executeSSHCommand(user, host, privateKeyPath, joinCommand)
	if err != nil {
		return fmt.Errorf("failed to join node: %v", err)
	}

	log.Printf("Node Join Output: %s", output)
	return nil
}

func getJoinCommand(user, host, privateKeyPath string) (string, error) {
	command := "kubeadm token create --print-join-command"
	output, err := executeSSHCommand(user, host, privateKeyPath, command)
	if err != nil {
		return "", fmt.Errorf("failed to get join command: %v", err)
	}

	return output, nil
}

func initControlPlane(user, host, privateKeyPath string) error {
	command := `
	sudo kubeadm init --control-plane-endpoint "192.168.1.100:6443" \
					--upload-certs --pod-network-cidr=10.244.0.0/16
	`
	output, err := executeSSHCommand(user, host, privateKeyPath, command)
	if err != nil {
		return fmt.Errorf("failed to initialize control plane: %v", err)
	}

	log.Printf("Control Plane Initialization Output: %s", output)
	return nil
}

func launchK8sNode(ctx context.Context, vmID string, socketPath string, kernelPath string, rootFSPath string) error {
    // Firecracker machine configuration
    cfg := firecracker.Config{
        SocketPath: socketPath,
        MachineCfg: firecracker.MachineConfiguration{
            VcpuCount:  4,
            MemSizeMib: 4096,
        },
        Drives: []firecracker.Drive{
            {
                DriveID:      firecracker.String("rootfs"),
                PathOnHost:   firecracker.String(rootFSPath),
                IsRootDevice: true,
                IsReadOnly:   false,
            },
        },
        KernelImagePath: kernelPath,
    }

    // Start Firecracker VM
    cmd := firecracker.NewCommandBuilder().WithSocketPath(socketPath).Build(ctx)
    machine, err := firecracker.NewMachine(ctx, cfg, firecracker.WithProcessRunner(cmd))
    if err != nil {
        return err
    }

    return machine.Start(ctx)
}
