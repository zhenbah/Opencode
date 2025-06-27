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
	grpcPort   = 9090
	httpPort   = 9091
	namespace  = "opencode-sessions"
	kubeconfig = ""
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

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runOrchestrator(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create Kubernetes runtime configuration
	kubeConfig := &models.KubernetesConfig{
		Namespace:     namespace,
		Kubeconfig:    kubeconfig,
		Image:         "ghcr.io/denysvitali/opencode:latest",
		CPURequest:    "250m",
		CPULimit:      "1000m",
		MemoryRequest: "512Mi",
		MemoryLimit:   "2Gi",
		StorageSize:   "10Gi",
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
