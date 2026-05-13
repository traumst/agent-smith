package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"agentsmith/src/agent/adapter/gemini"
	"agentsmith/src/agent/consent"
	"agentsmith/src/agent/loop"
	"agentsmith/src/api"
	"agentsmith/src/persistence/db"
	"agentsmith/src/persistence/settings"
	"agentsmith/src/test"
)

func main() {
	fmt.Println("Agent Smith starting up...")

	settingsPath := "data/settings.json"
	cfg, isTest := settings.Initialize(settingsPath)

	if isTest {
		fmt.Println("Running smoke test...")
		test.RunSmokeTest(cfg)
		return
	}

	fmt.Println("Initializing API server...")

	sqliteDB, memStore := db.Initialize("data/smith.db", "data/memory")
	defer sqliteDB.Close()

	refreshInterval, err := settings.ParseTimespan(cfg.ModelRefreshInterval)
	if err != nil {
		log.Printf("Warning: failed to parse ModelRefreshInterval, using 1h: %v\n", err)
		refreshInterval = time.Hour
	}
	registry := gemini.NewModelRegistry(cfg.ActiveModel, refreshInterval)

	agent := loop.Initialize(cfg, registry)
	pendingConsent := consent.Setup()

	handler := api.NewRouter(sqliteDB, registry, memStore, agent, pendingConsent, settingsPath)

	port := ":8080"
	fmt.Printf("Agent Smith ready at http://localhost%s\n", port)
	if err := http.ListenAndServe(port, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
