package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type MapRecord struct {
	ID         int
	AudioPath  string
	Path       string
	StarRating float64
	Title      string
}

func FindMap(dbPath string, onlineID int32, legacyID string) (*MapRecord, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `SELECT id, AudioPath, Path, StarRating, Title FROM Maps WHERE OnlineID = ? OR LegacyID = ? LIMIT 1`
	row := db.QueryRow(query, onlineID, legacyID)

	var m MapRecord
	err = row.Scan(&m.ID, &m.AudioPath, &m.Path, &m.StarRating, &m.Title)
	if err != nil {
		return nil, fmt.Errorf("map not found in database: %w", err)
	}

	return &m, nil
}
