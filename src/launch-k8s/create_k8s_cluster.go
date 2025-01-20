// package main

// import (
// 	"bytes"
// 	"context"
// 	"fmt"
// 	"log"
// 	"os"

// 	"github.com/firecracker-microvm/firecracker-go-sdk"
// 	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
// 	"golang.org/x/crypto/ssh"
// )

// func executeSSHCommand(user, host, privateKeyPath, command string) (string, error) {
// 	privateKeyBytes, err := os.ReadFile(privateKeyPath)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to read private key file: %v", err)
// 	}

// 	key, err := ssh.ParsePrivateKey(privateKeyBytes)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to parse private key: %v", err)
// 	}

// 	config := &ssh.ClientConfig{
// 		User:            user,
// 		Auth:            []ssh.AuthMethod{ssh.PublicKeys(key)},
// 		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
// 	}

// 	conn, err := ssh.Dial("tcp", host+":22", config)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to dial: %v", err)
// 	}
// 	defer conn.Close()

// 	session, err := conn.NewSession()
// 	if err != nil {
// 		return "", fmt.Errorf("failed to create session: %v", err)
// 	}
// 	defer session.Close()

// 	var output bytes.Buffer
// 	session.Stdout = &output
// 	if err := session.Run(command); err != nil {
// 		return "", fmt.Errorf("failed to run command: %v", err)
// 	}

// 	return output.String(), nil
// }

// func main() {
// 	ctx := context.Background()

// 	vmID := "vmID"
// 	socketPath := "./microvm-k8s-cluster.sock"
// 	kernelPath := "./setup/vmlinux-5.10.225"
// 	rootFSPath := "./setup/k8s-img-rootfs.ext4"

// 	// Firecracker microVM launch
// 	launchK8sNode(ctx, vmID, socketPath, kernelPath, rootFSPath)

// 	// Example VM configuration
// 	user := "root"
// 	privateKeyPath := "./setup/k8s-img.id_rsa"
// 	controlPlaneIP := "192.168.1.101"
// 	workerIPs := []string{"192.168.1.102", "192.168.1.103"}

// 	// Initialize control plane
// 	if err := initControlPlane(user, controlPlaneIP, privateKeyPath); err != nil {
// 		log.Fatalf("Failed to initialize control plane: %v", err)
// 	}

// 	// Retrieve join command
// 	joinCommand, err := getJoinCommand(user, controlPlaneIP, privateKeyPath)
// 	if err != nil {
// 		log.Fatalf("Failed to get join command: %v", err)
// 	}

// 	// Join worker nodes
// 	for _, workerIP := range workerIPs {
// 		if err := joinNode(user, workerIP, privateKeyPath, joinCommand); err != nil {
// 			log.Printf("Failed to join worker node %s: %v", workerIP, err)
// 		}
// 	}

// 	log.Println("Kubernetes cluster initialized successfully!")
// }

// func joinNode(user, host, privateKeyPath, joinCommand string) error {
// 	output, err := executeSSHCommand(user, host, privateKeyPath, joinCommand)
// 	if err != nil {
// 		return fmt.Errorf("failed to join node: %v", err)
// 	}

// 	log.Printf("Node Join Output: %s", output)
// 	return nil
// }

// func getJoinCommand(user, host, privateKeyPath string) (string, error) {
// 	command := "kubeadm token create --print-join-command"
// 	output, err := executeSSHCommand(user, host, privateKeyPath, command)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to get join command: %v", err)
// 	}

// 	return output, nil
// }

// func initControlPlane(user, host, privateKeyPath string) error {
// 	command := `
// 	sudo kubeadm init --control-plane-endpoint "192.168.1.100:6443" \
// 					--upload-certs --pod-network-cidr=10.244.0.0/16
// 	`
// 	output, err := executeSSHCommand(user, host, privateKeyPath, command)
// 	if err != nil {
// 		return fmt.Errorf("failed to initialize control plane: %v", err)
// 	}

// 	log.Printf("Control Plane Initialization Output: %s", output)
// 	return nil
// }

// func launchK8sNode(ctx context.Context, vmID string, socketPath string, kernelPath string, rootFSPath string) error {
// 	// Firecracker machine configuration
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

// 	// Start Firecracker VM
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
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"golang.org/x/crypto/ssh"
)

