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
	"github.com/opencode-ai/opencode/internal/orchestrator/session"
	orchestratorpb "github.com/opencode-ai/opencode/internal/proto/orchestrator/v1"
)

// Service implements the OrchestratorService
type Service struct {
	orchestratorpb.UnimplementedOrchestratorServiceServer

	config         *models.Config
	sessionManager models.SessionManager
}

// NewService creates a new orchestrator service
func NewService(ctx context.Context, config *models.Config) (*Service, error) {
	// Create default session store (in-memory)
	store := session.NewInMemorySessionStore()

	// Create service with the store
	service, err := NewServiceWithStore(config, store)
	if err != nil {
		return nil, err
	}

	// Start cleanup goroutine
	go service.cleanupExpiredSessions(ctx)

	return service, nil
}

func NewServiceWithStore(config *models.Config, sessionManager models.SessionManager) (*Service, error) {
	return &Service{
		config:         config,
		sessionManager: sessionManager,
	}, nil
}

// Health implements the health check
func (s *Service) Health(ctx context.Context, req *orchestratorpb.HealthRequest) (*orchestratorpb.HealthResponse, error) {
	count, _ := s.sessionManager.CountSessions(ctx, "")

	return &orchestratorpb.HealthResponse{
		Status:    orchestratorpb.HealthResponse_SERVING,
		Version:   "1.0.0",
		Timestamp: timestamppb.Now(),
		Details: map[string]string{
			"namespace":       s.config.Namespace,
			"active_sessions": fmt.Sprintf("%d", count),
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

	// Create PVC first
	if err := s.storageManager.CreatePVC(ctx, session); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create storage: %v", err)
	}

	// Create pod
	if err := s.podManager.CreatePod(ctx, session); err != nil {
		// Cleanup PVC on pod creation failure
		_ = s.storageManager.DeletePVC(ctx, session.Id)
		return nil, status.Errorf(codes.Internal, "failed to create pod: %v", err)
	}

	// Update session state
	session.State = orchestratorpb.SessionState_SESSION_STATE_CREATING
	session.UpdatedAt = timestamppb.Now()
	_ = s.sessionStore.UpdateSession(ctx, session)

	// Wait for pod to be ready (with timeout)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if err := s.podManager.WaitForPodReady(ctx, session.Id); err != nil {
			log.Printf("Pod failed to become ready for session %s: %v", session.Id, err)
			session.State = orchestratorpb.SessionState_SESSION_STATE_ERROR
			session.Status.Message = fmt.Sprintf("Pod failed to start: %v", err)
		} else {
			session.State = orchestratorpb.SessionState_SESSION_STATE_RUNNING
			session.Status.Ready = true
			session.Status.ReadyAt = timestamppb.Now()
			log.Printf("Session %s is ready", session.Id)
		}
		session.UpdatedAt = timestamppb.Now()
		_ = s.sessionStore.UpdateSession(ctx, session)
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

	// Update session status from pod
	if podStatus, err := s.podManager.GetPodStatus(ctx, session.Id); err == nil {
		session.Status = podStatus
		_ = s.sessionStore.UpdateSession(ctx, session)
	}

	// Update last accessed time
	_ = s.sessionStore.UpdateLastAccessed(ctx, session.Id)

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
	_ = s.sessionStore.UpdateSession(ctx, session)

	// Delete pod
	if err := s.podManager.DeletePod(ctx, session.Id); err != nil && !req.Force {
		return nil, status.Errorf(codes.Internal, "failed to delete pod: %v", err)
	}

	// Delete PVC
	if err := s.storageManager.DeletePVC(ctx, session.Id); err != nil && !req.Force {
		return nil, status.Errorf(codes.Internal, "failed to delete storage: %v", err)
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
	_ = s.sessionStore.UpdateLastAccessed(ctx, session.Id)

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
			expiredSessions, err := s.sessionStore.ListExpiredSessions(ctx, ttlSeconds)
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
