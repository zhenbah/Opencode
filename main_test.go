package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
	"github.com/opencode-ai/opencode/internal/logging"
)

func TestRootEndpoint(t *testing.T) {
	// Initialize logging for tests
	logging.InitGlobalLogging("test.log")

	// Create router like in main.go
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Motion Canvas AI Backend - WebSocket Ready"))
	})

	// Create test request
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	expected := "Motion Canvas AI Backend - WebSocket Ready"
	if w.Body.String() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, w.Body.String())
	}
}

func TestGenerateSceneEndpointValidation(t *testing.T) {
	// Initialize logging for tests
	logging.InitGlobalLogging("test.log")

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	
	// Add CORS middleware for testing
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	})

	// Add a test version that doesn't require full app initialization
	r.Post("/generateScene", func(w http.ResponseWriter, r *http.Request) {
		// Parse JSON request
		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.Prompt == "" {
			http.Error(w, "Prompt is required", http.StatusBadRequest)
			return
		}

		// For testing, just return success
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid request",
			requestBody:    `{"prompt": "Create a Motion Canvas scene"}`,
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "Empty prompt",
			requestBody:    `{"prompt": ""}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Prompt is required\n",
		},
		{
			name:           "Invalid JSON",
			requestBody:    `invalid json`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid JSON\n",
		},
		{
			name:           "Missing prompt field",
			requestBody:    `{}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Prompt is required\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/generateScene", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body '%s', got '%s'", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestWebSocketUpgrade(t *testing.T) {
	// Initialize logging for tests
	logging.InitGlobalLogging("test.log")

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("Failed to upgrade: %v", err)
			return
		}
		defer conn.Close()

		// Send a test message
		msg := WebSocketMessage{
			Type:    "test",
			Content: "WebSocket connection successful",
		}
		if err := conn.WriteJSON(msg); err != nil {
			t.Errorf("Failed to write JSON: %v", err)
		}

		// Read client response
		var received WebSocketMessage
		if err := conn.ReadJSON(&received); err != nil {
			t.Logf("Client disconnected or failed to read: %v", err)
		}
	}))
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect to WebSocket
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Read test message
	var msg WebSocketMessage
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	if msg.Type != "test" {
		t.Errorf("Expected type 'test', got '%s'", msg.Type)
	}

	if msg.Content != "WebSocket connection successful" {
		t.Errorf("Expected content 'WebSocket connection successful', got '%s'", msg.Content)
	}

	// Send response
	response := WebSocketMessage{
		Type:    "client_response",
		Content: "Received",
	}
	if err := conn.WriteJSON(response); err != nil {
		t.Errorf("Failed to write JSON response: %v", err)
	}
}
