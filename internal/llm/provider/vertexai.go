package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/vertex"
	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/llm/tools"
	"github.com/opencode-ai/opencode/internal/message"
	"google.golang.org/genai"
)

type VertexAIClient ProviderClient

func newVertexAIClient(opts providerClientOptions) VertexAIClient {
	if isClaudeModel(opts.model.APIModel) {
		client, err := newVertexAIClaudeClient(opts)
		if err != nil {
			logging.Error("Failed to create VertexAI Claude client", "error", err, "model", opts.model.APIModel)
			// Return error client instead of nil to prevent panics
			return &errorClient{err: fmt.Errorf("VertexAI Claude authentication failed: %w", err)}
		}
		return client
	}

	// Existing Gemini implementation (unchanged)
	geminiOpts := geminiOptions{}
	for _, o := range opts.geminiOptions {
		o(&geminiOpts)
	}

	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		Project:  os.Getenv("VERTEXAI_PROJECT"),
		Location: os.Getenv("VERTEXAI_LOCATION"),
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		logging.Error("Failed to create VertexAI Gemini client", "error", err)
		return &errorClient{err: fmt.Errorf("VertexAI Gemini authentication failed: %w", err)}
	}

	return &geminiClient{
		providerOptions: opts,
		options:         geminiOpts,
		client:          client,
	}
}

// Implementation reuses existing anthropicClient with VertexAI configuration
// This approach leverages the proven Claude message conversion logic from anthropic.go
// while configuring the Anthropic client to use VertexAI endpoints.

// isClaudeModel checks if a model is a Claude model by checking for the "claude-" prefix
func isClaudeModel(apiModel string) bool {
	return strings.HasPrefix(apiModel, "claude-")
}

// newVertexAIClaudeClient creates a new VertexAI Claude client using the official
// Anthropic SDK VertexAI integration. This automatically handles authentication,
// endpoint configuration, and API formatting for VertexAI Claude models.
//
// Required environment variables:
//   - VERTEXAI_PROJECT: Google Cloud project ID
//   - VERTEXAI_LOCATION: VertexAI location (e.g., us-central1)
//   - GOOGLE_APPLICATION_CREDENTIALS: Path to service account JSON (or use gcloud auth)
func newVertexAIClaudeClient(opts providerClientOptions) (VertexAIClient, error) {
	// Environment validation
	if err := validateVertexAIEnvironment(); err != nil {
		return nil, fmt.Errorf("VertexAI environment validation failed: %w", err)
	}
	
	project := os.Getenv("VERTEXAI_PROJECT")
	location := os.Getenv("VERTEXAI_LOCATION")
	
	// Use the official Anthropic SDK VertexAI integration
	// This handles all authentication, endpoint configuration, and API formatting automatically
	client := anthropic.NewClient(
		vertex.WithGoogleAuth(context.Background(), location, project),
	)
	
	// Configure Anthropic options from provider options
	anthropicOpts := anthropicOptions{}
	for _, o := range opts.anthropicOptions {
		o(&anthropicOpts)
	}
	
	return &anthropicClient{
		providerOptions: opts,
		options:         anthropicOpts,
		client:          client,
	}, nil
}

// validateVertexAIEnvironment validates required environment variables
func validateVertexAIEnvironment() error {
	project := os.Getenv("VERTEXAI_PROJECT")
	if project == "" {
		return fmt.Errorf("VERTEXAI_PROJECT environment variable is required")
	}
	
	location := os.Getenv("VERTEXAI_LOCATION")
	if location == "" {
		return fmt.Errorf("VERTEXAI_LOCATION environment variable is required")
	}
	
	return nil
}


// errorClient handles authentication failures gracefully without panics
type errorClient struct {
	err error
}

func (e *errorClient) send(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error) {
	return nil, e.err
}

func (e *errorClient) stream(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent {
	ch := make(chan ProviderEvent, 1)
	ch <- ProviderEvent{Type: EventError, Error: e.err}
	close(ch)
	return ch
}
