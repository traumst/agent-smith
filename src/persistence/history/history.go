package history

import (
	"database/sql"
	"fmt"
	"time"

	"smithai/src/agent/protocol"
)

// CreateTable initializes the chat_history table.
func CreateTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS chat_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_chat_history_session ON chat_history(session_id);
	`
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create chat_history table: %w", err)
	}
	return nil
}

// AddMessage inserts a new message into the history.
func AddMessage(db *sql.DB, sessionID string, msg protocol.Message) error {
	query := `INSERT INTO chat_history (session_id, role, content, created_at) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, sessionID, msg.Role, msg.Content, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}
	return nil
}

// GetHistory retrieves the chat history for a given session.
func GetHistory(db *sql.DB, sessionID string) ([]protocol.Message, error) {
	query := `SELECT role, content FROM chat_history WHERE session_id = ? ORDER BY created_at ASC`
	rows, err := db.Query(query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}
	defer rows.Close()

	var history []protocol.Message
	for rows.Next() {
		var msg protocol.Message
		if err := rows.Scan(&msg.Role, &msg.Content); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		history = append(history, msg)
	}

	return history, nil
}
