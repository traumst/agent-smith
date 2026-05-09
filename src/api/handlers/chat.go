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
	Agent *loop.Agent
	DB    *sql.DB
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
		fmt.Fprintf(w, "data: error: %v\n\n", err)
		flusher.Flush()
		return
	}

	var fullContent strings.Builder

	for resp := range stream {
		if resp.Error != nil {
			errStr := strings.ReplaceAll(resp.Error.Error(), "\n", "\ndata: ")
			fmt.Fprintf(w, "data: error: %s\n\n", errStr)
			flusher.Flush()
			break
		}

		if resp.Content != "" {
			fullContent.WriteString(resp.Content)
			// Replace newlines with \ndata: to preserve SSE format for multiline text
			content := strings.ReplaceAll(resp.Content, "\n", "\ndata: ")
			fmt.Fprintf(w, "data: %s\n\n", content)
			flusher.Flush()
		}

		if resp.Done {
			fmt.Fprintf(w, "data: \n\ndata: Tokens used: %d\n\n", resp.TokensUsed)
			flusher.Flush()
			break
		}
	}

	// Save the accumulated response to history
	if fullContent.Len() > 0 {
		assistantMsg := protocol.Message{Role: "assistant", Content: fullContent.String()}
		_ = history.AddMessage(h.DB, req.SessionID, assistantMsg)
	}
}
