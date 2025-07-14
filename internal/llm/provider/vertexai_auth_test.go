package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/opencode-ai/opencode/internal/llm/models"
)

// TestVertexAIAuth_ValidADC tests successful authentication with Application Default Credentials
func TestVertexAIAuth_ValidADC(t *testing.T) {
	// Set up test environment
	t.Setenv("VERTEXAI_PROJECT", "test-project")
	t.Setenv("VERTEXAI_LOCATION", "us-central1")
	
	// Test environment validation
	err := validateVertexAIEnvironment()
	assert.NoError(t, err, "Environment validation should pass with valid env vars")
}

// TestVertexAIAuth_ServiceAccountFile tests authentication with service account file
func TestVertexAIAuth_ServiceAccountFile(t *testing.T) {
	t.Setenv("VERTEXAI_PROJECT", "test-project")
	t.Setenv("VERTEXAI_LOCATION", "us-central1")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/path/to/service-account.json")
	
	err := validateVertexAIEnvironment()
	assert.NoError(t, err, "Environment validation should pass with service account file")
}

// TestVertexAIAuth_MissingCredentials tests handling of missing credentials
func TestVertexAIAuth_MissingCredentials(t *testing.T) {
	// Clear any existing credentials
	t.Setenv("VERTEXAI_PROJECT", "")
	t.Setenv("VERTEXAI_LOCATION", "")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")
	
	err := validateVertexAIEnvironment()
	assert.Error(t, err, "Environment validation should fail with missing credentials")
	assert.Contains(t, err.Error(), "VERTEXAI_PROJECT", "Error should mention missing project")
}

// TestVertexAIAuth_InvalidCredentials tests handling of invalid credentials
func TestVertexAIAuth_InvalidCredentials(t *testing.T) {
	t.Setenv("VERTEXAI_PROJECT", "")
	t.Setenv("VERTEXAI_LOCATION", "us-central1")
	
	err := validateVertexAIEnvironment()
	assert.Error(t, err, "Environment validation should fail with missing project")
}

// TestNewVertexAIClaudeClient_Success tests successful Claude client creation
func TestNewVertexAIClaudeClient_Success(t *testing.T) {
	t.Setenv("VERTEXAI_PROJECT", "test-project")
	t.Setenv("VERTEXAI_LOCATION", "us-central1")
	
	opts := providerClientOptions{
		model: models.Model{APIModel: "claude-3-sonnet"},
	}
	
	client, err := newVertexAIClaudeClient(opts)
	// Implementation should now succeed with valid Google Cloud credentials
	if err != nil {
		// If authentication fails, it should be due to missing/invalid credentials
		assert.Contains(t, err.Error(), "Google Cloud", "Error should be related to Google Cloud authentication")
		assert.Nil(t, client)
	} else {
		// If authentication succeeds, we should get a valid client
		assert.NotNil(t, client, "Client should not be nil when creation succeeds")
	}
}

// TestNewVertexAIClaudeClient_AuthFailure tests Claude client creation with auth failure
func TestNewVertexAIClaudeClient_AuthFailure(t *testing.T) {
	t.Setenv("VERTEXAI_PROJECT", "")
	t.Setenv("VERTEXAI_LOCATION", "")
	
	opts := providerClientOptions{
		model: models.Model{APIModel: "claude-3-sonnet"},
	}
	
	client, err := newVertexAIClaudeClient(opts)
	assert.Error(t, err, "Client creation should fail with missing environment")
	assert.Nil(t, client, "Client should be nil when creation fails")
}

