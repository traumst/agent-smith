package handlers

import (
	"database/sql"
	"html/template"
	"io/fs"
	"net/http"
	"strconv"

	"smithai/src/agent/adapter/gemini"
	"smithai/src/agent/protocol"
	"smithai/src/persistence/history"
	"smithai/src/persistence/settings"
)

// UIHandler serves the web interface.
type UIHandler struct {
	Templates    map[string]*template.Template
	DB           *sql.DB
	Registry     *gemini.ModelRegistry
	SettingsPath string
}

// Index renders the chat page.
func (h *UIHandler) Index(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	var hist []protocol.Message
	if sessionID != "" {
		hist, _ = history.GetHistory(h.DB, sessionID)
	}

	cfg, _ := settings.LoadSettings(h.SettingsPath)
	data := map[string]interface{}{
		"ActiveModel": h.Registry.GetActive(),
		"Models":      h.Registry.GetModels(),
		"Settings":    cfg,
		"History":     hist,
	}
	if err := h.Templates["chat.html"].Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Settings renders the settings page.
func (h *UIHandler) Settings(w http.ResponseWriter, r *http.Request) {
	cfg, err := settings.LoadSettings(h.SettingsPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := map[string]interface{}{
		"Settings": cfg,
	}
	if err := h.Templates["settings.html"].Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// HistoryList renders the history infinite scroll list.
func (h *UIHandler) HistoryList(w http.ResponseWriter, r *http.Request) {
	offsetStr := r.URL.Query().Get("offset")
	offset, _ := strconv.Atoi(offsetStr)
	limit := 10

	sessions, err := history.ListSessions(h.DB, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Sessions":   sessions,
		"NextOffset": offset + limit,
		"HasMore":    len(sessions) == limit,
	}

	if err := h.Templates["history_list.html"].Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// DeleteChat handles session deletion.
func (h *UIHandler) DeleteChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "Missing session_id", http.StatusBadRequest)
		return
	}
	if err := history.DeleteSession(h.DB, sessionID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// StaticHandler returns an http.Handler that serves embedded static files.
func StaticHandler(staticFS fs.FS) http.Handler {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic("failed to create sub filesystem for static: " + err.Error())
	}
	return http.StripPrefix("/static/", http.FileServer(http.FS(sub)))
}
