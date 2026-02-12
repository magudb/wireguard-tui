package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	wg "github.com/mlu/wireguard-tui/internal/wg"
)

// confirmAction is an interface for actions that can be executed after
// confirmation. The execute method runs the action and returns a tea.Msg
// indicating success or failure.
type confirmAction interface {
	execute() tea.Msg
}

// deleteAction deletes a WireGuard profile. If the interface is currently up,
// it is brought down first before the config file is removed.
type deleteAction struct {
	name string
}

func (d deleteAction) execute() tea.Msg {
	// If up, bring down first
	up, _ := wg.IsUp(d.name)
	if up {
		if err := wg.Down(d.name); err != nil {
			return errMsg{err}
		}
	}
	// Delete config file
	if err := wg.DeleteConfig(configDir, d.name); err != nil {
		return errMsg{err}
	}
	return deletedMsg{name: d.name}
}

type deletedMsg struct{ name string }

type confirmModel struct {
	message  string
	action   confirmAction
	selected int // 0 = yes, 1 = no
}

func newConfirmModel(msg string, action confirmAction) confirmModel {
	return confirmModel{
		message:  msg,
		action:   action,
		selected: 1, // Default to No
	}
}

func (a App) updateConfirm(msg tea.Msg) (App, tea.Cmd) {
	switch msg := msg.(type) {
	case deletedMsg:
		a.message = fmt.Sprintf("Deleted profile %q", msg.name)
		a.currentView = viewList
		return a, tea.Batch(loadProfiles(), clearMessageAfter(3*time.Second))

	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			a.confirm.selected = 0
		case "right", "l":
			a.confirm.selected = 1

		case "y":
			action := a.confirm.action
			a.currentView = viewList
			return a, func() tea.Msg {
				return action.execute()
			}

		case "n", "esc":
			a.currentView = viewDetail
			return a, nil

		case "enter":
			if a.confirm.selected == 0 {
				action := a.confirm.action
				a.currentView = viewList
				return a, func() tea.Msg {
					return action.execute()
				}
			}
			// selected == 1 (No) — go back to detail
			a.currentView = viewDetail
			return a, nil
		}
	}

	return a, nil
}

func (c confirmModel) view(width, height int) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Confirm"))
	b.WriteString("\n\n")

	b.WriteString("  " + c.message)
	b.WriteString("\n\n")

	// Render yes/no buttons
	yesStyle := lipgloss.NewStyle().Foreground(colorDim)
	noStyle := lipgloss.NewStyle().Foreground(colorDim)

	var yesText, noText string
	if c.selected == 0 {
		yesStyle = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
		yesText = yesStyle.Render("[ Yes ]")
	} else {
		yesText = yesStyle.Render("  Yes  ")
	}

	if c.selected == 1 {
		noStyle = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
		noText = noStyle.Render("[ No ]")
	} else {
		noText = noStyle.Render("  No  ")
	}

	b.WriteString("  " + yesText + "    " + noText)
	b.WriteString("\n\n")

	help := helpKey("y", "yes") + "  " + helpKey("n", "no")
	b.WriteString("  " + help)

	return b.String()
}
