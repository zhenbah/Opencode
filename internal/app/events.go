package app

// Import orchestrator here, assuming WorkerResult will be referenced.
// This might create a circular dependency if orchestrator also imports app for AppEvent.
// If so, WorkerResult might need to be defined in a common package, or AppEvent moved.
// For now, proceed and address if it becomes an issue during compilation.
import "github.com/opencode-ai/opencode/internal/orchestrator"

const (
	EventTypeUserInputCommand   = "UserInputCommand"
	EventTypeWorkerCompletion = "WorkerCompletion"
	EventTypeLogMessage         = "LogMessage" // Example for future use
	// Add more event types later (e.g., OrchestratorAgentResult, OrchestratorAgentError)
)

type AppEvent struct {
	Type string
	Data interface{}
}

type UserInputCommandData struct {
	Command string
}

type WorkerCompletionData struct {
	WorkerID string
	Result   orchestrator.WorkerResult // Re-use existing WorkerResult
}

type LogMessageData struct {
	Level   string // e.g., "INFO", "ERROR"
	Message string
	Fields  map[string]interface{}
}
