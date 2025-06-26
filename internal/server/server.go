package server

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/opencode-ai/opencode/internal/app"
	pb "github.com/opencode-ai/opencode/internal/proto/v1"
)

// OpenCodeServer implements the OpenCodeService
type OpenCodeServer struct {
	pb.UnimplementedOpenCodeServiceServer
	app *app.App
	// Current session state for single-session mode
	currentSessionID string
}

// NewOpenCodeServer creates a new OpenCode server instance
func NewOpenCodeServer(app *app.App) *OpenCodeServer {
	return &OpenCodeServer{
		app: app,
	}
}

// Health check for container status
func (s *OpenCodeServer) Health(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{
		Status:    pb.HealthStatus_HEALTH_STATUS_SERVING,
		Details:   map[string]string{"version": "1.0.0"},
		Timestamp: timestamppb.Now(),
	}, nil
}

// Get current session info (auto-created if none exists)
func (s *OpenCodeServer) GetSession(ctx context.Context, req *pb.GetSessionRequest) (*pb.GetSessionResponse, error) {
	// In single-session mode, auto-create session if none exists
	if s.currentSessionID == "" {
		session, err := s.app.Sessions.Create(ctx, "default")
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create session: %v", err)
		}
		s.currentSessionID = session.ID
	}

	session, err := s.app.Sessions.Get(ctx, s.currentSessionID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "session not found: %v", err)
	}

	return &pb.GetSessionResponse{
		Session: &pb.Session{
			Id:           session.ID,
			Title:        session.Title,
			CreatedAt:    timestamppb.New(time.Unix(session.CreatedAt, 0)),
			MessageCount: session.MessageCount,
		},
	}, nil
}

// Reset/clear the current session
func (s *OpenCodeServer) ResetSession(ctx context.Context, req *pb.ResetSessionRequest) (*pb.ResetSessionResponse, error) {
	if s.currentSessionID != "" {
		// Delete current session
		err := s.app.Sessions.Delete(ctx, s.currentSessionID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to delete session: %v", err)
		}
	}

	// Create new session
	title := req.Title
	if title == "" {
		title = "default"
	}
	session, err := s.app.Sessions.Create(ctx, title)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create new session: %v", err)
	}
	s.currentSessionID = session.ID

	return &pb.ResetSessionResponse{
		Session: &pb.Session{
			Id:           session.ID,
			Title:        session.Title,
			CreatedAt:    timestamppb.New(time.Unix(session.CreatedAt, 0)),
			MessageCount: 0,
		},
	}, nil
}

// Get session statistics
func (s *OpenCodeServer) GetSessionStats(ctx context.Context, req *pb.GetSessionStatsRequest) (*pb.GetSessionStatsResponse, error) {
	if s.currentSessionID == "" {
		return &pb.GetSessionStatsResponse{
			TotalMessages: 0,
			UserMessages: 0,
			AssistantMessages: 0,
			ToolMessages: 0,
		}, nil
	}

	session, err := s.app.Sessions.Get(ctx, s.currentSessionID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "session not found: %v", err)
	}

	return &pb.GetSessionStatsResponse{
		TotalMessages: session.MessageCount,
		UserMessages: 0, // TODO: Count by role
		AssistantMessages: 0,
		ToolMessages: 0,
		PromptTokens: session.PromptTokens,
		CompletionTokens: session.CompletionTokens,
		TotalCost: session.Cost,
		LastActivity: timestamppb.New(time.Unix(session.UpdatedAt, 0)),
		CurrentModel: "", // TODO: Get from agent
	}, nil
}

// Message methods
func (s *OpenCodeServer) SendMessage(req *pb.SendMessageRequest, stream pb.OpenCodeService_SendMessageServer) error {
	// TODO: Implement message sending with streaming response
	return status.Errorf(codes.Unimplemented, "SendMessage not implemented")
}

func (s *OpenCodeServer) ListMessages(ctx context.Context, req *pb.ListMessagesRequest) (*pb.ListMessagesResponse, error) {
	// TODO: Implement message listing
	return nil, status.Errorf(codes.Unimplemented, "ListMessages not implemented")
}

func (s *OpenCodeServer) StreamMessages(req *pb.StreamMessagesRequest, stream pb.OpenCodeService_StreamMessagesServer) error {
	// TODO: Implement message streaming
	return status.Errorf(codes.Unimplemented, "StreamMessages not implemented")
}

func (s *OpenCodeServer) ClearMessages(ctx context.Context, req *pb.ClearMessagesRequest) (*emptypb.Empty, error) {
	// TODO: Implement message clearing
	return &emptypb.Empty{}, nil
}

// File methods
func (s *OpenCodeServer) ListFiles(ctx context.Context, req *pb.ListFilesRequest) (*pb.ListFilesResponse, error) {
	// TODO: Implement file listing
	return nil, status.Errorf(codes.Unimplemented, "ListFiles not implemented")
}

func (s *OpenCodeServer) ReadFile(ctx context.Context, req *pb.ReadFileRequest) (*pb.ReadFileResponse, error) {
	// TODO: Implement file reading
	return nil, status.Errorf(codes.Unimplemented, "ReadFile not implemented")
}

func (s *OpenCodeServer) WriteFile(ctx context.Context, req *pb.WriteFileRequest) (*pb.WriteFileResponse, error) {
	// TODO: Implement file writing
	return nil, status.Errorf(codes.Unimplemented, "WriteFile not implemented")
}

func (s *OpenCodeServer) DeleteFile(ctx context.Context, req *pb.DeleteFileRequest) (*emptypb.Empty, error) {
	// TODO: Implement file deletion
	return &emptypb.Empty{}, nil
}

func (s *OpenCodeServer) GetFileChanges(ctx context.Context, req *pb.GetFileChangesRequest) (*pb.GetFileChangesResponse, error) {
	// TODO: Implement file change tracking
	return nil, status.Errorf(codes.Unimplemented, "GetFileChanges not implemented")
}

// Agent methods
func (s *OpenCodeServer) CancelAgent(ctx context.Context, req *pb.CancelAgentRequest) (*emptypb.Empty, error) {
	// TODO: Implement agent cancellation
	return &emptypb.Empty{}, nil
}

func (s *OpenCodeServer) GetAgentStatus(ctx context.Context, req *pb.GetAgentStatusRequest) (*pb.GetAgentStatusResponse, error) {
	// TODO: Implement agent status
	return nil, status.Errorf(codes.Unimplemented, "GetAgentStatus not implemented")
}

func (s *OpenCodeServer) ListModels(ctx context.Context, req *pb.ListModelsRequest) (*pb.ListModelsResponse, error) {
	// TODO: Implement model listing
	return nil, status.Errorf(codes.Unimplemented, "ListModels not implemented")
}

func (s *OpenCodeServer) SetModel(ctx context.Context, req *pb.SetModelRequest) (*pb.SetModelResponse, error) {
	// TODO: Implement model setting
	return nil, status.Errorf(codes.Unimplemented, "SetModel not implemented")
}

// getCurrentSession returns the current session ID, creating one if needed
func (s *OpenCodeServer) getCurrentSession(ctx context.Context) (string, error) {
	if s.currentSessionID == "" {
		// Auto-create session on first interaction
		session, err := s.app.Sessions.Create(ctx, "OpenCode API Session")
		if err != nil {
			return "", err
		}
		s.currentSessionID = session.ID
	}
	return s.currentSessionID, nil
}
