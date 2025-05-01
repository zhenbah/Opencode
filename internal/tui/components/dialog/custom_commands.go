package dialog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/tui/util"
)

// CustomCommandPrefix is the prefix used for custom commands loaded from files
const CustomCommandPrefix = "user:"

// LoadCustomCommands loads custom commands from the data directory
func LoadCustomCommands() ([]Command, error) {
	cfg := config.Get()
	if cfg == nil {
		return nil, fmt.Errorf("config not loaded")
	}

	dataDir := cfg.Data.Directory
	commandsDir := filepath.Join(dataDir, "commands")

	// Check if the commands directory exists
	if _, err := os.Stat(commandsDir); os.IsNotExist(err) {
		// Create the commands directory if it doesn't exist
		if err := os.MkdirAll(commandsDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create commands directory: %w", err)
		}
		// Return empty list since we just created the directory
		return []Command{}, nil
	}

	var commands []Command

	// Walk through the commands directory and load all .md files
	err := filepath.Walk(commandsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process markdown files
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			return nil
		}

		// Read the file content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read command file %s: %w", path, err)
		}

		// Get the command ID from the file name without the .md extension
		commandID := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
		
		// Get relative path from commands directory
		relPath, err := filepath.Rel(commandsDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}
		
		// Create the command ID from the relative path
		// Replace directory separators with colons
		commandIDPath := strings.ReplaceAll(filepath.Dir(relPath), string(filepath.Separator), ":")
		if commandIDPath != "." {
			commandID = commandIDPath + ":" + commandID
		}

		// Create a command
		command := Command{
			ID:          CustomCommandPrefix + commandID,
			Title:       commandID,
			Description: fmt.Sprintf("Custom command from %s", relPath),
			Handler: func(cmd Command) tea.Cmd {
				return util.CmdHandler(CommandRunCustomMsg{
					Content: string(content),
				})
			},
		}

		commands = append(commands, command)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to load custom commands: %w", err)
	}

	return commands, nil
}

// CommandRunCustomMsg is sent when a custom command is executed
type CommandRunCustomMsg struct {
	Content string
}