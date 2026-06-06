package tui

import "github.com/charmbracelet/lipgloss"

// Theme tokens (Premium Cyber-Noir Theme)
var (
	ColorPrimary   = lipgloss.Color("99")  // Purple/Violet
	ColorSecondary = lipgloss.Color("86")  // Cyan/Teal
	ColorFaint     = lipgloss.Color("237") // Dark Slate
	ColorWarning   = lipgloss.Color("204") // Amber/Coral

	StyleRoot = lipgloss.NewStyle().Padding(0, 1)
	
	StyleHeader = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(ColorFaint).
		PaddingBottom(1).
		MarginBottom(1)
		
	StyleBrand = lipgloss.NewStyle().
		Background(ColorPrimary).
		Foreground(lipgloss.Color("230")).
		Bold(true).
		Padding(0, 1)

	StyleStatus = lipgloss.NewStyle().Foreground(ColorSecondary)
	
	StyleTimeline = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1)
		
	StyleTelemetry = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderForeground(ColorFaint).
		Padding(0, 1)

	StyleMessagePeer = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(ColorSecondary).
		PaddingLeft(1)

	StyleMessageSys = lipgloss.NewStyle().Foreground(ColorWarning).Italic(true)

	StyleFooter = lipgloss.NewStyle().MarginTop(1)
	StylePrompt = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
)
