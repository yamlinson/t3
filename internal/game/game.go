// Package game manages the state and lifecycle of games
package game

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	game     *Game
	board    [9]rune
	cursor   int
	turn     rune
	selected bool
}

// GetModel returns a model for a game screen
func GetModel(g *Game) tea.Model {
	return Model{
		game: g,
		turn: 'X',
	}
}

// Init sets the initial state of a game screen
func (m Model) Init() tea.Cmd {
	return nil
}

// View returns the visual state of a queue screen
func (m Model) View() string {
	var s string
	cells := make([]string, 9)

	s += "Current turn: " + string(m.turn) + "\n\n"

	for i := range 9 {
		style := cellStyle
		if m.selected && i == m.cursor {
			style = selectedStyle
		}
		content := " "
		switch m.board[i] {
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
	s += "hjkl or arrow keys to move\n"
	s += "space to choose square\n"

	return s
}

// Update handles changes to a game screen
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			// TODO: Return to home
			return m, tea.Quit
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
			if m.board[m.cursor] == 0 {
				m.board[m.cursor] = m.turn
				if m.turn == 'X' {
					m.turn = 'O'
				} else {
					m.turn = 'X'
				}
			}
		}
	}
	return m, nil
}

// SwitchToGameModel instructs Bubble Tea to display a game
type SwitchToGameModel struct{}
