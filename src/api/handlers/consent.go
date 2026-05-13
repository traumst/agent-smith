package handlers

import (
	"encoding/json"
	"net/http"

	"agentsmith/src/agent/consent"
)

// ConsentHandler handles consent responses from the web UI.
type ConsentHandler struct {
	Pending *consent.PendingConsent
}

// Post receives a consent decision from the UI.
func (h *ConsentHandler) Post(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID     string `json:"id"`
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.ID == "" || req.Action == "" {
		http.Error(w, "missing id or action", http.StatusBadRequest)
		return
	}

	if !h.Pending.Respond(req.ID, req.Action) {
		http.Error(w, "no pending consent with that id", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}
