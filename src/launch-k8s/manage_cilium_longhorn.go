package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	apiv1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	storageSizeRequested, err := resource.ParseQuantity(request.StorageSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid storage size: " + err.Error()})
		return
	}

	// Create namespace
	ns := &apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: request.TenantID,
			Labels: map[string]string{
				"tenant": request.TenantID,
			},
		},
	}
	_, err = k8sClient.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create namespace: " + err.Error()})
		return
	}

	// Create network policy
	np := &netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-isolation", request.TenantID),
			Namespace: request.TenantID,
		},
		Spec: netv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			Ingress: []netv1.NetworkPolicyIngressRule{{
				From: []netv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"tenant": request.TenantID},
					},
				}},
			}},
			Egress: []netv1.NetworkPolicyEgressRule{{
				To: []netv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"tenant": request.TenantID},
					},
				}},
			}},
		},
	}
	_, err = k8sClient.NetworkingV1().NetworkPolicies(request.TenantID).Create(context.TODO(), np, metav1.CreateOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create network policy: " + err.Error()})
		return
	}

	// Create PVC
	pvc := &apiv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-storage", request.TenantID),
			Namespace: request.TenantID,
		},
		Spec: apiv1.PersistentVolumeClaimSpec{
			AccessModes: []apiv1.PersistentVolumeAccessMode{apiv1.ReadWriteOnce},
			Resources: apiv1.VolumeResourceRequirements{
				Requests: apiv1.ResourceList{
					apiv1.ResourceStorage: storageSizeRequested,
				},
			},
		},
	}
	_, err = k8sClient.CoreV1().PersistentVolumeClaims(request.TenantID).Create(context.TODO(), pvc, metav1.CreateOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create PVC: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Tenant created successfully"})
}
