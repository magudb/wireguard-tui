package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	wg "github.com/mlu/wireguard-tui/internal/wg"
)

type editorModel struct {
	profile *wg.Interface
}

func newEditorModel(profile *wg.Interface) editorModel {
	return editorModel{profile: profile}
}

func (a App) updateEditor(msg tea.Msg) (App, tea.Cmd) {
	return a, nil
}

func (e editorModel) view(width, height int) string {
	return "Editor (TODO)"
}
