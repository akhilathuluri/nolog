package tui

import (
	"fmt"
	"runtime"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.Width == 0 {
		return "Initializing..."
	}

	header := lipgloss.JoinHorizontal(lipgloss.Bottom,
		StyleBrand.Render("🔒 STEALTH ENGINE"),
		" ",
		StyleStatus.Render(fmt.Sprintf("Status: %s", func() string {
			if m.Session.PeerID != "" {
				return "Connected"
			}
			return "Waiting for peer"
		}())),
	)
	
	headerBlock := StyleHeader.Width(m.Width - 2).Render(header)

	var leftCol string
	if m.State == StateChat || m.State == StateAuthVerify {
		leftCol = StyleTimeline.Width(int(float64(m.Width)*0.70) - 2).Render(m.Timeline.View())
	} else if m.State == StateRoomJoin {
		leftCol = StyleTimeline.Width(int(float64(m.Width)*0.70) - 2).Render("\n  Paste Join Code:\n  " + m.Input.View())
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	ramMB := float64(mem.Alloc) / 1024 / 1024

	pingStr := m.Ping
	if pingStr == "" {
		if m.RoomID != "" {
			pingStr = "N/A"
		} else {
			pingStr = "..."
		}
	}

	metrics := fmt.Sprintf("Volatile RAM: %.1f MB\nKey Ratchet:  #%04d\nLive Ping:    %s", ramMB, m.MessageCount, pingStr)

	var telemetry string
	uploadCmd := fmt.Sprintf("scp -O -P 23234 <file> localhost:upload_%s", m.Identity.UniqueID)
	if m.RoomID != "" {
		telemetry = fmt.Sprintf("ID 📋:\n%s\n\nRoom:\n%s\n\nCrypto: ChaCha20-Poly1305\n%s\n\nUpload File:\n%s\n\n[Ctrl+Q] Panic Exit\n[Ctrl+Y] Copy ID\n[Ctrl+U] Copy Upload\n[Ctrl+L] Leave Room",
			m.Identity.UniqueID, m.RoomID, metrics, uploadCmd)
	} else {
		telemetry = fmt.Sprintf("ID 📋:\n%s\n\nPeer:\n%s\n\nCrypto: ChaCha20-Poly1305\n%s\n\nUpload File:\n%s\n\n[Ctrl+Q] Panic Exit\n[Ctrl+Y] Copy ID\n[Ctrl+U] Copy Upload\n[Ctrl+R] Create Room\n[Ctrl+J] Join Room",
			m.Identity.UniqueID, func() string {
				if m.Session.PeerID != "" {
					return m.Session.PeerID
				}
				return "Waiting for peer..."
			}(), metrics, uploadCmd)
	}

	if m.PendingFile != "" {
		telemetry += fmt.Sprintf("\n\nFile Ready:\n%s\n\n[Ctrl+D] Copy Download", m.PendingFile)
	}
		
	rightCol := StyleTelemetry.Width(int(float64(m.Width)*0.30) - 2).Render(telemetry)

	var mainBody string
	if m.Width < 80 {
		// Stacked view
		mainBody = lipgloss.JoinVertical(lipgloss.Left, leftCol, rightCol)
	} else {
		// Dual column
		mainBody = lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)
	}

	prompt := StylePrompt.Render("💬 [SYS]: _ ")
	if m.State == StateRoomJoin {
		prompt = StylePrompt.Render("🔑 [JOIN]: _ ")
	} else if m.State == StateAuthVerify {
		prompt = StylePrompt.Render("🛡️ [VERIFY y/n]: _ ")
	}
	footer := StyleFooter.Render(lipgloss.JoinHorizontal(lipgloss.Left, prompt, m.Input.View()))

	return StyleRoot.Render(lipgloss.JoinVertical(lipgloss.Left, headerBlock, mainBody, footer))
}
