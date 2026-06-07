package tui

import (
	"secure-chat/crypto"
	"secure-chat/manager"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type UIState int

const (
	StateChat UIState = iota
	StateRoomJoin
	StateAuthVerify
)

type Model struct {
	Session  *manager.Session
	Hub      *manager.Hub
	Identity *crypto.Identity
	Cipher   *crypto.CipherEngine

	Width  int
	Height int

	Timeline   viewport.Model
	Input      textinput.Model
	
	State       UIState
	Messages    []string
	Initiator   bool
	PendingFile string
	Spinner     spinner.Model
	
	RoomID      string
	RoomKey     []byte

	NextPriv []byte
	NextPub  []byte

	MessageCount int
	Ping         string
}

type peerMessageMsg []byte
type fileUploadMsg []byte
type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func waitForMessage(sub chan []byte) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-sub
		if !ok {
			return nil
		}
		return peerMessageMsg(msg)
	}
}

func waitForUpload(sub chan []byte) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-sub
		if !ok {
			return nil
		}
		return fileUploadMsg(msg)
	}
}

func InitialModel(sess *manager.Session, hub *manager.Hub, id *crypto.Identity, initiator bool) Model {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.CharLimit = 4096
	ti.Focus()

	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	return Model{
		Session:  sess,
		Hub:      hub,
		Identity: id,
		Input:    ti,
		State:    StateChat,
		Messages: []string{"[SYS] Session initialized. Waiting for peer..."},
		Initiator: initiator,
		Spinner:   s,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.Spinner.Tick, waitForMessage(m.Session.Incoming), waitForUpload(m.Session.Uploads), tickCmd())
}
