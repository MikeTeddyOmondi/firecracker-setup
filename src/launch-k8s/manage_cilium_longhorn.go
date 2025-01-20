package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var k8sClient *kubernetes.Clientset

func CreateTenant() {
	// Initialize Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	k8sClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Initialize Gin router
	router := gin.Default()
	router.POST("/create-tenant", createTenantHandler)
	router.Run(":8080")
}

// createTenantHandler creates a new namespace, network policy, and PVC for a tenant
func createTenantHandler(c *gin.Context) {
	type TenantRequest struct {
		TenantID     string `json:"tenant_id" binding:"required"`
		StorageSize  string `json:"storage_size" binding:"required"`
	}

	var request TenantRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create namespace
	ns := &v1.Namespace{
		ObjectMeta: v1meta.ObjectMeta{
			Name: request.TenantID,
			Labels: map[string]string{
				"tenant": request.TenantID,
			},
		},
	}
	_, err := k8sClient.CoreV1().Namespaces().Create(context.TODO(), ns, v1meta.CreateOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create namespace: " + err.Error()})
		return
	}

	// Create network policy
	np := &v1.NetworkPolicy{
		ObjectMeta: v1meta.ObjectMeta{
			Name:      fmt.Sprintf("%s-isolation", request.TenantID),
			Namespace: request.TenantID,
		},
		Spec: v1.NetworkPolicySpec{
			PodSelector: v1meta.LabelSelector{},
			Ingress: []v1.NetworkPolicyIngressRule{{
				From: []v1.NetworkPolicyPeer{{
					NamespaceSelector: &v1meta.LabelSelector{
						MatchLabels: map[string]string{"tenant": request.TenantID},
					},
				}},
			}},
			Egress: []v1.NetworkPolicyEgressRule{{
				To: []v1.NetworkPolicyPeer{{
					NamespaceSelector: &v1meta.LabelSelector{
						MatchLabels: map[string]string{"tenant": request.TenantID},
					},
				}},
			}},
		},
	}
	_, err = k8sClient.NetworkingV1().NetworkPolicies(request.TenantID).Create(context.TODO(), np, v1meta.CreateOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create network policy: " + err.Error()})
		return
	}

	// Create PVC
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: v1meta.ObjectMeta{
			Name:      fmt.Sprintf("%s-storage", request.TenantID),
			Namespace: request.TenantID,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: request.StorageSize,
				},
			},
		},
	}
	_, err = k8sClient.CoreV1().PersistentVolumeClaims(request.TenantID).Create(context.TODO(), pvc, v1meta.CreateOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create PVC: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Tenant created successfully"})
}
