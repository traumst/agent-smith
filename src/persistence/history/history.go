package history

import (
	"database/sql"
	"fmt"
	"time"

	"agentsmith/src/agent/protocol"
)

// CreateTable initializes the chat_history table.
func CreateTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS chat_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		model TEXT,
		tokens_used INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_chat_history_session ON chat_history(session_id);
	`
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create chat_history table: %w", err)
	}

	// Migration: add model column if it doesn't exist (for existing databases)
	// We ignore error here because if column exists it will fail
	_, _ = db.Exec("ALTER TABLE chat_history ADD COLUMN model TEXT")
	_, _ = db.Exec("ALTER TABLE chat_history ADD COLUMN tokens_used INTEGER")

	return nil
}

// AddMessage inserts a new message into the history.
func AddMessage(db *sql.DB, sessionID string, msg protocol.Message) error {
	query := `INSERT INTO chat_history (session_id, role, content, model, tokens_used, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := db.Exec(query, sessionID, msg.Role, msg.Content, msg.Model, msg.TokensUsed, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}
	return nil
}

// GetHistory retrieves the chat history for a given session.
func GetHistory(db *sql.DB, sessionID string) ([]protocol.Message, error) {
	query := `SELECT role, content, model, tokens_used, created_at FROM chat_history WHERE session_id = ? ORDER BY created_at ASC`
	rows, err := db.Query(query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}
	defer rows.Close()

	var history []protocol.Message
	for rows.Next() {
		var msg protocol.Message
		var model sql.NullString
		var createdAt string
		var tokensUsed sql.NullInt64
		if err := rows.Scan(&msg.Role, &msg.Content, &model, &tokensUsed, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		if model.Valid {
			msg.Model = model.String
		}
		if tokensUsed.Valid {
			msg.TokensUsed = int(tokensUsed.Int64)
		}
		msg.Timestamp = createdAt
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

// DeleteSession removes all messages associated with a session.
func DeleteSession(db *sql.DB, sessionID string) error {
	query := `DELETE FROM chat_history WHERE session_id = ?`
	_, err := db.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// BranchSession clones a session up to a certain message index (0-based) and adds a new message.
func BranchSession(db *sql.DB, oldSessionID, newSessionID string, upToIdx int, newMsg protocol.Message) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Get history for old session
	query := `SELECT role, content, model, tokens_used, created_at FROM chat_history WHERE session_id = ? ORDER BY created_at ASC`
	rows, err := tx.Query(query, oldSessionID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var history []protocol.Message
	for rows.Next() {
		var msg protocol.Message
		var model sql.NullString
		var createdAt string
		var tokensUsed sql.NullInt64
		if err := rows.Scan(&msg.Role, &msg.Content, &model, &tokensUsed, &createdAt); err != nil {
			return err
		}
		if model.Valid {
			msg.Model = model.String
		}
		if tokensUsed.Valid {
			msg.TokensUsed = int(tokensUsed.Int64)
		}
		history = append(history, msg)
	}

	// 2. Insert messages up to upToIdx into new session
	insertQuery := `INSERT INTO chat_history (session_id, role, content, model, tokens_used, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	now := time.Now().UTC()
	for i := 0; i < upToIdx && i < len(history); i++ {
		msg := history[i]
		// We use slightly incremented times to preserve order in case of identical timestamps
		_, err := tx.Exec(insertQuery, newSessionID, msg.Role, msg.Content, msg.Model, msg.TokensUsed, now.Add(time.Duration(i)*time.Millisecond))
		if err != nil {
			return err
		}
	}

	// 3. Insert the new/edited message
	_, err = tx.Exec(insertQuery, newSessionID, newMsg.Role, newMsg.Content, newMsg.Model, newMsg.TokensUsed, now.Add(time.Duration(upToIdx)*time.Millisecond))
	if err != nil {
		return err
	}

	return tx.Commit()
}
