package orchestrator

import (
	"context"

	"github.com/opencode-ai/opencode/internal/orchestrator/models"
	orchestratorpb "github.com/opencode-ai/opencode/internal/proto/orchestrator/v1"
)

// SimpleProxyManager implements ProxyManager using the runtime interface
type SimpleProxyManager struct {
	runtime models.Runtime
}

// ProxyHTTP forwards HTTP requests to the session endpoint
func (m *SimpleProxyManager) ProxyHTTP(ctx context.Context, sessionID, userID string, req *orchestratorpb.ProxyHTTPRequest) (*orchestratorpb.ProxyHTTPResponse, error) {
	// For now, return a simple response
	// TODO: Implement actual HTTP proxying
	return &orchestratorpb.ProxyHTTPResponse{
		StatusCode: 200,
		Body:       []byte("OK"),
		Headers:    make(map[string]string),
	}, nil
}

// GetSessionEndpoint returns the network endpoint for the session
func (m *SimpleProxyManager) GetSessionEndpoint(ctx context.Context, sessionID string) (string, error) {
	return m.runtime.GetSessionEndpoint(ctx, sessionID)
}
