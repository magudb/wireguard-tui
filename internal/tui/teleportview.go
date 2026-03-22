package tui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mlu/wireguard-tui/internal/teleport"
	wg "github.com/mlu/wireguard-tui/internal/wg"
)

var validProfileName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type teleportMode int

const (
	teleportSetup teleportMode = iota
)

type teleportModel struct {
	mode        teleportMode
	nameInput   textinput.Model
	pinInput    textinput.Model
	focusIndex  int // 0=name, 1=pin
	connecting  bool
	err         error
}

type teleportDoneMsg struct {
	name       string
	configText string
}

// teleportErrMsg is a teleport-specific error that bypasses the global errMsg handler.
type teleportErrMsg struct{ err error }

func newTeleportSetupModel() teleportModel {
	name := textinput.New()
	name.Placeholder = "e.g. amplifi-home"
	name.Focus()
	name.CharLimit = 15

	pin := textinput.New()
	pin.Placeholder = "e.g. AB123"
	pin.CharLimit = 10

	return teleportModel{
		mode:       teleportSetup,
		nameInput:  name,
		pinInput:   pin,
		focusIndex: 0,
	}
}

func (a App) updateTeleport(msg tea.Msg) (App, tea.Cmd) {
	switch msg := msg.(type) {
	case teleportDoneMsg:
		a.teleportView.connecting = false
		iface, err := wg.ParseConfigFromString(msg.configText)
		if err != nil {
			a.teleportView.err = fmt.Errorf("parsing generated config: %w", err)
			return a, nil
		}
		iface.Name = msg.name
		if err := wg.SaveConfig(configDir, iface); err != nil {
			a.teleportView.err = fmt.Errorf("saving config: %w", err)
			return a, nil
		}
		a.message = fmt.Sprintf("Teleport profile %q created", msg.name)
		a.currentView = viewList
		return a, tea.Batch(loadProfiles(), clearMessages())

	case teleportErrMsg:
		a.teleportView.connecting = false
		a.teleportView.err = msg.err
		return a, nil

	case tea.KeyMsg:
		if a.teleportView.connecting {
			return a, nil
		}

		switch msg.String() {
		case "esc":
			a.currentView = viewList
			return a, loadProfiles()

		case "tab", "shift+tab":
			if a.teleportView.mode == teleportSetup {
				if a.teleportView.focusIndex == 0 {
					a.teleportView.focusIndex = 1
					a.teleportView.nameInput.Blur()
					a.teleportView.pinInput.Focus()
				} else {
					a.teleportView.focusIndex = 0
					a.teleportView.pinInput.Blur()
					a.teleportView.nameInput.Focus()
				}
			}

		case "enter":
			if a.teleportView.mode == teleportSetup {
				name := strings.TrimSpace(a.teleportView.nameInput.Value())
				pin := strings.TrimSpace(a.teleportView.pinInput.Value())
				if name == "" {
					a.teleportView.err = fmt.Errorf("profile name is required")
					return a, nil
				}
				if !validProfileName.MatchString(name) {
					a.teleportView.err = fmt.Errorf("name must contain only letters, numbers, hyphens, underscores")
					return a, nil
				}
				if pin == "" {
					a.teleportView.err = fmt.Errorf("PIN is required")
					return a, nil
				}
				a.teleportView.connecting = true
				a.teleportView.err = nil
				return a, connectTeleport(pin, name)
			}
		}

		// Forward key to focused input
		if a.teleportView.mode == teleportSetup {
			var cmd tea.Cmd
			if a.teleportView.focusIndex == 0 {
				a.teleportView.nameInput, cmd = a.teleportView.nameInput.Update(msg)
			} else {
				a.teleportView.pinInput, cmd = a.teleportView.pinInput.Update(msg)
			}
			return a, cmd
		}
	}

	return a, nil
}

func connectTeleport(pin, name string) tea.Cmd {
	return func() tea.Msg {
		result, err := teleport.Connect(pin, name)
		if err != nil {
			return teleportErrMsg{err: err}
		}
		return teleportDoneMsg{name: result.Name, configText: result.ConfigText}
	}
}

// teleportToggleDoneMsg signals that a Teleport config was regenerated and the interface toggled.
type teleportToggleDoneMsg struct {
	name  string
	nowUp bool
}

// teleportToggleCmd regenerates the Teleport config before toggling.
// If the interface is up, it just brings it down (no regen needed).
// If the interface is down, it regenerates config via WebRTC then brings it up.
func teleportToggleCmd(name string) tea.Cmd {
	return func() tea.Msg {
		up, err := wg.IsUp(name)
		if err != nil {
			return errMsg{fmt.Errorf("checking interface state: %w", err)}
		}

		// Toggling OFF: just bring it down, no regen needed
		if up {
			if err := wg.Down(name); err != nil {
				return errMsg{err}
			}
			return teleportToggleDoneMsg{name: name, nowUp: false}
		}

		// Toggling ON: regenerate config first
		result, err := teleport.Connect("", name)
		if err != nil {
			return errMsg{fmt.Errorf("regenerating config: %w", err)}
		}

		iface, err := wg.ParseConfigFromString(result.ConfigText)
		if err != nil {
			return errMsg{fmt.Errorf("parsing generated config: %w", err)}
		}
		iface.Name = name

		if err := wg.SaveConfig(configDir, iface); err != nil {
			return errMsg{fmt.Errorf("saving config: %w", err)}
		}

		if err := wg.Up(name); err != nil {
			return errMsg{err}
		}

		return teleportToggleDoneMsg{name: name, nowUp: true}
	}
}

func (m teleportModel) view(width, height int) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Amplifi Teleport"))
	b.WriteString("\n\n")
	b.WriteString("  " + descStyle.Render("Connect to your Amplifi router via Teleport."))
	b.WriteString("\n")
	b.WriteString("  " + descStyle.Render("Get a PIN from the AmpliFi app → Teleport → Add Device."))
	b.WriteString("\n\n")

	b.WriteString("  " + labelStyle.Render("Profile Name:"))
	b.WriteString(m.nameInput.View())
	b.WriteString("\n\n")

	b.WriteString("  " + labelStyle.Render("PIN:"))
	b.WriteString(m.pinInput.View())
	b.WriteString("\n\n")

	if m.connecting {
		b.WriteString("  " + descStyle.Render("Connecting to Amplifi router..."))
	} else if m.err != nil {
		b.WriteString("  " + wrapError(m.err, width))
	}

	b.WriteString("\n")
	help := helpKey("tab", "next field") + "  " +
		helpKey("enter", "connect") + "  " +
		helpKey("esc", "back")
	b.WriteString(help)

	return b.String()
}
