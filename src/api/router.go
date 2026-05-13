package api

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"agentsmith/src/agent/adapter/gemini"
	"agentsmith/src/agent/consent"
	"agentsmith/src/agent/loop"
	"agentsmith/src/api/handlers"
	"agentsmith/src/api/middleware"
	"agentsmith/src/persistence/memory"
	"agentsmith/src/ui"
)

// NewRouter creates the main HTTP multiplexer and wires up all UI and API routes.
func NewRouter(sqliteDB *sql.DB, registry *gemini.ModelRegistry, memStore *memory.Store, agent *loop.Agent, pendingConsent *consent.PendingConsent, settingsPath string) http.Handler {
	templates, err := ui.LoadTemplates()
	if err != nil {
		log.Fatalf("Failed to load templates: %v\n", err)
	}

	mux := http.NewServeMux()

	// UI routes
	uiHandler := &handlers.UIHandler{
		Templates:    templates,
		DB:           sqliteDB,
		Registry:     registry,
		SettingsPath: settingsPath,
	}
	mux.HandleFunc("GET /", uiHandler.Index)
	mux.Handle("GET /static/", handlers.StaticHandler(ui.StaticFS))
	mux.HandleFunc("GET /ui/history", uiHandler.HistoryList)
	mux.HandleFunc("GET /ui/settings", uiHandler.Settings)
	mux.HandleFunc("POST /ui/delete", uiHandler.DeleteChat)
	mux.HandleFunc("POST /ui/branch", uiHandler.BranchChat)

	// API routes
	settingsHandler := &handlers.SettingsHandler{Path: settingsPath, Registry: registry}
	historyHandler := &handlers.HistoryHandler{DB: sqliteDB}
	memoryHandler := &handlers.MemoryHandler{Store: memStore}
	chatHandler := &handlers.ChatHandler{Agent: agent, DB: sqliteDB, ConsentChan: consent.Chan}
	consentHandler := &handlers.ConsentHandler{Pending: pendingConsent}

	withTimeout := middleware.Timeout(30 * time.Second)

	mux.Handle("GET /api/settings", withTimeout(http.HandlerFunc(settingsHandler.Get)))
	mux.Handle("POST /api/settings", withTimeout(http.HandlerFunc(settingsHandler.Post)))
	mux.Handle("GET /api/history/{session_id}", withTimeout(http.HandlerFunc(historyHandler.Get)))
	mux.Handle("POST /api/memory", withTimeout(http.HandlerFunc(memoryHandler.Post)))
	mux.Handle("POST /api/consent", withTimeout(http.HandlerFunc(consentHandler.Post)))
	mux.Handle("POST /api/chat", http.HandlerFunc(chatHandler.Post))

	var handler http.Handler = mux
	handler = middleware.Recovery(handler)
	handler = middleware.Logging(handler)
	return handler
}
