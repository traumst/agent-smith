package handlers

import (
	"database/sql"
	"html/template"
	"io/fs"
	"net/http"
	"strconv"

	"smithai/src/persistence/history"
)

// UIHandler serves the web interface.
type UIHandler struct {
	Templates *template.Template
	DB        *sql.DB
}

// Index renders the chat page.
func (h *UIHandler) Index(w http.ResponseWriter, r *http.Request) {
	if err := h.Templates.ExecuteTemplate(w, "chat.html", nil); err != nil {
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

	if err := h.Templates.ExecuteTemplate(w, "history_list.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// StaticHandler returns an http.Handler that serves embedded static files.
func StaticHandler(staticFS fs.FS) http.Handler {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic("failed to create sub filesystem for static: " + err.Error())
	}
	return http.StripPrefix("/static/", http.FileServer(http.FS(sub)))
}
