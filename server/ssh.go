package server

import (
	"log"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/bubbletea"

	tea "github.com/charmbracelet/bubbletea"
	"secure-chat/manager"
	"secure-chat/crypto"
	"secure-chat/tui"
)

func TeaHandler(hub *manager.Hub) bubbletea.Handler {
	return func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		id, err := crypto.GenerateIdentity()
		if err != nil {
			log.Println("Error generating identity:", err)
			return nil, nil
		}

		sess := manager.NewSession(id.UniqueID)
		hub.Register(sess)

		// Ensure session is unregistered when the SSH connection drops
		go func() {
			<-s.Context().Done()
			hub.Unregister(id.UniqueID)
			id.Wipe() // Zero out private keys when session ends
		}()

		// Check for target ID in arguments
		args := s.Command()
		initiator := false
		if len(args) > 0 {
			initiator = true
			targetID := args[0]
			err := hub.Pair(id.UniqueID, targetID)
			if err != nil {
				log.Printf("Failed to pair %s with %s: %v\n", id.UniqueID, targetID, err)
			} else {
				log.Printf("Successfully paired %s with %s\n", id.UniqueID, targetID)
				sess.Outgoing <- id.PublicKey
			}
		}

		m := tui.InitialModel(sess, hub, id, initiator)
		return m, []tea.ProgramOption{tea.WithAltScreen()}
	}
}
