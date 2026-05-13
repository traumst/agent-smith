package tools

import (
	"context"
	"fmt"
	"os/exec"

	"agentsmith/src/agent/consent"
	"agentsmith/src/agent/protocol"

	"github.com/kballard/go-shellquote"
)

// RegisterTerminalTools registers the terminal_exec tool.
func RegisterTerminalTools(d Dispatcher) {
	d.Register(protocol.ToolDef{
		Name:        "terminal_exec",
		Description: "Executes a command in the terminal. Uses exec.Command directly so shell operators (&&, |) are not evaluated.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "The command string to execute (e.g., 'ls -la' or 'git status').",
				},
			},
			"required": []string{"command"},
		},
	}, func(ctx context.Context, args any) (string, error) {
		argsMap, ok := args.(map[string]any)
		if !ok {
			return "", fmt.Errorf("invalid arguments format")
		}
		commandStr, _ := argsMap["command"].(string)
		if commandStr == "" {
			return "", fmt.Errorf("command is required")
		}

		// Require consent, subject is the command string
		action, err := consent.Require("terminal_exec", commandStr, map[string]string{"command": commandStr})
		if err != nil {
			return "", err
		}
		if action == "block" {
			return "", fmt.Errorf("action blocked by user")
		}

		// Parse string into args array.
		// This naturally prevents chain evaluation because "&&" becomes a literal argument.
		parsedArgs, err := shellquote.Split(commandStr)
		if err != nil {
			return "", fmt.Errorf("failed to parse command: %v", err)
		}

		if len(parsedArgs) == 0 {
			return "", fmt.Errorf("empty command")
		}

		cmd := exec.CommandContext(ctx, parsedArgs[0], parsedArgs[1:]...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return string(output) + "\nError: " + err.Error(), nil
		}

		return string(output), nil
	})
}
