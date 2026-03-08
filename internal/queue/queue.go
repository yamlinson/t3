// Package queue handles matchmaking queues
package queue

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yamlinson/t3/internal/event"
	"github.com/yamlinson/t3/internal/game"
	"github.com/yamlinson/t3/internal/player"
)

// Model describes a queue screen
type Model struct {
	textInput  textinput.Model
	errMsg     string
	state      uiState
	timer      timer.Model
	matchmaker *Matchmaker
	player     *player.Player
	match      *Match
	game       *game.Game
}

// GetModel returns a Model for a queue screen
func GetModel(p *player.Player, mm *Matchmaker) *Model {
	ti := textinput.New()
	if p.Name != "" {
		ti.SetValue(p.Name)
	} else {
		ti.Placeholder = "Your name"
	}
	ti.CharLimit = 20
	ti.Width = 30
	ti.Focus()

	return &Model{
		textInput:  ti,
		player:     p,
		matchmaker: mm,
	}
}

// Init sets the initial state of a queue screen
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// View returns the visual state of a queue screen
func (m Model) View() string {
	var s string
	switch m.state {
	case nameInput:
		s = fmt.Sprintf("Please enter your name...\n\n%s\n\n%s", m.textInput.View(), m.errMsg)
	case queued:
		s = fmt.Sprintf("Hello, %s!\n\nFinding an opponent...\n", m.player.Name)
		s += "Press 'q' to leave queue\n\n"
	case matched:
		seconds := int(m.timer.Timeout.Seconds())
		s = "Match found! Would you like to accept? (Y/n)\n\n"
		s += fmt.Sprintf("Time remaining: %d\n\n", seconds)
	case accepted:
		s = "Match accepted! Waiting for game start...\n\n"
	case declined:
		s = "Match declined. Press any key to return to home...\n\n"
	case gameReady:
		s = "Game found! Joining now...\n\n"
		s += fmt.Sprintf("Game ID: %s\n\n", m.game.ID)
	}
	return s
}

// Update handles changes to a queue screen
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case nameInput:
			switch msg.String() {
			case "enter":
				if m.textInput.Value() == "" {
					m.errMsg = "Name cannot be empty."
					return &m, nil
				}
				m.player.Name = m.textInput.Value()
				m.state = queued
				cmd = tea.Batch(
					m.player.WaitForEvent(),
					func() tea.Msg {
						m.matchmaker.AddPlayer(m.player)
						return playerAdded{}
					},
				)
			}
		case queued:
			switch msg.String() {
			case "q":
				m.matchmaker.DelPlayer(m.player)
				m.state = nameInput
				return &m, nil
			}
		case matched:
			switch msg.String() {
			case "enter", "Y", "y":
				m.state = accepted
				m.match.Responses <- AcceptMsg{
					PlayerID: m.player.ID,
					Accepted: true,
				}
				return &m, m.player.WaitForEvent()
			case "N", "n":
				m.state = declined
				return &m, nil
			}
		case declined:
			m.state = nameInput
			return &m, nil
		}
	case timer.TickMsg:
		if m.state == matched && int(m.timer.Timeout.Seconds()) == 0 {
			m.state = declined
		}
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		return &m, cmd
	case player.StreamEvent:
		switch msg.Type {
		case player.Matched:
			m.state = matched
			m.match = msg.Data["match"].(*Match)

			remaining := time.Until(m.match.AcceptTimeout)
			m.timer = timer.New(remaining)
			cmd = m.timer.Init()
		case player.Declined:
			m.state = declined
		case player.GameReady:
			m.state = gameReady
			m.game = msg.Data["game"].(*game.Game)
			cmd = func() tea.Msg {
				return event.SwitchToModel{Data: map[string]any{
					"model": "game",
					"game":  m.game,
				}}
			}
			m.timer.Stop()
		}
	case event.ShutdownMsg:
		m.matchmaker.DelPlayer(m.player)
		return &m, nil
	}

	if m.state == nameInput {
		m.textInput, cmd = m.textInput.Update(msg)
	}
	return &m, cmd
}

type uiState int

const (
	nameInput uiState = iota
	queued
	matched
	accepted
	declined
	gameReady
)

type playerAdded struct{}
