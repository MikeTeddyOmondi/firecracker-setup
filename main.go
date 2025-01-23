// package main

// import (
// 	"context"
// 	// "io"
// 	"log"

// 	// "path/filepath"

// 	"github.com/firecracker-microvm/firecracker-go-sdk"
// 	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
// )

// func main() {
// 	ctx := context.Background()

// 	// Configuration for multiple VMs
// 	vms := []struct {
// 		VMID       string
// 		SocketPath string
// 		KernelPath string
// 		RootFSPath string
// 	}{
// 		{"vm1", "/tmp/firecracker-vm1.sock", "./setup/vmlinux-5.10.225", "./setup/ubuntu-24.04.ext4"},
// 		{"vm2", "/tmp/firecracker-vm2.sock", "./setup/vmlinux-5.10.225", "./setup/ubuntu-24.04.ext4"},
// 	}

// 	// Launch all VMs
// 	for _, vm := range vms {
// 		go func(vmConfig struct {
// 			VMID       string
// 			SocketPath string
// 			KernelPath string
// 			RootFSPath string
// 		}) {
// 			err := launchFirecrackerInstance(ctx, vmConfig.VMID, vmConfig.SocketPath, vmConfig.KernelPath, vmConfig.RootFSPath)
// 			if err != nil {
// 				log.Printf("Failed to start VM %s: %v", vmConfig.VMID, err)
// 			}
// 		}(vm)
// 	}

// 	// Keep the main thread alive
// 	select {}
// }

// func launchFirecrackerInstance(ctx context.Context, vmID string, socketPath string, kernelPath string, rootFSPath string) error {
// 	var vcpuCount int64 = 1
// 	var memSizeMib int64 = 512
// 	smt := false

// 	driveID := "rootfs"
// 	isRootDevice := true
// 	isReadOnly := false
// 	pathOnHost := rootFSPath // "./setup-microvm/ubuntu-24.04.ext4" // "./setup-microvm/root-drive-with-ssh.img"

// 	cfg := firecracker.Config{
// 		SocketPath: socketPath,
// 		MachineCfg: models.MachineConfiguration{
// 			VcpuCount:  &vcpuCount,
// 			MemSizeMib: &memSizeMib,
// 			Smt:        &smt,
// 		},
// 		Drives: []models.Drive{
// 			{
// 				DriveID:      &driveID,
// 				IsRootDevice: &isRootDevice,
// 				IsReadOnly:   &isReadOnly,
// 				PathOnHost:   &pathOnHost,
// 			},
// 		},
// 		KernelImagePath: kernelPath,
// 	}

// 	// cmdBuilderOpts := firecracker.VMCommandBuilder{
// 	// 	bin:        "./setup/vmlinux-5.10.225",
// 	// 	args:       []string{},
// 	// 	socketPath: "",
// 	// 	stdin:      io.Reader,
// 	// 	stdout:     io.Writer,
// 	// 	stderr:     io.Writer,
// 	// }

// 	// Create the Firecracker command
// 	cmdBuilderOpts := firecracker.VMCommandBuilder{}
// 	cmd := firecracker.VMCommandBuilder(cmdBuilderOpts).
// 		WithSocketPath(socketPath).
// 		Build(ctx)

// 	// Start the Firecracker process
// 	machine, err := firecracker.NewMachine(ctx, cfg, firecracker.WithProcessRunner(cmd))
// 	if err != nil {
// 		return err
// 	}

// 	// Start the VM
// 	if err := machine.Start(ctx); err != nil {
// 		return err
// 	}

// 	log.Printf("Firecracker VM %s started successfully.", vmID)
// 	return nil
// }

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"firecracker-k8s/cluster"
)

func main() {
	// Parse command line flags
	name := flag.String("name", "", "Cluster name")
	nodes := flag.Int("nodes", 3, "Number of nodes")
	memory := flag.Int64("memory", 1024, "Memory per node in MB")
	vcpu := flag.Int64("vcpu", 1, "VCPUs per node")
	rootfs := flag.String("rootfs", "", "Path to root filesystem image")
	persistent := flag.Bool("persistent", false, "Enable persistent storage")
	subnet := flag.String("subnet", "172.16.0.0/24", "Subnet CIDR")
	gateway := flag.String("gateway", "172.16.0.1", "Gateway IP")
	flag.Parse()

	// Validate required flags
	if *name == "" || *rootfs == "" {
		log.Fatal("Cluster name and root filesystem path are required")
	}

	// Create cluster configuration
	config := cluster.ClusterConfig{
		Name:       *name,
		NodeCount:  *nodes,
		MemSizeMB:  *memory,
		VCPUCount:  *vcpu,
		RootDrive:  *rootfs,
		Persistent: *persistent,
		NetworkConfig: cluster.Network{
			SubnetCIDR: *subnet,
			Gateway:    *gateway,
		},
	}

	// Create new cluster instance
	c := cluster.NewCluster(config)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down cluster...")
		c.Cleanup()
		os.Exit(0)
	}()

	// Provision the cluster
	log.Printf("Provisioning cluster '%s' with %d nodes...", config.Name, config.NodeCount)
	if err := c.Provision(); err != nil {
		log.Fatalf("Failed to provision cluster: %v", err)
	}

	// Print cluster information
	printClusterInfo(c)

	// Keep the program running
	select {}
}

func printClusterInfo(c *cluster.Cluster) {
	fmt.Println("\nCluster Information:")
	fmt.Printf("Name: %s\n", c.Config.Name)
	fmt.Printf("Nodes: %d\n", len(c.Nodes))
	fmt.Println("\nNode Details:")
	
	for _, node := range c.Nodes {
		fmt.Printf("\nID: %s\n", node.ID)
		fmt.Printf("Role: %s\n", node.Role)
		fmt.Printf("IP: %s\n", node.IP)
		fmt.Printf("Socket: %s\n", node.Machine.Cfg.SocketPath)
	}

	fmt.Println("\nCluster is ready!")
	fmt.Println("Use 'kubectl' on the master node to manage the cluster")
}