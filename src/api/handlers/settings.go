package handlers

import (
	"encoding/json"
	"net/http"

	"smithai/src/agent/adapter/gemini"
	"smithai/src/persistence/settings"
)

// SettingsHandler manages the settings endpoints.
type SettingsHandler struct {
	Path     string
	Registry *gemini.ModelRegistry
}

// Get returns the current settings.
func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg, err := settings.LoadSettings(h.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

// Post updates the settings.
func (h *SettingsHandler) Post(w http.ResponseWriter, r *http.Request) {
	var cfg settings.Settings
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := settings.SaveSettings(h.Path, &cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.Registry.SetActive(cfg.ActiveModel)
	w.WriteHeader(http.StatusOK)
}
