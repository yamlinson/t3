// Package queue handles matchmaking queues
package queue

import (
	"sync"

	"github.com/charmbracelet/ssh"
	"github.com/google/uuid"
)

// Player represents a player waiting in queue
type Player struct {
	Name    string
	Session ssh.Session
	ID      uuid.UUID
	MatchC  chan Match
}

// Match represents the information sent to a player when a match is found
type Match struct {
	Opponent Player
}

// Matchmaker pairs players into matches
type Matchmaker struct {
	players []Player
	mu      sync.Mutex
}

// NewMatchmaker instantiates a new Matchmaker
func NewMatchmaker() *Matchmaker {
	return &Matchmaker{
		players: make([]Player, 0),
	}
}

// AddPlayer adds the given player to a Matchmaker
func (m *Matchmaker) AddPlayer(p Player) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.players = append(m.players, p)

	if len(m.players) >= 2 {
		p1, p2 := m.players[0], m.players[1]

		p1.MatchC <- Match{
			Opponent: p2,
		}
		p2.MatchC <- Match{
			Opponent: p1,
		}

		m.players = m.players[2:]
	}
}
