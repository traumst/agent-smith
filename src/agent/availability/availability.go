package availability

import (
	"encoding/csv"
	"fmt"
	"os"
	"sync"
	"time"
)

// Entry represents an item that is currently unavailable.
type Entry struct {
	ItemName string
	ItemType string
	WhyNot   string
	WhenNot  string
}

var (
	mu             sync.RWMutex
	unavailableFile = ".unavailable"
)

// IsAvailable checks if an item is NOT in the .unavailable file.
func IsAvailable(name string) bool {
	mu.RLock()
	defer mu.RUnlock()

	f, err := os.Open(unavailableFile)
	if err != nil {
		return true // File not found means everyone is available
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return true
	}

	for _, record := range records {
		if len(record) > 0 && record[0] == name {
			return false
		}
	}

	return true
}

// MarkUnavailable appends an item to the .unavailable file.
func MarkUnavailable(name, itemType, reason string) error {
	mu.Lock()
	defer mu.Unlock()

	f, err := os.OpenFile(unavailableFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open .unavailable: %w", err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	err = writer.Write([]string{
		name,
		itemType,
		reason,
		time.Now().UTC().Format(time.RFC3339),
	})

	if err != nil {
		return fmt.Errorf("failed to write to .unavailable: %w", err)
	}

	return nil
}
