package tui

import tea "github.com/charmbracelet/bubbletea"

type statusModel struct {
	name string
}

func newStatusModel(name string) statusModel {
	return statusModel{name: name}
}

func (s statusModel) init() tea.Cmd {
	return nil
}

func (a App) updateStatus(msg tea.Msg) (App, tea.Cmd) {
	return a, nil
}

func (s statusModel) view(width, height int) string {
	return "Status (TODO)"
}
