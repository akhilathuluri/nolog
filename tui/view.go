package tui

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.Width == 0 {
		return "Initializing Cyber-Core..."
	}

	header := lipgloss.JoinHorizontal(lipgloss.Bottom,
		StyleBrand.Render(" ⚡ STEALTH ENGINE "),
		" ",
		func() string {
			if m.Session.PeerID != "" {
				return StyleStatusConnected.Render(fmt.Sprintf("STATUS: SECURE 🔒 (%s)", m.Session.PeerID))
			}
			return StyleStatusWaiting.Render(fmt.Sprintf("AWAITING CONNECTION %s", m.Spinner.View()))
		}(),
	)

	var leftCol string
	leftWidth := int(float64(m.Width)*0.68) - 2
	leftHeight := m.Height - 6
	
	if m.State == StateChat || m.State == StateAuthVerify {
		leftCol = RenderPanel("📜 Secure Timeline", m.Timeline.View(), leftWidth, leftHeight, ColorAccent)
	} else if m.State == StateRoomJoin {
		leftCol = RenderPanel("🔑 Room Matrix", "\n  PASTE MATRIX JOIN CODE:\n\n  " + m.Input.View(), leftWidth, leftHeight, ColorAccent)
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

	metrics := fmt.Sprintf("RAM:   %.1f MB\nROT:   #%04d\nPING:  %s", ramMB, m.MessageCount, pingStr)

	uploadCmd := fmt.Sprintf("scp -P 23234 <file> localhost:upload_%s", m.Identity.UniqueID)
	
	var roomInfo string
	if m.RoomID != "" {
		roomInfo = fmt.Sprintf("MODE: ROOM (%s)", m.RoomID)
	} else {
		roomInfo = "MODE: PEER-TO-PEER"
	}

	rightWidth := int(float64(m.Width)*0.30) - 2
	
	identityContent := fmt.Sprintf("ID: %s\n%s", m.Identity.UniqueID, roomInfo)
	telemetryContent := fmt.Sprintf("CRYPTO: XChaCha20-P1305\n%s", metrics)

	shortcutsContent := "[Ctrl+Q] TERMINATE    [Ctrl+Y] COPY ID\n[Ctrl+U] COPY UPLOAD"
	if m.PendingFile != "" {
		shortcutsContent += "  [Ctrl+D] COPY DOWNLOAD\n"
	} else {
		shortcutsContent += "\n"
	}
	if m.RoomID != "" {
		shortcutsContent += "[Ctrl+L] LEAVE ROOM\n"
	} else {
		shortcutsContent += "[Ctrl+R] CREATE ROOM  [Ctrl+J] JOIN ROOM\n"
	}
	
	uploadContent := "UPLOAD URL:\n" + uploadCmd
	if m.PendingFile != "" {
		uploadContent += fmt.Sprintf("\n\nDOWNLOAD URL:\n%s", m.PendingFile)
	}

	// Build the right column panels
	idPanel := RenderPanel("🔑 Identity", identityContent, rightWidth, 4, ColorPrimary)
	telPanel := RenderPanel("⚡ Telemetry", telemetryContent, rightWidth, 5, ColorSecondary)
	cmdPanel := RenderPanel("⌨️ Commands", strings.TrimSpace(shortcutsContent), rightWidth, 5, ColorPrimary)
	scpPanel := RenderPanel("🌐 Network SCP", uploadContent, rightWidth, 8, ColorSecondary)

	rightCol := lipgloss.JoinVertical(lipgloss.Left, idPanel, telPanel, cmdPanel, scpPanel)

	var mainBody string
	if m.Width < 80 {
		mainBody = lipgloss.JoinVertical(lipgloss.Left, leftCol, rightCol)
	} else {
		mainBody = lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "  ", rightCol)
	}

	prompt := StylePrompt.Render("💬 [SYS]: _ ")
	if m.State == StateRoomJoin {
		prompt = StylePrompt.Render("🔑 [JOIN]: _ ")
	} else if m.State == StateAuthVerify {
		prompt = StylePrompt.Render("🛡️ [VERIFY y/n]: _ ")
	}
	
	footerContent := lipgloss.JoinHorizontal(lipgloss.Left, prompt, m.Input.View())
	footerPanel := RenderPanel("💬 Terminal Input", footerContent, m.Width - 4, 3, ColorFaint)

	return StyleRoot.Render(lipgloss.JoinVertical(lipgloss.Left, header, "", mainBody, footerPanel))
}
