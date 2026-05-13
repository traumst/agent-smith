package db

import (
	"database/sql"
	"log"

	"agentsmith/src/persistence/history"
	"agentsmith/src/persistence/logs"
	"agentsmith/src/persistence/memory"
	"agentsmith/src/persistence/refs"
	"agentsmith/src/persistence/vector"
)

// Initialize opens the database, runs migrations, and sets up the memory store.
func Initialize(dbPath, memPath string) (*sql.DB, *memory.Store) {
	sqliteDB, err := InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to init DB: %v\n", err)
	}

	// Run migrations
	tables := []func(*sql.DB) error{
		history.CreateTable,
		logs.CreateTable,
		refs.CreateTable,
		vector.CreateTable,
	}
	for _, create := range tables {
		if err := create(sqliteDB); err != nil {
			log.Fatalf("Failed to create table: %v\n", err)
		}
	}

	memStore, err := memory.NewStore(sqliteDB, memPath)
	if err != nil {
		log.Fatalf("Failed to create memory store: %v\n", err)
	}

	return sqliteDB, memStore
}
