package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// Request is a simple JSON-RPC request
type Request struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
}

// Response is a simple JSON-RPC response
type Response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			sendResponse(Response{
				JSONRPC: "2.0",
				Error:   map[string]any{"code": -32700, "message": "Parse error"},
			})
			continue
		}

		if req.Method == "ping" {
			sendResponse(Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  map[string]any{"status": "ok"},
			})
		} else {
			sendResponse(Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   map[string]any{"code": -32601, "message": "Method not found"},
			})
		}
	}
}

func sendResponse(resp Response) {
	out, _ := json.Marshal(resp)
	fmt.Println(string(out))
}
