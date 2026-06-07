package tui

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"secure-chat/crypto"

	"github.com/atotto/clipboard"
	"github.com/aymanbagabas/go-osc52/v2"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mdp/qrterminal/v3"
)


func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.State == StateAuthVerify {
			if msg.Type == tea.KeyRunes {
				r := msg.Runes[0]
				if r == 'y' || r == 'Y' {
					m.Messages = append(m.Messages, "[SYS] Fingerprint verified. Secure channel established!")
					m.State = StateChat
					m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
					m.Timeline.GotoBottom()
				} else if r == 'n' || r == 'N' {
					m.Messages = append(m.Messages, "[SYS] 🚨 FINGERPRINT MISMATCH! MITM DETECTED! 🚨")
					m.Messages = append(m.Messages, "[SYS] Application terminating to protect session.")
					m.State = StateChat
					m.Session.Outgoing <- []byte("SYS:DISCONNECT")
					m.Cipher = nil
					m.Session.PeerID = ""
					m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
					m.Timeline.GotoBottom()
					return m, tea.Quit
				}
			}
			return m, nil
		}

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

		if msg.Type == tea.KeyCtrlU {
			uploadCmd := fmt.Sprintf("scp -O -P 23234 <file> localhost:upload_%s", m.Identity.UniqueID)
			clipboard.WriteAll(uploadCmd)
			seq := osc52.New(uploadCmd).String()
			m.Messages = append(m.Messages, "[SYS] SCP Upload command copied to clipboard!")
			m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
			m.Timeline.GotoBottom()
			return m, tea.Printf("%s", seq)
		}

		if msg.Type == tea.KeyCtrlD {
			if m.PendingFile != "" {
				clipboard.WriteAll(m.PendingFile)
				seq := osc52.New(m.PendingFile).String()
				m.Messages = append(m.Messages, "[SYS] SCP Download command copied to clipboard!")
				if len(m.Messages) > 100 {
					m.Messages = m.Messages[len(m.Messages)-100:]
				}
				m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
				m.Timeline.GotoBottom()
				return m, tea.Printf("%s", seq)
			}
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
			m.Hub.CreateRoom(roomID) // Register room in Hub with 10 min TTL
			expiry := time.Now().Add(10 * time.Minute).Unix()
			joinCode := fmt.Sprintf("%s-%x-%d", roomID, roomKey, expiry)
			
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
					timestamp := time.Now().Format("15:04")
					m.Messages = append(m.Messages, fmt.Sprintf("[%s] 👤 You: %s", timestamp, text))
					m.MessageCount++
					m.Input.SetValue("")
					m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
					m.Timeline.GotoBottom()
					if m.Cipher != nil {
						var payload []byte
						if m.RoomID != "" {
							payload = []byte(fmt.Sprintf("MSG:%s:%s", m.Identity.UniqueID[:3], text))
						} else {
							payload = []byte(text)
						}
						cipherText, _ := m.Cipher.Encrypt(payload)
						if m.RoomID != "" {
							m.Hub.Broadcast(m.RoomID, m.Identity.UniqueID, cipherText)
						} else {
							m.Session.Outgoing <- cipherText
							m.Cipher.RatchetKey()
						}
					}
				}
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
				if len(parts) == 3 {
					roomID := parts[0]
					keyHex := parts[1]
					expiryStr := parts[2]
					
					expiry, errParse := strconv.ParseInt(expiryStr, 10, 64)
					if errParse == nil && time.Now().Unix() > expiry {
						m.Messages = append(m.Messages, "[SYS] 🚨 Invalid Join Code (EXPIRED).")
					} else {
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
							
							errJoin := m.Hub.JoinRoom(roomID, m.Identity.UniqueID)
							if errJoin != nil {
								m.Messages = append(m.Messages, "[SYS] 🚨 Invalid Join Code (Room no longer exists).")
								m.RoomID = ""
								m.RoomKey = nil
								m.Cipher = nil
								m.Session.PeerID = ""
							} else {
								m.Messages = append(m.Messages, "[SYS] Joined Room %s!", roomID)
							}
						} else {
							m.Messages = append(m.Messages, "[SYS] Invalid Join Code (Key Length Error).")
						}
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

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd

	case tickMsg:
		if m.Cipher != nil && m.RoomID == "" && m.Session.PeerID != "" {
			pingPayload := []byte(fmt.Sprintf("PING:%d", time.Now().UnixNano()))
			cipherText, _ := m.Cipher.Encrypt(pingPayload)
			m.Cipher.RatchetKey()
			select {
			case m.Session.Outgoing <- cipherText:
			default:
			}
		}
		return m, tickCmd()

	case peerMessageMsg:
		if bytes.Equal(msg, []byte("SYS:DISCONNECT")) {
			m.Messages = append(m.Messages, "[SYS] Peer has disconnected.")
			m.Session.PeerID = ""
			m.Cipher = nil
			m.Initiator = false // Reset initiator flag because any subsequent pair will be initiated by the peer
			m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
			m.Timeline.GotoBottom()
			return m, waitForMessage(m.Session.Incoming)
		}

		if m.Cipher == nil {
			peerPub := []byte(msg)
			if len(peerPub) != 32 {
				m.Messages = append(m.Messages, "[SYS] 🚨 Invalid Peer Public Key Length. Connection aborted.")
				m.Session.Outgoing <- []byte("SYS:DISCONNECT")
				m.Cipher = nil
				m.Session.PeerID = ""
				m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
				m.Timeline.GotoBottom()
				return m, tea.Quit
			}
			
			sharedKey, err := crypto.DeriveSharedKey(m.Identity.PrivateKey, peerPub)
			if err == nil {
				m.Cipher, _ = crypto.NewCipherEngine(sharedKey)
				peerFingerprint := crypto.FingerprintPubKey(peerPub)
				
				buf := new(bytes.Buffer)
				qrterminal.GenerateHalfBlock(peerFingerprint, qrterminal.L, buf)
				
				m.Messages = append(m.Messages, "[SYS] Cryptographic handshake completed.")
				m.Messages = append(m.Messages, buf.String())
				m.Messages = append(m.Messages, fmt.Sprintf("⚠️ VERIFY PEER FINGERPRINT: %s", peerFingerprint))
				m.Messages = append(m.Messages, fmt.Sprintf("⚠️ YOUR FINGERPRINT: %s", m.Identity.Fingerprint()))
				m.Messages = append(m.Messages, "Do the fingerprints match? [y/n]")
				m.State = StateAuthVerify
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
				if m.RoomID == "" {
					m.Cipher.RatchetKey()
				}
				if bytes.HasPrefix(decrypted, []byte("FILE:")) {
					parts := bytes.SplitN(decrypted, []byte(":"), 3)
					if len(parts) == 3 {
						filename := string(parts[1])
						fileData := parts[2]
						
						fileIdBytes := make([]byte, 16)
						rand.Read(fileIdBytes)
						secureHex := hex.EncodeToString(fileIdBytes)
						fileKey := "download_" + m.Identity.UniqueID + "_" + secureHex
						m.Hub.StoreFile(fileKey, fileData)
						
						m.Messages = append(m.Messages, fmt.Sprintf("Peer sent file: %s", filename))
						m.Messages = append(m.Messages, "[SYS] Download command added to sidebar!")
						m.PendingFile = fmt.Sprintf("scp -O -P 23234 localhost:%s ./%s", fileKey, filename)
					}
				} else if bytes.HasPrefix(decrypted, []byte("PING:")) {
					pongPayload := []byte(strings.Replace(string(decrypted), "PING:", "PONG:", 1))
					cipherText, _ := m.Cipher.Encrypt(pongPayload)
					if m.RoomID == "" {
						m.Cipher.RatchetKey()
					}
					select {
					case m.Session.Outgoing <- cipherText:
					default:
					}
				} else if bytes.HasPrefix(decrypted, []byte("PONG:")) {
					parts := bytes.Split(decrypted, []byte(":"))
					if len(parts) == 2 {
						var sentNano int64
						fmt.Sscanf(string(parts[1]), "%d", &sentNano)
						if sentNano > 0 {
							latency := time.Now().UnixNano() - sentNano
							m.Ping = fmt.Sprintf("%dms", latency/1e6)
						}
					}
				} else if bytes.HasPrefix(decrypted, []byte("MSG:")) {
					parts := bytes.SplitN(decrypted, []byte(":"), 3)
					if len(parts) == 3 {
						m.MessageCount++
						senderName := "Peer_" + string(parts[1])
						timestamp := time.Now().Format("15:04")
						m.Messages = append(m.Messages, fmt.Sprintf("[%s] 👤 %s: %s", timestamp, senderName, string(parts[2])))
					}
				} else {
					m.MessageCount++
					senderName := "Peer"
					if len(m.Session.PeerID) >= 3 {
						senderName = "Peer_" + m.Session.PeerID[:3]
					}
					timestamp := time.Now().Format("15:04")
					m.Messages = append(m.Messages, fmt.Sprintf("[%s] 👤 %s: %s", timestamp, senderName, string(decrypted)))
				}
			} else {
				m.Messages = append(m.Messages, "[SYS] Received malformed encrypted payload")
			}
			
			if len(m.Messages) > 100 {
				m.Messages = m.Messages[len(m.Messages)-100:]
			}
			m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
			m.Timeline.GotoBottom()
		}
		return m, waitForMessage(m.Session.Incoming)

	case fileUploadMsg:
		if m.Cipher == nil {
			m.Messages = append(m.Messages, "[SYS] 🚨 Cannot upload: No secure connection established!")
			if len(m.Messages) > 100 {
				m.Messages = m.Messages[len(m.Messages)-100:]
			}
			m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
			m.Timeline.GotoBottom()
			return m, waitForUpload(m.Session.Uploads)
		}

		cipherText, _ := m.Cipher.Encrypt([]byte(msg))
		if m.RoomID != "" {
			m.Hub.Broadcast(m.RoomID, m.Identity.UniqueID, cipherText)
		} else {
			m.Session.Outgoing <- cipherText
			m.Cipher.RatchetKey()
		}
		
		// Parse the filename from the payload to show in UI
		parts := bytes.SplitN([]byte(msg), []byte(":"), 3)
		if len(parts) >= 2 {
			filename := string(parts[1])
			m.Messages = append(m.Messages, fmt.Sprintf("[SYS] File '%s' securely transmitted!", filename))
		} else {
			m.Messages = append(m.Messages, "[SYS] File securely transmitted!")
		}
		
		m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
		m.Timeline.GotoBottom()
		return m, waitForUpload(m.Session.Uploads)

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		
		headerHeight := 3
		footerHeight := 2
		contentHeight := m.Height - headerHeight - footerHeight - 2
		
		m.Timeline.Width = int(float64(m.Width)*0.70) - 4
		m.Timeline.Height = contentHeight

		m.Timeline.SetContent(strings.Join(m.Messages, "\n"))
		m.Timeline.GotoBottom()
	}

	if m.State == StateChat || m.State == StateRoomJoin {
		m.Input, cmd = m.Input.Update(msg)
		cmds = append(cmds, cmd)
		m.Timeline, cmd = m.Timeline.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}
