// Package queue handles matchmaking queues
package queue

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yamlinson/t3/internal/game"
	"github.com/yamlinson/t3/internal/player"
)

// Matchmaker pairs players into matches
type Matchmaker struct {
	players []player.Player
	mu      sync.Mutex
	results chan matchResult
}

// NewMatchmaker instantiates a new Matchmaker
func NewMatchmaker() *Matchmaker {
	return &Matchmaker{
		players: make([]player.Player, 0),
		results: make(chan matchResult),
	}
}

// AddPlayer adds the given player to a Matchmaker
func (mm *Matchmaker) AddPlayer(p player.Player) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.players = append(mm.players, p)
	if len(mm.players) < 2 {
		return
	}

	p1, p2 := mm.players[0], mm.players[1]
	mm.players = mm.players[2:]

	match := newMatch()

	evt := player.StreamEvent{
		Type: player.Matched,
		Data: map[string]any{
			"MatchID":       match.ID,
			"AcceptTimeout": match.AcceptTimeout,
			"RespondTo":     (chan AcceptMsg)(match.Responses),
		},
	}

	p1.Stream <- evt
	p2.Stream <- evt

	go func() {
		ok := match.waitForResponses()
		mm.results <- matchResult{
			matchID: match.ID,
			players: [2]player.Player{p1, p2},
			ok:      ok,
		}
	}()
}

// WatchResults continuously watches a Matchmaker's Results channel
// As matches are accepted, WatchResults creates games and notifies players
func (mm *Matchmaker) WatchResults() {
	for r := range mm.results {
		p1, p2 := r.players[0], r.players[1]
		var evt player.StreamEvent
		if !r.ok {
			evt = player.StreamEvent{
				Type: player.Declined,
				Data: nil,
			}
		}
		g := game.NewGame(p1, p2)
		evt = player.StreamEvent{
			Type: player.GameReady,
			Data: map[string]any{
				"game": g,
			},
		}
		p1.Stream <- evt
		p2.Stream <- evt
	}
}

// AcceptMsg describes the data Match should receive on Responses
type AcceptMsg struct {
	PlayerID uuid.UUID
	Accepted bool
}

// Match represents the potential matches created by a Matchmaker
type Match struct {
	ID            uuid.UUID
	AcceptTimeout time.Time
	Responses     chan AcceptMsg
}

type matchResult struct {
	matchID uuid.UUID
	players [2]player.Player
	ok      bool
}

func newMatch() *Match {
	return &Match{
		ID:            uuid.New(),
		AcceptTimeout: time.Now().Add(15 * time.Second),
		Responses:     make(chan AcceptMsg, 2),
	}
}

func (m *Match) waitForResponses() bool {
	rcvd := map[uuid.UUID]bool{}

	timer := time.NewTimer(time.Until(m.AcceptTimeout))
	defer timer.Stop()

	for len(rcvd) < 2 {
		select {
		case <-timer.C:
			return false
		case msg := <-m.Responses:
			if msg.Accepted {
				rcvd[msg.PlayerID] = true
			}
		}
	}
	return true
}
