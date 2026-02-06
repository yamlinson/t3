// Package game manages the state and lifecycle of games
package game

import (
	"math/rand/v2"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yamlinson/t3/internal/player"
)

// Game represents the state of one game
type Game struct {
	mu      sync.Mutex
	ID      uuid.UUID
	Players [2]*player.Player
	Board   [9]rune
	Turns   map[*player.Player]rune
	Next    *player.Player
	Stream  chan StreamEvent
}

// StreamEvent contains the information sent to a game over its lifecycle
type StreamEvent struct {
	Type EventType
	Data map[string]any
}

// EventType defines the Types which can be associated with a StreamEvent
type EventType int

// PlayerTurn, PlayerQuit
// describe possible events a game might be notified of
const (
	PlayerTurn EventType = iota
	PlayerQuit
)

// NewGame creates a new game with the given pair of players
func NewGame(p1 *player.Player, p2 *player.Player) *Game {
	turns := make(map[*player.Player]rune)
	turns[p1] = 'X'
	turns[p2] = 'O'
	first := [2]*player.Player{p1, p2}[rand.IntN(2)]
	return &Game{
		ID:      uuid.New(),
		Players: [2]*player.Player{p1, p2},
		Turns:   turns,
		Next:    first,
		Stream:  make(chan StreamEvent),
	}
}

// WatchStream continuously watches the Game's Stream channel
// and handles StreamEvents based on their EventType
func (g *Game) WatchStream() {
	for evt := range g.Stream {
		switch evt.Type {
		case PlayerTurn:
			p := evt.Data["player"].(*player.Player)
			if p != g.Next {
				g.sendBoardUpdate(nil)
				continue
			}
			t := evt.Data["tile"].(int)
			if g.Board[t] != 0 {
				g.sendBoardUpdate(nil)
				continue
			}
			g.Board[t] = g.Turns[p]
			if p == g.Players[0] {
				g.Next = g.Players[1]
			} else {
				g.Next = g.Players[0]
			}
			g.sendBoardUpdate(nil)
		}
	}
}

func (g *Game) sendBoardUpdate(data map[string]any) {
	e := player.StreamEvent{
		Type: player.BoardUpdate,
		Data: data,
	}
	for _, p := range g.Players {
		go func(plr *player.Player) {
			select {
			case plr.Stream <- e:
			case <-time.After(60 * time.Second):
			}
		}(p)
	}
}
