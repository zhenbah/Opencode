package kubernetes

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	orchestratorpb "github.com/opencode-ai/opencode/internal/proto/orchestrator/v1"
	"github.com/opencode-ai/opencode/pkg/runtime"
)

// KubernetesRuntime implements the Runtime interface for Kubernetes
type KubernetesRuntime struct {
	client    kubernetes.Interface
	namespace string
	image     string
	resources *runtime.ResourceConfig
}

// Config holds Kubernetes-specific configuration
type Config struct {
	Namespace  string `json:"namespace"`
	Kubeconfig string `json:"kubeconfig"`
	Image      string `json:"image"`
}

var _ runtime.Runtime = (*KubernetesRuntime)(nil)

// NewKubernetesRuntime creates a new Kubernetes runtime
func NewKubernetesRuntime(config *runtime.Config) (*KubernetesRuntime, error) {
	// Extract Kubernetes-specific config
	kubeConfig := &Config{}
	if config.RuntimeConfig != nil {
		if ns, ok := config.RuntimeConfig["namespace"].(string); ok {
			kubeConfig.Namespace = ns
		}
		if kc, ok := config.RuntimeConfig["kubeconfig"].(string); ok {
			kubeConfig.Kubeconfig = kc
		}
		if img, ok := config.RuntimeConfig["image"].(string); ok {
			kubeConfig.Image = img
		}
	}

	// Set defaults
	if kubeConfig.Namespace == "" {
		kubeConfig.Namespace = "opencode-sessions"
	}
	if kubeConfig.Image == "" {
		kubeConfig.Image = "ghcr.io/denysvitali/opencode:latest"
	}

	var restConfig *rest.Config
	var err error

	if kubeConfig.Kubeconfig == "" {
		// Use in-cluster config
		restConfig, err = rest.InClusterConfig()
	} else {
		// Use provided kubeconfig
		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfig.Kubeconfig)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &KubernetesRuntime{
		client:    client,
		namespace: kubeConfig.Namespace,
		image:     kubeConfig.Image,
		resources: config.Resources,
	}, nil
}

// CreateSession creates a new session environment in Kubernetes
func (k *KubernetesRuntime) CreateSession(ctx context.Context, session *orchestratorpb.Session) error {
	// Create PVC first
	if err := k.createPVC(ctx, session); err != nil {
		return fmt.Errorf("failed to create PVC: %w", err)
	}

	// Create pod
	if err := k.createPod(ctx, session); err != nil {
		// Cleanup PVC on pod creation failure
		_ = k.deletePVC(ctx, session.Id)
		return fmt.Errorf("failed to create pod: %w", err)
	}

	return nil
}

// GetSessionStatus retrieves the current status of a session
func (k *KubernetesRuntime) GetSessionStatus(ctx context.Context, sessionID string) (*orchestratorpb.SessionStatus, error) {
	podName := k.getPodName(sessionID)
	pod, err := k.client.CoreV1().Pods(k.namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod status: %w", err)
	}

	status := &orchestratorpb.SessionStatus{
		PodName:   pod.Name,
		Ready:     false,
		Message:   string(pod.Status.Phase),
		CreatedAt: timestamppb.New(pod.CreationTimestamp.Time),
	}

	// Check if pod is ready
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			status.Ready = true
			status.ReadyAt = timestamppb.New(condition.LastTransitionTime.Time)
			break
		}
	}

	// Get pod IP for endpoint
	if pod.Status.PodIP != "" {
		status.Endpoint = fmt.Sprintf("http://%s:8081", pod.Status.PodIP)
	}

	return status, nil
}

// StartSession starts a stopped session
func (k *KubernetesRuntime) StartSession(ctx context.Context, sessionID string) error {
	// For Kubernetes, we don't typically "start" pods, we create them
	// This could be implemented with deployment scaling or similar
	return fmt.Errorf("start session not implemented for Kubernetes runtime")
}

// StopSession stops a running session
func (k *KubernetesRuntime) StopSession(ctx context.Context, sessionID string) error {
	podName := k.getPodName(sessionID)
	return k.client.CoreV1().Pods(k.namespace).Delete(ctx, podName, metav1.DeleteOptions{})
}

// DeleteSession removes a session and cleans up resources
func (k *KubernetesRuntime) DeleteSession(ctx context.Context, sessionID string) error {
	// Delete pod
	podName := k.getPodName(sessionID)
	err := k.client.CoreV1().Pods(k.namespace).Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil {
		// Don't fail if pod doesn't exist
	}

	// Delete PVC
	return k.deletePVC(ctx, sessionID)
}

// WaitForReady waits for a session to become ready
func (k *KubernetesRuntime) WaitForReady(ctx context.Context, sessionID string) error {
	podName := k.getPodName(sessionID)
	
	return wait.PollUntilContextTimeout(ctx, 5*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		pod, err := k.client.CoreV1().Pods(k.namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	})
}

