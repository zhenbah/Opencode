package provider

import (
	"context"
	"strings"

	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/opencode-ai/opencode/internal/llm/tools"
	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/message"
)

// ReasoningConfig holds configuration for reasoning requests
type ReasoningConfig struct {
	Model           string
	ReasoningEffort string
	MaxTokens       int
	Tools           []tools.BaseTool
}

// ReasoningHandler manages XAI reasoning content processing
type ReasoningHandler struct {
	client *xaiClient
}

// NewReasoningHandler creates a new reasoning handler
func NewReasoningHandler(client *xaiClient) *ReasoningHandler {
	return &ReasoningHandler{
		client: client,
	}
}

// ShouldUseReasoning determines if reasoning should be used for a request
func (r *ReasoningHandler) ShouldUseReasoning() bool {
	canReason := r.client.providerOptions.model.CanReason
	hasReasoningEffort := r.client.options.reasoningEffort != ""
	shouldApply := r.client.shouldApplyReasoningEffort()

	logging.Debug("Checking reasoning conditions",
		"model", r.client.providerOptions.model.ID,
		"can_reason", canReason,
		"reasoning_effort", r.client.options.reasoningEffort,
		"has_reasoning_effort", hasReasoningEffort,
		"should_apply", shouldApply)

	return canReason && hasReasoningEffort && shouldApply
}

// ProcessReasoningResponse handles reasoning content from API responses
func (r *ReasoningHandler) ProcessReasoningResponse(response *ProviderResponse) []ProviderEvent {
	var events []ProviderEvent

	// Send reasoning content as thinking delta first (if present)
	if response.ReasoningContent != "" {
		sanitizedReasoning := r.sanitizeReasoningContent(response.ReasoningContent)
		events = append(events, ProviderEvent{
			Type:     EventThinkingDelta,
			Thinking: sanitizedReasoning,
		})

		logging.Debug("Reasoning content processed",
			"original_length", len(response.ReasoningContent),
			"sanitized_length", len(sanitizedReasoning))
	}

	// Send regular content as delta if present
	if response.Content != "" {
		events = append(events, ProviderEvent{
			Type:    EventContentDelta,
			Content: response.Content,
		})
	}

	// Clear reasoning content from response before sending complete event
	response.ReasoningContent = ""

	// Send complete event
	events = append(events, ProviderEvent{
		Type:     EventComplete,
		Response: response,
	})

	return events
}

// sanitizeReasoningContent removes control characters that could corrupt terminal display
func (r *ReasoningHandler) sanitizeReasoningContent(content string) string {
	// Remove ANSI escape sequences (ESC character)
	content = strings.ReplaceAll(content, "\x1b", "")
	// Remove carriage returns (which can cause display issues)
	content = strings.ReplaceAll(content, "\r", "")
	// Remove other control characters that might cause issues
	content = strings.ReplaceAll(content, "\x00", "") // null
	content = strings.ReplaceAll(content, "\x07", "") // bell
	content = strings.ReplaceAll(content, "\x08", "") // backspace
	// Replace form feed with newline to preserve structure
	content = strings.ReplaceAll(content, "\x0c", "\n")
	return content
}

// BuildReasoningRequest creates a request body for reasoning models
func (r *ReasoningHandler) BuildReasoningRequest(ctx context.Context, messages []message.Message, tools []tools.BaseTool) map[string]interface{} {
	reqBody := map[string]interface{}{
		"model":      r.client.providerOptions.model.APIModel,
		"messages":   r.client.convertMessagesToAPI(messages),
		"max_tokens": r.client.providerOptions.maxTokens,
		"stream":     false, // Explicitly disable streaming for reasoning requests
	}

	// Add tools if provided
	if len(tools) > 0 {
		reqBody["tools"] = r.client.convertToolsToAPI(tools)
	}

	// Apply reasoning effort only if the model supports it
	// xAI grok models do not accept reasoning_effort parameter
	if r.client.options.reasoningEffort != "" && r.client.shouldApplyReasoningEffort() {
		reasoningEffort := r.client.options.reasoningEffort
		
		// Grok-3-mini models only support "low" or "high", not "medium"
		if (r.client.providerOptions.model.ID == models.XAIGrok3Mini || 
		    r.client.providerOptions.model.ID == models.XAIGrok3MiniFast) && 
		    reasoningEffort == "medium" {
			// Convert medium to high for Grok-3-mini models
			reasoningEffort = "high"
			logging.Debug("Converting reasoning effort from medium to high for Grok-3-mini")
		}
		
		reqBody["reasoning_effort"] = reasoningEffort
	}

	// Apply response format if configured
	if r.client.options.responseFormat != nil {
		reqBody["response_format"] = r.client.options.responseFormat
	}

	// Apply tool choice if configured
	if r.client.options.toolChoice != nil {
		reqBody["tool_choice"] = r.client.options.toolChoice
	}

	// Apply parallel tool calls if configured
	if r.client.options.parallelToolCalls != nil {
		reqBody["parallel_tool_calls"] = r.client.options.parallelToolCalls
	}

	return reqBody
}
