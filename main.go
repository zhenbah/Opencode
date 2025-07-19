package main

import (
	"context"
	"encoding/json"
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

type ChatRequest struct {
	Prompt string `json:"prompt"`
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

func (s *ChatServer) handleGenerateScene(w http.ResponseWriter, r *http.Request) {
	logging.Info("GenerateScene endpoint accessed", "remote_addr", r.RemoteAddr, "user_agent", r.UserAgent())

	// Parse JSON request
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logging.Error("Failed to decode JSON request", "error", err, "remote_addr", r.RemoteAddr)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	logging.Info("Received prompt request", "prompt_length", len(req.Prompt), "prompt_preview", truncateString(req.Prompt, 100))

	if req.Prompt == "" {
		logging.Warn("Empty prompt received", "remote_addr", r.RemoteAddr)
		http.Error(w, "Prompt is required", http.StatusBadRequest)
		return
	}

	// Upgrade to WebSocket
	logging.Info("Attempting to upgrade connection to WebSocket", "remote_addr", r.RemoteAddr)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logging.Error("Failed to upgrade to WebSocket", "error", err, "remote_addr", r.RemoteAddr)
		return
	}
	defer func() {
		logging.Info("Closing WebSocket connection", "remote_addr", r.RemoteAddr)
		conn.Close()
	}()

	logging.Info("WebSocket connection established successfully", "remote_addr", r.RemoteAddr)

	// Handle WebSocket connection
	s.handleWebSocketConnection(conn, req.Prompt)
}

func (s *ChatServer) handleWebSocketConnection(conn *websocket.Conn, prompt string) {
	logging.Info("Starting WebSocket connection handler", "prompt_length", len(prompt))
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		logging.Info("Cancelling WebSocket connection context")
		cancel()
	}()

	// Create session
	logging.Info("Creating new session for Motion Canvas Scene Generation")
	session, err := s.app.Sessions.Create(ctx, "Motion Canvas Scene Generation")
	if err != nil {
		logging.Error("Failed to create session", "error", err)
		s.sendError(conn, "Failed to create session: "+err.Error())
		return
	}
	logging.Info("Session created successfully", "session_id", session.ID)

	// Send session created message
	logging.Info("Sending session_created message to client", "session_id", session.ID)
	s.sendMessage(conn, WebSocketMessage{
		Type:      "session_created",
		SessionID: session.ID,
	})

	// Auto-approve permissions for this session
	logging.Info("Auto-approving permissions for session", "session_id", session.ID)
	s.app.Permissions.AutoApproveSession(session.ID)

	// Run the agent
	logging.Info("Starting CoderAgent for session", "session_id", session.ID, "prompt", truncateString(prompt, 200))
	done, err := s.app.CoderAgent.Run(ctx, session.ID, prompt)
	if err != nil {
		logging.Error("Failed to start CoderAgent", "error", err, "session_id", session.ID)
		s.sendError(conn, "Failed to start agent: "+err.Error())
		return
	}
	logging.Info("CoderAgent started successfully", "session_id", session.ID)

	// Subscribe to agent events
	logging.Info("Subscribing to agent events", "session_id", session.ID)
	eventChan := s.app.CoderAgent.Subscribe(ctx)
	logging.Info("Successfully subscribed to agent events", "session_id", session.ID)

	// Handle agent events and final result
	logging.Info("Starting event loop for session", "session_id", session.ID)
	for {
		select {
		case event := <-eventChan:
			logging.Debug("Received agent event", "event_type", event.Type, "session_id", event.Payload.SessionID, "target_session", session.ID)
			if event.Payload.SessionID == session.ID {
				logging.Info("Processing agent event for our session", "event_type", event.Type, "session_id", session.ID)
				s.handleAgentEvent(conn, event.Payload)
			} else {
				logging.Debug("Ignoring event for different session", "event_session", event.Payload.SessionID, "our_session", session.ID)
			}
		case result := <-done:
			logging.Info("Agent processing completed", "session_id", session.ID, "has_error", result.Error != nil)
			if result.Error != nil {
				logging.Error("Agent completed with error", "error", result.Error, "session_id", session.ID)
				s.sendError(conn, "Agent error: "+result.Error.Error())
			} else {
				logging.Info("Agent completed successfully", "session_id", session.ID, "content_length", len(result.Message.Content().String()))
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
			logging.Info("WebSocket connection handler completed", "session_id", session.ID)
			return
		case <-ctx.Done():
			logging.Info("Context cancelled, ending WebSocket connection handler", "session_id", session.ID)
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
	logging.Info("Starting app initialization")

	// Get current working directory
	logging.Debug("Getting current working directory")
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
	logging.Info("Configuration loaded successfully")

	// Connect to database (this also runs migrations)
	logging.Info("Connecting to database and running migrations")
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

	// Initialize logging
	logging.Info("Initializing global logging")
	logging.InitGlobalLogging("app.log")
	logging.Info("Motion Canvas AI Backend starting up")

	// Initialize app
	logging.Info("Initializing application context")
	ctx := context.Background()
	appInstance, err := initializeApp(ctx)
	if err != nil {
		logging.Error("Failed to initialize app", "error", err)
		log.Fatal("Failed to initialize app:", err)
	}
	defer func() {
		logging.Info("Shutting down application")
		appInstance.Shutdown()
		logging.Info("Application shutdown completed")
	}()

	// Create chat server
	logging.Info("Creating chat server instance")
	chatServer := &ChatServer{
		app:      appInstance,
		upgrader: upgrader,
	}
	logging.Info("Chat server created successfully")

	// Setup routes
	logging.Info("Setting up HTTP routes and middleware")
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

	logging.Debug("Registering generateScene endpoint")
	r.Post("/generateScene", chatServer.handleGenerateScene)

	logging.Info("All routes registered successfully")
	logging.Info("WebSocket server starting", "port", 3000, "endpoints", []string{"/", "/generateScene"})

	if err := http.ListenAndServe(":3000", r); err != nil {
		logging.Error("Server failed to start", "error", err, "port", 3000)
		log.Fatal("Server failed to start:", err)
	}
}
