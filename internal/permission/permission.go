package permission

import (
	"errors"
	"path/filepath"
	"slices"
	"sync"

	"github.com/google/uuid"
	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/logging" // Added for logging
	"github.com/opencode-ai/opencode/internal/pubsub"
)

var ErrorPermissionDenied = errors.New("permission denied")

type CreatePermissionRequest struct {
	SessionID   string `json:"session_id"`
	ToolName    string `json:"tool_name"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Params      any    `json:"params"`
	Path        string `json:"path"`
}

type PermissionRequest struct {
	ID          string `json:"id"`
	SessionID   string `json:"session_id"`
	ToolName    string `json:"tool_name"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Params      any    `json:"params"`
	Path        string `json:"path"`
}

type Service interface {
	pubsub.Suscriber[PermissionRequest]
	GrantPersistant(permission PermissionRequest)
	Grant(permission PermissionRequest)
	Deny(permission PermissionRequest)
	Request(opts CreatePermissionRequest) bool
	AutoApproveSession(sessionID string)
	AutoApproveAgent(agentID string)
}

type permissionService struct {
	*pubsub.Broker[PermissionRequest]

	sessionPermissions   []PermissionRequest
	pendingRequests      sync.Map
	autoApproveSessions  []string
	autoApprovedAgentIDs map[string]bool // New field for agent-specific auto-approval
	agentApprovalMutex   sync.RWMutex      // Mutex for autoApprovedAgentIDs
}

// AutoApproveAgent marks an agentID for automatic approval of all its tool use requests.
func (s *permissionService) AutoApproveAgent(agentID string) {
	s.agentApprovalMutex.Lock()
	defer s.agentApprovalMutex.Unlock()
	if s.autoApprovedAgentIDs == nil { // Ensure map is initialized
		s.autoApprovedAgentIDs = make(map[string]bool)
	}
	s.autoApprovedAgentIDs[agentID] = true
	logging.Info("Tool usage auto-approved for agent", "agentID", agentID)
}

func (s *permissionService) GrantPersistant(permission PermissionRequest) {
	respCh, ok := s.pendingRequests.Load(permission.ID)
	if ok {
		respCh.(chan bool) <- true
	}
	s.sessionPermissions = append(s.sessionPermissions, permission)
}

func (s *permissionService) Grant(permission PermissionRequest) {
	respCh, ok := s.pendingRequests.Load(permission.ID)
	if ok {
		respCh.(chan bool) <- true
	}
}

func (s *permissionService) Deny(permission PermissionRequest) {
	respCh, ok := s.pendingRequests.Load(permission.ID)
	if ok {
		respCh.(chan bool) <- false
	}
}

func (s *permissionService) Request(opts CreatePermissionRequest) bool {
	// Check for agent-specific auto-approval first
	s.agentApprovalMutex.RLock()
	isAutoApprovedAgent := s.autoApprovedAgentIDs[opts.SessionID]
	s.agentApprovalMutex.RUnlock()

	if isAutoApprovedAgent {
		logging.Info("Tool request auto-approved for agent", "agentID", opts.SessionID, "toolName", opts.ToolName, "action", opts.Action)
		// Note: The original Request method in the prompt has a different signature and logic for handling pending requests.
		// This simplified version directly returns true for auto-approved agents,
		// bypassing the pending request map and event publishing.
		// If detailed tracking of auto-approved requests is needed, that logic would be added here.
		return true
	}

	// Check for general session auto-approval (e.g., for orchestrator's own session)
	if slices.Contains(s.autoApproveSessions, opts.SessionID) {
		logging.Info("Tool request auto-approved for session", "sessionID", opts.SessionID, "toolName", opts.ToolName, "action", opts.Action)
		return true
	}

	// Proceed with existing logic for manual approval (should not be hit by agents or orchestrator if correctly configured)
	dir := filepath.Dir(opts.Path)
	if dir == "." {
		dir = config.WorkingDirectory()
	}
	permission := PermissionRequest{
		ID:          uuid.New().String(),
		Path:        dir,
		SessionID:   opts.SessionID,
		ToolName:    opts.ToolName,
		Description: opts.Description,
		Action:      opts.Action,
		Params:      opts.Params,
	}

	for _, p := range s.sessionPermissions {
		if p.ToolName == permission.ToolName && p.Action == permission.Action && p.SessionID == permission.SessionID && p.Path == permission.Path {
			return true
		}
	}

	respCh := make(chan bool, 1)

	s.pendingRequests.Store(permission.ID, respCh)
	defer s.pendingRequests.Delete(permission.ID)

	s.Publish(pubsub.CreatedEvent, permission)

	// Wait for the response with a timeout
	resp := <-respCh
	return resp
}

func (s *permissionService) AutoApproveSession(sessionID string) {
	s.autoApproveSessions = append(s.autoApproveSessions, sessionID)
}

func NewPermissionService() Service {
	return &permissionService{
		Broker:               pubsub.NewBroker[PermissionRequest](),
		sessionPermissions:   make([]PermissionRequest, 0),
		autoApproveSessions:  make([]string, 0), // Initialize this slice as well
		autoApprovedAgentIDs: make(map[string]bool), // Initialize the new map
	}
}
