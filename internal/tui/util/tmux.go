package util

import (
	"os"
	"os/exec"
	"strings"
)

// IsTmuxSession returns true if the current process is running inside tmux
func GetTmuxPane() string {
	tmuxPane := os.Getenv("TMUX_PANE")
	if os.Getenv("TMUX") == "" || tmuxPane == "" {
		return ""
	}
	return tmuxPane
}

// IsProcessFocused returns true if the current tmux pane is focused
// Returns true if not in tmux or if unable to determine focus state
func IsProcessFocused(tmuxPane string) bool {
	if tmuxPane == "" {
		return true
	}

	// Check if this specific pane is active
	cmd := exec.Command("tmux", "display-message", "-t", tmuxPane, "-p", "#{pane_active}")
	output, err := cmd.Output()
	if err != nil {
		return true // Default to focused if we can't determine
	}

	isActive := strings.TrimSpace(string(output)) == "1"
	return isActive
}

