package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
)

type ClusterConfig struct {
	Name          string
	NodeCount     int
	MemSizeMB     int64
	VCPUCount     int64
	RootDrive     string  // Path to root filesystem image
	NetworkConfig Network // Custom network configuration
	Persistent    bool    // Whether storage should persist after shutdown
}

type Network struct {
	SubnetCIDR string
	Gateway    string
}

type Node struct {
	ID       string
	Role     string // master or worker
	IP       string
	Machine  *firecracker.Machine
	RootPath string
}

type Cluster struct {
	Config      ClusterConfig
	Nodes       []*Node
	ctx         context.Context
	cancelFunc  context.CancelFunc
	joinCommand string
}

func NewCluster(config ClusterConfig) *Cluster {
	ctx, cancel := context.WithCancel(context.Background())
	return &Cluster{
		Config:     config,
		ctx:        ctx,
		cancelFunc: cancel,
	}
}

func (c *Cluster) Provision() error {
	// Create base working directory for the cluster
	baseDir := filepath.Join("/var/lib/firecracker", c.Config.Name)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create cluster directory: %v", err)
	}

	// Initialize nodes
	masterNode := &Node{
		ID:       fmt.Sprintf("%s-master", c.Config.Name),
		Role:     "master",
		IP:       c.Config.NetworkConfig.getNextIP("10"),
		RootPath: filepath.Join(baseDir, "master"),
	}

	workers := make([]*Node, c.Config.NodeCount-1)
	for i := range workers {
		workers[i] = &Node{
			ID:       fmt.Sprintf("%s-worker-%d", c.Config.Name, i),
			Role:     "worker",
			IP:       c.Config.NetworkConfig.getNextIP(fmt.Sprintf("%d", 20+i)),
			RootPath: filepath.Join(baseDir, fmt.Sprintf("worker-%d", i)),
		}
	}

	c.Nodes = append([]*Node{masterNode}, workers...)

	// Provision nodes in parallel
	var wg sync.WaitGroup
	errCh := make(chan error, len(c.Nodes))

	for _, node := range c.Nodes {
		wg.Add(1)
		go func(n *Node) {
			defer wg.Done()
			if err := c.provisionNode(n); err != nil {
				errCh <- fmt.Errorf("failed to provision node %s: %v", n.ID, err)
			}
		}(node)
	}

	wg.Wait()
	close(errCh)

	// Check for any provisioning errors
	for err := range errCh {
		if err != nil {
			c.Cleanup()
			return err
		}
	}

	// Configure Kubernetes
	if err := c.configureKubernetes(); err != nil {
		c.Cleanup()
		return fmt.Errorf("failed to configure kubernetes: %v", err)
	}

	return nil
}

func (c *Cluster) provisionNode(node *Node) error {
	// Create node directory
	if err := os.MkdirAll(node.RootPath, 0755); err != nil {
		return err
	}

	// Copy root filesystem
	rootDrive := filepath.Join(node.RootPath, "root.img")
	if err := copyFile(c.Config.RootDrive, rootDrive); err != nil {
		return err
	}

	var vcpuCount int64 = 1
	var memSizeMib int64 = 512
	smt := false
	
	vmID := "vm"
	// staticIP := "192.168.1.102"
	driveID := "rootfs"
	isRootDevice := true
	isReadOnly := false
	pathOnHost := "./setup/ubuntu-24.04.ext4" // "./setup-microvm/ubuntu-24.04.ext4" // "./setup-microvm/root-drive-with-ssh.img"
	socketPath := "/tmp/firecracker-vm.sock"
	kernelPath := "./setup/vmlinux-5.10.225"

	ifaceID := "eth0"
	// tapName := "tap-" + vmID
	macAddress := "AA:FC:00:00:00:0" + string(vmID[len(vmID)-1])

	// Configure network
	// networkInterfaces := []firecracker.NetworkInterface{{
	// 	StaticConfiguration: &firecracker.StaticNetworkConfiguration{
	// 		HostDevName: fmt.Sprintf("tap%s", node.ID),
	// 		IPConfiguration: &firecracker.IPConfiguration{
	// 			IPAddr:  node.IP,
	// 			Gateway: c.Config.NetworkConfig.Gateway,
	// 			Mask:    "255.255.255.0",
	// 		},
	// 	},
	// }}	
	networkInterfaces := []firecracker.NetworkInterface{
		{
			StaticConfiguration: &firecracker.StaticNetworkConfiguration{
				HostDevName: fmt.Sprintf("tap%s", node.ID), // tapName,
				MacAddress:  macAddress,
				IPConfiguration: &firecracker.IPConfiguration{
					IfName: ifaceID,
					IPAddr: net.IPNet{
						IP: net.ParseIP(node.IP), // net.IP{192, 168, 1, 100},
					},
				},
			},
		},
	}

	// Create machine configuration
	// config := firecracker.Config{
	// 	SocketPath:      filepath.Join(node.RootPath, "firecracker.sock"),
	// 	KernelImagePath: "/var/lib/firecracker/vmlinux", // Path to kernel image
	// 	Drives: []models.Drive{{
	// 		DriveID:      "1",
	// 		PathOnHost:   rootDrive,
	// 		IsRootDevice: true,
	// 		IsReadOnly:   false,
	// 	}},
	// 	NetworkInterfaces: networkInterfaces,
	// 	MachineCfg: models.MachineConfiguration{
	// 		VcpuCount:  c.Config.VCPUCount,
	// 		MemSizeMib: c.Config.MemSizeMB,
	// 		Smt:  true,
	// 	},
	// 	LogPath: filepath.Join(node.RootPath, "firecracker.log"),
	// }
	config := firecracker.Config{
		SocketPath: socketPath,
		MachineCfg: models.MachineConfiguration{
			VcpuCount:  &vcpuCount,
			MemSizeMib: &memSizeMib,
			Smt:        &smt,
		},
		Drives: []models.Drive{
			{
				DriveID:      &driveID,
				IsRootDevice: &isRootDevice,
				IsReadOnly:   &isReadOnly,
				PathOnHost:   &pathOnHost,
			},
		},
		KernelImagePath: kernelPath,
		NetworkInterfaces: networkInterfaces,
	}

	// Create and start the machine
	m, err := firecracker.NewMachine(c.ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create machine: %v", err)
	}

	if err := m.Start(c.ctx); err != nil {
		return fmt.Errorf("failed to start machine: %v", err)
	}

	node.Machine = m
	return nil
}

