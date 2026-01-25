// Package player manages the state and lifecycle of players
package player

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/google/uuid"
)

// Player represents a player in or waiting for a game
type Player struct {
	Name    string
	Session ssh.Session
	ID      uuid.UUID
	MatchC  chan MatchInfo
	ReadyC  chan uuid.UUID
}

// MatchInfo contains the information a player needs when notified of a new match
type MatchInfo struct {
	ID        uuid.UUID
	StartTime time.Time
}

// WaitForMatch places a player into the matchmaking queue
func (p *Player) WaitForMatch() tea.Cmd {
	return func() tea.Msg {
		return <-p.MatchC
	}
}
