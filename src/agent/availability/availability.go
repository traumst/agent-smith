package availability

import (
	"encoding/csv"
	"fmt"
	"os"
	"sync"
	"time"
)

// Entry represents an item that is currently unavailable or available.
type Entry struct {
	ItemName string
	ItemType string
	Reason   string
	Time     time.Time
}

var (
	mu sync.RWMutex
	// TODO these files contents should be stored in-memory while app is live
	unavailableFile = ".unavailable"
	availableFile   = ".available"
)

// IsAvailable checks if an item is NOT in the .unavailable file,
// or if it has been there for more than 4 hours.
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
			// Found in unavailable list. Check time.
			if len(record) >= 4 {
				t, err := time.Parse(time.RFC3339, record[3])
				if err == nil {
					// If it's been more than 4 hours, it's available again
					if time.Since(t) > 4*time.Hour {
						return true
					}
				}
			} else {
				fmt.Printf("XXX record in %s: %s\n", unavailableFile, record)
			}
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

// MarkAvailable appends an item to the .available file.
func MarkAvailable(name, itemType, reason string) error {
	mu.Lock()
	defer mu.Unlock()

	// Check if already in .available to avoid duplicates
	if exists(availableFile, name) {
		return nil
	}

	f, err := os.OpenFile(availableFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open .available: %w", err)
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
		return fmt.Errorf("failed to write to .available: %w", err)
	}

	return nil
}

// GetAvailable returns all items from the .available file.
func GetAvailable() ([]Entry, error) {
	mu.RLock()
	defer mu.RUnlock()

	f, err := os.Open(availableFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, fmt.Errorf("failed to open .available: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read .available: %w", err)
	}

	var entries []Entry
	for _, record := range records {
		if len(record) < 4 {
			continue
		}
		t, _ := time.Parse(time.RFC3339, record[3])
		entries = append(entries, Entry{
			ItemName: record[0],
			ItemType: record[1],
			Reason:   record[2],
			Time:     t,
		})
	}

	return entries, nil
}

// exists checks if an item is already in a file (internal use).
func exists(filename, name string) bool {
	f, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return false
	}

	for _, record := range records {
		if len(record) > 0 && record[0] == name {
			return true
		}
	}

	return false
}
