// Package home contains the home screen of the client
package home

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yamlinson/t3/internal/db"
	"github.com/yamlinson/t3/internal/event"
)

type keyMap struct {
	find key.Binding
	quit key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.find, k.quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.find, k.quit},
	}
}

var keys = keyMap{
	find: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "find game"),
	),
	quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

// Model describes a home screen
type Model struct {
	database   *db.DB
	table      table.Model
	help       help.Model
	titleStyle lipgloss.Style
	boxStyle   lipgloss.Style
}

// GetModel returns a model for a home screen
func GetModel(d *db.DB) *Model {
	tp, err := d.GetTopPlayers()
	if err != nil {
		fmt.Printf("Error getting players from database: %v", err)
	}

	var leaders []table.Row

	if tp != nil {
		for i, pl := range tp {
			row := table.Row{
				strconv.Itoa(i + 1),
				pl.Name,
				strconv.Itoa(pl.Wins),
				strconv.Itoa(pl.Draws),
				strconv.Itoa(pl.Losses),
			}
			leaders = append(leaders, row)
		}
	}

	columns := []table.Column{
		{Title: "Rank", Width: 6},
		{Title: "Player", Width: 15},
		{Title: "Wins", Width: 6},
		{Title: "Draws", Width: 6},
		{Title: "Losses", Width: 6},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(leaders),
		table.WithFocused(true),
		table.WithHeight(4),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("135")).
		Bold(true).
		Padding(1, 0, 1, 0).
		Align(lipgloss.Center)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Align(lipgloss.Center)

	help := help.New()
	help.Styles.ShortDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	help.Styles.ShortKey = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))

	return &Model{
		database:   d,
		table:      t,
		help:       help,
		titleStyle: titleStyle,
		boxStyle:   boxStyle,
	}
}

// Init sets the initial state of a home screen
func (m Model) Init() tea.Cmd {
	return nil
}

// View returns the visual state of a home screen
func (m Model) View() string {
	title := m.titleStyle.Render("t3")

	welcome := "Welcome to t3,\n"
	welcome += "multiplayer tic-tac-toe in your terminal!\n\n"
	welcome += "Press 'f' to search for a match...\n\n"

	welcomeBox := m.boxStyle.Render(welcome)

	leaderBoardHeader := lipgloss.NewStyle().
		Foreground(lipgloss.Color("204")).
		Bold(true).
		Padding(0, 0, 1, 0).
		Render("Top players")

	content := fmt.Sprintf(
		"%s\n%s\n%s\n%s\n\n%s",
		title,
		welcomeBox,
		leaderBoardHeader,
		m.table.View(),
		m.help.View(keys),
	)

	return fmt.Sprintf("\n%s\n\n", content)
}

// Update handles changes to a home screen
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.quit):
			return &m, tea.Quit
		case key.Matches(msg, keys.find):
			cmd = func() tea.Msg { return event.SwitchToModel{Data: map[string]any{"model": "queue"}} }
			return &m, cmd
		}
	}

	return &m, cmd
}
