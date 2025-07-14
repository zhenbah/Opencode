package models

const (
	ProviderVertexAI ModelProvider = "vertexai"

	// Gemini Models
	VertexAIGemini25Flash ModelID = "vertexai.gemini-2.5-flash"
	VertexAIGemini25      ModelID = "vertexai.gemini-2.5"

	// Claude Models
	VertexAIClaude4Sonnet ModelID = "vertexai.claude-sonnet-4"
	VertexAIClaude4Opus   ModelID = "vertexai.claude-opus-4"
)

var VertexAIModels = map[ModelID]Model{
	VertexAIGemini25Flash: {
		ID:                  VertexAIGemini25Flash,
		Name:                "VertexAI: Gemini 2.5 Flash",
		Provider:            ProviderVertexAI,
		APIModel:            "gemini-2.5-flash-preview-04-17",
		CostPer1MIn:         GeminiModels[Gemini25Flash].CostPer1MIn,
		CostPer1MInCached:   GeminiModels[Gemini25Flash].CostPer1MInCached,
		CostPer1MOut:        GeminiModels[Gemini25Flash].CostPer1MOut,
		CostPer1MOutCached:  GeminiModels[Gemini25Flash].CostPer1MOutCached,
		ContextWindow:       GeminiModels[Gemini25Flash].ContextWindow,
		DefaultMaxTokens:    GeminiModels[Gemini25Flash].DefaultMaxTokens,
		SupportsAttachments: true,
	},
	VertexAIGemini25: {
		ID:                  VertexAIGemini25,
		Name:                "VertexAI: Gemini 2.5 Pro",
		Provider:            ProviderVertexAI,
		APIModel:            "gemini-2.5-pro-preview-03-25",
		CostPer1MIn:         GeminiModels[Gemini25].CostPer1MIn,
		CostPer1MInCached:   GeminiModels[Gemini25].CostPer1MInCached,
		CostPer1MOut:        GeminiModels[Gemini25].CostPer1MOut,
		CostPer1MOutCached:  GeminiModels[Gemini25].CostPer1MOutCached,
		ContextWindow:       GeminiModels[Gemini25].ContextWindow,
		DefaultMaxTokens:    GeminiModels[Gemini25].DefaultMaxTokens,
		SupportsAttachments: true,
	},
	VertexAIClaude4Sonnet: {
		ID:                  VertexAIClaude4Sonnet,
		Name:                "VertexAI: Claude Sonnet 4",
		Provider:            ProviderVertexAI,
		APIModel:            "claude-sonnet-4",
		CostPer1MIn:         AnthropicModels[Claude4Sonnet].CostPer1MIn,
		CostPer1MInCached:   AnthropicModels[Claude4Sonnet].CostPer1MInCached,
		CostPer1MOut:        AnthropicModels[Claude4Sonnet].CostPer1MOut,
		CostPer1MOutCached:  AnthropicModels[Claude4Sonnet].CostPer1MOutCached,
		ContextWindow:       AnthropicModels[Claude4Sonnet].ContextWindow,
		DefaultMaxTokens:    AnthropicModels[Claude4Sonnet].DefaultMaxTokens,
		CanReason:           AnthropicModels[Claude4Sonnet].CanReason,
		SupportsAttachments: AnthropicModels[Claude4Sonnet].SupportsAttachments,
	},
	VertexAIClaude4Opus: {
		ID:                  VertexAIClaude4Opus,
		Name:                "VertexAI: Claude Opus 4",
		Provider:            ProviderVertexAI,
		APIModel:            "claude-opus-4",
		CostPer1MIn:         AnthropicModels[Claude4Opus].CostPer1MIn,
		CostPer1MInCached:   AnthropicModels[Claude4Opus].CostPer1MInCached,
		CostPer1MOut:        AnthropicModels[Claude4Opus].CostPer1MOut,
		CostPer1MOutCached:  AnthropicModels[Claude4Opus].CostPer1MOutCached,
		ContextWindow:       AnthropicModels[Claude4Opus].ContextWindow,
		DefaultMaxTokens:    AnthropicModels[Claude4Opus].DefaultMaxTokens,
		CanReason:           AnthropicModels[Claude4Opus].CanReason,
		SupportsAttachments: AnthropicModels[Claude4Opus].SupportsAttachments,
	},
}