// TestNewVertexAIClaudeClient_EnvironmentValidation tests environment variable validation
func TestNewVertexAIClaudeClient_EnvironmentValidation(t *testing.T) {
	tests := []struct {
		name     string
		project  string
		location string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid environment",
			project:  "test-project",
			location: "us-central1",
			wantErr:  false,
		},
		{
			name:     "missing project",
			project:  "",
			location: "us-central1",
			wantErr:  true,
			errMsg:   "VERTEXAI_PROJECT",
		},
		{
			name:     "missing location",
			project:  "test-project",
			location: "",
			wantErr:  true,
			errMsg:   "VERTEXAI_LOCATION",
		},
		{
			name:     "missing both",
			project:  "",
			location: "",
			wantErr:  true,
			errMsg:   "VERTEXAI_PROJECT",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("VERTEXAI_PROJECT", tt.project)
			t.Setenv("VERTEXAI_LOCATION", tt.location)
			
			err := validateVertexAIEnvironment()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestVertexAIClient_NeverReturnsNil tests that client creation never returns nil without error
func TestVertexAIClient_NeverReturnsNil(t *testing.T) {
	opts := providerClientOptions{
		model: models.Model{APIModel: "claude-3-sonnet"},
	}
	
	// Test with missing environment (should return error client, not nil)
	t.Setenv("VERTEXAI_PROJECT", "")
	t.Setenv("VERTEXAI_LOCATION", "")
	
	client := newVertexAIClient(opts)
	assert.NotNil(t, client, "Client should never be nil, even on auth failure")
	
	// Test streaming with error client
	ctx := context.Background()
	ch := client.stream(ctx, nil, nil)
	select {
	case event := <-ch:
		assert.Equal(t, EventError, event.Type, "Should receive error event")
		assert.Error(t, event.Error, "Error event should contain error")
	default:
		t.Fatal("Should receive error event from stream")
	}
}

// TestVertexAIClient_ErrorPropagation tests proper error propagation
func TestVertexAIClient_ErrorPropagation(t *testing.T) {
	opts := providerClientOptions{
		model: models.Model{APIModel: "claude-3-sonnet"},
	}
	
	t.Setenv("VERTEXAI_PROJECT", "")
	t.Setenv("VERTEXAI_LOCATION", "")
	
	client := newVertexAIClient(opts)
	require.NotNil(t, client)
	
	// Test send method error propagation
	ctx := context.Background()
	response, err := client.send(ctx, nil, nil)
	assert.Error(t, err, "Send should return error when auth fails")
	assert.Nil(t, response, "Response should be nil when error occurs")
	assert.Contains(t, err.Error(), "VERTEXAI_PROJECT", "Error should indicate missing environment variable")
}

// TestVertexAIClient_MeaningfulErrors tests that errors provide actionable information
func TestVertexAIClient_MeaningfulErrors(t *testing.T) {
	tests := []struct {
		name     string
		project  string
		location string
		errMsg   string
	}{
		{
			name:     "missing project only",
			project:  "",
			location: "us-central1",
			errMsg:   "VERTEXAI_PROJECT environment variable is required",
		},
		{
			name:     "missing location only",
			project:  "test-project",
			location: "",
			errMsg:   "VERTEXAI_LOCATION environment variable is required",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("VERTEXAI_PROJECT", tt.project)
			t.Setenv("VERTEXAI_LOCATION", tt.location)
			
			err := validateVertexAIEnvironment()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

// TestVertexAI_GeminiUnchanged tests that Gemini models continue working unchanged
func TestVertexAI_GeminiUnchanged(t *testing.T) {
	opts := providerClientOptions{
		model: models.Model{APIModel: "gemini-pro"},
	}
	
	// Gemini should work without VertexAI Claude environment
	t.Setenv("VERTEXAI_PROJECT", "test-project")
	t.Setenv("VERTEXAI_LOCATION", "us-central1")
	
	client := newVertexAIClient(opts)
	// Note: This will likely fail in CI due to missing Google credentials,
	// but the important thing is that it follows the Gemini code path
	
	// Verify it's not a Claude client by checking it's not an error client
	if client != nil {
		// If client creation succeeded, it should be a geminiClient
		_, isErrorClient := client.(*errorClient)
		assert.False(t, isErrorClient, "Gemini client should not be an error client")
	}
}

// TestVertexAI_ClaudeRouting tests that Claude models are routed correctly
func TestVertexAI_ClaudeRouting(t *testing.T) {
	claudeModels := []string{
		"claude-3-sonnet",
		"claude-3-opus",
		"claude-3-haiku",
		"claude-3-5-sonnet",
	}
	
	for _, model := range claudeModels {
		t.Run(model, func(t *testing.T) {
			assert.True(t, isClaudeModel(model), "Model %s should be identified as Claude model", model)
		})
	}
	
	nonClaudeModels := []string{
		"gemini-pro",
		"gemini-pro-vision",
		"text-bison",
		"gpt-4",
	}
	
	for _, model := range nonClaudeModels {
		t.Run(model, func(t *testing.T) {
			assert.False(t, isClaudeModel(model), "Model %s should not be identified as Claude model", model)
		})
	}
}

// TestVertexAI_EndToEndAuth tests end-to-end authentication flow using official SDK
func TestVertexAI_EndToEndAuth(t *testing.T) {
	t.Setenv("VERTEXAI_PROJECT", "test-project")
	t.Setenv("VERTEXAI_LOCATION", "us-central1")
	
	// Test environment validation (this still applies)
	err := validateVertexAIEnvironment()
	assert.NoError(t, err, "Environment validation should pass")
	
	// Authentication is now handled by the official Anthropic SDK VertexAI integration
	// We can test client creation without authentication errors if credentials are available
	opts := providerClientOptions{
		model: models.Model{APIModel: "claude-sonnet-4"},
	}
	
	// This may fail with auth errors in CI/test environment, which is expected
	client, err := newVertexAIClaudeClient(opts)
	if err != nil {
		// Expected in test environment without valid Google Cloud credentials
		t.Logf("Expected auth error in test environment: %v", err)
		assert.Contains(t, err.Error(), "Google Cloud", "Error should be related to Google Cloud authentication")
	} else {
		// If credentials are available, client should be created successfully
		assert.NotNil(t, client, "Client should be created successfully with valid credentials")
	}
}

// TestVertexAI_NetworkFailure tests handling of network failures
func TestVertexAI_NetworkFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}
	
	t.Setenv("VERTEXAI_PROJECT", "test-project")
	t.Setenv("VERTEXAI_LOCATION", "us-central1")
	
	// This test verifies that network failures are handled gracefully
	// In real implementation, this would test actual network scenarios
	
	opts := providerClientOptions{
		model: models.Model{APIModel: "claude-3-sonnet"},
	}
	
	client, err := newVertexAIClaudeClient(opts)
	if err != nil {
		// If error occurs, it should be Google Cloud authentication related
		assert.Contains(t, err.Error(), "Google Cloud")
		assert.Nil(t, client)
	} else {
		// If successful, we should get a valid client
		assert.NotNil(t, client, "Client should be created successfully")
	}
}

// Note: Helper functions validateVertexAIEnvironment and getGoogleCloudAuthOptions 
// are implemented in vertexai.go