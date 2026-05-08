package memory

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"smithai/internal/persistence/refs"
	"smithai/internal/persistence/vector"
)

// Store represents the long-term memory store.
type Store struct {
	dir string
	db  *sql.DB
}

// NewStore creates a new memory store that persists to the given directory.
func NewStore(db *sql.DB, dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create memory directory: %w", err)
	}
	return &Store{
		dir: dir,
		db:  db,
	}, nil
}

// SaveMemory writes a plaintext memory to disk, registers it, and inserts a dummy vector.
func (s *Store) SaveMemory(filename string, content string) error {
	path := filepath.Join(s.dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write memory file: %w", err)
	}

	refID, err := refs.AddRef(s.db, path, "local_memory")
	if err != nil {
		return fmt.Errorf("failed to add reference: %w", err)
	}

	// For Phase 1 dummy test, we insert a zero vector.
	// In reality, we'd extract keywords and call an embedding API.
	dummyVec := make([]float32, 1536)
	dummyVec[0] = 1.0 // Make it slightly non-zero

	if err := vector.UpsertVector(s.db, refID, dummyVec); err != nil {
		return fmt.Errorf("failed to upsert memory vector: %w", err)
	}

	return nil
}