func executeSSHCommand(user, host, privateKeyPath, command string) (string, error) {
	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read private key file: %v", err)
	}

	key, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %v", err)
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(key)},
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

	// Configuration
	user := "root"
	kernelPath := "./setup/vmlinux-5.10.225"
	rootFSPath := "./setup/k8s-img-rootfs.ext4"
	privateKeyPath := "./setup/k8s-img.id_rsa"

	// Define IP addresses
	controlPlaneIP := "192.168.1.100"
	workerIPs := []string{"192.168.1.101", "192.168.1.102"}
	vmIDs := []string{"control-plane", "worker-1", "worker-2"}
	socketPaths := []string{"./control-plane.sock", "./worker-1.sock", "./worker-2.sock"}

	// Launch microVMs
	for i, vmID := range vmIDs {
		socketPath := socketPaths[i]
		var ip string
		if vmID == "control-plane" {
			ip = controlPlaneIP
		} else {
			ip = workerIPs[i-1] // worker-1 and worker-2
		}

		if err := launchK8sNode(ctx, vmID, socketPath, kernelPath, rootFSPath, ip); err != nil {
			log.Fatalf("Failed to launch %s: %v", vmID, err)
		}

		// Wait for the VM to boot up
		time.Sleep(5 * time.Second) // Adjust as necessary

		if vmID == "control-plane" {
			if err := initControlPlane(user, controlPlaneIP, privateKeyPath); err != nil {
				log.Fatalf("Failed to initialize control plane: %v", err)
			}
		}
	}

	// Retrieve join command from control plane
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

func launchK8sNode(ctx context.Context, vmID string, socketPath string, kernelPath string, rootFSPath string, staticIP string) error {
	// Firecracker machine configuration
	var vcpuCount int64 = 1
	var memSizeMib int64 = 512
	smt := false

	driveID := "rootfs"
	isRootDevice := true
	isReadOnly := false
	pathOnHost := rootFSPath

	ifaceID := "eth0"
	tapName := "tap-" + vmID
	macAddress := "AA:FC:00:00:00:0" + string(vmID[len(vmID)-1])

	// networkInterface := []models.NetworkInterface{
	// 	{
	// 		IfaceID:     &ifaceID,
	//		HostDevName: &tapName,
	// 		GuestMac:    macAddress, // Unique MAC for each VM
	// 	},
	// }

	println("StaticIP: ", staticIP)

	networkInterface := []firecracker.NetworkInterface{
		{
			StaticConfiguration: &firecracker.StaticNetworkConfiguration{
				HostDevName: tapName,
				MacAddress:  macAddress,
				IPConfiguration: &firecracker.IPConfiguration{
					IfName: ifaceID,
					IPAddr: net.IPNet{
						IP: net.ParseIP(staticIP), // net.IP{192, 168, 1, 100}, // use staticIp
					},
				},
			},
		},
	}

	cfg := firecracker.Config{
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
		KernelImagePath:   kernelPath,
		NetworkInterfaces: networkInterface,
	}

	// Create the Firecracker command
	cmdBuilderOpts := firecracker.VMCommandBuilder{}
	cmd := firecracker.VMCommandBuilder(cmdBuilderOpts).
		WithSocketPath(socketPath).
		Build(ctx)

	// Start Firecracker VM
	machine, err := firecracker.NewMachine(ctx, cfg, firecracker.WithProcessRunner(cmd))
	if err != nil {
		return err
	}

	// Start the VM
	if err := machine.Start(ctx); err != nil {
		return err
	}

	log.Printf("Firecracker VM %s started successfully with IP %s.", vmID, staticIP)

	// Assign the static IP address to the TAP interface
	if err := assignStaticIP(staticIP, "tap-"+vmID); err != nil {
		return fmt.Errorf("failed to assign static IP: %v", err)
	}

	return nil
}

func assignStaticIP(staticIP, tapName string) error {
	// Command to assign the static IP address to the TAP interface
	cmd := fmt.Sprintf("sudo ip addr add %s/24 dev %s && sudo ip link set dev %s up", staticIP, tapName, tapName)

	// Execute the command using SSH (you need to implement SSH connection logic here)
	// For example, you can use the executeSSHCommand function you defined earlier
	_, err := executeSSHCommand("root", staticIP, "./setup/k8s-img.id_rsa", cmd) // Adjust the IP as needed
	if err != nil {
		return fmt.Errorf("failed to execute command on VM: %v", err)
	}

	return nil
}
