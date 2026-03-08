// Package db handles database connections and transactions
package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
	_ "github.com/mattn/go-sqlite3"    // SQLite driver
)

// DB wraps sql.DB with application methods
type DB struct {
	*sql.DB
	driver string
}

// Open creates a database connection based on environment
// Uses PostgreSQL if DATABASE_URL is set, otherwise SQLite
func Open() (*DB, error) {
	connStr := os.Getenv("DATABASE_URL")

	if connStr != "" {
		db, err := sql.Open("pgx", connStr)
		if err != nil {
			return nil, fmt.Errorf("open postgres: %w", err)
		}
		return &DB{DB: db, driver: "postgres"}, nil
	}

	// SQLite fallback - uses local file
	dbPath := os.Getenv("SQLITE_PATH")
	if dbPath == "" {
		dbPath = "/data/t3.db"
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite performance tuning
	_, _ = db.Exec("PRAGMA journal_mode=WAL;")
	_, _ = db.Exec("PRAGMA foreign_keys=ON;")

	return &DB{DB: db, driver: "sqlite"}, nil
}

// CreateSchema initializes the database tables
func (db *DB) CreateSchema() error {
	schema := `
		CREATE TABLE IF NOT EXISTS players (
			name VARCHAR(255) PRIMARY KEY,
			wins INTEGER DEFAULT 0,
			draws INTEGER DEFAULT 0,
			losses INTEGER DEFAULT 0,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`

	// SQLite-specific adjustments
	if db.driver == "sqlite" {
		schema = `
			CREATE TABLE IF NOT EXISTS players (
				name TEXT PRIMARY KEY,
				wins INTEGER DEFAULT 0,
				draws INTEGER DEFAULT 0,
				losses INTEGER DEFAULT 0,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		`
	}

	_, err := db.Exec(schema)
	return err
}

// GetTopPlayers returns the top 3 players by win count
func (db *DB) GetTopPlayers() ([]Player, error) {
	rows, err := db.Query(`
		SELECT name, wins, draws, losses
		FROM players
		ORDER BY wins DESC, draws DESC, losses ASC
		LIMIT 3;
		`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []Player

	for rows.Next() {
		var p Player
		if err := rows.Scan(&p.Name, &p.Wins, &p.Draws, &p.Losses); err != nil {
			return players, err
		}
		players = append(players, p)
	}
	if err = rows.Err(); err != nil {
		return players, err
	}
	return players, nil
}

// Close wraps the underlying db.Close
func (db *DB) Close() error {
	return db.DB.Close()
}
