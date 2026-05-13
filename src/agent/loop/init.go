package loop

import (
	"context"
	"log"
	"os"
	"time"

	"google.golang.org/genai"

	"agentsmith/src/agent/adapter/gemini"
	"agentsmith/src/agent/tools"
	"agentsmith/src/persistence/settings"
)

// Initialize sets up the Gemini client, model registry, background refresh, and the agent.
func Initialize(cfg *settings.Settings, registry *gemini.ModelRegistry) *Agent {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Println("Warning: GEMINI_API_KEY not set. Chat agent will fail.")
		return nil
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Printf("Failed to create genai client: %v\n", err)
		return nil
	}

	// Initial model refresh if needed
	if len(registry.GetModels()) == 0 {
		if err := registry.Refresh(ctx, client); err != nil {
			log.Printf("Warning: initial model refresh failed: %v\n", err)
		}
	}

	// Background refresh loop
	refreshInterval, _ := settings.ParseTimespan(cfg.ModelRefreshInterval)
	if refreshInterval == 0 {
		refreshInterval = time.Hour
	}
	go func() {
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()
		for range ticker.C {
			if err := registry.Refresh(context.Background(), client); err != nil {
				log.Printf("Error refreshing models: %v\n", err)
			}
		}
	}()

	geminiAdapter := gemini.NewAdapter(client, registry, cfg.GeminiRPM)
	dispatcher := tools.NewBasicDispatcher()

	tools.RegisterFSTools(dispatcher)
	tools.RegisterTerminalTools(dispatcher)
	tools.RegisterBrowserTools(dispatcher)
	tools.RegisterMCPTools(dispatcher)

	return NewAgent(geminiAdapter, dispatcher)
}
