package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	wg "github.com/mlu/wireguard-tui/internal/wg"
)

type exportModel struct {
	profile *wg.Interface
}

func newExportModel(profile *wg.Interface) exportModel {
	return exportModel{profile: profile}
}

func (a App) updateExport(msg tea.Msg) (App, tea.Cmd) {
	return a, nil
}

func (e exportModel) view(width, height int) string {
	return "Export (TODO)"
}
