package tui

import tea "github.com/charmbracelet/bubbletea"

type importModel struct{}

func newImportModel() importModel {
	return importModel{}
}

func (a App) updateImport(msg tea.Msg) (App, tea.Cmd) {
	return a, nil
}

func (i importModel) view(width, height int) string {
	return "Import (TODO)"
}
