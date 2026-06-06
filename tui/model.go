package tui

import (
	"secure-chat/crypto"
	"secure-chat/manager"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type UIState int

const (
	StateChat UIState = iota
	StateFilePicker
	StateRoomJoin
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
	FilePicker filepicker.Model
	
	State       UIState
	Messages    []string
	Initiator   bool
	PendingFile string
	
	RoomID      string
	RoomKey     []byte

	MessageCount int
	Ping         string
}

type peerMessageMsg []byte
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

func InitialModel(sess *manager.Session, hub *manager.Hub, id *crypto.Identity, initiator bool) Model {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Focus()

	fp := filepicker.New()
	fp.AllowedTypes = []string{} // Allow all files
	fp.CurrentDirectory = "."

	return Model{
		Session:  sess,
		Hub:      hub,
		Identity: id,
		Input:    ti,
		FilePicker: fp,
		State:    StateChat,
		Messages: []string{"[SYS] Session initialized. Waiting for peer..."},
		Initiator: initiator,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.FilePicker.Init(), waitForMessage(m.Session.Incoming), tickCmd())
}
