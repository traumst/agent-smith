package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"google.golang.org/genai"

	"smithai/src/agent/adapter/gemini"
	"smithai/src/agent/consent"
	"smithai/src/agent/loop"
	"smithai/src/agent/tools"
	"smithai/src/api/handlers"
	"smithai/src/api/middleware"
	"smithai/src/persistence/db"
	"smithai/src/persistence/history"
	"smithai/src/persistence/logs"
	"smithai/src/persistence/memory"
	"smithai/src/persistence/refs"
	"smithai/src/persistence/settings"
	"smithai/src/persistence/vector"
	"smithai/src/test"
	"smithai/src/ui"
)

func main() {
	testFlag := flag.Bool("test", false, "Run the CLI smoke test instead of starting the HTTP server")
	flag.Parse()

	fmt.Println("SmithAI starting up...")

	settingsPath := "data/settings.json"

	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatalf("Failed to create data dir: %v\n", err)
	}

	cfg, err := settings.LoadSettings(settingsPath)
	if err != nil {
		log.Fatalf("Failed to load settings: %v\n", err)
	}

	if err := settings.SaveSettings(settingsPath, cfg); err != nil {
		log.Fatalf("Failed to save settings: %v\n", err)
	}

	if *testFlag {
		fmt.Println("Running smoke test...")
		test.RunSmokeTest(cfg)
		return
	}

	fmt.Println("Initializing API server...")

	dbPath := "data/smith.db"
	sqliteDB, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to init DB: %v\n", err)
	}
	defer sqliteDB.Close()

	if err := history.CreateTable(sqliteDB); err != nil {
		log.Fatalf("Failed to create history table: %v\n", err)
	}
	if err := logs.CreateTable(sqliteDB); err != nil {
		log.Fatalf("Failed to create logs table: %v\n", err)
	}
	if err := refs.CreateTable(sqliteDB); err != nil {
		log.Fatalf("Failed to create refs table: %v\n", err)
	}
	if err := vector.CreateTable(sqliteDB); err != nil {
		log.Fatalf("Failed to create vector table: %v\n", err)
	}

	memStore, err := memory.NewStore(sqliteDB, "data/memory")
	if err != nil {
		log.Fatalf("Failed to create memory store: %v\n", err)
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	var agent *loop.Agent
	if apiKey == "" {
		log.Println("Warning: GEMINI_API_KEY not set. Chat agent will fail.")
	} else {
		ctx := context.Background()
		client, err := genai.NewClient(ctx, nil)
		if err != nil {
			log.Printf("Failed to create genai client: %v\n", err)
		} else {
			registry := gemini.NewModelRegistry()
			geminiAdapter := gemini.NewAdapter(client, registry.Active, cfg.GeminiRPM)
			dispatcher := tools.NewBasicDispatcher()

			tools.RegisterFSTools(dispatcher)
			tools.RegisterTerminalTools(dispatcher)
			tools.RegisterBrowserTools(dispatcher)
			tools.RegisterMCPTools(dispatcher)

			agent = loop.NewAgent(geminiAdapter, dispatcher)
		}
	}

	// Set up web-based consent prompter
	pendingConsent := consent.NewPendingConsent()
	consent.PromptFunc = func(toolName, subject string, args any) (string, error) {
		id := randomID()
		// Broadcast consent request via SSE (the chat handler will pick this up)
		consentReq := map[string]any{
			"id":      id,
			"tool":    toolName,
			"subject": subject,
			"args":    args,
		}
		data, _ := json.Marshal(consentReq)
		// Store in a global so the active SSE stream can read it
		broadcastConsent(string(data))
		// Block until UI responds
		action := pendingConsent.Wait(id)
		return consent.HandleConsentAction(action, subject), nil
	}

	// Load templates
	templates, err := ui.LoadTemplates()
	if err != nil {
		log.Fatalf("Failed to load templates: %v\n", err)
	}

	mux := http.NewServeMux()

	// UI routes
	uiHandler := &handlers.UIHandler{Templates: templates, DB: sqliteDB}
	mux.HandleFunc("GET /", uiHandler.Index)
	mux.Handle("GET /static/", handlers.StaticHandler(ui.StaticFS))
	mux.HandleFunc("GET /ui/history", uiHandler.HistoryList)

	// API routes
	settingsHandler := &handlers.SettingsHandler{Path: settingsPath}
	historyHandler := &handlers.HistoryHandler{DB: sqliteDB}
	memoryHandler := &handlers.MemoryHandler{Store: memStore}
	chatHandler := &handlers.ChatHandler{Agent: agent, DB: sqliteDB, ConsentChan: consentChan}
	consentHandler := &handlers.ConsentHandler{Pending: pendingConsent}

	withTimeout := middleware.Timeout(30 * time.Second)

	mux.Handle("GET /api/settings", withTimeout(http.HandlerFunc(settingsHandler.Get)))
	mux.Handle("POST /api/settings", withTimeout(http.HandlerFunc(settingsHandler.Post)))
	mux.Handle("GET /api/history/{session_id}", withTimeout(http.HandlerFunc(historyHandler.Get)))
	mux.Handle("POST /api/memory", withTimeout(http.HandlerFunc(memoryHandler.Post)))
	mux.Handle("POST /api/consent", withTimeout(http.HandlerFunc(consentHandler.Post)))

	// Chat stream can take a long time, no strict 30s timeout
	mux.Handle("POST /api/chat", http.HandlerFunc(chatHandler.Post))

	var handler http.Handler = mux
	handler = middleware.Recovery(handler)
	handler = middleware.Logging(handler)

	port := ":8080"
	fmt.Printf("SmithAI ready at http://localhost%s\n", port)
	if err := http.ListenAndServe(port, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func randomID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// consentBroadcast holds the latest consent request for SSE pickup.
// Simple approach: one active consent at a time.
var consentChan = make(chan string, 1)

func broadcastConsent(data string) {
	// Non-blocking send — if channel full, drain and resend
	select {
	case consentChan <- data:
	default:
		<-consentChan
		consentChan <- data
	}
}
