// Package game manages the state and lifecycle of games
package game

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yamlinson/t3/internal/event"
	"github.com/yamlinson/t3/internal/player"
)

var (
	cellStyle = lipgloss.NewStyle().
			Width(7).
			Height(3).
			Align(lipgloss.Center, lipgloss.Center).
			Border(lipgloss.RoundedBorder())

	selectedStyle = cellStyle.BorderForeground(lipgloss.Color("212"))
	playedXStyle  = cellStyle.Foreground(lipgloss.Color("99"))
	playedOStyle  = cellStyle.Foreground(lipgloss.Color("214"))
)

// Model describes a game screen
type Model struct {
	player  *player.Player
	game    *Game
	cursor  int
	state   gameState
	tickCmd tea.Cmd
}

// GetModel returns a model for a game screen
func GetModel(g *Game, p *player.Player) *Model {
	return &Model{
		game:   g,
		player: p,
	}
}

// Init sets the initial state of a game screen
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.player.WaitForEvent(),
		tickCmd(),
	)
}

// View returns the visual state of a queue screen
func (m Model) View() string {
	var s string
	cells := make([]string, 9)

	switch m.state {
	case gameInProgress:
		// Check time until TurnTimout occurs. Clamp positive
		remaining := max(0, 60-int(time.Since(m.game.TurnStart).Seconds()))
		s += fmt.Sprintf("Current turn: %s \t\t Time remaining: %s\n\n", string(m.game.Next.Name), formatTimer(remaining))
	case gameOver:
		s += fmt.Sprintf("Game over! %s won!\n\n", string(m.game.Winner.Name))
	}

	for i := range 9 {
		style := cellStyle
		content := " "
		switch m.game.Board[i] {
		case 'X':
			style = playedXStyle
			content = "X"
		case 'O':
			style = playedOStyle
			content = "O"
		}
		if i == m.cursor {
			style = selectedStyle
		}
		cells[i] = style.Render(content)
	}

	for i := range 3 {
		s += lipgloss.JoinHorizontal(lipgloss.Top, cells[i*3], cells[i*3+1], cells[i*3+2])
		if i < 2 {
			s += "\n"
		}
	}

	s += "\n\nControls:\n"
	s += "q to quit game\n"
	if m.state == gameInProgress {
		s += "hjkl or arrow keys to move\n"
		s += "space to choose square\n"
	}

	return s
}

// Update handles changes to a game screen
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.game.Winner != nil {
		m.state = gameOver
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case gameOver:
			if msg.String() == "q" {
				cmd = func() tea.Msg { return event.SwitchToMainModel{} }
			}
		case gameInProgress:
			switch msg.String() {
			case "q":
				e := StreamEvent{
					Type: PlayerQuit,
					Data: map[string]any{"player": m.player},
				}
				go func() {
					m.game.Stream <- e
				}()
				cmd = func() tea.Msg { return event.SwitchToMainModel{} }
			case "h", "left":
				if m.cursor%3 > 0 {
					m.cursor--
				}
			case "j", "down":
				if m.cursor < 6 {
					m.cursor += 3
				}
			case "k", "up":
				if m.cursor > 2 {
					m.cursor -= 3
				}
			case "l", "right":
				if m.cursor%3 < 2 {
					m.cursor++
				}
			case " ":
				if m.game.Board[m.cursor] == 0 && m.game.Next == m.player {
					evt := StreamEvent{
						Type: PlayerTurn,
						Data: map[string]any{
							"player": m.player,
							"tile":   m.cursor,
						},
					}
					go func() {
						m.game.Stream <- evt
					}()
				}
			}
		}
	case player.StreamEvent:
		switch msg.Type {
		case player.BoardUpdate:
		}
		cmd = m.player.WaitForEvent()
	case tickMsg:
		if m.state == gameOver {
			return m, nil
		}
		return m, tickCmd()
	}
	return &m, cmd
}

type gameState int

const (
	gameInProgress gameState = iota
	gameOver
)

// SwitchToGameModel instructs Bubble Tea to display a game
type SwitchToGameModel struct {
	Game *Game
}

var (
	timerNormal   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	timerWarning  = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	timerCritical = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

func formatTimer(remaining int) string {
	style := timerNormal
	if remaining <= 10 {
		style = timerCritical
	} else if remaining <= 30 {
		style = timerWarning
	}
	return style.Render(fmt.Sprintf("(%ds)", remaining))
}

type tickMsg struct{}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}
