package main

import (
    // "context"
    "fmt"
    // "log"
    "net/http"
    "os/exec"
    "sync"

    "github.com/gin-gonic/gin"
)

type FirecrackerInstance struct {
    ID          string
    SocketPath  string
    Process     *exec.Cmd
    ServicePort int // Assuming each service runs on a certain port
}

var (
    instances = make(map[string]*FirecrackerInstance)
    mu        sync.Mutex
)

func main() {
    r := gin.Default()

    r.POST("/create", createInstance)
    r.GET("/list", listInstances)
    r.POST("/stop/:id", stopInstance)

    r.Run(":8080")
}

// createInstance launches a new Firecracker VM with specified parameters
func createInstance(c *gin.Context) {
    var params struct {
        KernelPath string `json:"kernel_path" binding:"required"`
        RootfsPath string `json:"rootfs_path" binding:"required"`
        Port       int    `json:"port" binding:"required"`
    }
    if err := c.BindJSON(&params); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    id := fmt.Sprintf("instance-%d", len(instances)+1)
    socketPath := fmt.Sprintf("/tmp/firecracker-%s.socket", id)

    cmd := exec.Command("firecracker", "--api-sock", socketPath)
    if err := cmd.Start(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start Firecracker instance"})
        return
    }

    instance := &FirecrackerInstance{
        ID:          id,
        SocketPath:  socketPath,
        Process:     cmd,
        ServicePort: params.Port,
    }

    // Here you would add logic to configure the VM using Firecracker's API
    // This is pseudo-code; you'd need to implement actual network setup

    mu.Lock()
    instances[id] = instance
    mu.Unlock()

    c.JSON(http.StatusOK, gin.H{"id": id})
}

// listInstances returns all running instances
func listInstances(c *gin.Context) {
    mu.Lock()
    defer mu.Unlock()
    var instanceList []FirecrackerInstance
    for _, instance := range instances {
        instanceList = append(instanceList, *instance)
    }
    c.JSON(http.StatusOK, instanceList)
}

// stopInstance stops a specific Firecracker instance
func stopInstance(c *gin.Context) {
    id := c.Param("id")
    mu.Lock()
    instance, exists := instances[id]
    mu.Unlock()

    if !exists {
        c.JSON(http.StatusNotFound, gin.H{"error": "Instance not found"})
        return
    }

    if err := instance.Process.Process.Kill(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop instance"})
        return
    }

    mu.Lock()
    delete(instances, id)
    mu.Unlock()

    c.JSON(http.StatusOK, gin.H{"message": "Instance stopped"})
}