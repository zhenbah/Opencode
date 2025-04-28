package config

import (
	"testing"
)

func TestValidateHotkey(t *testing.T) {
	tests := []struct {
		name    string
		hotkey  string
		wantErr bool
	}{
		{
			name:    "valid single key",
			hotkey:  "a",
			wantErr: false,
		},
		{
			name:    "valid modifier+key",
			hotkey:  "ctrl+l",
			wantErr: false,
		},
		{
			name:    "valid multiple modifiers",
			hotkey:  "ctrl+alt+delete",
			wantErr: false,
		},
		{
			name:    "empty hotkey",
			hotkey:  "",
			wantErr: true,
		},
		{
			name:    "invalid modifier",
			hotkey:  "invalid+l",
			wantErr: true,
		},
		{
			name:    "invalid key",
			hotkey:  "ctrl+invalid",
			wantErr: true,
		},
		{
			name:    "empty part",
			hotkey:  "ctrl++l",
			wantErr: true,
		},
		{
			name:    "whitespace",
			hotkey:  "ctrl + l",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHotkey(tt.hotkey)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHotkey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateHotkeyConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  HotkeyConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: HotkeyConfig{
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
			},
			wantErr: false,
		},
		{
			name: "invalid config - empty hotkey",
			config: HotkeyConfig{
				Logs:          "",
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
			},
			wantErr: true,
		},
		{
			name: "invalid config - invalid modifier",
			config: HotkeyConfig{
				Logs:          "invalid+l",
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
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHotkeyConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHotkeyConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{
			name: "valid letter",
			key:  "a",
			want: true,
		},
		{
			name: "valid number",
			key:  "1",
			want: true,
		},
		{
			name: "valid special key",
			key:  "enter",
			want: true,
		},
		{
			name: "valid special character",
			key:  "?",
			want: true,
		},
		{
			name: "invalid key",
			key:  "invalid",
			want: false,
		},
		{
			name: "empty key",
			key:  "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidKey(tt.key); got != tt.want {
				t.Errorf("isValidKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidModifier(t *testing.T) {
	tests := []struct {
		name     string
		modifier string
		want     bool
	}{
		{
			name:     "valid modifier",
			modifier: "ctrl",
			want:     true,
		},
		{
			name:     "valid modifier uppercase",
			modifier: "CTRL",
			want:     true,
		},
		{
			name:     "invalid modifier",
			modifier: "invalid",
			want:     false,
		},
		{
			name:     "empty modifier",
			modifier: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidModifier(tt.modifier); got != tt.want {
				t.Errorf("isValidModifier() = %v, want %v", got, tt.want)
			}
		})
	}
} 