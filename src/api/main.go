package main

import (
	"context"
	"log"
	"net/http"
	// "time"

	"github.com/gin-gonic/gin"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/cio"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	client    *containerd.Client
	k8sClient *kubernetes.Clientset
)

func main() {
	// Initialize Gin Router
	r := gin.Default()

	// Setup Kubernetes and Containerd clients
	setupClients()

	// Routes
	r.POST("/deploy", deployMicroVM)
	// r.POST("/pause/:id", pauseMicroVM)
	// r.POST("/resume/:id", resumeMicroVM)
	r.DELETE("/delete/:id", deleteMicroVM)

	// Run server
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to run API server: %v", err)
	}
}

// setupClients initializes Kubernetes and Containerd clients
func setupClients() {
	// Kubernetes Client
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to create in-cluster config: %v", err)
	}
	k8sClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Containerd Client
	client, err = containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		log.Fatalf("Failed to connect to containerd: %v", err)
	}
}

// deployMicroVM deploys an OCI image as a Firecracker microVM
func deployMicroVM(c *gin.Context) {
	var request struct {
		Image       string `json:"image" binding:"required"`
		MicroVMID   string `json:"microvm_id" binding:"required"`
		Namespace   string `json:"namespace" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := namespaces.WithNamespace(context.Background(), request.Namespace)

	// Pull image if not already available
	image, err := client.Pull(ctx, request.Image, containerd.WithPullUnpack)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to pull image: " + err.Error()})
		return
	}

	// Create a new microVM task
	container, err := client.NewContainer(
		ctx,
		request.MicroVMID,
		containerd.WithNewSnapshot(request.MicroVMID+"-snapshot", image),
		containerd.WithNewSpec(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create container: " + err.Error()})
		return
	}

	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task: " + err.Error()})
		return
	}

	// Start the task
	if err := task.Start(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start task: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "MicroVM deployed successfully", "id": request.MicroVMID})
}

// pauseMicroVM pauses a running Firecracker microVM
// func pauseMicroVM(c *gin.Context) {
// 	id := c.Param("id")
// 	ctx := namespaces.WithNamespace(context.Background(), "default")

// 	task, err := client.LoadTask(ctx, id)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load task: " + err.Error()})
// 		return
// 	}

// 	if err := task.Pause(ctx); err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to pause microVM: " + err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"status": "MicroVM paused successfully", "id": id})
// }

// // resumeMicroVM resumes a paused Firecracker microVM
// func resumeMicroVM(c *gin.Context) {
// 	id := c.Param("id")
// 	ctx := namespaces.WithNamespace(context.Background(), "default")

// 	task, err := client.LoadTask(ctx, id)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load task: " + err.Error()})
// 		return
// 	}

// 	if err := task.Resume(ctx); err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resume microVM: " + err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"status": "MicroVM resumed successfully", "id": id})
// }

// deleteMicroVM deletes a Firecracker microVM
func deleteMicroVM(c *gin.Context) {
	id := c.Param("id")
	ctx := namespaces.WithNamespace(context.Background(), "default")

	container, err := client.LoadContainer(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load container: " + err.Error()})
		return
	}

	if err := container.Delete(ctx, containerd.WithSnapshotCleanup); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete microVM: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "MicroVM deleted successfully", "id": id})
}
