// main is the entrypoint of t3
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/google/uuid"
	"github.com/yamlinson/t3/internal/player"
	"github.com/yamlinson/t3/internal/queue"
)

const (
	host = "localhost"
	port = "2222"
)

func main() {
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Error("Could not start server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}

var mm *queue.Matchmaker = queue.NewMatchmaker()

func teaHandler(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
	ti := textinput.New()
	ti.Placeholder = "Your name"
	ti.CharLimit = 20
	ti.Width = 30
	ti.Focus()

	p := player.Player{
		Name:    "",
		Session: sess,
		ID:      uuid.New(),
		MatchC:  make(chan uuid.UUID, 1),
	}

	initialModel := model{
		textInput:  ti,
		player:     p,
		matchmaker: mm,
	}

	return initialModel, []tea.ProgramOption{tea.WithAltScreen()}
}

type uiState int

const (
	nameInput uiState = iota
	waitingInQueue
	matched
)

type model struct {
	textInput  textinput.Model
	errMsg     string
	state      uiState
	player     player.Player
	matchmaker *queue.Matchmaker
	gameID     uuid.UUID
	// term      string
	// profile   string
	width  int
	height int
	// bg        string
	txtStyle  lipgloss.Style
	quitStyle lipgloss.Style
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			switch m.state {
			case nameInput:
				if m.textInput.Value() == "" {
					m.errMsg = "Name cannot be empty."
					return m, nil
				}
				m.player.Name = m.textInput.Value()
				m.state = waitingInQueue
				return m, tea.Batch(
					queue.WaitForMatch(&m.player),
					func() tea.Msg {
						m.matchmaker.AddPlayer(m.player)
						return playerAddedMsg{}
					},
				)
			}
		case "ctrl+c":
			return m, tea.Quit
		}
	case queue.MatchFound:
		m.state = matched
		m.gameID = msg.GameID
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return startGameMsg{}
		})
	}

	if m.state == nameInput {
		m.textInput, cmd = m.textInput.Update(msg)
	}
	return m, cmd
}

func (m model) View() string {
	var s string
	switch m.state {
	case nameInput:
		s = fmt.Sprintf("Please enter your name...\n%s\n\n%s", m.textInput.View(), m.errMsg)
	case waitingInQueue:
		s = fmt.Sprintf("Hello, %s!\n\nFinding an opponent...\n", m.player.Name)
	case matched:
		s = "Matched!\n\nGame starting soon...\n"
	}
	return m.txtStyle.Render(s) + "\n\n" + m.quitStyle.Render("Press 'ctrl+c' to quit\n")
}

type (
	startGameMsg   struct{}
	playerAddedMsg struct{}
)
