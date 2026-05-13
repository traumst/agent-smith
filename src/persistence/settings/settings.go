package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"agentsmith/src/agent/consent"
	"agentsmith/src/agent/protocol"
)

// Settings encapsulates the overall configuration for the agent.
type Settings struct {
	ActiveModel          string                `json:"activeModel"`
	SystemPrompt         protocol.SystemPrompt `json:"systemPrompt"`
	GeminiRPM            int                   `json:"geminiRPM"`
	ModelRefreshInterval string                `json:"modelRefreshInterval"`
	Whitelist            string                `json:"whitelist"`
	Blacklist            string                `json:"blacklist"`
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

	// Set defaults for missing fields
	if s.ModelRefreshInterval == "" {
		s.ModelRefreshInterval = DefaultSettings().ModelRefreshInterval
	}
	if s.GeminiRPM == 0 {
		s.GeminiRPM = DefaultSettings().GeminiRPM
	}

	// Load consent lists
	if s.Whitelist, err = consent.ReadList(".whitelist"); err != nil {
		return nil, err
	}
	if s.Blacklist, err = consent.ReadList(".blacklist"); err != nil {
		return nil, err
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

	// Save consent lists
	if err := consent.WriteList(".whitelist", s.Whitelist); err != nil {
		return err
	}
	if err := consent.WriteList(".blacklist", s.Blacklist); err != nil {
		return err
	}
	return nil
}

// DefaultSettings returns a set of basic default settings.
func DefaultSettings() *Settings {
	return &Settings{
		ActiveModel: "gemini-2.0-flash",
		SystemPrompt: protocol.SystemPrompt{
			Competence:   "You are an expert, helpful AI assistant named Agent Smith.",
			Mood:         "Direct, concise, and professional.",
			Instructions: "Prioritize answering the user's questions clearly.",
		},
		GeminiRPM:            5,
		ModelRefreshInterval: "1:0:0.000",
	}
}

// ParseTimespan parses a string in H:M:S.ms format and returns a time.Duration.
func ParseTimespan(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("empty timespan")
	}
	var h, m, s_sec int
	var ms int
	_, err := fmt.Sscanf(s, "%d:%d:%d.%d", &h, &m, &s_sec, &ms)
	if err != nil {
		// Try without milliseconds if it fails
		_, err = fmt.Sscanf(s, "%d:%d:%d", &h, &m, &s_sec)
		if err != nil {
			return 0, fmt.Errorf("invalid timespan format, expected H:M:S.ms: %w", err)
		}
	}

	total := time.Duration(h)*time.Hour +
		time.Duration(m)*time.Minute +
		time.Duration(s_sec)*time.Second +
		time.Duration(ms)*time.Millisecond

	return total, nil
}
