package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	wg "github.com/mlu/wireguard-tui/internal/wg"
)

type importModel struct {
	pathInput textinput.Model
	preview   string
	parsed    *wg.Interface
	err       error
	confirmed bool
}

// importDoneMsg is sent after an import config has been successfully saved.
type importDoneMsg struct{ name string }

func newImportModel() importModel {
	ti := textinput.New()
	ti.Placeholder = "/path/to/config.conf"
	ti.CharLimit = 256
	ti.Focus()

	return importModel{
		pathInput: ti,
	}
}

func (a App) updateImport(msg tea.Msg) (App, tea.Cmd) {
	im := &a.importView

	switch msg := msg.(type) {
	case importDoneMsg:
		a.message = fmt.Sprintf("Imported profile %q", msg.name)
		a.currentView = viewList
		return a, loadProfiles()

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if im.parsed == nil {
				// First enter: read and parse the file
				path := strings.TrimSpace(im.pathInput.Value())
				if path == "" {
					im.err = fmt.Errorf("file path is required")
					return a, nil
				}

				f, err := os.Open(path)
				if err != nil {
					im.err = fmt.Errorf("opening file: %w", err)
					return a, nil
				}
				defer f.Close()

				iface, err := wg.ParseConfig(f)
				if err != nil {
					im.err = fmt.Errorf("parsing config: %w", err)
					return a, nil
				}

				// Derive name from filename (strip .conf extension)
				base := filepath.Base(path)
				name := strings.TrimSuffix(base, ".conf")
				iface.Name = name

				im.parsed = iface
				im.preview = wg.MarshalConfig(iface)
				im.err = nil
				im.pathInput.Blur()
				return a, nil
			}

			// Second enter: confirm import — save to /etc/wireguard/
			iface := im.parsed
			name := iface.Name
			return a, func() tea.Msg {
				if err := wg.SaveConfig(configDir, iface); err != nil {
					return errMsg{err: err}
				}
				return importDoneMsg{name: name}
			}

		case "esc":
			if im.parsed != nil {
				// Clear preview, go back to path input
				im.parsed = nil
				im.preview = ""
				im.err = nil
				im.pathInput.Focus()
				return a, nil
			}
			// At path input: go back to list
			a.currentView = viewList
			return a, nil
		}

		// Delegate text input updates when path input is focused
		if im.parsed == nil {
			var cmd tea.Cmd
			im.pathInput, cmd = im.pathInput.Update(msg)
			return a, cmd
		}
	}

	return a, nil
}

func (i importModel) view(width, height int) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Import Profile"))
	b.WriteString("\n\n")

	if i.parsed != nil {
		// Preview mode
		name := i.parsed.Name
		b.WriteString("  " + descStyle.Render("Preview of "+name+".conf:"))
		b.WriteString("\n\n")

		// Show config preview in a box
		configBox := boxStyle.Render(i.preview)
		b.WriteString(configBox)
		b.WriteString("\n\n")

		b.WriteString("  " + labelStyle.Render("Import as:") + valueStyle.Render(name))
		b.WriteString("\n\n")

		help := helpKey("enter", "confirm import") + "  " + helpKey("esc", "back")
		b.WriteString(help)
	} else {
		// Path input mode
		b.WriteString("  " + labelStyle.Render("File path:") + i.pathInput.View())
		b.WriteString("\n\n")

		if i.err != nil {
			b.WriteString("  " + errorStyle.Render(i.err.Error()))
			b.WriteString("\n\n")
		}

		help := helpKey("enter", "load") + "  " + helpKey("esc", "cancel")
		b.WriteString(help)
	}

	return b.String()
}
