package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/db"
	"github.com/opencode-ai/opencode/internal/format"
	"github.com/opencode-ai/opencode/internal/history"
	"github.com/opencode-ai/opencode/internal/llm/agent"
	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/lsp"
	"github.com/opencode-ai/opencode/internal/message"
	"github.com/opencode-ai/opencode/internal/permission"
	"github.com/opencode-ai/opencode/internal/session"
	"github.com/opencode-ai/opencode/internal/tui/theme"
	"bytes" // Added for HTTP client body
	"io"    // Added for reading response body
	"net/http"      // Added for HTTP client
	"github.com/opencode-ai/opencode/internal/orchestrator" // Added for orchestrator
	"encoding/json"                                        // Added for worker mode
	"os"                                                   // Added for worker mode
)

type App struct {
	Sessions    session.Service
	Messages    message.Service
	History     history.Service
	Permissions permission.Service

	CoderAgent agent.Service

	LSPClients map[string]*lsp.Client

	clientsMutex sync.RWMutex

	watcherCancelFuncs []context.CancelFunc
	cancelFuncsMutex   sync.Mutex
	watcherWG          sync.WaitGroup

	IsWorkerMode bool                   // Added to indicate worker mode
	Orchestrator orchestrator.Service // Added for orchestrator
	eventQueue   chan AppEvent        // Added for event system
}

func New(ctx context.Context, conn *sql.DB, isWorker bool) (*App, error) { // Modified to accept isWorker
	q := db.New(conn)
	sessions := session.NewService(q)
	messages := message.NewService(q)
	files := history.NewService(q, conn)

	app := &App{
		Sessions:    sessions,
		Messages:    messages,
		History:     files,
		Permissions: permission.NewPermissionService(),
		LSPClients:  make(map[string]*lsp.Client),
		IsWorkerMode: isWorker, // Set IsWorkerMode
	}

	if !isWorker {
		app.eventQueue = make(chan AppEvent, 50) // Initialize event queue for orchestrator mode
	}

	// Initialize theme based on configuration
	// Theme is not needed for worker mode
	if !app.IsWorkerMode {
		app.initTheme()
	}

	// Initialize LSP clients in the background, only if not in worker mode
	if !app.IsWorkerMode {
		go app.initLSPClients(ctx)
	}

	var err error
	app.CoderAgent, err = agent.NewAgent(
		config.AgentCoder,
		app.Sessions,
		app.Messages,
		agent.CoderAgentTools(
			app.Permissions,
			app.Sessions,
			app.Messages,
			app.History,
			app.LSPClients,
		),
	)
	if err != nil {
		logging.Error("Failed to create coder agent", err)
		return nil, err
	}

	// Initialize Orchestrator service if not in worker mode
	if !isWorker {
		exePath, err := os.Executable()
		if err != nil {
			// Handle error or decide on a fallback path. For now, log and potentially continue without orchestrator.
			logging.Error("Failed to get executable path for orchestrator", "error", err)
			// Depending on requirements, you might return an error here:
			// return nil, fmt.Errorf("failed to initialize orchestrator: could not get executable path: %w", err)
		} else {
			// Pass eventQueue to Orchestrator service
			app.Orchestrator = orchestrator.NewService(exePath, app.eventQueue)
			logging.Info("Orchestrator service initialized", "executablePath", exePath)
			// Start the orchestrator's API server
			if err := app.Orchestrator.StartAPIServer(); err != nil {
				// This is a non-fatal error for now, allows opencode to run without orchestrator API
				logging.Error("Failed to start orchestrator API server", "error", err)
			}
		}
	}

	return app, nil
}

// initTheme sets the application theme based on the configuration
func (app *App) initTheme() {
	cfg := config.Get()
	if cfg == nil || cfg.TUI.Theme == "" {
		return // Use default theme
	}

	// Try to set the theme from config
	err := theme.SetTheme(cfg.TUI.Theme)
	if err != nil {
		logging.Warn("Failed to set theme from config, using default theme", "theme", cfg.TUI.Theme, "error", err)
	} else {
		logging.Debug("Set theme from config", "theme", cfg.TUI.Theme)
	}
}

