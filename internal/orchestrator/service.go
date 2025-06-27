package orchestrator

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/opencode-ai/opencode/internal/orchestrator/models"
	"github.com/opencode-ai/opencode/internal/orchestrator/runtime/kubernetes"
	"github.com/opencode-ai/opencode/internal/orchestrator/session"
	orchestratorpb "github.com/opencode-ai/opencode/internal/proto/orchestrator/v1"
)

// Service implements the OrchestratorService
type Service struct {
	orchestratorpb.UnimplementedOrchestratorServiceServer

	config         *models.Config
	sessionManager models.SessionManager
	runtime        models.Runtime
	proxyManager   *ProxyManager
	connectionPool *ConnectionPool
}

// NewService creates a new orchestrator service
func NewService(ctx context.Context, config *models.Config) (*Service, error) {
	// Create runtime based on configuration
	var rt models.Runtime
	var err error

	switch config.RuntimeConfig.GetType() {
	case "kubernetes":
		kubeConfig, ok := config.RuntimeConfig.(*models.KubernetesConfig)
		if !ok {
			return nil, fmt.Errorf("invalid kubernetes configuration")
		}
		rt, err = kubernetes.NewRuntime(kubeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create kubernetes runtime: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported runtime type: %s", config.RuntimeConfig.GetType())
	}

	// Create session manager (in-memory store)
	store := session.NewInMemorySessionStore()

	// Create connection pool for efficient HTTP handling
	poolConfig := DefaultPoolConfig()
	connectionPool := NewConnectionPool(rt, poolConfig)

	// Create advanced proxy manager with connection pooling
	proxyManager := NewProxyManager(rt, store)

	// Create service with the components
	service, err := NewServiceWithComponents(config, store, rt, proxyManager, connectionPool)
	if err != nil {
		return nil, err
	}

	// Start background monitoring
	go service.cleanupExpiredSessions(ctx)
	go proxyManager.MonitorSessionHealth(ctx)

	return service, nil
}

func NewServiceWithComponents(config *models.Config, sessionManager models.SessionManager, runtime models.Runtime, proxyManager *ProxyManager, connectionPool *ConnectionPool) (*Service, error) {
	return &Service{
		config:         config,
		sessionManager: sessionManager,
		runtime:        runtime,
		proxyManager:   proxyManager,
		connectionPool: connectionPool,
	}, nil
}

// Health implements the health check
func (s *Service) Health(ctx context.Context, req *orchestratorpb.HealthRequest) (*orchestratorpb.HealthResponse, error) {
	count, _ := s.sessionManager.CountSessions(ctx, "")

	// Check runtime health
	runtimeHealthy := true
	if err := s.runtime.HealthCheck(ctx); err != nil {
		runtimeHealthy = false
	}

	status := orchestratorpb.HealthResponse_SERVING
	if !runtimeHealthy {
		status = orchestratorpb.HealthResponse_NOT_SERVING
	}

	return &orchestratorpb.HealthResponse{
		Status:    status,
		Version:   "1.0.0",
		Timestamp: timestamppb.Now(),
		Details: map[string]string{
			"runtime_type":    s.config.RuntimeConfig.GetType(),
			"active_sessions": fmt.Sprintf("%d", count),
			"runtime_healthy": fmt.Sprintf("%t", runtimeHealthy),
		},
	}, nil
}

// CreateSession creates a new session
func (s *Service) CreateSession(ctx context.Context, req *orchestratorpb.CreateSessionRequest) (*orchestratorpb.CreateSessionResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	session, err := s.sessionManager.CreateSession(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create session: %v", err)
	}

	// Create session in runtime
	if err := s.runtime.CreateSession(ctx, session); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create session in runtime: %v", err)
	}

	// Update session state
	session.State = orchestratorpb.SessionState_SESSION_STATE_CREATING
	session.UpdatedAt = timestamppb.Now()
	_ = s.sessionManager.UpdateSession(ctx, session)

	// Wait for session to be ready (with timeout)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if err := s.runtime.WaitForSessionReady(ctx, session.Id); err != nil {
			log.Printf("Session failed to become ready for session %s: %v", session.Id, err)
			session.State = orchestratorpb.SessionState_SESSION_STATE_ERROR
			if session.Status == nil {
				session.Status = &orchestratorpb.SessionStatus{}
			}
			session.Status.Message = fmt.Sprintf("Session failed to start: %v", err)
		} else {
			session.State = orchestratorpb.SessionState_SESSION_STATE_RUNNING
			if session.Status == nil {
				session.Status = &orchestratorpb.SessionStatus{}
			}
			session.Status.Ready = true
			session.Status.ReadyAt = timestamppb.Now()
			log.Printf("Session %s is ready", session.Id)
		}
		session.UpdatedAt = timestamppb.Now()
		_ = s.sessionManager.UpdateSession(ctx, session)
	}()

	return &orchestratorpb.CreateSessionResponse{
		Session: session,
	}, nil
}

