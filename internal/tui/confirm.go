package tui

import tea "github.com/charmbracelet/bubbletea"

type confirmModel struct {
	message string
}

func newConfirmModel(msg string, action interface{}) confirmModel {
	return confirmModel{message: msg}
}

func (a App) updateConfirm(msg tea.Msg) (App, tea.Cmd) {
	return a, nil
}

func (c confirmModel) view(width, height int) string {
	return "Confirm (TODO)"
}
