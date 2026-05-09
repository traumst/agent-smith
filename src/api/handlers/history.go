package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"smithai/src/persistence/history"
)

// HistoryHandler manages the chat history endpoints.
type HistoryHandler struct {
	DB *sql.DB
}

// Get retrieves the chat history for a specific session ID.
func (h *HistoryHandler) Get(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("session_id")
	if sessionID == "" {
		http.Error(w, "missing session_id parameter", http.StatusBadRequest)
		return
	}

	hist, err := history.GetHistory(h.DB, sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(hist)
}
