package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	colorGreen  = lipgloss.Color("42")
	colorRed    = lipgloss.Color("196")
	colorDim    = lipgloss.Color("240")
	colorAccent = lipgloss.Color("63")
	colorWhite  = lipgloss.Color("255")

	// Title
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorAccent).
		MarginBottom(1)

	// Status indicators
	statusUp = lipgloss.NewStyle().
		Foreground(colorGreen).
		Bold(true).
		Render("● UP")

	statusDown = lipgloss.NewStyle().
		Foreground(colorDim).
		Render("○ DOWN")

	// Key hints
	keyStyle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true)

	descStyle = lipgloss.NewStyle().
		Foreground(colorDim)

	// Error / success messages
	errorStyle = lipgloss.NewStyle().
		Foreground(colorRed).
		Bold(true)

	successStyle = lipgloss.NewStyle().
		Foreground(colorGreen)

	// Detail labels
	labelStyle = lipgloss.NewStyle().
		Foreground(colorDim).
		Width(20)

	valueStyle = lipgloss.NewStyle().
		Foreground(colorWhite)

	// Border box
	boxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Padding(1, 2)
)

func helpKey(key, desc string) string {
	return keyStyle.Render("["+key+"]") + " " + descStyle.Render(desc)
}
