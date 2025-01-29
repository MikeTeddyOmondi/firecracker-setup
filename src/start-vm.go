package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"syscall"
)

func main() {
	// Firecracker socket path
	socketPath := "/tmp/firecracker.socket"

	// Define configurations for the microVM
	kernelPath := "./setup/vmlinux-5.10.225"    // Path to your kernel image
	drivePath := "./setup/ubuntu-24.04.ext4" // Path to your root filesystem image
	machineConfig := map[string]interface{}{
		"vcpu_count":   1,
		"mem_size_mib": 512,
	}
	bootSource := map[string]interface{}{
		"kernel_image_path": kernelPath,
		"boot_args":         "console=ttyS0 reboot=k panic=1 pci=off",
	}
	driveConfig := map[string]interface{}{
		"drive_id":       "rootfs",
		"path_on_host":   drivePath,
		"is_root_device": true,
		"is_read_only":   false,
	}

	// Helper function to send PUT requests to Firecracker API
	sendPutRequest := func(endpoint string, data interface{}) error {
		body, err := json.Marshal(data)
		if err != nil {
			return err
		}

		req, err := http.NewRequest("PUT", "http://unix/"+endpoint, bytes.NewBuffer(body))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return (&net.Dialer{
						Control: func(_, addr string, c syscall.RawConn) error {
							return c.Control(func(fd uintptr) {
								// Use syscall.Connect instead of addrUnix.Connect
								syscall.Connect(int(fd), &syscall.SockaddrUnix{Name: socketPath})
							})
						},
					}).DialContext(ctx, "unix", socketPath)
				},
			},
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			return fmt.Errorf("status: %s, body: %s", resp.Status, bodyBytes)
		}
		return nil
	}

	// Set machine configuration
	if err := sendPutRequest("/machine-config", machineConfig); err != nil {
		fmt.Println("Failed to set machine config:", err)
		os.Exit(1)
	}

	// Set boot source
	if err := sendPutRequest("/boot-source", bootSource); err != nil {
		fmt.Println("Failed to set boot source:", err)
		os.Exit(1)
	}

	// Add root drive
	if err := sendPutRequest("/drives/rootfs", driveConfig); err != nil {
		fmt.Println("Failed to add drive:", err)
		os.Exit(1)
	}

	// Start the microVM
	if err := sendPutRequest("/actions", map[string]string{"action_type": "InstanceStart"}); err != nil {
		fmt.Println("Failed to start instance:", err)
		os.Exit(1)
	}

	fmt.Println("MicroVM started successfully.")
}
