package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

func TarballToExt4() {
	// Define paths
	tarGzPath := "k8s-img.tar.gz"
	rootfsDir := "rootfs"
	ext4File := "k8s-img-rootfs.ext4"
	mountPoint := "/mnt"

	// Step 1: Extract tarball
	err := extractTarGz(tarGzPath, rootfsDir)
	if err != nil {
		fmt.Printf("Error extracting tarball: %v\n", err)
		return
	}
	fmt.Println("Tarball extracted successfully.")

	// Step 2: Create and format ext4 filesystem
	err = createExt4File(ext4File, 512) // 512 MB
	if err != nil {
		fmt.Printf("Error creating ext4 file: %v\n", err)
		return
	}
	fmt.Println("Ext4 filesystem created successfully.")

	// Step 3: Mount the ext4 filesystem
	err = mountExt4(ext4File, mountPoint)
	if err != nil {
		fmt.Printf("Error mounting ext4 file: %v\n", err)
		return
	}
	defer unmountExt4(mountPoint)

	// Step 4: Copy extracted files to the mounted ext4
	err = copyDir(rootfsDir, mountPoint)
	if err != nil {
		fmt.Printf("Error copying files: %v\n", err)
		return
	}
	fmt.Println("Files copied successfully.")

	// Step 5: Unmount the ext4 filesystem
	err = unmountExt4(mountPoint)
	if err != nil {
		fmt.Printf("Error unmounting ext4 file: %v\n", err)
		return
	}
	fmt.Println("Ext4 filesystem unmounted successfully.")
}

// Extract tar.gz file to a directory
func extractTarGz(tarGzPath, outputDir string) error {
	file, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(outputDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}
	return nil
}

// Create and format an ext4 filesystem
func createExt4File(ext4Path string, sizeMB int) error {
	// Create the file with specified size
	err := exec.Command("dd", "if=/dev/zero", fmt.Sprintf("of=%s", ext4Path),
		"bs=1M", fmt.Sprintf("count=%d", sizeMB)).Run()
	if err != nil {
		return fmt.Errorf("failed to create ext4 file: %w", err)
	}

	// Format the file as ext4
	err = exec.Command("mkfs.ext4", ext4Path).Run()
	if err != nil {
		return fmt.Errorf("failed to format ext4 file: %w", err)
	}
	return nil
}

// Mount ext4 filesystem
func mountExt4(ext4Path, mountPoint string) error {
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point: %w", err)
	}
	return exec.Command("mount", "-o", "loop", ext4Path, mountPoint).Run()
}

// Unmount ext4 filesystem
func unmountExt4(mountPoint string) error {
	return exec.Command("umount", mountPoint).Run()
}

// Copy directory contents
func copyDir(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, srcFile); err != nil {
			return err
		}

		return os.Chmod(destPath, info.Mode())
	})
}
