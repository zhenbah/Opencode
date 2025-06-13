package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/opencode-ai/opencode/internal/app"
	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/db"
	"github.com/opencode-ai/opencode/internal/format"
	"github.com/opencode-ai/opencode/internal/llm/agent"
	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/pubsub"
	"github.com/opencode-ai/opencode/internal/tui"
	"github.com/opencode-ai/opencode/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "opencode",
	Short: "Terminal-based AI assistant for software development",
	Long: `OpenCode is a powerful terminal-based AI assistant that helps with software development tasks.
It provides an interactive chat interface with AI capabilities, code analysis, and LSP integration
to assist developers in writing, debugging, and understanding code directly from the terminal.`,
	Example: `
  # Run in interactive mode
  opencode

  # Run with debug logging
  opencode -d

  # Run with debug logging in a specific directory
  opencode -d -c /path/to/project

  # Print version
  opencode -v

  # Run a single non-interactive prompt
  opencode -p "Explain the use of context in Go"

  # Run a single non-interactive prompt with JSON output format
  opencode -p "Explain the use of context in Go" -f json
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If the help flag is set, show the help message
		if cmd.Flag("help").Changed {
			cmd.Help()
			return nil
		}
		if cmd.Flag("version").Changed {
			fmt.Println(version.Version)
			return nil
		}

		// Load the config
		debug, _ := cmd.Flags().GetBool("debug")
		cwd, _ := cmd.Flags().GetString("cwd")
		prompt, _ := cmd.Flags().GetString("prompt")
		outputFormat, _ := cmd.Flags().GetString("output-format")
		quiet, _ := cmd.Flags().GetBool("quiet")
		// Worker mode flags
		workerMode, _ := cmd.Flags().GetBool("worker-mode")
		taskFile, _ := cmd.Flags().GetString("task-file")
		agentID, _ := cmd.Flags().GetString("agent-id")
		orchestratorAPI, _ := cmd.Flags().GetString("orchestrator-api")
		taskID, _ := cmd.Flags().GetString("task-id") // Read the new task-id flag

		// Validate format option
		if !format.IsValid(outputFormat) {
			return fmt.Errorf("invalid format option: %s\n%s", outputFormat, format.GetHelpText())
		}

		if cwd != "" {
			err := os.Chdir(cwd)
			if err != nil {
				return fmt.Errorf("failed to change directory: %v", err)
			}
		}
		if cwd == "" {
			c, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current working directory: %v", err)
			}
			cwd = c
		}
		_, err := config.Load(cwd, debug)
		if err != nil {
			return err
		}

		// Connect DB, this will also run migrations
		conn, err := db.Connect()
		if err != nil {
			return err
		}

		// Create main context for the application
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Pass workerMode to app.New
		appInstance, err := app.New(ctx, conn, workerMode)
		if err != nil {
			logging.Error("Failed to create app: %v", err)
			return err
		}
		// Defer shutdown here so it runs for all modes
		defer appInstance.Shutdown()

		// Initialize MCP tools early for all modes, if not in worker mode
		// Workers might not need MCP tools, or might have a different initialization.
		// For now, let's assume they don't initialize MCP tools the same way.
		if !workerMode {
			initMCPTools(ctx, appInstance)
		}

		// Worker mode execution
		if workerMode {
			logging.Info("Starting in Worker Mode", "agentID", agentID, "taskFile", taskFile)
			// Ensure worker mode does not fall through to interactive or non-interactive prompt mode
			if prompt != "" {
				logging.Warn("Prompt flag (-p) is ignored when --worker-mode is active.")
			}
			// Pass taskID to RunWorkerMode
			return appInstance.RunWorkerMode(ctx, taskFile, agentID, orchestratorAPI, taskID)
		}

		// Non-interactive prompt mode (if not worker mode)
		if prompt != "" {
			// Run non-interactive flow using the App method
			return appInstance.RunNonInteractive(ctx, prompt, outputFormat, quiet)
		}

		// Orchestrator CLI mode (default if no other mode is specified)
		logging.Info("Starting in Orchestrator CLI mode")
		// The setupSubscriptions function and its TUI specific logic are removed.
		// If general event logging or handling is needed, it should be done directly.
		// For example, logging events can be directly handled by the logging package subscribers if any.
		err = appInstance.RunOrchestratorCLI(ctx)
		if err != nil {
			logging.Error("Orchestrator CLI error", "error", err)
			return err
		}
		logging.Info("Orchestrator CLI exited")
		return nil
	},
}

// attemptTUIRecovery is no longer needed as TUI is removed for orchestrator.
// func attemptTUIRecovery(program *tea.Program) {
// 	logging.Info("Attempting to recover TUI after panic")
// 	program.Quit()
// }

func initMCPTools(ctx context.Context, app *app.App) {
	go func() {
		defer logging.RecoverPanic("MCP-goroutine", nil)

		// Create a context with timeout for the initial MCP tools fetch
		ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// Set this up once with proper error handling
		agent.GetMcpTools(ctxWithTimeout, app.Permissions)
		logging.Info("MCP message handling goroutine exiting")
	}()
}

func setupSubscriber[T any](
	ctx context.Context,
	wg *sync.WaitGroup,
	name string,
	subscriber func(context.Context) <-chan pubsub.Event[T],
	outputCh chan<- tea.Msg,
) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer logging.RecoverPanic(fmt.Sprintf("subscription-%s", name), nil)

		subCh := subscriber(ctx)

		for {
			select {
			case event, ok := <-subCh:
				if !ok {
					logging.Info("subscription channel closed", "name", name)
					return
				}

				var msg tea.Msg = event

				select {
				case outputCh <- msg:
				case <-time.After(2 * time.Second):
					logging.Warn("message dropped due to slow consumer", "name", name)
				case <-ctx.Done():
					logging.Info("subscription cancelled", "name", name)
					return
				}
			case <-ctx.Done():
				logging.Info("subscription cancelled", "name", name)
				return
			}
		}
	}()
}

// setupSubscriptions was TUI specific and is removed.
// If any of these subscriptions are critical for non-TUI operation (e.g. logging specific events),
// they would need to be re-implemented to directly call handlers or log.
// For now, removing it as per the goal to eliminate TUI dependencies for orchestrator CLI.
/*
func setupSubscriptions(app *app.App, parentCtx context.Context) (chan tea.Msg, func()) {
	// ... implementation ...
}
*/

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("help", "h", false, "Help")
	rootCmd.Flags().BoolP("version", "v", false, "Version")
	rootCmd.Flags().BoolP("debug", "d", false, "Debug")
	rootCmd.Flags().StringP("cwd", "c", "", "Current working directory")
	rootCmd.Flags().StringP("prompt", "p", "", "Prompt to run in non-interactive mode")

	// Add format flag with validation logic
	rootCmd.Flags().StringP("output-format", "f", format.Text.String(),
		"Output format for non-interactive mode (text, json)")

	// Add quiet flag to hide spinner in non-interactive mode
	rootCmd.Flags().BoolP("quiet", "q", false, "Hide spinner in non-interactive mode")

	// Worker mode flags
	rootCmd.Flags().BoolP("worker-mode", "w", false, "Run in worker agent mode")
	rootCmd.Flags().String("task-file", "", "Path to JSON file defining the task for the worker")
	rootCmd.Flags().String("agent-id", "", "Unique ID for this worker agent")
	rootCmd.Flags().String("orchestrator-api", "", "API endpoint of the orchestrator for reporting")
	rootCmd.Flags().String("task-id", "", "Unique ID for the task assigned to the worker") // New task-id flag

	// Register custom validation for the format flag
	rootCmd.RegisterFlagCompletionFunc("output-format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return format.SupportedFormats, cobra.ShellCompDirectiveNoFileComp
	})
}
