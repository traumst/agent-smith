package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"smithai/src/agent/loop"
	"smithai/src/agent/protocol"
	"smithai/src/persistence/history"
)

// ChatHandler manages the agent interaction endpoint via SSE.
type ChatHandler struct {
	Agent      *loop.Agent
	DB         *sql.DB
	ConsentChan <-chan string // receives consent request JSON from main
}

// Post handles chat requests and streams responses via Server-Sent Events.
func (h *ChatHandler) Post(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID    string                `json:"session_id"`
		UserPrompt   string                `json:"user_prompt"`
		SystemPrompt protocol.SystemPrompt `json:"system_prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		http.Error(w, "missing session_id", http.StatusBadRequest)
		return
	}

	hist, err := history.GetHistory(h.DB, req.SessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if req.UserPrompt != "" {
		userMsg := protocol.Message{Role: "user", Content: req.UserPrompt}
		if err := history.AddMessage(h.DB, req.SessionID, userMsg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	agentReq := protocol.Request{
		SystemPrompt: req.SystemPrompt,
		UserPrompt:   req.UserPrompt,
		History:      hist,
		Stream:       true,
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ctx := r.Context()
	stream, err := h.Agent.Run(ctx, &agentReq)
	if err != nil {
		writeSSE(w, flusher, "error", fmt.Sprintf("error: %v", err))
		return
	}

	var fullContent strings.Builder

	for {
		select {
		case consentData := <-h.ConsentChan:
			writeSSE(w, flusher, "consent", consentData)

		case resp, ok := <-stream:
			if !ok {
				// stream closed
				writeSSE(w, flusher, "done", "")
				goto save
			}

			if resp.Error != nil {
				errStr := strings.ReplaceAll(resp.Error.Error(), "\n", " ")
				writeSSE(w, flusher, "error", errStr)
				goto save
			}

			if resp.Content != "" {
				fullContent.WriteString(resp.Content)
				writeSSE(w, flusher, "message", resp.Content)
			}

			if resp.Done {
				writeSSE(w, flusher, "done", fmt.Sprintf("Tokens used: %d", resp.TokensUsed))
				goto save
			}
		}
	}

save:
	if fullContent.Len() > 0 {
		assistantMsg := protocol.Message{Role: "assistant", Content: fullContent.String()}
		_ = history.AddMessage(h.DB, req.SessionID, assistantMsg)
	}
}

// writeSSE writes a single SSE event with proper formatting.
func writeSSE(w http.ResponseWriter, flusher http.Flusher, event, data string) {
	// SSE multiline data: each line must be prefixed with "data: "
	lines := strings.Split(data, "\n")
	fmt.Fprintf(w, "event: %s\n", event)
	for _, line := range lines {
		fmt.Fprintf(w, "data: %s\n", line)
	}
	fmt.Fprint(w, "\n")
	flusher.Flush()
}
