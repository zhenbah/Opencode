package app

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"encoding/json" // Added for tool call pretty print

	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/pb"
	"github.com/opencode-ai/opencode/internal/orchestrator" // For TaskDefinition (if /spawn implemented here)
	"github.com/google/uuid"                             // For TaskID generation in /spawn
)

const OrchestratorSessionID = "orchestrator_session"

func (a *App) RunOrchestratorCLI(ctx context.Context) error {
	fmt.Println("OpenCode Orchestrator CLI. Type '/quit' to exit, '/spawn <prompt>' to spawn a worker, '/result <workerID>' to check result.")

	// Auto-approve permissions for the orchestrator session
	if a.Permissions != nil {
		a.Permissions.AutoApproveSession(OrchestratorSessionID)
		logging.Info("Permissions auto-approved for orchestrator session", "sessionID", OrchestratorSessionID)
	}

	// Context for managing the input goroutine
	inputCtx, cancelInput := context.WithCancel(ctx)
	defer cancelInput() // Ensure input goroutine is signalled to stop on exit

	// Goroutine for reading user input
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			select {
			case <-inputCtx.Done(): // Check if context was cancelled
				logging.Debug("Input goroutine: context done, exiting.")
				return
			default:
				// Prompt needs to be printed by the main loop to avoid interleaving with event messages.
				// This goroutine just reads.
				input, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						logging.Info("Input goroutine: EOF received, sending quit command.")
						a.eventQueue <- AppEvent{Type: EventTypeUserInputCommand, Data: UserInputCommandData{Command: "/quit"}}
						return
					}
					logging.Error("Input goroutine: Error reading input", "error", err)
					// Potentially send an error event or a quit event
					a.eventQueue <- AppEvent{Type: EventTypeUserInputCommand, Data: UserInputCommandData{Command: "/quit"}}
					return
				}
				a.eventQueue <- AppEvent{Type: EventTypeUserInputCommand, Data: UserInputCommandData{Command: strings.TrimSpace(input)}}
			}
		}
	}()

	// Main event loop
	fmt.Print("Orchestrator> ") // Initial prompt
	for {
		select {
		case <-ctx.Done(): // Main context cancellation (e.g. Ctrl+C from OS)
			fmt.Println("\nShutting down Orchestrator CLI due to context cancellation...")
			cancelInput() // Signal input goroutine to stop
			return nil
		case event := <-a.eventQueue:
			switch event.Type {
			case EventTypeUserInputCommand:
				data := event.Data.(UserInputCommandData)
				command := data.Command

				if command == "/quit" {
					fmt.Println("Exiting Orchestrator CLI.")
					cancelInput() // Signal input goroutine to stop
					return nil   // Exit RunOrchestratorCLI
				}

				if strings.HasPrefix(command, "/spawn ") {
					prompt := strings.TrimSpace(strings.TrimPrefix(command, "/spawn "))
					if prompt == "" {
						fmt.Fprintln(os.Stderr, "Error: /spawn command requires a prompt.")
					} else if a.Orchestrator == nil {
						fmt.Fprintln(os.Stderr, "Error: Orchestrator service not available for /spawn.")
					} else {
						taskID := uuid.NewString()
						taskDef := orchestrator.TaskDefinition{Prompt: prompt, TaskID: taskID}
						workerID, err := a.Orchestrator.SpawnWorker(ctx, taskDef) // Use app context for worker
						if err != nil {
							fmt.Fprintf(os.Stderr, "Error spawning worker: %v\n", err)
							logging.Error("Failed to spawn worker", "prompt", prompt, "error", err)
						} else {
							fmt.Printf("Worker %s spawned for task %s (prompt: %s)\n", workerID, taskID, prompt)
							logging.Info("Worker spawned via CLI", "workerID", workerID, "taskID", taskID, "prompt", prompt)
						}
					}
				} else if strings.HasPrefix(command, "/result ") {
					workerID := strings.TrimSpace(strings.TrimPrefix(command, "/result "))
					if workerID == "" {
						fmt.Fprintln(os.Stderr, "Error: /result command requires a worker ID.")
					} else if a.Orchestrator == nil {
						fmt.Fprintln(os.Stderr, "Error: Orchestrator service not available for /result.")
					} else {
						result, found := a.Orchestrator.GetWorkerResult(workerID)
						if !found {
							fmt.Printf("No result found for worker ID: %s (it may be running or an invalid ID).\n", workerID)
						} else {
							fmt.Printf("Result for worker %s (Task %s):\n  Status: %s\n", result.AgentID, result.TaskID, result.Status)
							if result.Error != "" {
								fmt.Printf("  Error: %s\n", result.Error)
							}
							if result.Result != "" {
								fmt.Printf("  Output: %s\n", result.Result)
							}
						}
					}
				} else if command == "" {
					// Just reprint prompt for empty input
				} else {
					// Default: Treat input as a prompt for the Orchestrator's CoderAgent
					logging.Info("Sending prompt to Orchestrator's CoderAgent", "sessionID", OrchestratorSessionID, "prompt", command)
					if a.CoderAgent == nil {
						fmt.Fprintln(os.Stderr, "Error: CoderAgent not available.")
					} else {
						doneChan, err := a.CoderAgent.Run(ctx, OrchestratorSessionID, command)
						if err != nil {
							fmt.Fprintf(os.Stderr, "Error running CoderAgent: %v\n", err)
							logging.Error("Error running CoderAgent for orchestrator", "error", err)
						} else {
							fmt.Println("Orchestrator's CoderAgent is thinking...")
							result := <-doneChan // This remains blocking for now

							if result.Error != nil {
								fmt.Fprintf(os.Stderr, "\nCoderAgent Error: %v\n", result.Error)
								logging.Error("CoderAgent run for orchestrator failed", "error", result.Error)
							}
							if result.Message != nil && result.Message.Content().String() != "" {
								fmt.Println("\n--- Agent Response ---")
								fmt.Println(result.Message.Content().String())
								fmt.Println("----------------------")
							}
							if len(result.ToolCalls) > 0 {
								fmt.Println("\n--- Tool Calls Requested ---")
								for _, tc := range result.ToolCalls {
									if tc.ToolName != "" {
										fmt.Printf("- Tool: %s\n", tc.ToolName)
										argsMap := tc.Arguments.(map[string]interface{})
										argsJson, _ := json.MarshalIndent(argsMap, "  ", "  ")
										fmt.Printf("  Args: %s\n", string(argsJson))
									} else if tc.ToolResult != nil && tc.ToolResult.ToolName != "" {
										fmt.Printf("- Tool Result for: %s (ID: %s)\n", tc.ToolResult.ToolName, tc.ToolResult.ToolUseID)
									}
								}
								fmt.Println("--------------------------")
							}
							if result.Message == nil && len(result.ToolCalls) == 0 && result.Error == nil {
								fmt.Println("\nAgent finished with no text output or tool calls.")
							}
						}
					}
				}
				fmt.Print("Orchestrator> ") // Re-print prompt after handling user command

			case EventTypeWorkerCompletion:
				data := event.Data.(WorkerCompletionData)
				// Async message, so print it and then re-print the prompt line
				fmt.Printf("\n[Worker Event] Worker %s completed. Task: %s. Status: %s\n", data.WorkerID, data.Result.TaskID, data.Result.Status)
				if data.Result.Error != "" {
					fmt.Printf("[Worker Event] Error: %s\n", data.Result.Error)
				}
				// TODO: Optionally print data.Result.Result if needed/desired here.
				fmt.Print("Orchestrator> ") // Re-print prompt

			default:
				logging.Warn("Unknown event type received in CLI", "type", event.Type)
				fmt.Print("Orchestrator> ") // Re-print prompt
			}
		}
	}
}
