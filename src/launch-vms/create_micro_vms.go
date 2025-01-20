package main

import (
	"context"
	"log"
	"path/filepath"

	"github.com/firecracker-microvm/firecracker-go-sdk"
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
		{"vm1", "/tmp/firecracker-vm1.sock", "./vmlinux", "./rootfs-vm1.ext4"},
		{"vm2", "/tmp/firecracker-vm2.sock", "./vmlinux", "./rootfs-vm2.ext4"},
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
	cfg := firecracker.Config{
		SocketPath: socketPath,
		MachineCfg: firecracker.MachineConfiguration{
			VCpuCount:   2,
			MemSizeMib:  512,
			HtEnabled:   true,
		},
		Drives: []firecracker.Drive{
			{
				DriveID:      firecracker.String("rootfs"),
				PathOnHost:   firecracker.String(rootFSPath),
				IsRootDevice: firecracker.Bool(true),
				IsReadOnly:   firecracker.Bool(false),
			},
		},
		KernelImagePath: kernelPath,
	}

	// Create the Firecracker command
	cmd := firecracker.NewCommandBuilder().
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

