package tui

import tea "github.com/charmbracelet/bubbletea"

type wizardModel struct{}

func newWizardModel() wizardModel {
	return wizardModel{}
}

func (a App) updateWizard(msg tea.Msg) (App, tea.Cmd) {
	return a, nil
}

func (w wizardModel) view(width, height int) string {
	return "Wizard (TODO)"
}
