package tui

import (
	"fmt"

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
	if m.State == StateChat {
		leftCol = StyleTimeline.Width(int(float64(m.Width)*0.70) - 2).Render(m.Timeline.View())
	} else {
		leftCol = StyleTimeline.Width(int(float64(m.Width)*0.70) - 2).Render(m.FilePicker.View())
	}

	telemetry := fmt.Sprintf("ID 📋:\n%s\n\nPeer:\n%s\n\nCrypto: ChaCha20-Poly1305\n\n[Ctrl+Q] Panic Exit\n[Ctrl+F] Send File\n[Ctrl+Y] Copy ID",
		m.Identity.UniqueID, func() string {
			if m.Session.PeerID != "" {
				return m.Session.PeerID
			}
			return "None"
		}())
		
	if m.PendingFile != "" {
		telemetry += fmt.Sprintf("\n\nFile Ready:\nscp -O -P 23234 localhost:download_%s ./%s\n\n[Ctrl+G] Copy SCP", m.Identity.UniqueID, m.PendingFile)
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
	if m.State == StateFilePicker {
		prompt = StylePrompt.Render("📁 [FILE]: _ ")
	}
	footer := StyleFooter.Render(lipgloss.JoinHorizontal(lipgloss.Left, prompt, m.Input.View()))

	return StyleRoot.Render(lipgloss.JoinVertical(lipgloss.Left, headerBlock, mainBody, footer))
}
