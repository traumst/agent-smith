package main

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/genai"

	"smithai/src/agent/adapter/gemini"
	"smithai/src/agent/loop"
	"smithai/src/agent/protocol"
	"smithai/src/agent/tools"
	"smithai/src/persistence/db"
	"smithai/src/persistence/history"
	"smithai/src/persistence/logs"
	"smithai/src/persistence/memory"
	"smithai/src/persistence/refs"
	"smithai/src/persistence/settings"
	"smithai/src/persistence/vector"
)

func main() {
	fmt.Println("SmithAI starting up...")

	settingsPath := "data/settings.json"

	// Ensure data directory exists
	if err := os.MkdirAll("data", 0755); err != nil {
		fmt.Printf("Failed to create data dir: %v\n", err)
		return
	}

	cfg, err := settings.LoadSettings(settingsPath)
	if err != nil {
		fmt.Printf("Failed to load settings: %v\n", err)
		return
	}

	// Try saving to ensure it writes correctly
	if err := settings.SaveSettings(settingsPath, cfg); err != nil {
		fmt.Printf("Failed to save settings: %v\n", err)
		return
	}

	fmt.Printf("Configured Mood: %s\n", cfg.SystemPrompt.Mood)

	req := protocol.Request{
		SystemPrompt: cfg.SystemPrompt,
		UserPrompt:   "Hello! Please use the 'dummy_test_tool' to say 'hello world'.",
		Stream:       true,
	}

	fmt.Printf("Dummy request created: %+v\n", req)

	// Phase 2: Smoke Test
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("Warning: GEMINI_API_KEY not set, skipping Phase 2 Agent Loop test.")
	} else {
		ctx := context.Background()
		client, err := genai.NewClient(ctx, nil) // Uses GEMINI_API_KEY env var automatically
		if err != nil {
			fmt.Printf("Failed to create genai client: %v\n", err)
		} else {
			geminiAdapter := gemini.NewAdapter(client, "gemini-2.5-flash")
			dispatcher := tools.NewBasicDispatcher()

			// Register Dummy Tool
			dispatcher.Register(protocol.ToolDef{
				Name:        "dummy_test_tool",
				Description: "A dummy tool to test tool invocation.",
				// We pass a very simple generic object for arguments schema (though empty works for tests often)
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"message": map[string]any{
							"type": "string",
						},
					},
				},
			}, func(ctx context.Context, args any) (string, error) {
				return fmt.Sprintf("Dummy tool executed with args: %+v", args), nil
			})

			agent := loop.NewAgent(geminiAdapter, dispatcher)
			fmt.Println("\n--- Starting Agent Loop ---")
			
			stream, err := agent.Run(ctx, &req)
			if err != nil {
				fmt.Printf("Agent Run failed: %v\n", err)
			} else {
				for resp := range stream {
					if resp.Error != nil {
						fmt.Printf("\n[Error from stream]: %v\n", resp.Error)
						break
					}
					if resp.Done {
						fmt.Printf("\n[Stream Complete. Tokens used: %d]\n", resp.TokensUsed)
						break
					}
					// Print text chunk
					if resp.Content != "" {
						fmt.Print(resp.Content)
					}
				}
			}
			fmt.Println("\n--- End of Agent Loop ---")
		}
	}

	// SQLite Testing
	dbPath := "data/smith.db"
	sqliteDB, err := db.InitDB(dbPath)
	if err != nil {
		fmt.Printf("Failed to init DB: %v\n", err)
		return
	}
	defer sqliteDB.Close()

	if err := history.CreateTable(sqliteDB); err != nil {
		fmt.Printf("Failed to create history table: %v\n", err)
		return
	}
	if err := logs.CreateTable(sqliteDB); err != nil {
		fmt.Printf("Failed to create logs table: %v\n", err)
		return
	}
	if err := refs.CreateTable(sqliteDB); err != nil {
		fmt.Printf("Failed to create refs table: %v\n", err)
		return
	}

	// Test history insert
	sessionID := "test-session-123"
	if err := history.AddMessage(sqliteDB, sessionID, protocol.Message{Role: "user", Content: "Hello from DB test!"}); err != nil {
		fmt.Printf("Failed to add message: %v\n", err)
		return
	}

	hist, err := history.GetHistory(sqliteDB, sessionID)
	if err != nil {
		fmt.Printf("Failed to get history: %v\n", err)
		return
	}
	fmt.Printf("Retrieved History: %+v\n", hist)

	if err := vector.CreateTable(sqliteDB); err != nil {
		fmt.Printf("Failed to create vector table: %v\n", err)
		return
	}

	// Test Memory & Vector
	memStore, err := memory.NewStore(sqliteDB, "data/memory")
	if err != nil {
		fmt.Printf("Failed to create memory store: %v\n", err)
		return
	}

	if err := memStore.SaveMemory("test_memory.txt", "Smith is an AI agent built in Go."); err != nil {
		fmt.Printf("Failed to save memory: %v\n", err)
		return
	}

	// Search for dummy vector (all zeros with first element 1.0)
	queryVec := make([]float32, 1536)
	queryVec[0] = 1.0
	results, err := vector.Search(sqliteDB, queryVec, 5)
	if err != nil {
		fmt.Printf("Failed to search vector table: %v\n", err)
		return
	}
	fmt.Printf("Vector Search Results: %+v\n", results)
}
