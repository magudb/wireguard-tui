package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	wg "github.com/mlu/wireguard-tui/internal/wg"
)

type detailModel struct {
	profile *wg.Interface
	isUp    bool
}

type toggledMsg struct {
	name  string
	nowUp bool
}

func newDetailModel(profile *wg.Interface, isUp bool) detailModel {
	return detailModel{
		profile: profile,
		isUp:    isUp,
	}
}

func (a App) updateDetail(msg tea.Msg) (App, tea.Cmd) {
	switch msg := msg.(type) {
	case toggledMsg:
		a.detail.isUp = msg.nowUp
		state := "DOWN"
		if msg.nowUp {
			state = "UP"
		}
		a.message = fmt.Sprintf("%s is now %s", msg.name, state)
		return a, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			a.currentView = viewList
			return a, loadProfiles()

		case "e":
			a.editor = newEditorModel(a.detail.profile)
			a.currentView = viewEditor
			return a, nil

		case "s":
			a.status = newStatusModel(a.detail.profile.Name)
			a.currentView = viewStatus
			return a, a.status.init()

		case "t":
			name := a.detail.profile.Name
			return a, func() tea.Msg {
				nowUp, err := wg.Toggle(name)
				if err != nil {
					return errMsg{err}
				}
				return toggledMsg{name: name, nowUp: nowUp}
			}

		case "x":
			a.exportView = newExportModel(a.detail.profile)
			a.currentView = viewExport
			return a, nil

		case "d":
			name := a.detail.profile.Name
			a.confirm = newConfirmModel(
				fmt.Sprintf("Delete profile %q?", name),
				deleteAction{name: name},
			)
			a.currentView = viewConfirm
			return a, nil
		}
	}

	return a, nil
}

func truncateKey(key string, max int) string {
	if len(key) > max {
		return key[:max] + "..."
	}
	return key
}

func (d detailModel) view(width, height int) string {
	if d.profile == nil {
		return "No profile selected"
	}

	var b strings.Builder
	p := d.profile

	b.WriteString(titleStyle.Render("Profile: " + p.Name))
	b.WriteString("\n\n")

	// Status
	status := statusDown
	if d.isUp {
		status = statusUp
	}
	b.WriteString("  " + labelStyle.Render("Status:") + status + "\n")
	b.WriteString("\n")

	// Address
	if p.Address != "" {
		b.WriteString("  " + labelStyle.Render("Address:") + valueStyle.Render(p.Address) + "\n")
	}

	// Listen Port
	if p.ListenPort != 0 {
		b.WriteString("  " + labelStyle.Render("Listen Port:") + valueStyle.Render(fmt.Sprintf("%d", p.ListenPort)) + "\n")
	}

	// DNS
	if p.DNS != "" {
		b.WriteString("  " + labelStyle.Render("DNS:") + valueStyle.Render(p.DNS) + "\n")
	}

	// MTU
	if p.MTU != 0 {
		b.WriteString("  " + labelStyle.Render("MTU:") + valueStyle.Render(fmt.Sprintf("%d", p.MTU)) + "\n")
	}

	b.WriteString("\n")

	// Peers count
	b.WriteString("  " + labelStyle.Render("Peers:") + valueStyle.Render(fmt.Sprintf("%d", len(p.Peers))) + "\n")

	// Individual peers
	for i, peer := range p.Peers {
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  Peer %d:\n", i+1))

		if peer.PublicKey != "" {
			b.WriteString("    " + labelStyle.Render("Public Key:") + valueStyle.Render(truncateKey(peer.PublicKey, 20)) + "\n")
		}
		if peer.Endpoint != "" {
			b.WriteString("    " + labelStyle.Render("Endpoint:") + valueStyle.Render(peer.Endpoint) + "\n")
		}
		if peer.AllowedIPs != "" {
			b.WriteString("    " + labelStyle.Render("Allowed IPs:") + valueStyle.Render(peer.AllowedIPs) + "\n")
		}
		if peer.PersistentKeepalive != 0 {
			b.WriteString("    " + labelStyle.Render("Keepalive:") + valueStyle.Render(fmt.Sprintf("%d", peer.PersistentKeepalive)) + "\n")
		}
	}

	b.WriteString("\n")
	help := helpKey("e", "edit") + "  " +
		helpKey("s", "status") + "  " +
		helpKey("t", "toggle") + "  " +
		helpKey("x", "export") + "  " +
		helpKey("d", "delete") + "  " +
		helpKey("esc", "back")
	b.WriteString(help)

	return b.String()
}
