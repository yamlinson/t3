// Package player manages the state and lifecycle of players
package player

import (
	"github.com/charmbracelet/ssh"
	"github.com/google/uuid"
)

// Player represents a player in or waiting for a game
type Player struct {
	Name    string
	Session ssh.Session
	ID      uuid.UUID
	MatchC  chan uuid.UUID
}
