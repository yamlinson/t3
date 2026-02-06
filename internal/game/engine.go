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
	Winner  *player.Player
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
			g.checkWin()
			if g.Winner != nil {
				g.sendBoardUpdate(nil)
				continue
			}
			if p == g.Players[0] {
				g.Next = g.Players[1]
			} else {
				g.Next = g.Players[0]
			}
			g.sendBoardUpdate(nil)
		case PlayerQuit:
			p := evt.Data["player"].(*player.Player)
			if p == g.Players[0] {
				g.Winner = g.Players[1]
			} else {
				g.Winner = g.Players[0]
			}
			g.sendBoardUpdate(nil)
		}
	}
}

func (g *Game) checkWin() {
	if g.Winner != nil {
		return
	}
	var winningRune rune
	for i := range 3 {
		// Check vertical win
		if g.Board[i] != 0 &&
			g.Board[i] == g.Board[i+3] &&
			g.Board[i] == g.Board[i+6] {
			winningRune = g.Board[1]
		}
		// Check horizontal win
		j := i * 3
		if g.Board[j] != 0 &&
			g.Board[j] == g.Board[j+1] &&
			g.Board[j] == g.Board[j+2] {
			winningRune = g.Board[j]
		}
	}
	// Check diagonal wins
	if g.Board[0] != 0 &&
		g.Board[0] == g.Board[4] &&
		g.Board[0] == g.Board[8] {
		winningRune = g.Board[0]
	}
	if g.Board[2] != 0 &&
		g.Board[2] == g.Board[4] &&
		g.Board[2] == g.Board[6] {
		winningRune = g.Board[2]
	}
	if winningRune != 0 {
		if g.Turns[g.Players[0]] == winningRune {
			g.Winner = g.Players[0]
		} else {
			g.Winner = g.Players[1]
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
