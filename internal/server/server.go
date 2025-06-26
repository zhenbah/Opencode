package server

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/opencode-ai/opencode/internal/app"
	"github.com/opencode-ai/opencode/internal/message"
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
	// Ensure we have a session
	sessionID, err := s.getCurrentSession(stream.Context())
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get session: %v", err)
	}

	// Convert proto parts to internal message format
	parts := make([]message.ContentPart, len(req.Parts))
	for i, part := range req.Parts {
		parts[i] = convertProtoContentPart(part)
	}

	// Create message request
	msgReq := message.CreateMessageParams{
		Role:  message.User,
		Parts: parts,
		Model: "gpt-4", // TODO: Get from request or config
	}

	// Create the user message
	userMsg, err := s.app.Messages.Create(stream.Context(), sessionID, msgReq)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to create message: %v", err)
	}

	// Send message started event
	err = stream.Send(&pb.SendMessageResponse{
		Response: &pb.SendMessageResponse_MessageStarted{
			MessageStarted: &pb.MessageStarted{
				MessageId: userMsg.ID,
				Timestamp: timestamppb.New(time.Unix(userMsg.CreatedAt, 0)),
			},
		},
	})
	if err != nil {
		return err
	}

	// TODO: Implement actual AI response generation
	// For now, send a simple echo response
	assistantParts := []message.ContentPart{
		message.TextContent{Text: "Echo: " + getTextFromParts(parts)},
	}

	assistantMsg, err := s.app.Messages.Create(stream.Context(), sessionID, message.CreateMessageParams{
		Role:  message.Assistant,
		Parts: assistantParts,
		Model: "gpt-4",
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to create assistant message: %v", err)
	}

	// Send content delta
	err = stream.Send(&pb.SendMessageResponse{
		Response: &pb.SendMessageResponse_ContentDelta{
			ContentDelta: &pb.ContentDelta{
				Text: getTextFromParts(assistantParts),
			},
		},
	})
	if err != nil {
		return err
	}

	// Send completion message
	return stream.Send(&pb.SendMessageResponse{
		Response: &pb.SendMessageResponse_MessageCompleted{
			MessageCompleted: &pb.MessageCompleted{
				MessageId:    assistantMsg.ID,
				FinishReason: pb.FinishReason_FINISH_REASON_END_TURN,
				Timestamp:    timestamppb.New(time.Unix(assistantMsg.CreatedAt, 0)),
				Usage: &pb.TokenUsage{
					PromptTokens:     100, // TODO: Calculate actual usage
					CompletionTokens: 50,
					Cost:             0.001,
				},
			},
		},
	})
}

func (s *OpenCodeServer) ListMessages(ctx context.Context, req *pb.ListMessagesRequest) (*pb.ListMessagesResponse, error) {
	// Ensure we have a session
	sessionID, err := s.getCurrentSession(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get session: %v", err)
	}

	// Get messages from the service
	messages, err := s.app.Messages.List(ctx, sessionID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list messages: %v", err)
	}

	// Apply role filter if specified
	if req.RoleFilter != pb.MessageRole_MESSAGE_ROLE_UNSPECIFIED {
		filtered := make([]message.Message, 0)
		targetRole := convertProtoMessageRole(req.RoleFilter)
		for _, msg := range messages {
			if msg.Role == targetRole {
				filtered = append(filtered, msg)
			}
		}
		messages = filtered
	}

	// Apply pagination
	offset := int32(0)
	limit := int32(50)
	if req.Pagination != nil {
		if req.Pagination.Offset > 0 {
			offset = req.Pagination.Offset
		}
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
	}

	total := int32(len(messages))
	start := offset
	end := offset + limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	pagedMessages := messages[start:end]

	// Convert messages to proto format
	protoMessages := make([]*pb.Message, len(pagedMessages))
	for i, msg := range pagedMessages {
		protoMessages[i] = convertMessageToProto(msg)
	}

	return &pb.ListMessagesResponse{
		Messages: protoMessages,
		Pagination: &pb.PaginationResponse{
			Total:    total,
			Limit:    limit,
			Offset:   offset,
			HasMore:  end < total,
		},
	}, nil
}

