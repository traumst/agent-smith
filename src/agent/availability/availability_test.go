package availability

import (
	"os"
	"testing"
	"time"
)

func TestAvailability(t *testing.T) {
	// Cleanup
	os.Remove(".available")
	os.Remove(".unavailable")
	defer os.Remove(".available")
	defer os.Remove(".unavailable")

	name := "test-item"
	
	// Initially available
	if !IsAvailable(name) {
		t.Errorf("Expected %s to be available initially", name)
	}

	// Mark unavailable
	err := MarkUnavailable(name, "test", ReasonUnknownError)
	if err != nil {
		t.Fatalf("MarkUnavailable failed: %v", err)
	}

	if IsAvailable(name) {
		t.Errorf("Expected %s to be unavailable after marking", name)
	}

	// Wait 4 hours is hard to test without mocking time, 
	// but let's try to overwrite the file with an old timestamp.
	MarkUnavailable(name, "test", ReasonUnknownError) // This appends
	
	// Manual overwrite to simulate 5 hours ago
	f, _ := os.Create(".unavailable")
	oldTime := time.Now().Add(-5 * time.Hour).UTC().Format(time.RFC3339)
	f.WriteString(name + ",test,resource_exhausted," + oldTime + "\n")
	f.Close()

	if !IsAvailable(name) {
		t.Errorf("Expected %s to be available again after 5 hours", name)
	}

	// Test MarkAvailable
	err = MarkAvailable(name, "test", ReasonDiscovered)
	if err != nil {
		t.Fatalf("MarkAvailable failed: %v", err)
	}

	available, err := GetAvailable()
	if err != nil {
		t.Fatalf("GetAvailable failed: %v", err)
	}

	found := false
	for _, entry := range available {
		if entry.ItemName == name {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected %s to be in .available", name)
	}
}
