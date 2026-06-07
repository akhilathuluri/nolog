package tui

import (
	"strings"
	"github.com/charmbracelet/lipgloss"
)

// Theme tokens (Gruvbox Dashboard Theme)
var (
	ColorPrimary   = lipgloss.Color("#D79921") // Gruvbox Yellow/Gold
	ColorSecondary = lipgloss.Color("#A89984") // Gruvbox Gray/Beige
	ColorAccent    = lipgloss.Color("#E5C07B") // Muted Gold
	ColorFaint     = lipgloss.Color("#504945") // Gruvbox Dark Gray
	ColorText      = lipgloss.Color("#EBDBB2") // Gruvbox Light/Off-White
	ColorWarning   = lipgloss.Color("#CC241D") // Gruvbox Red

	StyleRoot = lipgloss.NewStyle().Padding(1, 2).Foreground(ColorText)
	
	StyleBrand = lipgloss.NewStyle().
		Background(ColorPrimary).
		Foreground(lipgloss.Color("#282828")).
		Bold(true).
		Padding(0, 2).
		MarginRight(2)

	StyleStatusConnected = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Padding(0, 1)
	StyleStatusWaiting   = lipgloss.NewStyle().Foreground(ColorPrimary).Italic(true).Padding(0, 1)

	StyleMessagePeer = lipgloss.NewStyle().
		Foreground(ColorText).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(ColorSecondary).
		PaddingLeft(1)

	StyleMessageSys = lipgloss.NewStyle().Foreground(ColorWarning).Italic(true)

	StylePrompt = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
)

// RenderPanel simulates a dashboard pane with an embedded title like ╭─ 📜 Title ─╮
func RenderPanel(title string, content string, width, height int, borderColor lipgloss.Color) string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), false, true, true, true).
		BorderForeground(borderColor).
		Width(width - 2). // Lipgloss width includes borders, but if we construct top line manually we must match widths. Wait, lipgloss Width sets the content width.
		Height(height - 1).
		Padding(0, 1).
		Render(content)

	innerWidth := width - 2
	if innerWidth < 0 {
		innerWidth = 0
	}
	
	topBorderRunes := []rune(strings.Repeat("─", innerWidth))
	titleStr := " " + title + " "
	titleRunes := []rune(titleStr)
	
	if innerWidth > len(titleRunes)+2 {
		copy(topBorderRunes[1:], titleRunes)
	}

	topLine := "╭" + string(topBorderRunes) + "╮"
	topLine = lipgloss.NewStyle().Foreground(borderColor).Render(topLine)

	return lipgloss.JoinVertical(lipgloss.Left, topLine, box)
}
