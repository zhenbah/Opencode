package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/opencode-ai/opencode/internal/orchestrator"
	"github.com/opencode-ai/opencode/internal/orchestrator/models"
	orchestratorpb "github.com/opencode-ai/opencode/internal/proto/orchestrator/v1"
)

var (
	grpcPort           int
	httpPort           int
	namespace          string
	kubeconfig         string
	kubernetesCPUReq   string
	kubernetesCPULimit string
	kubernetesMemReq   string
	kubernetesMemLimit string
	kubernetesStorage  string
	kubernetesImage    string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "orchestrator",
		Short: "OpenCode Kubernetes Orchestrator",
		Long:  "Manages OpenCode sessions as Kubernetes pods with persistent storage",
		RunE:  runOrchestrator,
	}

	rootCmd.Flags().IntVar(&grpcPort, "grpc-port", 9090, "gRPC server port")
	rootCmd.Flags().IntVar(&httpPort, "http-port", 9091, "HTTP gateway port")
	rootCmd.Flags().StringVar(&namespace, "namespace", "opencode-sessions", "Kubernetes namespace for sessions")
	rootCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (empty for in-cluster)")
	rootCmd.Flags().StringVar(&kubernetesCPUReq, "kubernetes.cpu-request", "250m", "CPU request for session pods")
	rootCmd.Flags().StringVar(&kubernetesCPULimit, "kubernetes.cpu-limit", "1000m", "CPU limit for session pods")
	rootCmd.Flags().StringVar(&kubernetesMemReq, "kubernetes.memory-request", "512Mi", "Memory request for session pods")
	rootCmd.Flags().StringVar(&kubernetesMemLimit, "kubernetes.memory-limit", "2Gi", "Memory limit for session pods")
	rootCmd.Flags().StringVar(&kubernetesStorage, "kubernetes.storage-size", "10Gi", "Storage size for session workspaces")
	rootCmd.Flags().StringVar(&kubernetesImage, "kubernetes.image", "ghcr.io/denysvitali/opencode:latest", "Container image for session pods")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runOrchestrator(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create Kubernetes runtime configuration
	kubeConfig := &models.KubernetesConfig{
		Namespace:  namespace,
		Kubeconfig: kubeconfig,
		Image:      kubernetesImage,
		Resources: models.ResourceRequirements{
			Requests: models.ResourceList{
				CPU:    kubernetesCPUReq,
				Memory: kubernetesMemReq,
			},
			Limits: models.ResourceList{
				CPU:    kubernetesCPULimit,
				Memory: kubernetesMemLimit,
			},
		},
		StorageSize: kubernetesStorage,
	}

	orchestratorSvc, err := orchestrator.NewService(ctx, &models.Config{
		RuntimeConfig: kubeConfig,
		SessionTTL:    24 * time.Hour,
	})
	if err != nil {
		return fmt.Errorf("failed to create orchestrator service: %w", err)
	}

	// Start gRPC server
	grpcServer := grpc.NewServer()
	orchestratorpb.RegisterOrchestratorServiceServer(grpcServer, orchestratorSvc)
	reflection.Register(grpcServer)

	grpcAddr := fmt.Sprintf(":%d", grpcPort)
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", grpcAddr, err)
	}

	// Start HTTP gateway
	gwMux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := orchestratorpb.RegisterOrchestratorServiceHandlerFromEndpoint(ctx, gwMux, grpcAddr, opts); err != nil {
		return fmt.Errorf("failed to register gateway: %w", err)
	}

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", httpPort),
		Handler: gwMux,
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start servers
	go func() {
		log.Printf("Starting gRPC server on %s", grpcAddr)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	go func() {
		log.Printf("Starting HTTP gateway on :%d", httpPort)
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Orchestrator stopped")
	return nil
}
