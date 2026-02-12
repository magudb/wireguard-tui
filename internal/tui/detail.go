package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	wg "github.com/mlu/wireguard-tui/internal/wg"
)

type detailModel struct {
	profile *wg.Interface
	isUp    bool
}

func newDetailModel(profile *wg.Interface, isUp bool) detailModel {
	return detailModel{
		profile: profile,
		isUp:    isUp,
	}
}

func (a App) updateDetail(msg tea.Msg) (App, tea.Cmd) {
	return a, nil
}

func (d detailModel) view(width, height int) string {
	return "Detail view (TODO)"
}
