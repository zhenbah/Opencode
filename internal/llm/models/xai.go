package models

const (
	ProviderXAI ModelProvider = "xai"

	XAIGrok3Beta         ModelID = "grok-3-beta"
	XAIGrok3MiniBeta     ModelID = "grok-3-mini-beta"
	XAIGrok3FastBeta     ModelID = "grok-3-fast-beta"
	XAiGrok3MiniFastBeta ModelID = "grok-3-mini-fast-beta"
	XAIGrok40709         ModelID = "grok-4-0709"
)

var XAIModels = map[ModelID]Model{
	XAIGrok3Beta: {
		ID:                 XAIGrok3Beta,
		Name:               "Grok3 Beta",
		Provider:           ProviderXAI,
		APIModel:           "grok-3-beta",
		CostPer1MIn:        3.0,
		CostPer1MInCached:  0,
		CostPer1MOut:       15,
		CostPer1MOutCached: 0,
		ContextWindow:      131_072,
		DefaultMaxTokens:   20_000,
	},
	XAIGrok3MiniBeta: {
		ID:                 XAIGrok3MiniBeta,
		Name:               "Grok3 Mini Beta",
		Provider:           ProviderXAI,
		APIModel:           "grok-3-mini-beta",
		CostPer1MIn:        0.3,
		CostPer1MInCached:  0,
		CostPer1MOut:       0.5,
		CostPer1MOutCached: 0,
		ContextWindow:      131_072,
		DefaultMaxTokens:   20_000,
	},
	XAIGrok3FastBeta: {
		ID:                 XAIGrok3FastBeta,
		Name:               "Grok3 Fast Beta",
		Provider:           ProviderXAI,
		APIModel:           "grok-3-fast-beta",
		CostPer1MIn:        5,
		CostPer1MInCached:  0,
		CostPer1MOut:       25,
		CostPer1MOutCached: 0,
		ContextWindow:      131_072,
		DefaultMaxTokens:   20_000,
	},
	XAiGrok3MiniFastBeta: {
		ID:                 XAiGrok3MiniFastBeta,
		Name:               "Grok3 Mini Fast Beta",
		Provider:           ProviderXAI,
		APIModel:           "grok-3-mini-fast-beta",
		CostPer1MIn:        0.6,
		CostPer1MInCached:  0,
		CostPer1MOut:       4.0,
		CostPer1MOutCached: 0,
		ContextWindow:      131_072,
		DefaultMaxTokens:   20_000,
	},
	XAIGrok40709: {
		ID:                 XAIGrok40709,
		Name:               "Grok 4 0709",
		Provider:           ProviderXAI,
		APIModel:           "grok-4-0709",
		CostPer1MIn:        3.0,
		CostPer1MInCached:  0.75,
		CostPer1MOut:       15.0,
		CostPer1MOutCached: 0.75,
		ContextWindow:      256_000,
		DefaultMaxTokens:   20_000,
		CanReason:          true,
		SupportsAttachments: true,
	},
}
