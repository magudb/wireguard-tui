package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type viewType int

const (
	viewList viewType = iota
	viewDetail
	viewWizard
	viewEditor
	viewStatus
	viewImport
	viewExport
	viewConfirm
)

const configDir = "/etc/wireguard"

// Custom message types
type errMsg struct{ err error }
type clearErrMsg struct{}
type navigateMsg struct{ view viewType }
type refreshMsg struct{}

// clearMessages returns a command that clears err and message after 3 seconds.
func clearMessages() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return clearErrMsg{}
	})
}

// App is the root Bubble Tea model. It manages navigation between views.
type App struct {
	currentView viewType

	list       listModel
	detail     detailModel
	wizard     wizardModel
	editor     editorModel
	status     statusModel
	importView importModel
	exportView exportModel
	confirm    confirmModel

	width   int
	height  int
	err     error
	message string
}

// NewApp creates a new App starting at the list view.
func NewApp() App {
	return App{
		currentView: viewList,
		list:        newListModel(),
	}
}

// Init implements tea.Model. It loads profiles on startup.
func (a App) Init() tea.Cmd {
	return loadProfiles()
}

// Update implements tea.Model.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

	case errMsg:
		a.err = msg.err
		return a, clearMessages()

	case clearErrMsg:
		a.err = nil
		a.message = ""
		return a, nil

	case navigateMsg:
		a.currentView = msg.view
		return a, nil
	}

	// Delegate to the current view's update method
	var cmd tea.Cmd
	switch a.currentView {
	case viewList:
		a, cmd = a.updateList(msg)
	case viewDetail:
		a, cmd = a.updateDetail(msg)
	case viewWizard:
		a, cmd = a.updateWizard(msg)
	case viewEditor:
		a, cmd = a.updateEditor(msg)
	case viewStatus:
		a, cmd = a.updateStatus(msg)
	case viewImport:
		a, cmd = a.updateImport(msg)
	case viewExport:
		a, cmd = a.updateExport(msg)
	case viewConfirm:
		a, cmd = a.updateConfirm(msg)
	}

	return a, cmd
}

// View implements tea.Model.
func (a App) View() string {
	var content string
	switch a.currentView {
	case viewList:
		content = a.list.view(a.width, a.height)
	case viewDetail:
		content = a.detail.view(a.width, a.height)
	case viewWizard:
		content = a.wizard.view(a.width, a.height)
	case viewEditor:
		content = a.editor.view(a.width, a.height)
	case viewStatus:
		content = a.status.view(a.width, a.height)
	case viewImport:
		content = a.importView.view(a.width, a.height)
	case viewExport:
		content = a.exportView.view(a.width, a.height)
	case viewConfirm:
		content = a.confirm.view(a.width, a.height)
	}

	if a.err != nil {
		content += "\n" + errorStyle.Render("Error: "+a.err.Error())
	}
	if a.message != "" {
		content += "\n" + successStyle.Render(a.message)
	}

	return content
}