func (s *OpenCodeServer) StreamMessages(req *pb.StreamMessagesRequest, stream pb.OpenCodeService_StreamMessagesServer) error {
	// TODO: Implement message streaming
	return status.Errorf(codes.Unimplemented, "StreamMessages not implemented")
}

func (s *OpenCodeServer) ClearMessages(ctx context.Context, req *pb.ClearMessagesRequest) (*emptypb.Empty, error) {
	// Ensure we have a session
	sessionID, err := s.getCurrentSession(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get session: %v", err)
	}

	// Clear all messages in the session
	err = s.app.Messages.DeleteSessionMessages(ctx, sessionID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to clear messages: %v", err)
	}

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

// Helper functions for proto conversion

func convertProtoContentPart(part *pb.ContentPart) message.ContentPart {
	switch content := part.Content.(type) {
	case *pb.ContentPart_Text:
		return message.TextContent{Text: content.Text.Text}
	case *pb.ContentPart_Binary:
		return message.BinaryContent{
			Path:     content.Binary.Path,
			MIMEType: content.Binary.MimeType,
			Data:     content.Binary.Data,
		}
	default:
		// Return empty text content for unsupported types
		return message.TextContent{Text: ""}
	}
}

func convertProtoMessageRole(role pb.MessageRole) message.MessageRole {
	switch role {
	case pb.MessageRole_MESSAGE_ROLE_USER:
		return message.User
	case pb.MessageRole_MESSAGE_ROLE_ASSISTANT:
		return message.Assistant
	case pb.MessageRole_MESSAGE_ROLE_TOOL:
		return message.Tool
	default:
		return message.User
	}
}

func convertMessageToProto(msg message.Message) *pb.Message {
	protoRole := pb.MessageRole_MESSAGE_ROLE_USER
	switch msg.Role {
	case message.Assistant:
		protoRole = pb.MessageRole_MESSAGE_ROLE_ASSISTANT
	case message.Tool:
		protoRole = pb.MessageRole_MESSAGE_ROLE_TOOL
	case message.System:
		protoRole = pb.MessageRole_MESSAGE_ROLE_USER // Map system to user for API
	}

	protoParts := make([]*pb.ContentPart, len(msg.Parts))
	for i, part := range msg.Parts {
		protoParts[i] = convertContentPartToProto(part)
	}

	return &pb.Message{
		Id:        msg.ID,
		Role:      protoRole,
		Parts:     protoParts,
		CreatedAt: timestamppb.New(time.Unix(msg.CreatedAt, 0)),
		UpdatedAt: timestamppb.New(time.Unix(msg.UpdatedAt, 0)),
		Model:     string(msg.Model),
	}
}

func convertContentPartToProto(part message.ContentPart) *pb.ContentPart {
	switch p := part.(type) {
	case message.TextContent:
		return &pb.ContentPart{
			Content: &pb.ContentPart_Text{
				Text: &pb.TextContent{
					Text: p.Text,
				},
			},
		}
	case message.BinaryContent:
		return &pb.ContentPart{
			Content: &pb.ContentPart_Binary{
				Binary: &pb.BinaryContent{
					Path:     p.Path,
					MimeType: p.MIMEType,
					Data:     p.Data,
				},
			},
		}
	case message.ToolCall:
		return &pb.ContentPart{
			Content: &pb.ContentPart_ToolCall{
				ToolCall: &pb.ToolCallContent{
					Id:    p.ID,
					Name:  p.Name,
					Input: p.Input,
				},
			},
		}
	case message.ToolResult:
		return &pb.ContentPart{
			Content: &pb.ContentPart_ToolResult{
				ToolResult: &pb.ToolResultContent{
					ToolCallId: p.ToolCallID,
					Content:    p.Content,
					Metadata:   p.Metadata,
					IsError:    p.IsError,
				},
			},
		}
	default:
		// Return empty text content for unsupported types
		return &pb.ContentPart{
			Content: &pb.ContentPart_Text{
				Text: &pb.TextContent{
					Text: "",
				},
			},
		}
	}
}

func getTextFromParts(parts []message.ContentPart) string {
	text := ""
	for _, part := range parts {
		if textPart, ok := part.(message.TextContent); ok {
			text += textPart.Text + " "
		}
	}
	return text
}
