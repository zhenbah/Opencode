package models

import (
	"strings"
	"testing"
)

func TestVertexAIClaudeModels(t *testing.T) {
	// Test that Claude Sonnet 4 model is correctly defined
	claude4Sonnet, exists := SupportedModels[VertexAIClaude4Sonnet]
	if !exists {
		t.Errorf("VertexAI Claude Sonnet 4 model not found in SupportedModels")
		return
	}

	// Verify model properties
	if claude4Sonnet.ID != VertexAIClaude4Sonnet {
		t.Errorf("Expected ID %s, got %s", VertexAIClaude4Sonnet, claude4Sonnet.ID)
	}
	if claude4Sonnet.Name != "VertexAI: Claude Sonnet 4" {
		t.Errorf("Expected name 'VertexAI: Claude Sonnet 4', got %s", claude4Sonnet.Name)
	}
	if claude4Sonnet.Provider != ProviderVertexAI {
		t.Errorf("Expected provider %s, got %s", ProviderVertexAI, claude4Sonnet.Provider)
	}
	if claude4Sonnet.APIModel != "claude-sonnet-4" {
		t.Errorf("Expected API model 'claude-sonnet-4', got %s", claude4Sonnet.APIModel)
	}
	if !claude4Sonnet.CanReason {
		t.Errorf("Expected Claude Sonnet 4 to support reasoning")
	}
	if !claude4Sonnet.SupportsAttachments {
		t.Errorf("Expected Claude Sonnet 4 to support attachments")
	}

	// Test that Claude Opus 4 model is correctly defined
	claude4Opus, exists := SupportedModels[VertexAIClaude4Opus]
	if !exists {
		t.Errorf("VertexAI Claude Opus 4 model not found in SupportedModels")
		return
	}

	// Verify model properties
	if claude4Opus.ID != VertexAIClaude4Opus {
		t.Errorf("Expected ID %s, got %s", VertexAIClaude4Opus, claude4Opus.ID)
	}
	if claude4Opus.Name != "VertexAI: Claude Opus 4" {
		t.Errorf("Expected name 'VertexAI: Claude Opus 4', got %s", claude4Opus.Name)
	}
	if claude4Opus.Provider != ProviderVertexAI {
		t.Errorf("Expected provider %s, got %s", ProviderVertexAI, claude4Opus.Provider)
	}
	if claude4Opus.APIModel != "claude-opus-4" {
		t.Errorf("Expected API model 'claude-opus-4', got %s", claude4Opus.APIModel)
	}
	if !claude4Opus.SupportsAttachments {
		t.Errorf("Expected Claude Opus 4 to support attachments")
	}

	// Check reasoning capability - should match the Anthropic model
	anthropicOpusModel := AnthropicModels[Claude4Opus]
	if claude4Opus.CanReason != anthropicOpusModel.CanReason {
		t.Errorf("Expected CanReason to match Anthropic model: %v, got %v", anthropicOpusModel.CanReason, claude4Opus.CanReason)
	}

	// Test that pricing is inherited correctly from Anthropic models
	anthropicSonnet := AnthropicModels[Claude4Sonnet]
	if claude4Sonnet.CostPer1MIn != anthropicSonnet.CostPer1MIn {
		t.Errorf("Expected inherited input cost %f, got %f", anthropicSonnet.CostPer1MIn, claude4Sonnet.CostPer1MIn)
	}
	if claude4Sonnet.ContextWindow != anthropicSonnet.ContextWindow {
		t.Errorf("Expected inherited context window %d, got %d", anthropicSonnet.ContextWindow, claude4Sonnet.ContextWindow)
	}

	anthropicOpus := AnthropicModels[Claude4Opus]
	if claude4Opus.CostPer1MIn != anthropicOpus.CostPer1MIn {
		t.Errorf("Expected inherited input cost %f, got %f", anthropicOpus.CostPer1MIn, claude4Opus.CostPer1MIn)
	}
	if claude4Opus.ContextWindow != anthropicOpus.ContextWindow {
		t.Errorf("Expected inherited context window %d, got %d", anthropicOpus.ContextWindow, claude4Opus.ContextWindow)
	}
}

func TestVertexAIProviderPriority(t *testing.T) {
	// Test that VertexAI provider is included in the popularity rankings
	priority, exists := ProviderPopularity[ProviderVertexAI]
	if !exists {
		t.Errorf("VertexAI provider not found in ProviderPopularity")
		return
	}
	
	// VertexAI should have a reasonable priority (not 0)
	if priority <= 0 {
		t.Errorf("Expected positive priority for VertexAI provider, got %d", priority)
	}
}

// Test model routing for all defined models
func TestVertexAI_AllModelRouting(t *testing.T) {
	claudeModels := []ModelID{
		VertexAIClaude4Sonnet,
		VertexAIClaude4Opus,
	}

	geminiModels := []ModelID{
		VertexAIGemini25Flash,
		VertexAIGemini25,
	}

	// Test Claude models route correctly
	for _, modelID := range claudeModels {
		t.Run(string(modelID), func(t *testing.T) {
			model := SupportedModels[modelID]
			if !strings.HasPrefix(model.APIModel, "claude-") {
				t.Errorf("Claude model %s should have 'claude-' prefix, got %s", modelID, model.APIModel)
			}
		})
	}

	// Test Gemini models route correctly  
	for _, modelID := range geminiModels {
		t.Run(string(modelID), func(t *testing.T) {
			model := SupportedModels[modelID]
			if strings.HasPrefix(model.APIModel, "claude-") {
				t.Errorf("Gemini model %s should not have 'claude-' prefix, got %s", modelID, model.APIModel)
			}
		})
	}
}

// Test model definitions for required fields
func TestVertexAI_ClaudeModelDefinitions(t *testing.T) {
	claudeModels := []ModelID{
		VertexAIClaude4Sonnet,
		VertexAIClaude4Opus,
	}

	for _, modelID := range claudeModels {
		t.Run(string(modelID), func(t *testing.T) {
			model := SupportedModels[modelID]

			// Verify required fields
			if model.APIModel == "" {
				t.Errorf("API model should not be empty")
			}
			if model.Name == "" {
				t.Errorf("Display name should not be empty")
			}
			if model.ContextWindow <= 0 {
				t.Errorf("Context window should be positive, got %d", model.ContextWindow)
			}
			if model.DefaultMaxTokens <= 0 {
				t.Errorf("Max output tokens should be positive, got %d", model.DefaultMaxTokens)
			}

			// Verify Claude-specific requirements
			if !model.SupportsAttachments {
				t.Errorf("Claude models should support attachments")
			}
		})
	}
}