package vector

import (
	"database/sql"
	"fmt"

	"github.com/asg017/sqlite-vec-go-bindings/cgo"
)

// CreateTable initializes the virtual table for vector lookups.
func CreateTable(db *sql.DB) error {
	// The vector table uses vec0. We assume an embedding size of 1536 (e.g. OpenAI).
	query := `
	CREATE VIRTUAL TABLE IF NOT EXISTS vector_memory USING vec0(
		ref_id INTEGER PRIMARY KEY,
		embedding float[1536]
	);
	`
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create vector_memory table: %w", err)
	}
	return nil
}

// UpsertVector inserts or updates a vector associated with a reference ID.
func UpsertVector(db *sql.DB, refID int, embedding []float32) error {
	vecBytes, err := vec.SerializeFloat32(embedding)
	if err != nil {
		return fmt.Errorf("failed to serialize float32 vector: %w", err)
	}

	// sqlite-vec doesn't support UPSERT, so we DELETE then INSERT.
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM vector_memory WHERE ref_id = ?", refID); err != nil {
		return fmt.Errorf("failed to delete existing vector: %w", err)
	}

	if _, err := tx.Exec("INSERT INTO vector_memory(ref_id, embedding) VALUES (?, ?)", refID, vecBytes); err != nil {
		return fmt.Errorf("failed to insert vector: %w", err)
	}

	return tx.Commit()
}

// SearchResult represents a single match from a vector search.
type SearchResult struct {
	RefID    int
	Distance float64
}

// Search performs a similarity search.
func Search(db *sql.DB, queryVec []float32, limit int) ([]SearchResult, error) {
	vecBytes, err := vec.SerializeFloat32(queryVec)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize float32 vector: %w", err)
	}

	// We use the distance parameter to sort by nearest neighbors.
	query := `
		SELECT ref_id, distance
		FROM vector_memory
		WHERE embedding MATCH ? AND k = ?
		ORDER BY distance
	`
	rows, err := db.Query(query, vecBytes, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search vector table: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var res SearchResult
		if err := rows.Scan(&res.RefID, &res.Distance); err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		results = append(results, res)
	}

	return results, nil
}