func (c *Cluster) configureKubernetes() error {
	// Initialize master node
	master := c.Nodes[0]
	if err := c.initializeMaster(master); err != nil {
		return err
	}

	// Join worker nodes
	for _, worker := range c.Nodes[1:] {
		if err := c.joinWorker(worker); err != nil {
			return err
		}
	}

	return nil
}

func (c *Cluster) Cleanup() {
	c.cancelFunc()
	for _, node := range c.Nodes {
		if node.Machine != nil {
			if err := node.Machine.Shutdown(c.ctx); err != nil {
				log.Printf("Error shutting down node %s: %v", node.ID, err)
			}
		}

		// Clean up node directory if not persistent
		if !c.Config.Persistent {
			if err := os.RemoveAll(node.RootPath); err != nil {
				log.Printf("Error cleaning up node directory %s: %v", node.RootPath, err)
			}
		}
	}
}

// Helper functions would be implemented here:
// - copyFile: Copy root filesystem image
// - initializeMaster: Initialize Kubernetes master node
// - joinWorker: Join worker nodes to the cluster
// - Network.getNextIP: Generate next available IP in subnet

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	// Create destination file
	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer destFile.Close()

	// Copy the file
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}

	// Sync to ensure write is complete
	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %v", err)
	}

	return nil
}

// initializeMaster initializes the Kubernetes master node
func (c *Cluster) initializeMaster(master *Node) error {
	// Wait for machine to be ready
	if err := c.waitForSSH(master); err != nil {
		return fmt.Errorf("master node SSH not ready: %v", err)
	}

	// Initialize kubeadm
	initCommand := fmt.Sprintf(`kubeadm init \
		--apiserver-advertise-address=%s \
		--pod-network-cidr=%s \
		--node-name=%s`,
		master.IP,
		c.Config.NetworkConfig.SubnetCIDR,
		master.ID,
	)

	if err := c.executeCommand(master, initCommand); err != nil {
		return fmt.Errorf("failed to initialize master: %v", err)
	}

	// Install CNI network plugin (using Calico as example)
	calicoCommand := "kubectl apply -f https://docs.projectcalico.org/manifests/calico.yaml"
	if err := c.executeCommand(master, calicoCommand); err != nil {
		return fmt.Errorf("failed to install Calico: %v", err)
	}

	// Get join command for workers
	joinCommand, err := c.getJoinCommand(master)
	if err != nil {
		return fmt.Errorf("failed to get join command: %v", err)
	}
	c.joinCommand = joinCommand

	return nil
}

// joinWorker joins a worker node to the cluster
func (c *Cluster) joinWorker(worker *Node) error {
	// Wait for machine to be ready
	if err := c.waitForSSH(worker); err != nil {
		return fmt.Errorf("worker node SSH not ready: %v", err)
	}

	// Join the cluster using the stored join command
	if err := c.executeCommand(worker, c.joinCommand); err != nil {
		return fmt.Errorf("failed to join worker to cluster: %v", err)
	}

	return nil
}

// getNextIP generates the next available IP in the subnet
func (n *Network) getNextIP(host string) string {
	// Parse the subnet CIDR
	_, ipNet, err := net.ParseCIDR(n.SubnetCIDR)
	if err != nil {
		log.Printf("Error parsing CIDR: %v, using fallback IP", err)
		return fmt.Sprintf("192.168.%s.100", host)
	}

	// Get the first three octets of the subnet
	firstThreeOctets := strings.Join(strings.Split(ipNet.IP.String(), ".")[:3], ".")

	// Return IP with provided host portion
	return fmt.Sprintf("%s.%s", firstThreeOctets, host)
}

// waitForSSH waits for SSH to become available on the node
func (c *Cluster) waitForSSH(node *Node) error {
	timeout := time.After(2 * time.Minute)
	tick := time.Tick(2 * time.Second)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for SSH on node %s", node.ID)
		case <-tick:
			if err := c.checkSSH(node); err == nil {
				return nil
			}
		}
	}
}

// checkSSH attempts to establish SSH connection
func (c *Cluster) checkSSH(node *Node) error {
	cmd := exec.Command("ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "ConnectTimeout=5",
		fmt.Sprintf("root@%s", node.IP),
		"echo", "hello")

	return cmd.Run()
}

// executeCommand executes a command on the node via SSH
func (c *Cluster) executeCommand(node *Node, command string) error {
	cmd := exec.Command("ssh",
		"-o", "StrictHostKeyChecking=no",
		fmt.Sprintf("root@%s", node.IP),
		command)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %v, output: %s", err, string(output))
	}

	return nil
}

// getJoinCommand retrieves the kubeadm join command from the master
func (c *Cluster) getJoinCommand(master *Node) (string, error) {
	cmd := exec.Command("ssh",
		"-o", "StrictHostKeyChecking=no",
		fmt.Sprintf("root@%s", master.IP),
		"kubeadm token create --print-join-command")

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get join command: %v", err)
	}

	return strings.TrimSpace(string(output)), nil
}
