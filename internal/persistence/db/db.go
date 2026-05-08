package db

import (
	"database/sql"
	"fmt"

	vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
)

// InitDB opens a SQLite database connection and sets appropriate pragmas.
func InitDB(dsn string) (*sql.DB, error) {
	vec.Auto()
	// Add pragmas to connection string for WAL mode and foreign keys
	db, err := sql.Open("sqlite3", dsn+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
