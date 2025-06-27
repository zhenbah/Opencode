package runtime

import (
	"context"

	orchestratorpb "github.com/opencode-ai/opencode/internal/proto/orchestrator/v1"
)

// Runtime defines the interface for different session execution environments
type Runtime interface {
	// CreateSession creates a new session environment
	CreateSession(ctx context.Context, session *orchestratorpb.Session) error

	// GetSessionStatus retrieves the current status of a session
	GetSessionStatus(ctx context.Context, sessionID string) (*orchestratorpb.SessionStatus, error)

	// StartSession starts a stopped session
	StartSession(ctx context.Context, sessionID string) error

	// StopSession stops a running session
	StopSession(ctx context.Context, sessionID string) error

	// DeleteSession removes a session and cleans up resources
	DeleteSession(ctx context.Context, sessionID string) error

	// WaitForReady waits for a session to become ready
	WaitForReady(ctx context.Context, sessionID string) error

	// GetSessionEndpoint returns the network endpoint for a session
	GetSessionEndpoint(ctx context.Context, sessionID string) (string, error)

	// Health checks the health of the runtime
	Health(ctx context.Context) error

	// Close cleans up runtime resources
	Close() error
}

// Config holds runtime-specific configuration
type Config struct {
	// Type specifies the runtime type (kubernetes, docker, firecracker, etc.)
	Type string `json:"type"`

	// Generic resource configuration
	Resources *ResourceConfig `json:"resources,omitempty"`

	// Runtime-specific configuration as a map
	RuntimeConfig map[string]interface{} `json:"runtime_config,omitempty"`
}

// ResourceConfig defines generic resource limits
type ResourceConfig struct {
	CPURequest    string `json:"cpu_request,omitempty"`
	CPULimit      string `json:"cpu_limit,omitempty"`
	MemoryRequest string `json:"memory_request,omitempty"`
	MemoryLimit   string `json:"memory_limit,omitempty"`
	StorageSize   string `json:"storage_size,omitempty"`
}

// Session represents a generic session with runtime-agnostic properties
type Session struct {
	ID       string
	UserID   string
	Name     string
	State    orchestratorpb.SessionState
	Endpoint string
	Labels   map[string]string
}
