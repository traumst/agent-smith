package test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"google.golang.org/genai"

	"agentsmith/src/agent/adapter/gemini"
	"agentsmith/src/agent/loop"
	"agentsmith/src/agent/protocol"
	"agentsmith/src/agent/tools"
	"agentsmith/src/persistence/db"
	"agentsmith/src/persistence/history"
	"agentsmith/src/persistence/logs"
	"agentsmith/src/persistence/memory"
	"agentsmith/src/persistence/refs"
	"agentsmith/src/persistence/settings"
	"agentsmith/src/persistence/vector"
)

// RunSmokeTest executes the Phase 2 test flow.
func RunSmokeTest(cfg *settings.Settings) {
	fmt.Printf("Configured Mood: %s\n", cfg.SystemPrompt.Mood)

	selectedModel := "models/gemini-2.5-flash-lite"
	fmt.Printf("grug use model: %s\n\n", selectedModel)

	req := protocol.Request{
		SystemPrompt: cfg.SystemPrompt,
		UserPrompt:   "Hello! Please perform the following tasks:\n1. Write a file called 'test.txt' with the content 'hello world'.\n2. Run the terminal command 'ls'.\n3. Summarize the web page 'https://htmx.org/attributes/hx-get/'.\n4. Ping the MCP server to verify integration.",
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
			registry := gemini.NewModelRegistry(selectedModel, time.Hour)
			geminiAdapter := gemini.NewAdapter(client, registry, cfg.GeminiRPM)
			dispatcher := tools.NewBasicDispatcher()

			tools.RegisterFSTools(dispatcher)
			tools.RegisterTerminalTools(dispatcher)
			tools.RegisterBrowserTools(dispatcher)
			tools.RegisterMCPTools(dispatcher)

			agent := loop.NewAgent(geminiAdapter, dispatcher)
			fmt.Println("--- Starting Agent Loop ---")

			stream, err := agent.Run(ctx, &req)
			if err != nil {
				fmt.Printf("Agent Run failed: %v\n", err)
			} else {
				var fullOutput strings.Builder
				for resp := range stream {
					if resp.Error != nil {
						fmt.Printf("\n[Error from stream]: %v\n", resp.Error)
						break
					}
					if resp.Done {
						fmt.Printf("\n[Stream Complete. Tokens used: %d]\n", resp.TokensUsed)
						break
					}
					if resp.Content != "" {
						fmt.Printf("[LLM response] %s\n", resp.Content)
						fullOutput.WriteString(resp.Content)
					}
				}

				// Assert browser tool actually loaded the page
				if strings.Contains(strings.ToLower(fullOutput.String()), "hx-get") {
					fmt.Println("\n[PASS] browser_fetch: page loaded, hx-get found")
				} else {
					fmt.Println("\n[FAIL] browser_fetch: hx-get NOT found in output")
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

	_, err = history.GetHistory(sqliteDB, sessionID)
	if err != nil {
		fmt.Printf("Failed to get history: %v\n", err)
		return
	}

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

	if err := memStore.SaveMemory("test_memory.txt", "Agent Smith is an AI agent built in Go."); err != nil {
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