// RunNonInteractive handles the execution flow when a prompt is provided via CLI flag.
func (a *App) RunNonInteractive(ctx context.Context, prompt string, outputFormat string, quiet bool) error {
	logging.Info("Running in non-interactive mode")

	// Start spinner if not in quiet mode
	var spinner *format.Spinner
	if !quiet {
		spinner = format.NewSpinner("Thinking...")
		spinner.Start()
		defer spinner.Stop()
	}

	const maxPromptLengthForTitle = 100
	titlePrefix := "Non-interactive: "
	var titleSuffix string

	if len(prompt) > maxPromptLengthForTitle {
		titleSuffix = prompt[:maxPromptLengthForTitle] + "..."
	} else {
		titleSuffix = prompt
	}
	title := titlePrefix + titleSuffix

	sess, err := a.Sessions.Create(ctx, title)
	if err != nil {
		return fmt.Errorf("failed to create session for non-interactive mode: %w", err)
	}
	logging.Info("Created session for non-interactive run", "session_id", sess.ID)

	// Automatically approve all permission requests for this non-interactive session
	a.Permissions.AutoApproveSession(sess.ID)

	done, err := a.CoderAgent.Run(ctx, sess.ID, prompt)
	if err != nil {
		return fmt.Errorf("failed to start agent processing stream: %w", err)
	}

	result := <-done
	if result.Error != nil {
		if errors.Is(result.Error, context.Canceled) || errors.Is(result.Error, agent.ErrRequestCancelled) {
			logging.Info("Agent processing cancelled", "session_id", sess.ID)
			return nil
		}
		return fmt.Errorf("agent processing failed: %w", result.Error)
	}

	// Stop spinner before printing output
	if !quiet && spinner != nil {
		spinner.Stop()
	}

	// Get the text content from the response
	content := "No content available"
	if result.Message.Content().String() != "" {
		content = result.Message.Content().String()
	}

	fmt.Println(format.FormatOutput(content, outputFormat))

	logging.Info("Non-interactive run completed", "session_id", sess.ID)

	return nil
}

// Shutdown performs a clean shutdown of the application
func (app *App) Shutdown() {
	// Stop Orchestrator API server first if it exists
	if app.Orchestrator != nil {
		// Create a context for shutdown, e.g., with a timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := app.Orchestrator.StopAPIServer(shutdownCtx); err != nil {
			logging.Error("Failed to stop orchestrator API server", "error", err)
		} else {
			logging.Info("Orchestrator API server stopped")
		}
	}

	// Cancel all watcher goroutines
	app.cancelFuncsMutex.Lock()
	for _, cancel := range app.watcherCancelFuncs {
		cancel()
	}
	app.cancelFuncsMutex.Unlock()
	app.watcherWG.Wait()

	// Perform additional cleanup for LSP clients
	app.clientsMutex.RLock()
	clients := make(map[string]*lsp.Client, len(app.LSPClients))
	maps.Copy(clients, app.LSPClients)
	app.clientsMutex.RUnlock()

	for name, client := range clients {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := client.Shutdown(shutdownCtx); err != nil {
			logging.Error("Failed to shutdown LSP client", "name", name, "error", err)
		}
		cancel()
	}
}

// initLSPClients initializes LSP clients for different languages.
// This function is intended to be run as a goroutine.
// In worker mode, this function should be a no-op.
func (app *App) initLSPClients(ctx context.Context) {
	if app.IsWorkerMode {
		logging.Debug("Skipping LSP client initialization in worker mode")
		return
	}

	// Get the languages from the configuration
	cfg := config.Get()
	if cfg == nil || len(cfg.LSP.Languages) == 0 {
		logging.Debug("No LSP languages configured, skipping LSP client initialization")
		return
	}

	logging.Debug("Initializing LSP clients for languages", "languages", cfg.LSP.Languages)

	var wg sync.WaitGroup
	for _, lang := range cfg.LSP.Languages {
		wg.Add(1)
		go func(language string) {
			defer wg.Done()
			client, err := lsp.NewClient(ctx, language, config.WorkingDirectory())
			if err != nil {
				logging.Error("Failed to initialize LSP client", "language", language, "error", err)
				return
			}
			app.clientsMutex.Lock()
			app.LSPClients[language] = client
			app.clientsMutex.Unlock()
			logging.Info("LSP client initialized", "language", language)
		}(lang)
	}
	wg.Wait()
	logging.Debug("All LSP clients initialized")
}

