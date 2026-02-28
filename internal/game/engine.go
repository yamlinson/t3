// Package game manages the state and lifecycle of games
package game

import (
	"log"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yamlinson/t3/internal/db"
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
	database  *db.DB
	done      chan struct{}
	endOnce   sync.Once
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
func NewGame(p1 *player.Player, p2 *player.Player, database *db.DB) *Game {
	turns := make(map[*player.Player]rune)
	turns[p1] = 'X'
	turns[p2] = 'O'
	first := [2]*player.Player{p1, p2}[rand.IntN(2)]
	return &Game{
		ID:       uuid.New(),
		Players:  [2]*player.Player{p1, p2},
		Turns:    turns,
		Next:     first,
		Stream:   make(chan StreamEvent),
		State:    InProgress,
		database: database,
		done:     make(chan struct{}),
	}
}

// WatchStream continuously watches the Game's Stream channel
// and handles StreamEvents based on their EventType
func (g *Game) WatchStream() {
	if g.State == InProgress && g.turnTimer == nil {
		g.turnTimer = g.startTurnTimer()
	}

	for {
		select {
		case evt, ok := <-g.Stream:
			if !ok {
				g.cleanup()
				return
			}
			g.handleEvent(evt)
		case <-g.done:
			g.cleanup()
			return
		}
	}
}

func (g *Game) handleEvent(evt StreamEvent) {
	switch evt.Type {
	case PlayerTurn:
		p := evt.Data["player"].(*player.Player)
		if p != g.Next {
			g.sendBoardUpdate(nil)
		}
		t := evt.Data["tile"].(int)
		if g.Board[t] != 0 {
			g.sendBoardUpdate(nil)
		}
		g.stopTurnTimer()
		g.Board[t] = g.Turns[p]
		g.checkWin()
		if g.State == Win || g.State == Draw {
			g.sendBoardUpdate(nil)
			g.end(g.State, g.Winner)
		}
		g.Next = g.otherPlayer(p)
		g.turnTimer = g.startTurnTimer()
		g.sendBoardUpdate(nil)
	case TurnTimeout:
		g.sendBoardUpdate(nil)
		g.end(Forfeit, g.otherPlayer(g.Next))
	case PlayerQuit:
		p := evt.Data["player"].(*player.Player)
		g.sendBoardUpdate(nil)
		g.end(Forfeit, g.otherPlayer(p))
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
			winningRune = g.Board[i]
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

func (g *Game) writeResult() {
	switch g.State {
	case Draw:
		if err := g.database.RecordMatch(
			[]string{g.Players[0].Name, g.Players[1].Name},
			db.Draw,
		); err != nil {
			log.Fatal(err)
		}
	case Forfeit, Win:
		if err := g.database.RecordMatch(
			[]string{g.Winner.Name, g.otherPlayer(g.Winner).Name},
			db.Win,
		); err != nil {
			log.Fatal(err)
		}
	}
}

func (g *Game) end(state State, winner *player.Player) {
	g.endOnce.Do(func() {
		g.State = state
		g.Winner = winner
		g.writeResult()
		close(g.done)
	})
}

func (g *Game) cleanup() {
	if g.turnTimer != nil {
		g.turnTimer.Stop()
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
