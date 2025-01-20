package main

import (
	"context"
	// "io"
	"log"

	// "path/filepath"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
)

func main() {
	ctx := context.Background()

	// Configuration for multiple VMs
	vms := []struct {
		VMID       string
		SocketPath string
		KernelPath string
		RootFSPath string
	}{
		{"vm1", "/tmp/firecracker-vm1.sock", "./setup/vmlinux-5.10.225", "./setup/ubuntu-24.04.ext4"},
		{"vm2", "/tmp/firecracker-vm2.sock", "./setup/vmlinux-5.10.225", "./setup/ubuntu-24.04.ext4"},
	}

	// Launch all VMs
	for _, vm := range vms {
		go func(vmConfig struct {
			VMID       string
			SocketPath string
			KernelPath string
			RootFSPath string
		}) {
			err := launchFirecrackerInstance(ctx, vmConfig.VMID, vmConfig.SocketPath, vmConfig.KernelPath, vmConfig.RootFSPath)
			if err != nil {
				log.Printf("Failed to start VM %s: %v", vmConfig.VMID, err)
			}
		}(vm)
	}

	// Keep the main thread alive
	select {}
}

func launchFirecrackerInstance(ctx context.Context, vmID string, socketPath string, kernelPath string, rootFSPath string) error {
	var vcpuCount int64 = 1
	var memSizeMib int64 = 512
	smt := false

	driveID := "rootfs"
	isRootDevice := true
	isReadOnly := false
	pathOnHost := rootFSPath // "./setup-microvm/ubuntu-24.04.ext4" // "./setup-microvm/root-drive-with-ssh.img"

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
		KernelImagePath: kernelPath,
	}

	// cmdBuilderOpts := firecracker.VMCommandBuilder{
	// 	bin:        "./setup/vmlinux-5.10.225",
	// 	args:       []string{},
	// 	socketPath: "",
	// 	stdin:      io.Reader,
	// 	stdout:     io.Writer,
	// 	stderr:     io.Writer,
	// }

	// Create the Firecracker command
	cmdBuilderOpts := firecracker.VMCommandBuilder{}
	cmd := firecracker.VMCommandBuilder(cmdBuilderOpts).
		WithSocketPath(socketPath).
		Build(ctx)

	// Start the Firecracker process
	machine, err := firecracker.NewMachine(ctx, cfg, firecracker.WithProcessRunner(cmd))
	if err != nil {
		return err
	}

	// Start the VM
	if err := machine.Start(ctx); err != nil {
		return err
	}

	log.Printf("Firecracker VM %s started successfully.", vmID)
	return nil
}
