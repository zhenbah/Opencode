package runtime

import (
	"context"

	orchestratorpb "github.com/opencode-ai/opencode/internal/proto/orchestrator/v1"
)

// Runtime defines the interface for different execution environments
// This abstraction allows the orchestrator to work with different runtimes
// like Kubernetes, Docker, Firecracker, etc.
type Runtime interface {
	// CreateSession creates a new session in the runtime environment
	CreateSession(ctx context.Context, session *orchestratorpb.Session) error

	// GetSessionStatus returns the current status of a session
	GetSessionStatus(ctx context.Context, sessionID string) (*orchestratorpb.SessionStatus, error)

	// DeleteSession removes a session from the runtime environment
	DeleteSession(ctx context.Context, sessionID string) error

	// WaitForSessionReady waits for a session to become ready
	WaitForSessionReady(ctx context.Context, sessionID string) error

	// GetSessionEndpoint returns the network endpoint for the session
	GetSessionEndpoint(ctx context.Context, sessionID string) (string, error)

	// ListSessions returns all sessions managed by this runtime
	ListSessions(ctx context.Context) ([]*orchestratorpb.Session, error)

	// HealthCheck performs a health check of the runtime
	HealthCheck(ctx context.Context) error

	// Close cleans up runtime resources
	Close() error
}

// Config defines the base configuration interface for all runtimes
type Config interface {
	// GetType returns the runtime type
	GetType() string
	
	// Validate validates the configuration
	Validate() error
}

// Factory creates runtime instances
type Factory interface {
	// CreateRuntime creates a new runtime instance
	CreateRuntime(config Config) (Runtime, error)

	// SupportedTypes returns the list of supported runtime types
	SupportedTypes() []string
}
