package config

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
)

// HotkeyConfig defines the structure for all remappable hotkeys
type HotkeyConfig struct {
	// Global hotkeys
	Logs          string `yaml:"logs"`          // Default: "ctrl+l"
	Quit          string `yaml:"quit"`          // Default: "ctrl+c"
	Help          string `yaml:"help"`          // Default: "ctrl+?"
	SwitchSession string `yaml:"switch_session"` // Default: "ctrl+a"
	Commands      string `yaml:"commands"`      // Default: "ctrl+k"

	// Chat page hotkeys
	NewSession string `yaml:"new_session"` // Default: "ctrl+n"
	Cancel     string `yaml:"cancel"`      // Default: "esc"

	// Message navigation hotkeys
	PageDown     string `yaml:"page_down"`      // Default: "pgdown"
	PageUp       string `yaml:"page_up"`        // Default: "pgup"
	HalfPageUp   string `yaml:"half_page_up"`   // Default: "ctrl+u"
	HalfPageDown string `yaml:"half_page_down"` // Default: "ctrl+d"

	// Dialog navigation hotkeys
	Up     string `yaml:"up"`     // Default: "up"
	Down   string `yaml:"down"`   // Default: "down"
	Enter  string `yaml:"enter"`  // Default: "enter"
	Escape string `yaml:"escape"` // Default: "esc"
	J      string `yaml:"j"`      // Default: "j"
	K      string `yaml:"k"`      // Default: "k"
	Left   string `yaml:"left"`   // Default: "left"
	Right  string `yaml:"right"`  // Default: "right"
	Tab    string `yaml:"tab"`    // Default: "tab"
}

// DefaultHotkeyConfig returns the default hotkey configuration
func DefaultHotkeyConfig() HotkeyConfig {
	return HotkeyConfig{
		Logs:          "ctrl+l",
		Quit:          "ctrl+c",
		Help:          "ctrl+?",
		SwitchSession: "ctrl+a",
		Commands:      "ctrl+k",
		NewSession:    "ctrl+n",
		Cancel:        "esc",
		PageDown:      "pgdown",
		PageUp:        "pgup",
		HalfPageUp:    "ctrl+u",
		HalfPageDown:  "ctrl+d",
		Up:            "up",
		Down:          "down",
		Enter:         "enter",
		Escape:        "esc",
		J:             "j",
		K:             "k",
		Left:          "left",
		Right:         "right",
		Tab:           "tab",
	}
}

// GetKeyBinding creates a new key.Binding from the given key string
func GetKeyBinding(keyStr string, helpKey, helpDesc string) key.Binding {
	return key.NewBinding(
		key.WithKeys(keyStr),
		key.WithHelp(helpKey, helpDesc),
	)
}

// ValidateHotkey validates a hotkey string
func ValidateHotkey(keyStr string) error {
	if keyStr == "" {
		return fmt.Errorf("hotkey cannot be empty")
	}

	// Split into parts for modifier+key combinations
	parts := strings.Split(keyStr, "+")
	
	// Check each part
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return fmt.Errorf("invalid hotkey format: empty part in '%s'", keyStr)
		}

		// Last part is the key, others are modifiers
		if i == len(parts)-1 {
			// Validate key
			if !isValidKey(part) {
				return fmt.Errorf("invalid key '%s' in hotkey '%s'", part, keyStr)
			}
		} else {
			// Validate modifier
			if !isValidModifier(part) {
				return fmt.Errorf("invalid modifier '%s' in hotkey '%s'", part, keyStr)
			}
		}
	}

	return nil
}

