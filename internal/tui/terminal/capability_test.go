package terminal

import (
	"os"
	"testing"
)

func TestDetectColorProfile(t *testing.T) {
	// Save original environment
	originalNoColor := os.Getenv("NO_COLOR")
	originalColorterm := os.Getenv("COLORTERM")
	originalTerm := os.Getenv("TERM")
	originalColors := os.Getenv("COLORS")

	// Restore environment after test
	defer func() {
		setOrUnset("NO_COLOR", originalNoColor)
		setOrUnset("COLORTERM", originalColorterm)
		setOrUnset("TERM", originalTerm)
		setOrUnset("COLORS", originalColors)
	}()

	tests := []struct {
		name      string
		noColor   string
		colorterm string
		term      string
		colors    string
		expected  ColorProfile
	}{
		{
			name:     "NO_COLOR set",
			noColor:  "1",
			expected: ProfileNoColor,
		},
		{
			name:      "COLORTERM=truecolor",
			colorterm: "truecolor",
			term:      "xterm-256color", // Add a valid term to avoid termenv fallback
			expected:  ProfileTrueColor,
		},
		{
			name:      "COLORTERM=24bit",
			colorterm: "24bit",
			term:      "xterm-256color",
			expected:  ProfileTrueColor,
		},
		{
			name:     "TERM=xterm-256color",
			term:     "xterm-256color",
			expected: Profile256Color,
		},
		{
			name:     "TERM=screen-256color",
			term:     "screen-256color",
			expected: Profile256Color,
		},
		{
			name:     "TERM=xterm",
			term:     "xterm",
			expected: Profile256Color,
		},
		{
			name:     "TERM=dumb",
			term:     "dumb",
			expected: ProfileNoColor,
		},
		{
			name:     "TERM=vt100",
			term:     "vt100",
			expected: Profile256Color, // Conservative fallback
		},
		{
			name:     "COLORS=16777216",
			term:     "unknown",
			colors:   "16777216",
			expected: ProfileTrueColor,
		},
		{
			name:     "COLORS=256",
			term:     "unknown",
			colors:   "256",
			expected: Profile256Color,
		},
		{
			name:     "COLORS=16",
			term:     "unknown",
			colors:   "16",
			expected: Profile16Color,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Unsetenv("NO_COLOR")
			os.Unsetenv("COLORTERM")
			os.Unsetenv("TERM")
			os.Unsetenv("COLORS")

			// Set test environment
			if tt.noColor != "" {
				os.Setenv("NO_COLOR", tt.noColor)
			}
			if tt.colorterm != "" {
				os.Setenv("COLORTERM", tt.colorterm)
			}
			if tt.term != "" {
				os.Setenv("TERM", tt.term)
			}
			if tt.colors != "" {
				os.Setenv("COLORS", tt.colors)
			}

			result := DetectColorProfile()
			if result != tt.expected {
				t.Errorf("DetectColorProfile() = %v (%s), expected %v (%s)",
					result, result.String(), tt.expected, tt.expected.String())
			}
		})
	}
}

func TestColorProfileChromaFormatter(t *testing.T) {
	tests := []struct {
		profile  ColorProfile
		expected string
	}{
		{ProfileNoColor, "terminal"},
		{Profile16Color, "terminal16"},
		{Profile256Color, "terminal256"},
		{ProfileTrueColor, "terminal16m"},
	}

	for _, tt := range tests {
		t.Run(tt.profile.String(), func(t *testing.T) {
			result := tt.profile.ChromaFormatter()
			if result != tt.expected {
				t.Errorf("ChromaFormatter() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestHasColorSupport(t *testing.T) {
	// Save original environment
	originalNoColor := os.Getenv("NO_COLOR")
	originalTerm := os.Getenv("TERM")

	// Restore environment after test
	defer func() {
		setOrUnset("NO_COLOR", originalNoColor)
		setOrUnset("TERM", originalTerm)
	}()

	// Test NO_COLOR disables colors
	os.Setenv("NO_COLOR", "1")
	os.Unsetenv("TERM")
	if HasColorSupport() {
		t.Error("HasColorSupport() should return false when NO_COLOR is set")
	}

	// Test color terminal
	os.Unsetenv("NO_COLOR")
	os.Setenv("TERM", "xterm-256color")
	if !HasColorSupport() {
		t.Error("HasColorSupport() should return true for color terminal")
	}
}

func TestTerminalCapabilities(t *testing.T) {
	// Save original environment
	originalColorterm := os.Getenv("COLORTERM")
	originalTerm := os.Getenv("TERM")
	originalNoColor := os.Getenv("NO_COLOR")

	// Restore environment after test
	defer func() {
		setOrUnset("COLORTERM", originalColorterm)
		setOrUnset("TERM", originalTerm)
		setOrUnset("NO_COLOR", originalNoColor)
	}()

	// Clear NO_COLOR to ensure we're testing color capabilities
	os.Unsetenv("NO_COLOR")

	// Test true color support
	os.Setenv("COLORTERM", "truecolor")
	os.Setenv("TERM", "xterm-256color")
	if !HasTrueColorSupport() {
		t.Error("HasTrueColorSupport() should return true for truecolor terminal")
	}

	// Test 256 color support
	os.Unsetenv("COLORTERM")
	os.Setenv("TERM", "xterm-256color")
	if !Has256ColorSupport() {
		t.Error("Has256ColorSupport() should return true for 256-color terminal")
	}

	// Test basic terminal
	os.Setenv("TERM", "dumb")
	if HasTrueColorSupport() || Has256ColorSupport() {
		t.Error("Dumb terminal should not support advanced colors")
	}
}

// Helper function to set environment variable or unset if empty
func setOrUnset(key, value string) {
	if value == "" {
		os.Unsetenv(key)
	} else {
		os.Setenv(key, value)
	}
}
