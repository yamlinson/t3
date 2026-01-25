// Package game manages the state and lifecycle of games
package game

import (
	"time"

	"github.com/google/uuid"
	"github.com/yamlinson/t3/internal/player"
)

// Game represents the state of one game
type Game struct {
	ID        uuid.UUID
	Players   [2]player.Player
	StartTime time.Time
}

// NewGame creates a new game with the given pair of players
func NewGame(p1 player.Player, p2 player.Player) *Game {
	return &Game{
		ID:        uuid.New(),
		Players:   [2]player.Player{p1, p2},
		StartTime: time.Now().Add(15 * time.Second),
	}
}