// GetSession retrieves session information
func (s *Service) GetSession(ctx context.Context, req *orchestratorpb.GetSessionRequest) (*orchestratorpb.GetSessionResponse, error) {
	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}

	session, err := s.sessionManager.GetSession(ctx, req.SessionId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "session not found: %v", err)
	}

	// Update session status from runtime
	if sessionStatus, err := s.runtime.GetSessionStatus(ctx, session.Id); err == nil {
		session.Status = sessionStatus
		_ = s.sessionManager.UpdateSession(ctx, session)
	}

	// Update last accessed time
	_ = s.sessionManager.UpdateLastAccessed(ctx, session.Id)

	return &orchestratorpb.GetSessionResponse{
		Session: session,
	}, nil
}

// ListSessions lists user sessions
func (s *Service) ListSessions(ctx context.Context, req *orchestratorpb.ListSessionsRequest) (*orchestratorpb.ListSessionsResponse, error) {
	sessions, nextToken, err := s.sessionManager.ListSessions(ctx, req.UserId, req.PageSize, req.PageToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list sessions: %v", err)
	}

	return &orchestratorpb.ListSessionsResponse{
		Sessions:      sessions,
		NextPageToken: nextToken,
		TotalSize:     int32(len(sessions)),
	}, nil
}

// DeleteSession deletes a session
func (s *Service) DeleteSession(ctx context.Context, req *orchestratorpb.DeleteSessionRequest) (*emptypb.Empty, error) {
	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}

	// Get session to verify ownership
	session, err := s.sessionManager.GetSession(ctx, req.SessionId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "session not found: %v", err)
	}

	// Update state to stopping
	session.State = orchestratorpb.SessionState_SESSION_STATE_STOPPING
	session.UpdatedAt = timestamppb.Now()
	_ = s.sessionManager.UpdateSession(ctx, session)

	// Delete session from runtime
	if err := s.runtime.DeleteSession(ctx, session.Id); err != nil && !req.Force {
		return nil, status.Errorf(codes.Internal, "failed to delete session from runtime: %v", err)
	}

	// Remove from registry
	if err := s.sessionManager.DeleteSession(ctx, req.SessionId, req.UserId, req.Force); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete session: %v", err)
	}

	log.Printf("Session %s deleted", req.SessionId)
	return &emptypb.Empty{}, nil
}

// ProxyHTTP proxies HTTP requests to sessions
func (s *Service) ProxyHTTP(ctx context.Context, req *orchestratorpb.ProxyHTTPRequest) (*orchestratorpb.ProxyHTTPResponse, error) {
	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}

	// Verify session exists and is ready
	session, err := s.sessionManager.GetSession(ctx, req.SessionId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "session not found: %v", err)
	}

	if session.State != orchestratorpb.SessionState_SESSION_STATE_RUNNING {
		return nil, status.Errorf(codes.FailedPrecondition, "session is not ready")
	}

	// Update last accessed time
	_ = s.sessionManager.UpdateLastAccessed(ctx, session.Id)

	// Proxy the request
	return s.proxyManager.ProxyHTTP(ctx, req.SessionId, req.UserId, req)
}

// ProxyStream handles streaming proxy requests
func (s *Service) ProxyStream(stream orchestratorpb.OrchestratorService_ProxyStreamServer) error {
	// TODO: Implement streaming proxy
	return status.Error(codes.Unimplemented, "streaming proxy not yet implemented")
}

// cleanupExpiredSessions removes expired sessions
func (s *Service) cleanupExpiredSessions(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ttlSeconds := int64(s.config.SessionTTL.Seconds())
			expiredSessions, err := s.sessionManager.ListExpiredSessions(ctx, ttlSeconds)
			if err != nil {
				log.Printf("Failed to get expired sessions: %v", err)
				continue
			}

			// Delete expired sessions
			for _, session := range expiredSessions {
				log.Printf("Cleaning up expired session: %s", session.Id)
				deleteReq := &orchestratorpb.DeleteSessionRequest{
					SessionId: session.Id,
					Force:     true,
				}
				_, _ = s.DeleteSession(ctx, deleteReq)
			}
		}
	}
}
