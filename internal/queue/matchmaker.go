// Package queue handles matchmaking queues
package queue

import (
	"sync"

	"github.com/yamlinson/t3/internal/game"
	"github.com/yamlinson/t3/internal/player"
)

// Matchmaker pairs players into matches
type Matchmaker struct {
	players []player.Player
	mu      sync.Mutex
}

// NewMatchmaker instantiates a new Matchmaker
func NewMatchmaker() *Matchmaker {
	return &Matchmaker{
		players: make([]player.Player, 0),
	}
}

// AddPlayer adds the given player to a Matchmaker
func (m *Matchmaker) AddPlayer(p player.Player) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.players = append(m.players, p)

	if len(m.players) >= 2 {
		p1, p2 := m.players[0], m.players[1]

		g := game.NewGame(p1, p2)

		p1.MatchC <- g.ID
		p2.MatchC <- g.ID

		m.players = m.players[2:]
	}
}
