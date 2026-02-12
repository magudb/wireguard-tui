package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	wg "github.com/mlu/wireguard-tui/internal/wg"
)

type exportModel struct {
	profile   *wg.Interface
	showQR    bool
	qrString  string
	confText  string
	pathInput textinput.Model
	saving    bool
	err       error
	message   string
}

func newExportModel(profile *wg.Interface) exportModel {
	confText := wg.MarshalConfig(profile)

	qrString := ""
	qr, err := wg.GenerateQRString(profile)
	if err == nil {
		qrString = qr
	}

	ti := textinput.New()
	ti.Placeholder = fmt.Sprintf("/home/user/%s.conf", profile.Name)
	ti.CharLimit = 256
	ti.SetValue(fmt.Sprintf("%s.conf", profile.Name))

	return exportModel{
		profile:   profile,
		showQR:    false,
		qrString:  qrString,
		confText:  confText,
		pathInput: ti,
		err:       err, // capture QR generation error if any
	}
}

// exportSavedMsg is sent after an export file has been saved.
type exportSavedMsg struct{ path string }

func (a App) updateExport(msg tea.Msg) (App, tea.Cmd) {
	ex := &a.exportView

	switch msg := msg.(type) {
	case exportSavedMsg:
		ex.saving = false
		ex.message = fmt.Sprintf("Saved to %s", msg.path)
		ex.err = nil
		ex.pathInput.Blur()
		return a, nil

	case tea.KeyMsg:
		key := msg.String()

		if ex.saving {
			switch key {
			case "enter":
				// Write config to the specified path
				path := strings.TrimSpace(ex.pathInput.Value())
				if path == "" {
					ex.err = fmt.Errorf("file path is required")
					return a, nil
				}
				confText := ex.confText
				return a, func() tea.Msg {
					if err := os.WriteFile(path, []byte(confText), 0600); err != nil {
						return errMsg{err: err}
					}
					return exportSavedMsg{path: path}
				}

			case "esc":
				// Cancel save mode
				ex.saving = false
				ex.err = nil
				ex.pathInput.Blur()
				return a, nil
			}

			// Delegate to path input
			var cmd tea.Cmd
			ex.pathInput, cmd = ex.pathInput.Update(msg)
			return a, cmd
		}

		// Not in save mode
		switch key {
		case "q":
			// Toggle QR display
			ex.showQR = !ex.showQR
			return a, nil

		case "c":
			// Show config text
			ex.showQR = false
			return a, nil

		case "s":
			// Enter save mode
			ex.saving = true
			ex.message = ""
			ex.err = nil
			ex.pathInput.Focus()
			return a, nil

		case "esc":
			// Go back to detail view
			a.currentView = viewDetail
			return a, nil
		}
	}

	return a, nil
}

func (e exportModel) view(width, height int) string {
	var b strings.Builder

	if e.saving {
		// Save mode
		b.WriteString(titleStyle.Render("Export: " + e.profile.Name))
		b.WriteString("\n\n")

		b.WriteString("  " + labelStyle.Render("Save to:") + e.pathInput.View())
		b.WriteString("\n\n")

		if e.err != nil {
			b.WriteString("  " + errorStyle.Render(e.err.Error()))
			b.WriteString("\n\n")
		}

		help := helpKey("enter", "save") + "  " + helpKey("esc", "cancel")
		b.WriteString(help)

		return b.String()
	}

	if e.showQR {
		// QR mode
		b.WriteString(titleStyle.Render("Export: " + e.profile.Name + " (QR Code)"))
		b.WriteString("\n\n")

		if e.qrString != "" {
			b.WriteString(e.qrString)
			b.WriteString("\n")
			b.WriteString("  " + descStyle.Render("Scan with WireGuard mobile app"))
			b.WriteString("\n\n")
		} else {
			b.WriteString("  " + errorStyle.Render("QR code generation failed"))
			b.WriteString("\n\n")
		}

		help := helpKey("c", "show config") + "  " + helpKey("s", "save to file") + "  " + helpKey("esc", "back")
		b.WriteString(help)
	} else {
		// Config text mode
		b.WriteString(titleStyle.Render("Export: " + e.profile.Name))
		b.WriteString("\n\n")

		configBox := boxStyle.Render(e.confText)
		b.WriteString(configBox)
		b.WriteString("\n\n")

		if e.message != "" {
			b.WriteString("  " + successStyle.Render(e.message))
			b.WriteString("\n\n")
		}

		help := helpKey("q", "show QR") + "  " + helpKey("s", "save to file") + "  " + helpKey("esc", "back")
		b.WriteString(help)
	}

	return b.String()
}