// RunWorkerMode runs the application in worker mode.
// Now accepts taskID from CLI.
func (a *App) RunWorkerMode(ctx context.Context, taskFilePath string, agentID string, orchestratorAPI string, taskID string) error {
	logging.Info("Starting OpenCode in worker mode", "agentID", agentID, "taskFile", taskFilePath, "orchestratorAPI", orchestratorAPI, "taskID", taskID)

	// Define a struct for the output - now includes TaskID
	type WorkerOutput struct {
		AgentID         string `json:"agent_id"`
		TaskID          string `json:"task_id"` // Added TaskID
		TaskFilePath    string `json:"task_file_path"`
		OrchestratorAPI string `json:"orchestrator_api"`
		Status          string `json:"status"`
		Result          string `json:"result,omitempty"`
		Error           string `json:"error,omitempty"`
	}

	output := WorkerOutput{
		AgentID:         agentID,
		TaskID:          taskID, // Set TaskID from parameter
		TaskFilePath:    taskFilePath,
		OrchestratorAPI: orchestratorAPI,
	}

	if taskFilePath == "" {
		err := errors.New("task file path is required for worker mode")
		logging.Error("Worker mode initialization error", "error", err)
		output.Status = "failed"
		output.Error = err.Error()
		// Marshal and print to stdout even on early exit
		jsonData, _ := json.Marshal(output)
		fmt.Println(string(jsonData))
		return err
	}

	if agentID == "" {
		err := errors.New("agent ID is required for worker mode")
		logging.Error("Worker mode initialization error", "error", err)
		output.Status = "failed"
		output.Error = err.Error()
		jsonData, _ := json.Marshal(output)
		fmt.Println(string(jsonData))
		return err
	}

	// Read task file
	taskData, err := os.ReadFile(taskFilePath)
	if err != nil {
		logging.Error("Failed to read task file", "path", taskFilePath, "error", err)
		output.Status = "failed"
		output.Error = fmt.Sprintf("failed to read task file: %v", err)
		jsonData, _ := json.Marshal(output)
		fmt.Println(string(jsonData))
		return err
	}

	// Task file now contains both task_prompt and task_id
	var taskFileContent struct {
		TaskPrompt string `json:"task_prompt"`
		TaskID     string `json:"task_id"` // Expect task_id from file as well
	}
	if err := json.Unmarshal(taskData, &taskFileContent); err != nil {
		logging.Error("Failed to parse task file JSON", "path", taskFilePath, "error", err)
		output.Status = "failed"
		output.Error = fmt.Sprintf("failed to parse task file JSON: %v", err)
		jsonData, _ := json.Marshal(output) // Use existing output struct
		fmt.Println(string(jsonData))
		return err
	}

	// Ensure the taskID from CLI matches the one in the file, or prioritize one.
	// For now, let's use the taskID from the CLI if provided, otherwise from the file.
	// The output.TaskID is already set from the CLI parameter.
	// If CLI taskID is empty, we might want to use taskFileContent.TaskID.
	// However, the orchestrator is expected to provide it via CLI.
	// We also need to ensure the output.TaskID is correctly set for the final JSON.
	// The `taskID` parameter to this function is the one passed via CLI.
	// The `taskFileContent.TaskID` is from the JSON file.
	// The `output.TaskID` should be the definitive one for the output JSON.
	// Let's ensure output.TaskID is what we want. It's already set from the CLI taskID parameter.
	// If for some reason the CLI taskID was empty but the file one wasn't, we could use it:
	if output.TaskID == "" && taskFileContent.TaskID != "" {
		output.TaskID = taskFileContent.TaskID
		logging.Warn("TaskID from CLI was empty, using TaskID from task file", "taskID", output.TaskID)
	}


	if taskFileContent.TaskPrompt == "" {
		err := errors.New("task_prompt not found or empty in task file")
		logging.Error("Worker mode task definition error", "error", err)
		output.Status = "failed"
		output.Error = err.Error()
		jsonData, _ := json.Marshal(output)
		fmt.Println(string(jsonData))
		return err
	}

	// Auto-approve permissions for this agent
	logging.Info("Auto-approving permissions for agent", "agentID", agentID)
	a.Permissions.AutoApproveAgent(agentID) // Conceptual call

	logging.Info("Executing task for agent", "agentID", agentID, "taskID", output.TaskID, "prompt", taskFileContent.TaskPrompt)
	// Using agentID as sessionID for now
	done, err := a.CoderAgent.Run(ctx, agentID, taskFileContent.TaskPrompt) // Use prompt from parsed file
	if err != nil {
		logging.Error("Failed to start agent processing for worker", "agentID", agentID, "taskID", output.TaskID, "error", err)
		output.Status = "failed"
		output.Error = fmt.Sprintf("failed to start agent processing: %v", err)
		jsonData, _ := json.Marshal(output) // Use existing output struct
		fmt.Println(string(jsonData))
		return err
	}

	result := <-done
	if result.Error != nil {
		if errors.Is(result.Error, context.Canceled) || errors.Is(result.Error, agent.ErrRequestCancelled) {
			logging.Info("Agent processing cancelled for worker", "agentID", agentID)
			output.Status = "cancelled"
			output.Error = result.Error.Error()
		} else {
			logging.Error("Agent processing failed for worker", "agentID", agentID, "error", result.Error)
			output.Status = "failed"
			output.Error = result.Error.Error()
		}
	} else {
		logging.Info("Agent processing completed for worker", "agentID", agentID)
		output.Status = "completed"
		if result.Message != nil && result.Message.Content() != nil {
			output.Result = result.Message.Content().String()
		} else {
			output.Result = "No content in result message."
		}
	}

	// Attempt to report result via API
	jsonData, jsonErr := json.MarshalIndent(output, "", "  ")
	if jsonErr != nil {
		logging.Error("Worker failed to marshal output for API/stdout", "agentID", agentID, "taskID", output.TaskID, "error", jsonErr)
		// Cannot easily report this error via API if marshalling itself failed.
		// Stdout becomes critical here.
		fmt.Printf("{\"agent_id\": \"%s\", \"task_id\": \"%s\", \"status\": \"failed\", \"error\": \"failed to marshal output: %v\"}\n", agentID, output.TaskID, jsonErr)
		return jsonErr // Return the marshalling error
	}

	if orchestratorAPI != "" {
		reqBody := bytes.NewBuffer(jsonData)
		resp, httpErr := http.Post(orchestratorAPI, "application/json", reqBody)
		if httpErr != nil {
			logging.Error("Worker failed to report result via API", "agentID", agentID, "taskID", output.TaskID, "api_url", orchestratorAPI, "error", httpErr)
			// Fallback to stdout
			fmt.Println(string(jsonData))
		} else {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				logging.Info("Worker successfully reported result via API", "agentID", agentID, "taskID", output.TaskID, "api_url", orchestratorAPI)
				// Optionally print a minimal message to stdout indicating API success
				// fmt.Printf("{\"agent_id\": \"%s\", \"task_id\": \"%s\", \"status\": \"reported_via_api\"}\n", agentID, output.TaskID)
			} else {
				logging.Error("Orchestrator API returned non-OK status", "agentID", agentID, "taskID", output.TaskID, "api_url", orchestratorAPI, "status_code", resp.StatusCode)
				// Fallback to stdout
				bodyBytes, _ := io.ReadAll(resp.Body)
				logging.Error("Orchestrator API response body", "body", string(bodyBytes))
				fmt.Println(string(jsonData))
			}
		}
	} else {
		logging.Warn("Orchestrator API URL not provided, printing result to stdout", "agentID", agentID, "taskID", output.TaskID)
		// Fallback to stdout if no API URL
		fmt.Println(string(jsonData))
	}

	if result.Error != nil {
		return result.Error // Return the original agent error
	}

	return nil
}
