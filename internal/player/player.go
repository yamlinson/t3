// Package player manages the state and lifecycle of players
package player

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/google/uuid"
)

// Player represents a player in or waiting for a game
type Player struct {
	Name    string
	Session ssh.Session
	ID      uuid.UUID
	Stream  chan StreamEvent
}

// StreamEvent contains the information a player needs when notified of an event on their stream
type StreamEvent struct {
	Type EventType
	Data map[string]any
}

// EventType defines the Types which can be associated with an Event
type EventType int

// Queued, Matched, Accepted, Declined, and GameReady
// describe possible events a player might be notified of
const (
	Queued EventType = iota
	Matched
	Accepted
	Declined
	GameReady
)

// WaitForEvent listens for an event on the player's stream
func (p *Player) WaitForEvent() tea.Cmd {
	return func() tea.Msg {
		return <-p.Stream
	}
}
