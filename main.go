package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"google.golang.org/genai"

	"smithai/src/agent/adapter/gemini"
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
			geminiAdapter := gemini.NewAdapter(client, registry.Models[0].Stable, cfg.GeminiRPM)
			dispatcher := tools.NewBasicDispatcher()

			tools.RegisterFSTools(dispatcher)
			tools.RegisterTerminalTools(dispatcher)
			tools.RegisterBrowserTools(dispatcher)
			tools.RegisterMCPTools(dispatcher)

			agent = loop.NewAgent(geminiAdapter, dispatcher)
		}
	}

	mux := http.NewServeMux()

	settingsHandler := &handlers.SettingsHandler{Path: settingsPath}
	historyHandler := &handlers.HistoryHandler{DB: sqliteDB}
	memoryHandler := &handlers.MemoryHandler{Store: memStore}
	chatHandler := &handlers.ChatHandler{Agent: agent, DB: sqliteDB}

	withTimeout := middleware.Timeout(30 * time.Second)

	mux.Handle("GET /api/settings", withTimeout(http.HandlerFunc(settingsHandler.Get)))
	mux.Handle("POST /api/settings", withTimeout(http.HandlerFunc(settingsHandler.Post)))
	mux.Handle("GET /api/history/{session_id}", withTimeout(http.HandlerFunc(historyHandler.Get)))
	mux.Handle("POST /api/memory", withTimeout(http.HandlerFunc(memoryHandler.Post)))

	// Chat stream can take a long time, no strict 30s timeout
	mux.Handle("POST /api/chat", http.HandlerFunc(chatHandler.Post))

	var handler http.Handler = mux
	handler = middleware.Recovery(handler)
	handler = middleware.Logging(handler)

	port := ":8080"
	fmt.Printf("Listening on %s\n", port)
	if err := http.ListenAndServe(port, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
