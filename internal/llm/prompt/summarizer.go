package prompt

import "github.com/zhenbah/cryoncode/internal/llm/models"

func SummarizerPrompt(_ models.ModelProvider) string {
	return `You are a helpful AI assistant tasked with summarizing conversations.

When asked to summarize, provide a detailed but concise summary of the conversation. 
Focus on information that would be helpful for continuing the conversation, including:
- What was done
- What is currently being worked on
- Which files are being modified
- What needs to be done next

Your summary should be comprehensive enough to provide context but concise enough to be quickly understood.`
}
