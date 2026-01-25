// Package queue handles matchmaking queues
package queue

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/yamlinson/t3/internal/player"
)

// MatchFound represents the information sent to a player when a match is found
// It is primarily used as a means of notifying Bubble Tea of a certain event
type MatchFound struct {
	GameID uuid.UUID
}

// WaitForMatch places a player into the matchmaking queue
func WaitForMatch(p *player.Player) tea.Cmd {
	return func() tea.Msg {
		m := <-p.MatchC
		return MatchFound{GameID: m}
	}
}
