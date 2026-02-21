// Package game manages the state and lifecycle of games
package game

import (
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
	"github.com/yamlinson/t3/internal/player"
)

// Game represents the state of one game
type Game struct {
	ID        uuid.UUID
	Players   [2]*player.Player
	Board     [9]rune
	Turns     map[*player.Player]rune
	Next      *player.Player
	turnTimer *time.Timer
	TurnStart time.Time
	Stream    chan StreamEvent
	State     State
	Winner    *player.Player
}

// StreamEvent contains the information sent to a game over its lifecycle
type StreamEvent struct {
	Type EventType
	Data map[string]any
}

// EventType defines the Types which can be associated with a StreamEvent
type EventType int

// PlayerTurn, PlayerQuit, TurnTimeout
// describe possible events a game might be notified of
const (
	PlayerTurn EventType = iota
	PlayerQuit
	TurnTimeout
)

// State defines the possible states of a game
type State int

// InProgress, Draw, Forfeit, Win
// are the possible states of a game
const (
	InProgress State = iota
	Draw
	Forfeit
	Win
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
		State:   InProgress,
	}
}

// WatchStream continuously watches the Game's Stream channel
// and handles StreamEvents based on their EventType
func (g *Game) WatchStream() {
	if g.State == InProgress && g.turnTimer == nil {
		g.turnTimer = g.startTurnTimer()
	}
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
			g.stopTurnTimer()
			g.Board[t] = g.Turns[p]
			g.checkWin()
			if g.State == Win || g.State == Draw {
				g.sendBoardUpdate(nil)
				continue
			}
			g.Next = g.otherPlayer(p)
			g.turnTimer = g.startTurnTimer()
			g.sendBoardUpdate(nil)
		case TurnTimeout:
			g.State = Forfeit
			g.Winner = g.otherPlayer(g.Next)
			g.sendBoardUpdate(nil)
		case PlayerQuit:
			p := evt.Data["player"].(*player.Player)
			g.stopTurnTimer()
			g.State = Forfeit
			g.Winner = g.otherPlayer(p)
			g.sendBoardUpdate(nil)
		}
	}
	if g.turnTimer != nil {
		g.turnTimer.Stop()
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
		g.State = Win
		if g.Turns[g.Players[0]] == winningRune {
			g.Winner = g.Players[0]
		} else {
			g.Winner = g.Players[1]
		}
	}
	// Check draw
	for i := range 9 {
		if g.Board[i] == 0 { // Open tile still available
			return
		}
	}
	g.State = Draw
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

func (g *Game) otherPlayer(p *player.Player) *player.Player {
	if p == g.Players[0] {
		return g.Players[1]
	}
	return g.Players[0]
}

func (g *Game) startTurnTimer() *time.Timer {
	g.TurnStart = time.Now()
	return time.AfterFunc(60*time.Second, func() {
		g.Stream <- StreamEvent{
			Type: TurnTimeout,
			Data: map[string]interface{}{},
		}
	})
}

func (g *Game) stopTurnTimer() {
	if g.turnTimer != nil {
		g.turnTimer.Stop()
	}
}
