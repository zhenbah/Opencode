package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/llm/tools"
	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/lsp"
)

const (
	serverName    = "mcp-language-server"
	serverVersion = "0.1.0"
)

func main() {
	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handlers
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChan
		logging.Info("Received shutdown signal, closing LSP connections...")
		cancel()
		// Give some time for graceful shutdown
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()

	// Load configuration
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	// Initialize configuration
	cfg, err := config.Load(workingDir, os.Getenv("OPENCODE_DEBUG") == "true")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize LSP clients
	lspClients := make(map[string]*lsp.Client)
	for name, clientConfig := range cfg.LSP {
		if clientConfig.Disabled {
			continue
		}

		logging.Info("Creating LSP client", "name", name, "command", clientConfig.Command, "args", clientConfig.Args)
		
		// Create the LSP client
		lspClient, err := lsp.NewClient(ctx, clientConfig.Command, clientConfig.Args...)
		if err != nil {
			logging.Error("Failed to create LSP client", "name", name, "error", err)
			continue
		}

		// Create a longer timeout for initialization (some servers take time to start)
		initCtx, initCancel := context.WithTimeout(ctx, 30*time.Second)
		
		// Initialize with the initialization context
		_, err = lspClient.InitializeLSPClient(initCtx, workingDir)
		if err != nil {
			logging.Error("Initialize failed", "name", name, "error", err)
			// Clean up the client to prevent resource leaks
			lspClient.Close()
			initCancel()
			continue
		}

		// Wait for the server to be ready
		if err := lspClient.WaitForServerReady(initCtx); err != nil {
			logging.Error("Server failed to become ready", "name", name, "error", err)
			// We'll continue anyway, as some functionality might still work
			lspClient.SetServerState(lsp.StateError)
		} else {
			logging.Info("LSP server is ready", "name", name)
			lspClient.SetServerState(lsp.StateReady)
		}

		initCancel()
		logging.Info("LSP client initialized", "name", name)
		
		// Store the client
		lspClients[name] = lspClient
	}

	// Create hooks for the MCP server
	hooks := &server.Hooks{}

	// Create MCP server
	mcpServer := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(true),
		server.WithLogging(),
		server.WithHooks(hooks),
		server.WithInstructions(`
This MCP server provides language server protocol (LSP) capabilities to OpenCode.
It allows you to get diagnostics (errors, warnings, hints) from language servers.

Available tools:
- diagnostics: Get diagnostic information from language servers
`),
	)

	// Add the diagnostics tool
	diagnosticsTool := tools.NewDiagnosticsTool(lspClients)
	mcpServer.AddTool(convertToolInfo(diagnosticsTool.Info()), handleDiagnosticsTool(diagnosticsTool))

	// Serve the MCP server over stdio
	logging.Info("Starting MCP-LSP server over stdio")
	if err := server.ServeStdio(mcpServer); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// convertToolInfo converts a tools.ToolInfo to mcp.Tool
func convertToolInfo(info tools.ToolInfo) mcp.Tool {
	// For the diagnostics tool, provide a more concise description
	description := info.Description
	if info.Name == tools.DiagnosticsToolName {
		description = "Get LSP diagnostics for a specific file or the whole project. Use after you've made file changes and want to check for errors or warnings in your code. Helpful for debugging and ensuring code quality."
	}
	
	return mcp.Tool{
		Name:        info.Name,
		Description: description,
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: info.Parameters,
			Required:   info.Required,
		},
	}
}

// handleDiagnosticsTool creates a handler function for the diagnostics tool
func handleDiagnosticsTool(diagnosticsTool tools.BaseTool) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Make a copy of the arguments to preserve originals
		args := make(map[string]interface{})
		for k, v := range request.Params.Arguments {
			args[k] = v
		}

		// Custom parsing for ensuring file path is absolute
		if args != nil {
			if filePath, ok := args["file_path"].(string); ok && filePath != "" {
				// Store the original path format
				args["original_path"] = filePath

				// Ensure file path is absolute
				if !filepath.IsAbs(filePath) {
					wd, err := os.Getwd()
					if err == nil {
						absPath := filepath.Join(wd, filePath)
						args["file_path"] = absPath
					} else {
						return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
					}
				}
			}
		}

		// Convert the arguments to JSON
		paramsBytes, err := json.Marshal(args)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal arguments: %w", err)
		}

		// Create a tool call
		call := tools.ToolCall{
			Name:  request.Params.Name,
			Input: string(paramsBytes),
		}

		// Run the tool
		response, err := diagnosticsTool.Run(ctx, call)
		if err != nil {
			return nil, fmt.Errorf("tool execution error: %w", err)
		}

		// Return the result
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: response.Content,
				},
			},
		}, nil
	}
}