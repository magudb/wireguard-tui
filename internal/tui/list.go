package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	wg "github.com/mlu/wireguard-tui/internal/wg"
)

type listModel struct {
	profiles []*wg.Interface
	active   map[string]bool
	cursor   int
}

func newListModel() listModel {
	return listModel{
		active: make(map[string]bool),
	}
}

type profilesLoadedMsg struct {
	profiles []*wg.Interface
	active   map[string]bool
}

func loadProfiles() tea.Cmd {
	return func() tea.Msg {
		profiles, err := wg.LoadConfigsFromDir(configDir)
		if err != nil {
			return errMsg{err: err}
		}

		activeList, err := wg.ListInterfaces()
		if err != nil {
			return errMsg{err: err}
		}

		active := make(map[string]bool)
		for _, name := range activeList {
			active[name] = true
		}

		return profilesLoadedMsg{
			profiles: profiles,
			active:   active,
		}
	}
}

func (a App) updateList(msg tea.Msg) (App, tea.Cmd) {
	switch msg := msg.(type) {
	case profilesLoadedMsg:
		a.list.profiles = msg.profiles
		a.list.active = msg.active
		if a.list.cursor >= len(a.list.profiles) && len(a.list.profiles) > 0 {
			a.list.cursor = len(a.list.profiles) - 1
		}
		return a, nil

	case toggledMsg:
		a.list.active[msg.name] = msg.nowUp
		state := "DOWN"
		if msg.nowUp {
			state = "UP"
		}
		a.message = fmt.Sprintf("%s is now %s", msg.name, state)
		return a, clearMessages()

	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return a, tea.Quit
		case "up", "k":
			if a.list.cursor > 0 {
				a.list.cursor--
			}
		case "down", "j":
			if a.list.cursor < len(a.list.profiles)-1 {
				a.list.cursor++
			}
		case "enter":
			if len(a.list.profiles) > 0 {
				p := a.list.profiles[a.list.cursor]
				isUp := a.list.active[p.Name]
				a.detail = newDetailModel(p, isUp)
				a.currentView = viewDetail
			}
		case "n":
			a.wizard = newWizardModel()
			a.currentView = viewWizard
		case "i":
			a.importView = newImportModel()
			a.currentView = viewImport
		case "a":
			a.teleportView = newTeleportSetupModel()
			a.currentView = viewTeleport
			return a, nil
		case "t":
			if len(a.list.profiles) > 0 {
				name := a.list.profiles[a.list.cursor].Name
				return a, func() tea.Msg {
					nowUp, err := wg.Toggle(name)
					if err != nil {
						return errMsg{err}
					}
					return toggledMsg{name: name, nowUp: nowUp}
				}
			}
		}

	case refreshMsg:
		return a, loadProfiles()
	}

	return a, nil
}

func (l listModel) view(width, height int) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("WireGuard TUI"))
	b.WriteString("\n")

	if len(l.profiles) == 0 {
		b.WriteString("\n")
		b.WriteString(descStyle.Render("No profiles found in " + configDir))
		b.WriteString("\n")
		b.WriteString(descStyle.Render("Press [n] to create a new profile or [i] to import one."))
		b.WriteString("\n")
	} else {
		nameStyle := lipgloss.NewStyle().Bold(true).Width(15)
		addrStyle := lipgloss.NewStyle().Foreground(colorDim).Width(20)

		for i, p := range l.profiles {
			cursor := "  "
			if i == l.cursor {
				cursor = "> "
			}

			status := statusDown
			if l.active[p.Name] {
				status = statusUp
			}

			peerCount := fmt.Sprintf("%d peer", len(p.Peers))
			if len(p.Peers) != 1 {
				peerCount += "s"
			}

			line := cursor +
				nameStyle.Render(p.Name) + " " +
				addrStyle.Render(p.Address) + " " +
				status + "  " +
				descStyle.Render(peerCount)

			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	help := helpKey("n", "new") + "  " +
		helpKey("a", "amplifi") + "  " +
		helpKey("t", "toggle") + "  " +
		helpKey("i", "import") + "  " +
		helpKey("q", "quit")
	b.WriteString(help)

	return b.String()
}
