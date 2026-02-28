// t3 is a tic-tac-toe server with a TUI client served over SSH
package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/google/uuid"
	"github.com/yamlinson/t3/internal/db"
	"github.com/yamlinson/t3/internal/event"
	"github.com/yamlinson/t3/internal/game"
	"github.com/yamlinson/t3/internal/player"
	"github.com/yamlinson/t3/internal/queue"
)

const (
	host = "localhost"
	port = "2222"
)

func main() {
	database, err := db.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	if err := database.CreateSchema(); err != nil {
		log.Fatal(err)
	}

	mm := queue.NewMatchmaker(database)
	go mm.WatchResults()

	teaHandler := func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
		p := &player.Player{
			Name:    "",
			Session: sess,
			ID:      uuid.New(),
			Stream:  make(chan player.StreamEvent),
		}

		initialModel := initMainModel(p, mm, database)

		return initialModel, []tea.ProgramOption{tea.WithAltScreen()}
	}

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

type mainModel struct {
	activeModel tea.Model
	player      *player.Player
	matchmaker  *queue.Matchmaker
	game        *game.Game
	width       int
	height      int
	txtStyle    lipgloss.Style
	quitStyle   lipgloss.Style
	database    *db.DB
}

func initMainModel(p *player.Player, mm *queue.Matchmaker, d *db.DB) *mainModel {
	return &mainModel{
		activeModel: queue.GetModel(p, mm),
		player:      p,
		matchmaker:  mm,
		database:    d,
	}
}

func (m mainModel) Init() tea.Cmd {
	return nil
}

func (m mainModel) View() string {
	content := m.activeModel.View()
	styled := m.txtStyle.Render(content)
	quit := m.quitStyle.Render("Press 'ctrl+c' to exit\n")
	s := styled + "\n\n" + quit
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		s,
	)
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.activeModel, _ = m.activeModel.Update(event.ShutdownMsg{})
			return &m, tea.Quit
		}
	case game.SwitchToGameModel:
		g := msg.Game
		m.activeModel = game.GetModel(g, m.player)
		return &m, m.activeModel.Init()
	case event.SwitchToMainModel:
		m.activeModel = queue.GetModel(m.player, m.matchmaker)
		return &m, m.activeModel.Init()
	}
	m.activeModel, cmd = m.activeModel.Update(msg)
	return &m, cmd
}
