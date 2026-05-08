package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"

	"smithai/src/agent/protocol"
)

// Request represents a JSON-RPC request
type mcpRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
}

// RegisterMCPTools registers the mcp_ping tool.
func RegisterMCPTools(d Dispatcher) {
	d.Register(protocol.ToolDef{
		Name:        "mcp_ping",
		Description: "Pings the local Ping MCP server to verify that MCP integration and standard IO communication is working properly.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{},
		},
	}, func(ctx context.Context, args any) (string, error) {
		// Launch the ping_mcp server
		cmd := exec.CommandContext(ctx, "go", "run", "cmd/ping_mcp/main.go")
		
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return "", fmt.Errorf("failed to get stdin pipe: %v", err)
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return "", fmt.Errorf("failed to get stdout pipe: %v", err)
		}

		if err := cmd.Start(); err != nil {
			return "", fmt.Errorf("failed to start ping_mcp server: %v", err)
		}
		
		defer func() {
			stdin.Close()
			cmd.Process.Kill()
			cmd.Wait()
		}()

		// Prepare request
		req := mcpRequest{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "ping",
		}
		reqBytes, _ := json.Marshal(req)
		
		// Send request
		_, err = io.WriteString(stdin, string(reqBytes)+"\n")
		if err != nil {
			return "", fmt.Errorf("failed to write to MCP server: %v", err)
		}

		// Read response
		reader := bufio.NewReader(stdout)
		respLine, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read from MCP server: %v", err)
		}

		return fmt.Sprintf("MCP server responded: %s", respLine), nil
	})
}
