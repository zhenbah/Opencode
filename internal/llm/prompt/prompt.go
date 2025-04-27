package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/llm/models"
)

func GetAgentPrompt(agentName config.AgentName, provider models.ModelProvider) string {
	basePrompt := ""
	switch agentName {
	case config.AgentCoder:
		basePrompt = CoderPrompt(provider)
	case config.AgentTitle:
		basePrompt = TitlePrompt(provider)
	case config.AgentTask:
		basePrompt = TaskPrompt(provider)
	default:
		basePrompt = "You are a helpful assistant"
	}

	if agentName == config.AgentCoder || agentName == config.AgentTask {
		// Add context from project-specific instruction files if they exist
		contextContent := getContextFromFiles()
		if contextContent != "" {
			return fmt.Sprintf("%s\n\n# Project-Specific Context\n%s", basePrompt, contextContent)
		}
	}
	return basePrompt
}

// getContextFromFiles checks for the existence of context files and returns their content
func getContextFromFiles() string {
	var (
		cfg            = config.Get()
		workDir        = cfg.WorkingDir
		contextFiles   = cfg.ContextFiles
		contextContent strings.Builder
	)

	for _, file := range contextFiles {
		filePath := filepath.Join(workDir, file)
		content, err := os.ReadFile(filePath)
		if err == nil {
			contextContent.WriteRune('\n')
			contextContent.WriteString(string(content))
			contextContent.WriteRune('\n')
		}
	}

	return contextContent.String()
}