// GetSessionEndpoint returns the network endpoint for a session
func (k *KubernetesRuntime) GetSessionEndpoint(ctx context.Context, sessionID string) (string, error) {
	podName := k.getPodName(sessionID)
	pod, err := k.client.CoreV1().Pods(k.namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get pod: %w", err)
	}

	if pod.Status.PodIP == "" {
		return "", fmt.Errorf("pod IP not yet assigned")
	}

	return fmt.Sprintf("http://%s:8081", pod.Status.PodIP), nil
}

// Health checks the health of the runtime
func (k *KubernetesRuntime) Health(ctx context.Context) error {
	_, err := k.client.CoreV1().Namespaces().Get(ctx, k.namespace, metav1.GetOptions{})
	return err
}

// Close cleans up runtime resources
func (k *KubernetesRuntime) Close() error {
	// No explicit cleanup needed for Kubernetes client
	return nil
}

// Helper methods

func (k *KubernetesRuntime) getPodName(sessionID string) string {
	// Use first 8 characters of session ID to keep pod name short
	shortID := sessionID
	if len(sessionID) > 8 {
		shortID = sessionID[:8]
	}
	return fmt.Sprintf("opencode-session-%s", shortID)
}

func (k *KubernetesRuntime) getPVCName(sessionID string) string {
	// Use first 8 characters of session ID to keep PVC name short
	shortID := sessionID
	if len(sessionID) > 8 {
		shortID = sessionID[:8]
	}
	return fmt.Sprintf("opencode-storage-%s", shortID)
}

func (k *KubernetesRuntime) createPod(ctx context.Context, session *orchestratorpb.Session) error {
	podName := k.getPodName(session.Id)
	pvcName := k.getPVCName(session.Id)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: k.namespace,
			Labels: map[string]string{
				"app":        "opencode-session",
				"session-id": session.Id[:8], // Truncate for valid label value
				"user-id":    session.UserId,
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:  "opencode",
					Image: k.image,
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 8081,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "workspace",
							MountPath: "/workspace",
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  "SESSION_ID",
							Value: session.Id,
						},
						{
							Name:  "USER_ID",
							Value: session.UserId,
						},
					},
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/health",
								Port:   intstr.FromInt(8081),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						InitialDelaySeconds: 30,
						PeriodSeconds:       10,
						TimeoutSeconds:      5,
						FailureThreshold:    3,
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/health",
								Port:   intstr.FromInt(8081),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						InitialDelaySeconds: 5,
						PeriodSeconds:       5,
						TimeoutSeconds:      3,
						FailureThreshold:    3,
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "workspace",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
			},
		},
	}

	// Apply resource limits if configured
	if k.resources != nil {
		container := &pod.Spec.Containers[0]
		container.Resources = corev1.ResourceRequirements{}

		if k.resources.CPURequest != "" || k.resources.MemoryRequest != "" {
			container.Resources.Requests = make(corev1.ResourceList)
			if k.resources.CPURequest != "" {
				container.Resources.Requests[corev1.ResourceCPU] = resource.MustParse(k.resources.CPURequest)
			}
			if k.resources.MemoryRequest != "" {
				container.Resources.Requests[corev1.ResourceMemory] = resource.MustParse(k.resources.MemoryRequest)
			}
		}

		if k.resources.CPULimit != "" || k.resources.MemoryLimit != "" {
			container.Resources.Limits = make(corev1.ResourceList)
			if k.resources.CPULimit != "" {
				container.Resources.Limits[corev1.ResourceCPU] = resource.MustParse(k.resources.CPULimit)
			}
			if k.resources.MemoryLimit != "" {
				container.Resources.Limits[corev1.ResourceMemory] = resource.MustParse(k.resources.MemoryLimit)
			}
		}
	}

	_, err := k.client.CoreV1().Pods(k.namespace).Create(ctx, pod, metav1.CreateOptions{})
	return err
}

func (k *KubernetesRuntime) createPVC(ctx context.Context, session *orchestratorpb.Session) error {
	pvcName := k.getPVCName(session.Id)
	
	storageSize := "10Gi"
	if k.resources != nil && k.resources.StorageSize != "" {
		storageSize = k.resources.StorageSize
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: k.namespace,
			Labels: map[string]string{
				"app":        "opencode-session",
				"session-id": session.Id[:8], // Truncate for valid label value
				"user-id":    session.UserId,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(storageSize),
				},
			},
		},
	}

	_, err := k.client.CoreV1().PersistentVolumeClaims(k.namespace).Create(ctx, pvc, metav1.CreateOptions{})
	return err
}

func (k *KubernetesRuntime) deletePVC(ctx context.Context, sessionID string) error {
	pvcName := k.getPVCName(sessionID)
	return k.client.CoreV1().PersistentVolumeClaims(k.namespace).Delete(ctx, pvcName, metav1.DeleteOptions{})
}
