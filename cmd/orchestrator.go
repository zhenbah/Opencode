package cmd

import (
	"context"
	"errors"
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
	grpcPort     int
	httpPort     int
	namespace    string
	kubeconfig   string
	runtimeImage string
	cpuReq       string
	memoryReq    string
	memoryLimit  string
	storageSize  string
)

func init() {
	orchestratorCmd := &cobra.Command{
		Use:   "orchestrator",
		Short: "OpenCode Kubernetes Orchestrator",
		Long:  "Manages OpenCode sessions as Kubernetes pods with persistent storage",
		RunE:  runOrchestrator,
	}

	orchestratorCmd.Flags().IntVar(&grpcPort, "grpc-port", 9090, "gRPC server port")
	orchestratorCmd.Flags().IntVar(&httpPort, "http-port", 9091, "HTTP gateway port")
	orchestratorCmd.Flags().StringVar(&namespace, "namespace", "opencode-sessions", "Kubernetes namespace for sessions")
	orchestratorCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (empty for in-cluster)")
	orchestratorCmd.Flags().StringVar(&runtimeImage, "runtime-image", "ghcr.io/denysvitali/opencode:latest", "Containr image for OpenCode runtime environment")
	orchestratorCmd.Flags().StringVar(&cpuReq, "cpu-request", "50m", "CPU request for session pods")
	orchestratorCmd.Flags().StringVar(&memoryReq, "memory-request", "128Mi", "Memory request for session pods")
	orchestratorCmd.Flags().StringVar(&memoryLimit, "memory-limit", "256Mi", "Memory limit for session pods")
	orchestratorCmd.Flags().StringVar(&storageSize, "storage-size", "10Gi", "Persistent storage size for session pods")

	rootCmd.AddCommand(orchestratorCmd)
}

func runOrchestrator(*cobra.Command, []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create Kubernetes runtime configuration
	kubeConfig := &models.KubernetesConfig{
		Namespace:     namespace,
		Kubeconfig:    kubeconfig,
		Image:         runtimeImage,
		CPURequest:    cpuReq,
		CPULimit:      cpuReq, // For now, use same value
		MemoryRequest: memoryReq,
		MemoryLimit:   memoryLimit,
		StorageSize:   storageSize,
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
		if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
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
