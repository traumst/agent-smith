package test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

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

// RunSmokeTest executes the Phase 2 test flow.
func RunSmokeTest(cfg *settings.Settings) {
	fmt.Printf("Configured Mood: %s\n", cfg.SystemPrompt.Mood)

	registry := gemini.NewModelRegistry(time.Hour)
	selectedModel := selectModelInteractive(registry.Models)
	registry.SetActive(selectedModel)
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
			geminiAdapter := gemini.NewAdapter(client, selectedModel, cfg.GeminiRPM)
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

func selectModelInteractive(models []gemini.ModelTier) string {
	// Save current stty state
	cmd := exec.Command("stty", "-g")
	cmd.Stdin = os.Stdin
	state, err := cmd.Output()
	if err != nil {
		// Fallback if stty not available
		return models[0].Stable
	}

	// Set raw mode
	cmdRaw := exec.Command("stty", "-icanon", "-echo")
	cmdRaw.Stdin = os.Stdin
	cmdRaw.Run()

	// Restore original state
	defer func() {
		cmdRestore := exec.Command("stty", strings.TrimSpace(string(state)))
		cmdRestore.Stdin = os.Stdin
		cmdRestore.Run()
	}()

	// Hide cursor
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h")

	selectedIndex := 0

	for {
		// Clear current line
		fmt.Print("\r\033[K")
		fmt.Println("\npick model (up/down arrows, space/enter to select):")
		for i, m := range models {
			if i == selectedIndex {
				fmt.Printf("\033[K> %s\n", m.Name)
			} else {
				fmt.Printf("\033[K  %s\n", m.Name)
			}
		}

		b := make([]byte, 3)
		os.Stdin.Read(b)

		if b[0] == '\n' || b[0] == '\r' || b[0] == ' ' {
			// Clear the menu lines we drew
			fmt.Printf("\033[%dA\033[J", len(models)+2)
			return models[selectedIndex].Stable
		}

		if b[0] == 27 && b[1] == '[' {
			if b[2] == 'A' {
				selectedIndex--
				if selectedIndex < 0 {
					selectedIndex = len(models) - 1
				}
			} else if b[2] == 'B' {
				selectedIndex++
				if selectedIndex >= len(models) {
					selectedIndex = 0
				}
			}
		} else if b[0] == 3 {
			fmt.Printf("\033[%dA\033[J", len(models)+2)
			fmt.Print("\033[?25h")
			os.Exit(1)
		}

		fmt.Printf("\033[%dA", len(models)+2)
	}
}