// ValidateHotkeyConfig validates all hotkeys in the configuration
func ValidateHotkeyConfig(config HotkeyConfig) error {
	// Validate global hotkeys
	if err := ValidateHotkey(config.Logs); err != nil {
		return fmt.Errorf("logs hotkey: %w", err)
	}
	if err := ValidateHotkey(config.Quit); err != nil {
		return fmt.Errorf("quit hotkey: %w", err)
	}
	if err := ValidateHotkey(config.Help); err != nil {
		return fmt.Errorf("help hotkey: %w", err)
	}
	if err := ValidateHotkey(config.SwitchSession); err != nil {
		return fmt.Errorf("switch_session hotkey: %w", err)
	}
	if err := ValidateHotkey(config.Commands); err != nil {
		return fmt.Errorf("commands hotkey: %w", err)
	}

	// Validate chat page hotkeys
	if err := ValidateHotkey(config.NewSession); err != nil {
		return fmt.Errorf("new_session hotkey: %w", err)
	}
	if err := ValidateHotkey(config.Cancel); err != nil {
		return fmt.Errorf("cancel hotkey: %w", err)
	}

	// Validate message navigation hotkeys
	if err := ValidateHotkey(config.PageDown); err != nil {
		return fmt.Errorf("page_down hotkey: %w", err)
	}
	if err := ValidateHotkey(config.PageUp); err != nil {
		return fmt.Errorf("page_up hotkey: %w", err)
	}
	if err := ValidateHotkey(config.HalfPageUp); err != nil {
		return fmt.Errorf("half_page_up hotkey: %w", err)
	}
	if err := ValidateHotkey(config.HalfPageDown); err != nil {
		return fmt.Errorf("half_page_down hotkey: %w", err)
	}

	// Validate dialog navigation hotkeys
	if err := ValidateHotkey(config.Up); err != nil {
		return fmt.Errorf("up hotkey: %w", err)
	}
	if err := ValidateHotkey(config.Down); err != nil {
		return fmt.Errorf("down hotkey: %w", err)
	}
	if err := ValidateHotkey(config.Enter); err != nil {
		return fmt.Errorf("enter hotkey: %w", err)
	}
	if err := ValidateHotkey(config.Escape); err != nil {
		return fmt.Errorf("escape hotkey: %w", err)
	}
	if err := ValidateHotkey(config.J); err != nil {
		return fmt.Errorf("j hotkey: %w", err)
	}
	if err := ValidateHotkey(config.K); err != nil {
		return fmt.Errorf("k hotkey: %w", err)
	}
	if err := ValidateHotkey(config.Left); err != nil {
		return fmt.Errorf("left hotkey: %w", err)
	}
	if err := ValidateHotkey(config.Right); err != nil {
		return fmt.Errorf("right hotkey: %w", err)
	}
	if err := ValidateHotkey(config.Tab); err != nil {
		return fmt.Errorf("tab hotkey: %w", err)
	}

	return nil
}

// isValidKey checks if a key is valid
func isValidKey(key string) bool {
	// List of valid keys
	validKeys := map[string]bool{
		"a": true, "b": true, "c": true, "d": true, "e": true, "f": true, "g": true,
		"h": true, "i": true, "j": true, "k": true, "l": true, "m": true, "n": true,
		"o": true, "p": true, "q": true, "r": true, "s": true, "t": true, "u": true,
		"v": true, "w": true, "x": true, "y": true, "z": true,
		"0": true, "1": true, "2": true, "3": true, "4": true, "5": true, "6": true,
		"7": true, "8": true, "9": true,
		"up": true, "down": true, "left": true, "right": true,
		"enter": true, "space": true, "tab": true, "esc": true, "backspace": true,
		"delete": true, "home": true, "end": true, "pgup": true, "pgdown": true,
		"f1": true, "f2": true, "f3": true, "f4": true, "f5": true, "f6": true,
		"f7": true, "f8": true, "f9": true, "f10": true, "f11": true, "f12": true,
		"?": true, "!": true, "@": true, "#": true, "$": true, "%": true, "^": true,
		"&": true, "*": true, "(": true, ")": true, "-": true, "_": true, "=": true,
		"+": true, "[": true, "]": true, "{": true, "}": true, "\\": true, "|": true,
		";": true, ":": true, "'": true, "\"": true, ",": true, "<": true, ".": true,
		">": true, "/": true, "`": true, "~": true,
	}

	return validKeys[strings.ToLower(key)]
}

// isValidModifier checks if a modifier is valid
func isValidModifier(modifier string) bool {
	validModifiers := map[string]bool{
		"ctrl": true, "alt": true, "shift": true, "cmd": true, "super": true,
	}
	return validModifiers[strings.ToLower(modifier)]
} 