package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
	"github.com/opencode-ai/opencode/internal/app"
	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/db"
	"github.com/opencode-ai/opencode/internal/llm/agent"
	"github.com/opencode-ai/opencode/internal/logging"
)

// Helper function to truncate strings for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

type WebSocketMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id,omitempty"`
	Content   string `json:"content,omitempty"`
	Error     string `json:"error,omitempty"`
}

type ChatServer struct {
	app      *app.App
	upgrader websocket.Upgrader
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

func (s *ChatServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	logging.Debug("WebSocket endpoint accessed", "remote_addr", r.RemoteAddr, "user_agent", r.UserAgent())

	// Upgrade to WebSocket
	logging.Info("Attempting to upgrade connection to WebSocket", "remote_addr", r.RemoteAddr)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logging.Error("Failed to upgrade to WebSocket", "error", err, "remote_addr", r.RemoteAddr)
		return
	}
	defer func() {
		logging.Debug("Closing WebSocket connection", "remote_addr", r.RemoteAddr)
		conn.Close()
	}()

	logging.Debug("WebSocket connection established successfully", "remote_addr", r.RemoteAddr)
	s.handleWebSocketConnection(conn)
}

func (s *ChatServer) handleWebSocketConnection(conn *websocket.Conn) {
	logging.Debug("Starting WebSocket connection handler")
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		logging.Info("Cancelling WebSocket connection context")
		cancel()
	}()

	// Wait for initial message containing the prompt
	var promptMessage struct {
		Prompt string `json:"prompt"`
	}

	if err := conn.ReadJSON(&promptMessage); err != nil {
		logging.Error("Failed to read prompt message from WebSocket", "error", err)
		s.sendError(conn, "Failed to read prompt message")
		return
	}

	if promptMessage.Prompt == "" {
		logging.Warn("Empty prompt received from WebSocket")
		s.sendError(conn, "Prompt is required")
		return
	}

	logging.Debug("Received prompt from WebSocket", "prompt_length", len(promptMessage.Prompt), "prompt_preview", truncateString(promptMessage.Prompt, 100))

	// Create session
	logging.Debug("Creating new session for Motion Canvas Scene Generation")
	session, err := s.app.Sessions.Create(ctx, "Motion Canvas Scene Generation")
	if err != nil {
		logging.Error("Failed to create session", "error", err)
		s.sendError(conn, "Failed to create session: "+err.Error())
		return
	}
	// logging.Debug("Session created successfully", "session_id", session.ID)

	// Send session created message
	s.sendMessage(conn, WebSocketMessage{
		Type:      "session_created",
		SessionID: session.ID,
	})

	// Auto-approve permissions for this session
	// logging.Debug("Auto-approving permissions for session", "session_id", session.ID)
	s.app.Permissions.AutoApproveSession(session.ID)

	// Run the agent
	logging.Debug("Starting CoderAgent for session", "session_id", session.ID, "prompt", truncateString(promptMessage.Prompt, 200))
	done, err := s.app.CoderAgent.Run(ctx, session.ID, promptMessage.Prompt)
	if err != nil {
		logging.Error("Failed to start CoderAgent", "error", err, "session_id", session.ID)
		s.sendError(conn, "Failed to start agent: "+err.Error())
		return
	}
	logging.Debug("CoderAgent started successfully", "session_id", session.ID)

	// Subscribe to agent events
	logging.Debug("Subscribing to agent events", "session_id", session.ID)
	eventChan := s.app.CoderAgent.Subscribe(ctx)
	logging.Debug("Successfully subscribed to agent events", "session_id", session.ID)

	// Handle agent events and final result
	logging.Debug("Starting event loop for session", "session_id", session.ID)
	for {
		select {
		case event := <-eventChan:
			logging.Debug("Received agent event", "event_type", event.Type, "session_id", event.Payload.SessionID, "target_session", session.ID)
			if event.Payload.SessionID == session.ID {
				logging.Debug("Processing agent event for our session", "event_type", event.Type, "session_id", session.ID)
				s.handleAgentEvent(conn, event.Payload)
			} else {
				logging.Debug("Ignoring event for different session", "event_session", event.Payload.SessionID, "our_session", session.ID)
			}
		case result := <-done:
			logging.Debug("Agent processing completed", "session_id", session.ID, "has_error", result.Error != nil)
			if result.Error != nil {
				logging.Error("Agent completed with error", "error", result.Error, "session_id", session.ID)
				s.sendError(conn, "Agent error: "+result.Error.Error())
			} else {
				logging.Debug("Agent completed successfully", "session_id", session.ID, "content_length", len(result.Message.Content().String()))
				s.sendMessage(conn, WebSocketMessage{
					Type:      "agent_response",
					Content:   result.Message.Content().String(),
					SessionID: session.ID,
				})
				s.sendMessage(conn, WebSocketMessage{
					Type:      "agent_done",
					SessionID: session.ID,
				})
			}
			logging.Debug("WebSocket connection handler completed", "session_id", session.ID)
			return
		case <-ctx.Done():
			logging.Debug("Context cancelled, ending WebSocket connection handler", "session_id", session.ID)
			return
		}
	}
}

