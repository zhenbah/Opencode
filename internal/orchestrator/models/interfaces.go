package models

import (
	"context"
	"time"

	orchestratorpb "github.com/opencode-ai/opencode/internal/proto/orchestrator/v1"
)

// Config holds the orchestrator configuration
type Config struct {
	Namespace  string
	Kubeconfig string
	Image      string
	Resources  *ResourceConfig
	SessionTTL time.Duration
}

// ResourceConfig defines resource limits for sessions
type ResourceConfig struct {
	CPURequest    string
	CPULimit      string
	MemoryRequest string
	MemoryLimit   string
	StorageSize   string
}

// SessionManager defines the interface for session persistence
type SessionManager interface {
	// CreateSession creates a new session
	CreateSession(ctx context.Context, session *orchestratorpb.CreateSessionRequest) (*orchestratorpb.Session, error)

	// GetSession retrieves a session by ID
	GetSession(ctx context.Context, sessionID string, userID string) (*orchestratorpb.Session, error)

	// GetSessionByUser retrieves a session by ID and verifies user ownership
	GetSessionByUser(ctx context.Context, sessionID, userID string) (*orchestratorpb.Session, error)

	// ListSessions lists all sessions for a user with pagination
	ListSessions(ctx context.Context, userID string, limit int32, pageToken string) ([]*orchestratorpb.Session, string, error)

	// UpdateSession updates an existing session
	UpdateSession(ctx context.Context, session *orchestratorpb.Session) error

	// DeleteSession removes a session
	DeleteSession(ctx context.Context, sessionID string, userID string, force bool) error

	// CountSessions returns the total number of sessions for a user
	CountSessions(ctx context.Context, userID string) (int, error)

	// ListExpiredSessions returns sessions that haven't been accessed within the TTL
	ListExpiredSessions(ctx context.Context, ttl int64) ([]*orchestratorpb.Session, error)

	// UpdateLastAccessed updates the last accessed time for a session
	UpdateLastAccessed(ctx context.Context, sessionID string) error

	// Close closes the session store connection
	Close() error
}

// PodManager handles Kubernetes pod operations
type PodManager interface {
	CreatePod(ctx context.Context, session *orchestratorpb.Session) error
	DeletePod(ctx context.Context, sessionID string) error
	WaitForPodReady(ctx context.Context, sessionID string) error
	GetPodStatus(ctx context.Context, sessionID string) (*orchestratorpb.SessionStatus, error)
}

// StorageManager handles persistent storage
type StorageManager interface {
	CreatePVC(ctx context.Context, session *orchestratorpb.Session) error
	DeletePVC(ctx context.Context, sessionID string) error
	GetPVCStatus(ctx context.Context, sessionID string) (string, error)
}

// ProxyManager handles request proxying to sessions
type ProxyManager interface {
	ProxyHTTP(ctx context.Context, sessionID, userID string, req *orchestratorpb.ProxyHTTPRequest) (*orchestratorpb.ProxyHTTPResponse, error)
	GetSessionEndpoint(ctx context.Context, sessionID string) (string, error)
}
