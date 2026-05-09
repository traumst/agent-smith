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

// SessionSummary represents a brief overview of a chat session.
type SessionSummary struct {
	SessionID    string    `json:"session_id"`
	FirstMessage string    `json:"first_message"`
	MessageCount int       `json:"message_count"`
	CreatedAt    time.Time `json:"created_at"`
}

// ListSessions retrieves a summary of past sessions, ordered by most recent first.
func ListSessions(db *sql.DB, limit, offset int) ([]SessionSummary, error) {
	query := `
		SELECT 
			session_id,
			MIN(created_at) as created_at,
			COUNT(id) as message_count,
			(SELECT content FROM chat_history ch2 WHERE ch2.session_id = ch1.session_id ORDER BY created_at ASC LIMIT 1) as first_message
		FROM chat_history ch1
		GROUP BY session_id
		ORDER BY MAX(created_at) DESC
		LIMIT ? OFFSET ?
	`
	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []SessionSummary
	for rows.Next() {
		var s SessionSummary
		var createdAt string
		if err := rows.Scan(&s.SessionID, &createdAt, &s.MessageCount, &s.FirstMessage); err != nil {
			return nil, fmt.Errorf("failed to scan session summary: %w", err)
		}
		
		// Parse sqlite datetime string
		if t, err := time.Parse("2006-01-02 15:04:05", createdAt); err == nil {
			s.CreatedAt = t
		} else if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			s.CreatedAt = t
		}
		
		// Truncate first message for display
		if len(s.FirstMessage) > 50 {
			s.FirstMessage = s.FirstMessage[:47] + "..."
		}
		
		sessions = append(sessions, s)
	}

	return sessions, nil
}
