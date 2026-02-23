package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	wg "github.com/mlu/wireguard-tui/internal/wg"
)

type statusTickMsg struct{}

type statusDataMsg struct {
	status *wg.InterfaceStatus
	err    error
}

type statusModel struct {
	name    string
	status  *wg.InterfaceStatus
	loading bool
	err     error
}

func newStatusModel(name string) statusModel {
	return statusModel{name: name, loading: true}
}

func (s statusModel) init() tea.Cmd {
	return fetchStatus(s.name)
}

func fetchStatus(name string) tea.Cmd {
	return func() tea.Msg {
		st, err := wg.GetStatus(name)
		return statusDataMsg{status: st, err: err}
	}
}

func statusTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return statusTickMsg{}
	})
}

func (a App) updateStatus(msg tea.Msg) (App, tea.Cmd) {
	switch msg := msg.(type) {
	case statusDataMsg:
		a.status.status = msg.status
		a.status.err = msg.err
		a.status.loading = false
		return a, statusTick()

	case statusTickMsg:
		return a, fetchStatus(a.status.name)

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			a.currentView = viewDetail
			return a, nil
		}
	}

	return a, nil
}

// formatHandshake formats a time.Duration as a human-readable "ago" string.
func formatHandshake(d time.Duration) string {
	if d == 0 {
		return "never"
	}

	totalSeconds := int(d.Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	var parts []string
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}

	return strings.Join(parts, "") + " ago"
}

func (s statusModel) view(width, height int) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf("Status: %s (live)", s.name)))
	b.WriteString("\n\n")

	if s.loading {
		b.WriteString("  Loading...\n")
		b.WriteString("\n")
		b.WriteString(helpKey("esc", "back"))
		return b.String()
	}

	if s.err != nil {
		b.WriteString("  " + errorStyle.Render(fmt.Sprintf("Error: %s", s.err.Error())) + "\n")
		b.WriteString("\n")
		b.WriteString("  " + descStyle.Render("Refreshing every 2s...") + "\n")
		b.WriteString("\n")
		b.WriteString(helpKey("esc", "back"))
		return b.String()
	}

	st := s.status

	// Interface info
	if st.PublicKey != "" {
		b.WriteString("  " + labelStyle.Render("Public Key:") + valueStyle.Render(st.PublicKey) + "\n")
	}
	if st.ListenPort != 0 {
		b.WriteString("  " + labelStyle.Render("Listen Port:") + valueStyle.Render(fmt.Sprintf("%d", st.ListenPort)) + "\n")
	}

	b.WriteString("\n")

	// Peers
	peerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		PaddingLeft(1).
		PaddingRight(1)

	for i, peer := range st.Peers {
		var peerContent strings.Builder

		truncatedKey := truncateKey(peer.PublicKey, 12)
		fmt.Fprintf(&peerContent, "Peer %d: %s\n", i+1, valueStyle.Render(truncatedKey))

		if peer.Endpoint != "" {
			peerContent.WriteString("  " + labelStyle.Render("Endpoint:") + valueStyle.Render(peer.Endpoint) + "\n")
		}
		if peer.AllowedIPs != "" {
			peerContent.WriteString("  " + labelStyle.Render("Allowed IPs:") + valueStyle.Render(peer.AllowedIPs) + "\n")
		}

		peerContent.WriteString("  " + labelStyle.Render("Latest Handshake:") + valueStyle.Render(formatHandshake(peer.LatestHandshake)) + "\n")

		if peer.TransferRx != "" || peer.TransferTx != "" {
			rx := peer.TransferRx
			tx := peer.TransferTx
			if rx == "" {
				rx = "0 B"
			}
			if tx == "" {
				tx = "0 B"
			}
			peerContent.WriteString("  " + labelStyle.Render("Transfer:") + valueStyle.Render(fmt.Sprintf("\u2193 %s  \u2191 %s", rx, tx)) + "\n")
		}

		if peer.PersistentKeepalive > 0 {
			peerContent.WriteString("  " + labelStyle.Render("Keepalive:") + valueStyle.Render(fmt.Sprintf("every %ds", peer.PersistentKeepalive)) + "\n")
		}

		rendered := peerStyle.Render(strings.TrimRight(peerContent.String(), "\n"))
		// Indent each line of the rendered box
		for j, line := range strings.Split(rendered, "\n") {
			if j > 0 {
				b.WriteString("\n")
			}
			b.WriteString("  " + line)
		}
		b.WriteString("\n")
	}

	if len(st.Peers) == 0 {
		b.WriteString("  " + descStyle.Render("No peers") + "\n")
	}

	b.WriteString("\n")
	b.WriteString("  " + descStyle.Render("Refreshing every 2s...") + "\n")
	b.WriteString("\n")
	b.WriteString(helpKey("esc", "back"))

	return b.String()
}
