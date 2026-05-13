package tools

import (
	"context"
	"fmt"
	"os"

	"agentsmith/src/agent/consent"
	"agentsmith/src/agent/protocol"
)

// RegisterFSTools registers the fs_read and fs_write tools with the dispatcher.
func RegisterFSTools(d Dispatcher) {
	d.Register(protocol.ToolDef{
		Name:        "fs_read",
		Description: "Reads the content of a file from the local file system.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "The absolute or relative path to the file.",
				},
			},
			"required": []string{"path"},
		},
	}, func(ctx context.Context, args any) (string, error) {
		argsMap, ok := args.(map[string]any)
		if !ok {
			return "", fmt.Errorf("invalid arguments format")
		}
		path, _ := argsMap["path"].(string)
		if path == "" {
			return "", fmt.Errorf("path is required")
		}

		// Read is considered safe enough not to require consent in this phase,
		// but we could add it if desired.
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	})

	d.Register(protocol.ToolDef{
		Name:        "fs_write",
		Description: "Writes content to a file on the local file system.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "The absolute or relative path to the file.",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "The content to write to the file.",
				},
			},
			"required": []string{"path", "content"},
		},
	}, func(ctx context.Context, args any) (string, error) {
		argsMap, ok := args.(map[string]any)
		if !ok {
			return "", fmt.Errorf("invalid arguments format")
		}
		path, _ := argsMap["path"].(string)
		content, _ := argsMap["content"].(string)

		if path == "" {
			return "", fmt.Errorf("path is required")
		}

		// Write requires user consent
		action, err := consent.Require("fs_write", path, map[string]string{"path": path, "content_length": fmt.Sprintf("%d", len(content))})
		if err != nil {
			return "", err
		}
		if action == "block" {
			return "", fmt.Errorf("action blocked by user")
		}

		err = os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			return "", err
		}
		return "File written successfully.", nil
	})
}
