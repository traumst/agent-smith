package settings

import (
	"flag"
	"log"
	"os"
)

// Initialize handles flag parsing, data directory creation, and settings loading/saving.
func Initialize(path string) (*Settings, bool) {
	testFlag := flag.Bool("test", false, "Run the CLI smoke test instead of starting the HTTP server")
	flag.Parse()

	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatalf("Failed to create data dir: %v\n", err)
	}

	cfg, err := LoadSettings(path)
	if err != nil {
		log.Fatalf("Failed to load settings: %v\n", err)
	}

	if err := SaveSettings(path, cfg); err != nil {
		log.Fatalf("Failed to save settings: %v\n", err)
	}

	return cfg, *testFlag
}
