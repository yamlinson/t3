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

// You can wire any Bubble Tea model up to the middleware with a function that
// handles the incoming ssh.Session. Here we just grab the terminal info and
// pass it to the new model. You can also return tea.ProgramOptions (such as
// tea.WithAltScreen) on a session by session basis.
func teaHandler(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
	// This should never fail, as we are using the activeterm middleware.
	// pty, _, _ := s.Pty()

	// When running a Bubble Tea app over SSH, you shouldn't use the default
	// lipgloss.NewStyle function.
	// That function will use the color profile from the os.Stdin, which is the
	// server, not the client.
	// We provide a MakeRenderer function in the bubbletea middleware package,
	// so you can easily get the correct renderer for the current session, and
	// use it to create the styles.
	// The recommended way to use these styles is to then pass them down to
	// your Bubble Tea model.
	// renderer := bubbletea.MakeRenderer(s)
	// txtStyle := renderer.NewStyle().Foreground(lipgloss.Color("10"))
	// quitStyle := renderer.NewStyle().Foreground(lipgloss.Color("8"))

	// bg := "light"
	// if renderer.HasDarkBackground() {
	// 	bg = "dark"
	// }

	ti := textinput.New()
	ti.Placeholder = "Your name"
	ti.CharLimit = 20
	ti.Width = 30
	ti.Focus()

	p := queue.Player{
		Name:    "",
		Session: sess,
		ID:      uuid.New(),
		MatchC:  make(chan queue.Match, 1),
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
	player     queue.Player
	matchmaker *queue.Matchmaker
	opponent   queue.Player
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
					waitForMatch(&m.player),
					func() tea.Msg {
						m.matchmaker.AddPlayer(m.player)
						return playerAddedMsg{}
					},
				)
			}
		case "ctrl+c":
			return m, tea.Quit
		}
	case matchFound:
		m.state = matched
		m.opponent = msg.opponent
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
		s = fmt.Sprintf("Matched with %s!\n\nGame starting soon...\n", m.opponent.Name)
	}
	return m.txtStyle.Render(s) + "\n\n" + m.quitStyle.Render("Press 'ctrl+c' to quit\n")
}

type matchFound struct {
	opponent queue.Player
}

type (
	startGameMsg   struct{}
	playerAddedMsg struct{}
)

func waitForMatch(p *queue.Player) tea.Cmd {
	return func() tea.Msg {
		m := <-p.MatchC
		return matchFound{opponent: m.Opponent}
	}
}
