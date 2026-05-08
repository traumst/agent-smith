package refs

import (
	"database/sql"
	"fmt"
	"time"
)

// Reference represents a pointer to a local file or web URL.
type Reference struct {
	ID        int
	URI       string
	Source    string
	CreatedAt time.Time
}

// CreateTable initializes the references table.
func CreateTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS "references" (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		uri TEXT NOT NULL UNIQUE,
		source TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create references table: %w", err)
	}
	return nil
}

// AddRef inserts a new reference and returns its ID.
func AddRef(db *sql.DB, uri, source string) (int, error) {
	query := `INSERT INTO "references" (uri, source, created_at) VALUES (?, ?, ?)
			  ON CONFLICT(uri) DO UPDATE SET source=excluded.source
			  RETURNING id`
	var id int
	err := db.QueryRow(query, uri, source, time.Now().UTC()).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to insert reference: %w", err)
	}
	return id, nil
}
