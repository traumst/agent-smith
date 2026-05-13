package handlers

import (
	"encoding/json"
	"net/http"

	"agentsmith/src/persistence/memory"
)

// MemoryHandler manages long-term memory endpoints.
type MemoryHandler struct {
	Store *memory.Store
}

// Post saves a new memory file.
func (h *MemoryHandler) Post(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Filename string `json:"filename"`
		Content  string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Filename == "" || req.Content == "" {
		http.Error(w, "missing filename or content", http.StatusBadRequest)
		return
	}

	if err := h.Store.SaveMemory(req.Filename, req.Content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
