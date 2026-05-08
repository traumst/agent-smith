package settings

import (
	"encoding/json"
	"fmt"
	"os"

	"smithai/internal/agent/protocol"
)

// Settings encapsulates the overall configuration for the agent.
type Settings struct {
	SystemPrompt protocol.SystemPrompt `json:"systemPrompt"`
}

// LoadSettings reads the settings from a JSON file on disk.
func LoadSettings(path string) (*Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default settings if file doesn't exist
			return DefaultSettings(), nil
		}
		return nil, fmt.Errorf("failed to read settings file: %w", err)
	}

	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("failed to parse settings JSON: %w", err)
	}

	return &s, nil
}

// SaveSettings writes the settings to a JSON file on disk.
func SaveSettings(path string, s *Settings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode settings: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	return nil
}

// DefaultSettings returns a set of basic default settings.
func DefaultSettings() *Settings {
	return &Settings{
		SystemPrompt: protocol.SystemPrompt{
			Competence:   "You are an expert, helpful AI assistant named Smith.",
			Mood:         "Direct, concise, and professional.",
			Instructions: "Prioritize answering the user's questions clearly.",
		},
	}
}
