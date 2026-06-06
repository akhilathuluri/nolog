package tui

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"secure-chat/crypto"

	"github.com/atotto/clipboard"
	"github.com/aymanbagabas/go-osc52/v2"
	tea "github.com/charmbracelet/bubbletea"
)

type fileUploadCompleteMsg struct {
	success bool
	msg     string
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyCtrlQ {
			// Panic Exit
			m.Identity.Wipe()
			m.Session.Close()
			return m, tea.Quit
		}

		if msg.Type == tea.KeyCtrlY {
			// Local server clipboard fallback
			clipboard.WriteAll(m.Identity.UniqueID)
			
			// Remote SSH OSC 52
			seq := osc52.New(m.Identity.UniqueID).String()
			
			m.Messages = append(m.Messages, "[SYS] Unique ID copied to clipboard!")
			m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
			m.Timeline.GotoBottom()

			return m, tea.Printf("%s", seq)
		}

		if msg.Type == tea.KeyCtrlG && m.PendingFile != "" {
			scpCmd := fmt.Sprintf("scp -O -P 23234 localhost:download_%s ./%s", m.Identity.UniqueID, m.PendingFile)
			
			clipboard.WriteAll(scpCmd)
			seq := osc52.New(scpCmd).String()
			
			m.Messages = append(m.Messages, "[SYS] SCP command copied to clipboard!")
			m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
			m.Timeline.GotoBottom()

			return m, tea.Printf("%s", seq)
		}

		if msg.Type == tea.KeyCtrlR {
			if m.RoomID != "" {
				m.Hub.LeaveRoom(m.RoomID, m.Identity.UniqueID)
				m.Messages = append(m.Messages, fmt.Sprintf("[SYS] Left previous Room %s", m.RoomID))
			} else if m.Session.PeerID != "" {
				m.Session.Outgoing <- []byte("SYS:DISCONNECT")
				m.Messages = append(m.Messages, "[SYS] Disconnected from 1-to-1 peer.")
			}

			roomKey := make([]byte, 32)
			rand.Read(roomKey)
			roomID := fmt.Sprintf("ROOM_%s", m.Identity.UniqueID[:8])
			joinCode := fmt.Sprintf("%s-%x", roomID, roomKey)
			
			m.RoomID = roomID
			m.RoomKey = roomKey
			m.Cipher, _ = crypto.NewCipherEngine(roomKey)
			m.Session.PeerID = roomID // Block 1-to-1 pairings
			m.Hub.JoinRoom(roomID, m.Identity.UniqueID)
			
			clipboard.WriteAll(joinCode)
			seq := osc52.New(joinCode).String()
			
			m.Messages = append(m.Messages, fmt.Sprintf("[SYS] Room %s created!", roomID))
			m.Messages = append(m.Messages, "[SYS] Join Code copied to clipboard!")
			m.Messages = append(m.Messages, fmt.Sprintf("Code: %s", joinCode[:30]))
			m.Messages = append(m.Messages, fmt.Sprintf("      %s", joinCode[30:]))
			m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
			m.Timeline.GotoBottom()
			return m, tea.Printf("%s", seq)
		}

		if msg.Type == tea.KeyCtrlJ {
			m.State = StateRoomJoin
			m.Input.Placeholder = "Paste Join Code..."
			m.Input.SetValue("")
			return m, nil
		}

		if msg.Type == tea.KeyCtrlL {
			if m.RoomID != "" {
				m.Hub.LeaveRoom(m.RoomID, m.Identity.UniqueID)
				m.Messages = append(m.Messages, fmt.Sprintf("[SYS] Left Room %s", m.RoomID))
				m.RoomID = ""
				m.RoomKey = nil
				m.Session.PeerID = ""
				m.Cipher = nil
				m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
				m.Timeline.GotoBottom()
			}
			return m, nil
		}

		if m.State == StateChat {
			if msg.Type == tea.KeyEnter {
				text := m.Input.Value()
				if text != "" {
					m.Messages = append(m.Messages, fmt.Sprintf("You: %s", text))
					m.Input.SetValue("")
					m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
					m.Timeline.GotoBottom()
					if m.Cipher != nil {
						cipherText, _ := m.Cipher.Encrypt([]byte(text))
						if m.RoomID != "" {
							m.Hub.Broadcast(m.RoomID, m.Identity.UniqueID, cipherText)
						} else {
							m.Session.Outgoing <- cipherText
						}
					}
				}
				return m, nil
			}
			if msg.Type == tea.KeyCtrlF {
				m.State = StateFilePicker
				return m, nil
			}
		} else if m.State == StateFilePicker {
			if msg.Type == tea.KeyEsc {
				m.State = StateChat
				return m, nil
			}
		} else if m.State == StateRoomJoin {
			if msg.Type == tea.KeyEsc {
				m.State = StateChat
				m.Input.Placeholder = "Type a message..."
				return m, nil
			}
			if msg.Type == tea.KeyEnter {
				code := m.Input.Value()
				code = strings.TrimSpace(code)
				code = strings.ReplaceAll(code, "\n", "")
				code = strings.ReplaceAll(code, "\r", "")
				code = strings.ReplaceAll(code, " ", "")
				parts := strings.Split(code, "-")
				if len(parts) == 2 {
					roomID := parts[0]
					keyHex := parts[1]
					keyBytes, err := hex.DecodeString(keyHex)
					if err == nil && len(keyBytes) == 32 {
						if m.RoomID != "" {
							m.Hub.LeaveRoom(m.RoomID, m.Identity.UniqueID)
							m.Messages = append(m.Messages, fmt.Sprintf("[SYS] Left previous Room %s", m.RoomID))
						} else if m.Session.PeerID != "" {
							m.Session.Outgoing <- []byte("SYS:DISCONNECT")
							m.Messages = append(m.Messages, "[SYS] Disconnected from 1-to-1 peer.")
						}

						m.RoomID = roomID
						m.RoomKey = keyBytes
						m.Cipher, _ = crypto.NewCipherEngine(keyBytes)
						m.Session.PeerID = roomID // Block 1-to-1 pairings
						m.Hub.JoinRoom(roomID, m.Identity.UniqueID)
						m.Messages = append(m.Messages, fmt.Sprintf("[SYS] Joined Room %s!", roomID))
					} else {
						m.Messages = append(m.Messages, "[SYS] Invalid Join Code (Key Length Error).")
					}
				} else {
					m.Messages = append(m.Messages, "[SYS] Invalid Join Code Format.")
				}
				m.State = StateChat
				m.Input.Placeholder = "Type a message..."
				m.Input.SetValue("")
				m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
				m.Timeline.GotoBottom()
				return m, nil
			}
		}

	case peerMessageMsg:
		if bytes.Equal(msg, []byte("SYS:DISCONNECT")) {
			m.Messages = append(m.Messages, "[SYS] Peer has disconnected.")
			m.Session.PeerID = ""
			m.Cipher = nil
			m.PendingFile = ""
			m.Initiator = false // Reset initiator flag because any subsequent pair will be initiated by the peer
			m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
			m.Timeline.GotoBottom()
			return m, waitForMessage(m.Session.Incoming)
		}

		if m.Cipher == nil {
			peerPub := []byte(msg)
			sharedKey, err := crypto.DeriveSharedKey(m.Identity.PrivateKey, peerPub)
			if err == nil {
				m.Cipher, _ = crypto.NewCipherEngine(sharedKey)
				m.Messages = append(m.Messages, "[SYS] Secure encrypted channel established!")
				if !m.Initiator {
					m.Session.Outgoing <- m.Identity.PublicKey
				}
			} else {
				m.Messages = append(m.Messages, "[SYS] Cryptographic handshake failed!")
			}
			m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
			m.Timeline.GotoBottom()
		} else {
			decrypted, err := m.Cipher.Decrypt([]byte(msg))
			if err == nil {
				if bytes.HasPrefix(decrypted, []byte("FILE:")) {
					parts := bytes.SplitN(decrypted, []byte(":"), 3)
					if len(parts) == 3 {
						filename := string(parts[1])
						fileData := parts[2]
						
						fileKey := "download_" + m.Identity.UniqueID
						m.Hub.StoreFile(fileKey, fileData)
						
						m.Messages = append(m.Messages, fmt.Sprintf("Peer sent file: %s", filename))
						m.Messages = append(m.Messages, "[SYS] Download command added to sidebar!")
						m.PendingFile = filename
					}
				} else {
					m.Messages = append(m.Messages, fmt.Sprintf("Peer: %s", string(decrypted)))
				}
			} else {
				m.Messages = append(m.Messages, "[SYS] Received malformed encrypted payload")
			}
			m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
			m.Timeline.GotoBottom()
		}
		return m, waitForMessage(m.Session.Incoming)

	case fileUploadCompleteMsg:
		m.Messages = append(m.Messages, msg.msg)
		m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
		m.Timeline.GotoBottom()
		return m, nil

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		
		headerHeight := 3
		footerHeight := 2
		contentHeight := m.Height - headerHeight - footerHeight - 2
		
		m.Timeline.Width = int(float64(m.Width)*0.70) - 4
		m.Timeline.Height = contentHeight
		
		m.FilePicker.Height = contentHeight

		m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
		m.Timeline.GotoBottom()
	}

	if m.State == StateChat || m.State == StateRoomJoin {
		m.Input, cmd = m.Input.Update(msg)
		cmds = append(cmds, cmd)
		m.Timeline, cmd = m.Timeline.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.State == StateFilePicker {
		m.FilePicker, cmd = m.FilePicker.Update(msg)
		cmds = append(cmds, cmd)
		
		if didSelect, path := m.FilePicker.DidSelectFile(msg); didSelect {
			m.State = StateChat
			m.Messages = append(m.Messages, fmt.Sprintf("[SYS] Uploading %s...", filepath.Base(path)))
			m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
			m.Timeline.GotoBottom()
			
			// Read file, encrypt, and send
			uploadCmd := func() tea.Msg {
				data, err := os.ReadFile(path)
				if err != nil || len(data) > 10*1024*1024 { 
					return fileUploadCompleteMsg{false, "[SYS] Upload failed: file too large or unreadable"}
				}
				filename := filepath.Base(path)
				payload := append([]byte("FILE:"+filename+":"), data...)
				cipherText, _ := m.Cipher.Encrypt(payload)
				if m.RoomID != "" {
					m.Hub.Broadcast(m.RoomID, m.Identity.UniqueID, cipherText)
				} else {
					m.Session.Outgoing <- cipherText
				}
				return fileUploadCompleteMsg{true, "[SYS] File securely transmitted!"}
			}
			cmds = append(cmds, uploadCmd)
		}
	}

	return m, tea.Batch(cmds...)
}