func (s *ChatServer) handleAgentEvent(conn *websocket.Conn, event agent.AgentEvent) {
	logging.Debug("Handling agent event", "event_type", event.Type, "session_id", event.SessionID)
	switch event.Type {
	case agent.AgentEventTypeResponse:
		logging.Info("Processing agent response event", "session_id", event.SessionID, "content_length", len(event.Message.Content().String()))
		s.sendMessage(conn, WebSocketMessage{
			Type:    "agent_response",
			Content: event.Message.Content().String(),
		})
	case agent.AgentEventTypeError:
		logging.Error("Processing agent error event", "error", event.Error, "session_id", event.SessionID)
		s.sendError(conn, event.Error.Error())
	default:
		logging.Warn("Unknown agent event type", "event_type", event.Type, "session_id", event.SessionID)
	}
}

func (s *ChatServer) sendMessage(conn *websocket.Conn, msg WebSocketMessage) {
	logging.Debug("Sending WebSocket message", "type", msg.Type, "session_id", msg.SessionID, "content_length", len(msg.Content))
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := conn.WriteJSON(msg); err != nil {
		logging.Error("Failed to send WebSocket message", "error", err, "message_type", msg.Type, "session_id", msg.SessionID)
	} else {
		logging.Debug("WebSocket message sent successfully", "type", msg.Type, "session_id", msg.SessionID)
	}
}

func (s *ChatServer) sendError(conn *websocket.Conn, errorMsg string) {
	logging.Warn("Sending error message to client", "error", errorMsg)
	s.sendMessage(conn, WebSocketMessage{
		Type:  "error",
		Error: errorMsg,
	})
}

func initializeApp(ctx context.Context) (*app.App, error) {
	cwd, err := os.Getwd()
	if err != nil {
		logging.Error("Failed to get current working directory", "error", err)
		return nil, err
	}
	logging.Info("Current working directory obtained", "cwd", cwd)

	// Load config
	logging.Info("Loading configuration", "cwd", cwd)
	_, err = config.Load(cwd, false)
	if err != nil {
		logging.Error("Failed to load configuration", "error", err, "cwd", cwd)
		return nil, err
	}
	dbConn, err := db.Connect()
	if err != nil {
		logging.Error("Failed to connect to database", "error", err)
		return nil, err
	}
	logging.Info("Database connection established successfully")

	// Create app
	logging.Info("Creating app instance")
	appInstance, err := app.New(ctx, dbConn)
	if err != nil {
		logging.Error("Failed to create app instance", "error", err)
		return nil, err
	}
	logging.Info("App instance created successfully")

	return appInstance, nil
}

func main() {
	defer logging.RecoverPanic("main", func() {
		logging.ErrorPersist("Application terminated due to unhandled panic")
	})

	logging.InitGlobalLogging("app.log")

	// Initialize app
	logging.Info("Initializing application context")
	ctx := context.Background()
	appInstance, err := initializeApp(ctx)
	if err != nil {
		logging.Error("Failed to initialize app", "error", err)
		log.Fatal("Failed to initialize app:", err)
	}
	defer func() {
		appInstance.Shutdown()
	}()

	// Create chat server
	logging.Debug("Creating chat server instance")
	chatServer := &ChatServer{
		app:      appInstance,
		upgrader: upgrader,
	}
	logging.Debug("Chat server created successfully")

	// Setup routes
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// Add simple CORS handler
	logging.Debug("Adding CORS middleware")
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "*")

			if r.Method == "OPTIONS" {
				logging.Debug("Handling CORS preflight request", "origin", r.Header.Get("Origin"))
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	logging.Debug("Registering root endpoint")
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		logging.Debug("Root endpoint accessed", "remote_addr", r.RemoteAddr)
		w.Write([]byte("Motion Canvas AI Backend - WebSocket Ready"))
	})

	logging.Debug("Registering WebSocket endpoint")
	r.Get("/ws", chatServer.handleWebSocket)

	logging.Debug("WebSocket server starting", "port", 3000, "endpoints", []string{"/", "/ws"})

	if err := http.ListenAndServe(":3000", r); err != nil {
		logging.Error("Server failed to start", "error", err, "port", 3000)
		log.Fatal("Server failed to start:", err)
	}
}
