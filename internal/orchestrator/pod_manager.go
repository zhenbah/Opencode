package orchestrator

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

	"github.com/opencode-ai/opencode/internal/orchestrator/models"
	orchestratorpb "github.com/opencode-ai/opencode/internal/proto/orchestrator/v1"
)

// KubernetesPodManager implements PodManager using Kubernetes API
type KubernetesPodManager struct {
	client    kubernetes.Interface
	namespace string
	config    *models.Config
}

// NewKubernetesPodManager creates a new Kubernetes pod manager
func NewKubernetesPodManager(config *models.Config) (*KubernetesPodManager, error) {
	var kubeConfig *rest.Config
	var err error

	if config.Kubeconfig == "" {
		// Use in-cluster config
		kubeConfig, err = rest.InClusterConfig()
	} else {
		// Use provided kubeconfig
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", config.Kubeconfig)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
	}

	client, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &KubernetesPodManager{
		client:    client,
		namespace: config.Namespace,
		config:    config,
	}, nil
}

// CreatePod creates a Kubernetes pod for the session
func (m *KubernetesPodManager) CreatePod(ctx context.Context, session *orchestratorpb.Session) error {
	podName := session.Status.PodName

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: m.namespace,
			Labels: map[string]string{
				"app":        "opencode-session",
				"session-id": session.Id,
				"user-id":    session.UserId,
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:  "opencode",
					Image: session.Config.Image,
					Ports: []corev1.ContainerPort{
						{
							Name:          "grpc",
							ContainerPort: 8080,
							Protocol:      corev1.ProtocolTCP,
						},
						{
							Name:          "http",
							ContainerPort: 8081,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(session.Config.Resources.CpuRequest),
							corev1.ResourceMemory: resource.MustParse(session.Config.Resources.MemoryRequest),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(session.Config.Resources.CpuLimit),
							corev1.ResourceMemory: resource.MustParse(session.Config.Resources.MemoryLimit),
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
							Name:  "GRPC_PORT",
							Value: "8080",
						},
						{
							Name:  "HTTP_PORT",
							Value: "8081",
						},
					},
					Command: []string{"./opencode", "server"},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/health",
								Port: intstr.FromInt(8081),
							},
						},
						InitialDelaySeconds: 5,
						PeriodSeconds:       5,
					},
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/health",
								Port: intstr.FromInt(8081),
							},
						},
						InitialDelaySeconds: 15,
						PeriodSeconds:       20,
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "workspace",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: session.Status.PvcName,
						},
					},
				},
			},
		},
	}

	// Add environment variables from session config
	if session.Config.Environment != nil {
		for key, value := range session.Config.Environment {
			pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{
				Name:  key,
				Value: value,
			})
		}
	}

	_, err := m.client.CoreV1().Pods(m.namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create pod: %w", err)
	}

	return nil
}

// DeletePod deletes a Kubernetes pod
func (m *KubernetesPodManager) DeletePod(ctx context.Context, sessionID string) error {
	podName := fmt.Sprintf("opencode-session-%s", sessionID[:8])

	err := m.client.CoreV1().Pods(m.namespace).Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete pod: %w", err)
	}

	return nil
}

// WaitForPodReady waits for a pod to become ready
func (m *KubernetesPodManager) WaitForPodReady(ctx context.Context, sessionID string) error {
	podName := fmt.Sprintf("opencode-session-%s", sessionID[:8])

	return wait.PollUntilContextTimeout(ctx, 2*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		pod, err := m.client.CoreV1().Pods(m.namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		// Check if pod is ready
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		// Check if pod failed
		if pod.Status.Phase == corev1.PodFailed {
			return false, fmt.Errorf("pod failed: %s", pod.Status.Message)
		}

		return false, nil
	})
}

// GetPodStatus returns the current status of a pod
func (m *KubernetesPodManager) GetPodStatus(ctx context.Context, sessionID string) (*orchestratorpb.SessionStatus, error) {
	podName := fmt.Sprintf("opencode-session-%s", sessionID[:8])

	pod, err := m.client.CoreV1().Pods(m.namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod: %w", err)
	}

	status := &orchestratorpb.SessionStatus{
		PodName:          podName,
		PodNamespace:     m.namespace,
		PvcName:          fmt.Sprintf("opencode-pvc-%s", sessionID[:8]),
		InternalEndpoint: fmt.Sprintf("%s.%s.svc.cluster.local:8081", podName, m.namespace),
		Ready:            false,
		Message:          string(pod.Status.Phase),
	}

	// Check if pod is ready
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			status.Ready = true
			status.ReadyAt = timestamppb.Now()
			break
		}
	}

	return status, nil
}
