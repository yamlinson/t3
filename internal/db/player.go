// Package db handles database connections and transactions
package db

import (
	"database/sql"
	"fmt"
	"time"
)

// Player describes a player entry in the database
type Player struct {
	Name      string
	Wins      int
	Draws     int
	Losses    int
	UpdatedAt time.Time
}

// MatchOutcome describes a player's match result
type MatchOutcome int

// Win, Draw, and Loss are the possible outcomes of a match
const (
	Win MatchOutcome = iota
	Draw
	Loss
)

// RecordMatch updates player stats, creating the player if new
func (db *DB) RecordMatch(players []string, outcome MatchOutcome) error {
	if len(players) != 2 {
		return fmt.Errorf("expected 2 players, got %d", len(players))
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Both players get same outcome for draws
	if outcome == Draw {
		for _, name := range players {
			if err := db.recordInTx(tx, name, outcome); err != nil {
				return err
			}
		}
		return tx.Commit()
	}

	// Win/Loss: first player wins, second loses
	if err := db.recordInTx(tx, players[0], Win); err != nil {
		return err
	}
	if err := db.recordInTx(tx, players[1], Loss); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) recordInTx(tx *sql.Tx, name string, outcome MatchOutcome) error {
	var wins, draws, losses int
	switch outcome {
	case Win:
		wins = 1
	case Draw:
		draws = 1
	case Loss:
		losses = 1
	}

	now := time.Now()

	if db.driver == "sqlite" {
		_, err := tx.Exec(`
			INSERT INTO players (name, wins, draws, losses, updated_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (name) DO UPDATE SET
				wins = players.wins + ?,
				draws = players.draws + ?,
				losses = players.losses + ?,
				updated_at = ?`,
			name, wins, draws, losses, now,
			wins, draws, losses, now)
		return err
	}

	_, err := tx.Exec(`
		INSERT INTO players (name, wins, draws, losses, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (name) DO UPDATE SET
			wins = players.wins + $2,
			draws = players.draws + $3,
			losses = players.losses + $4,
			updated_at = $5`,
		name, wins, draws, losses, now)
	return err
}
