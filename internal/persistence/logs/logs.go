package logs

import (
	"database/sql"
	"fmt"
	"time"
)

// CreateTable initializes the usage_logs table.
func CreateTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS usage_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type TEXT NOT NULL,
		details TEXT NOT NULL,
		tokens_used INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create usage_logs table: %w", err)
	}
	return nil
}

// AddLog inserts a new usage log entry.
func AddLog(db *sql.DB, eventType, details string, tokensUsed int) error {
	query := `INSERT INTO usage_logs (event_type, details, tokens_used, created_at) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, eventType, details, tokensUsed, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to insert usage log: %w", err)
	}
	return nil
}
