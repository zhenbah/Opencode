package orchestrator

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/opencode-ai/opencode/internal/orchestrator/models"
	orchestratorpb "github.com/opencode-ai/opencode/internal/proto/orchestrator/v1"
)

// KubernetesStorageManager implements StorageManager using Kubernetes PVC
type KubernetesStorageManager struct {
	client    kubernetes.Interface
	namespace string
	config    *models.Config
}

// NewKubernetesStorageManager creates a new Kubernetes storage manager
func NewKubernetesStorageManager(config *models.Config) (*KubernetesStorageManager, error) {
	var kubeConfig *rest.Config
	var err error

	if config.Kubeconfig == "" {
		kubeConfig, err = rest.InClusterConfig()
	} else {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", config.Kubeconfig)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
	}

	client, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &KubernetesStorageManager{
		client:    client,
		namespace: config.Namespace,
		config:    config,
	}, nil
}

// CreatePVC creates a persistent volume claim for the session
func (m *KubernetesStorageManager) CreatePVC(ctx context.Context, session *orchestratorpb.Session) error {
	pvcName := session.Status.PvcName

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: m.namespace,
			Labels: map[string]string{
				"app":        "opencode-session",
				"session-id": session.Id,
				"user-id":    session.UserId,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(session.Config.StorageSize),
				},
			},
		},
	}

	_, err := m.client.CoreV1().PersistentVolumeClaims(m.namespace).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create PVC: %w", err)
	}

	return nil
}

// DeletePVC deletes a persistent volume claim
func (m *KubernetesStorageManager) DeletePVC(ctx context.Context, sessionID string) error {
	pvcName := fmt.Sprintf("opencode-pvc-%s", sessionID[:8])

	err := m.client.CoreV1().PersistentVolumeClaims(m.namespace).Delete(ctx, pvcName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete PVC: %w", err)
	}

	return nil
}

// GetPVCStatus returns the status of a PVC
func (m *KubernetesStorageManager) GetPVCStatus(ctx context.Context, sessionID string) (string, error) {
	pvcName := fmt.Sprintf("opencode-pvc-%s", sessionID[:8])

	pvc, err := m.client.CoreV1().PersistentVolumeClaims(m.namespace).Get(ctx, pvcName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get PVC: %w", err)
	}

	return string(pvc.Status.Phase), nil
}
