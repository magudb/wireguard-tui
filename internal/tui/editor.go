package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	wg "github.com/mlu/wireguard-tui/internal/wg"
)

// editorModel holds the state for the profile editor form.
type editorModel struct {
	profile    *wg.Interface
	inputs     []textinput.Model // 4 interface fields
	focusIndex int

	// Peer editing
	peers       []wg.Peer
	peerIdx     int               // which peer is selected (-1 = none)
	peerInputs  []textinput.Model // 5 inputs for current peer
	peerFocus   int
	editingPeer bool

	err error
}

// editorSavedMsg is sent after the editor successfully saves a config.
type editorSavedMsg struct {
	profile *wg.Interface
}

// Interface field indices
const (
	editorFieldAddress    = 0
	editorFieldListenPort = 1
	editorFieldDNS        = 2
	editorFieldMTU        = 3
	editorFieldCount      = 4
)

// newEditorModel creates a new editor model pre-populated with the profile's values.
func newEditorModel(profile *wg.Interface) editorModel {
	e := editorModel{
		profile: profile,
		peerIdx: -1,
	}

	// Copy peers so we don't mutate the original
	e.peers = make([]wg.Peer, len(profile.Peers))
	copy(e.peers, profile.Peers)

	// Initialize interface field inputs
	e.inputs = make([]textinput.Model, editorFieldCount)

	// Address
	e.inputs[editorFieldAddress] = textinput.New()
	e.inputs[editorFieldAddress].Placeholder = "10.0.0.1/24"
	e.inputs[editorFieldAddress].SetValue(profile.Address)
	e.inputs[editorFieldAddress].CharLimit = 43
	e.inputs[editorFieldAddress].Focus()

	// Listen Port
	e.inputs[editorFieldListenPort] = textinput.New()
	e.inputs[editorFieldListenPort].Placeholder = "51820"
	if profile.ListenPort != 0 {
		e.inputs[editorFieldListenPort].SetValue(strconv.Itoa(profile.ListenPort))
	}
	e.inputs[editorFieldListenPort].CharLimit = 5

	// DNS
	e.inputs[editorFieldDNS] = textinput.New()
	e.inputs[editorFieldDNS].Placeholder = "1.1.1.1, 8.8.8.8"
	e.inputs[editorFieldDNS].SetValue(profile.DNS)
	e.inputs[editorFieldDNS].CharLimit = 100

	// MTU
	e.inputs[editorFieldMTU] = textinput.New()
	e.inputs[editorFieldMTU].Placeholder = ""
	if profile.MTU != 0 {
		e.inputs[editorFieldMTU].SetValue(strconv.Itoa(profile.MTU))
	}
	e.inputs[editorFieldMTU].CharLimit = 5

	// Initialize peer inputs (empty, populated when editing a peer)
	e.peerInputs = makeEditorPeerInputs()

	return e
}

// makeEditorPeerInputs creates a fresh set of 5 text inputs for peer editing.
func makeEditorPeerInputs() []textinput.Model {
	inputs := make([]textinput.Model, peerSubStepCount)

	inputs[peerStepPubKey] = textinput.New()
	inputs[peerStepPubKey].Placeholder = "base64 public key"
	inputs[peerStepPubKey].CharLimit = 44

	inputs[peerStepAllowedIPs] = textinput.New()
	inputs[peerStepAllowedIPs].Placeholder = "0.0.0.0/0, ::/0"
	inputs[peerStepAllowedIPs].CharLimit = 200

	inputs[peerStepEndpoint] = textinput.New()
	inputs[peerStepEndpoint].Placeholder = "host:port (optional)"
	inputs[peerStepEndpoint].CharLimit = 100

	inputs[peerStepPSK] = textinput.New()
	inputs[peerStepPSK].Placeholder = "base64 preshared key (optional)"
	inputs[peerStepPSK].CharLimit = 44

	inputs[peerStepKeepalive] = textinput.New()
	inputs[peerStepKeepalive].Placeholder = "25"
	inputs[peerStepKeepalive].CharLimit = 5

	return inputs
}

// populatePeerInputs fills the peer inputs from a peer struct.
func populatePeerInputs(inputs []textinput.Model, peer wg.Peer) []textinput.Model {
	inputs[peerStepPubKey].SetValue(peer.PublicKey)
	inputs[peerStepAllowedIPs].SetValue(peer.AllowedIPs)
	inputs[peerStepEndpoint].SetValue(peer.Endpoint)
	inputs[peerStepPSK].SetValue(peer.PresharedKey)
	if peer.PersistentKeepalive != 0 {
		inputs[peerStepKeepalive].SetValue(strconv.Itoa(peer.PersistentKeepalive))
	} else {
		inputs[peerStepKeepalive].SetValue("")
	}
	return inputs
}

// buildPeerFromInputs constructs a Peer from the current peer inputs.
func buildPeerFromInputs(inputs []textinput.Model) wg.Peer {
	keepalive := 0
	if ka := strings.TrimSpace(inputs[peerStepKeepalive].Value()); ka != "" {
		keepalive, _ = strconv.Atoi(ka)
	}
	return wg.Peer{
		PublicKey:           strings.TrimSpace(inputs[peerStepPubKey].Value()),
		AllowedIPs:          strings.TrimSpace(inputs[peerStepAllowedIPs].Value()),
		Endpoint:            strings.TrimSpace(inputs[peerStepEndpoint].Value()),
		PresharedKey:        strings.TrimSpace(inputs[peerStepPSK].Value()),
		PersistentKeepalive: keepalive,
	}
}

func (a App) updateEditor(msg tea.Msg) (App, tea.Cmd) {
	e := &a.editor

	switch msg := msg.(type) {
	case editorSavedMsg:
		// Return to detail view with the updated profile
		isUp, _ := wg.IsUp(msg.profile.Name)
		a.detail = newDetailModel(msg.profile, isUp)
		a.currentView = viewDetail
		a.message = fmt.Sprintf("Saved profile %q", msg.profile.Name)
		if isUp {
			a.message += " (restart interface for changes to take effect)"
		}
		return a, nil

	case tea.KeyMsg:
		key := msg.String()

		// Ctrl+S: save from anywhere
		if key == "ctrl+s" {
			return a.editorSave()
		}

		if e.editingPeer {
			return a.editorUpdatePeerEdit(msg)
		}

		return a.editorUpdateMain(msg)
	}

	return a, nil
}

// editorUpdateMain handles key events when not editing a peer.
func (a App) editorUpdateMain(msg tea.KeyMsg) (App, tea.Cmd) {
	e := &a.editor
	key := msg.String()

	switch key {
	case "esc":
		// Cancel: go back to detail view without saving
		a.currentView = viewDetail
		return a, nil

	case "tab", "down":
		e.inputs[e.focusIndex].Blur()
		e.focusIndex++
		if e.focusIndex >= editorFieldCount {
			e.focusIndex = 0
		}
		e.inputs[e.focusIndex].Focus()
		return a, nil

	case "shift+tab", "up":
		e.inputs[e.focusIndex].Blur()
		e.focusIndex--
		if e.focusIndex < 0 {
			e.focusIndex = editorFieldCount - 1
		}
		e.inputs[e.focusIndex].Focus()
		return a, nil

	case "enter":
		// If on the last interface field, save
		if e.focusIndex == editorFieldCount-1 {
			return a.editorSave()
		}
		// Otherwise, move to the next field
		e.inputs[e.focusIndex].Blur()
		e.focusIndex++
		e.inputs[e.focusIndex].Focus()
		return a, nil

	case "a":
		// Add a new peer
		newPeer := wg.Peer{
			AllowedIPs: "0.0.0.0/0, ::/0",
		}
		e.peers = append(e.peers, newPeer)
		e.peerIdx = len(e.peers) - 1
		e.editingPeer = true
		e.peerFocus = 0
		e.inputs[e.focusIndex].Blur()
		e.peerInputs = makeEditorPeerInputs()
		e.peerInputs = populatePeerInputs(e.peerInputs, newPeer)
		e.peerInputs[0].Focus()
		return a, nil

	case "d":
		// Delete the selected (or last) peer
		if len(e.peers) > 0 {
			idx := e.peerIdx
			if idx < 0 || idx >= len(e.peers) {
				idx = len(e.peers) - 1
			}
			e.peers = append(e.peers[:idx], e.peers[idx+1:]...)
			e.peerIdx = -1
		}
		return a, nil

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		// Select a peer by number key
		idx, _ := strconv.Atoi(key)
		idx-- // 1-indexed to 0-indexed
		if idx < len(e.peers) {
			e.peerIdx = idx
			e.editingPeer = true
			e.peerFocus = 0
			e.inputs[e.focusIndex].Blur()
			e.peerInputs = makeEditorPeerInputs()
			e.peerInputs = populatePeerInputs(e.peerInputs, e.peers[idx])
			e.peerInputs[0].Focus()
		}
		return a, nil
	}

	// Delegate to the focused text input
	var cmd tea.Cmd
	e.inputs[e.focusIndex], cmd = e.inputs[e.focusIndex].Update(msg)
	return a, cmd
}

// editorUpdatePeerEdit handles key events when editing a peer.
func (a App) editorUpdatePeerEdit(msg tea.KeyMsg) (App, tea.Cmd) {
	e := &a.editor
	key := msg.String()

	switch key {
	case "esc":
		// Exit peer edit mode back to interface fields
		e.peerInputs[e.peerFocus].Blur()
		e.editingPeer = false
		e.inputs[e.focusIndex].Focus()
		return a, nil

	case "tab", "down":
		e.peerInputs[e.peerFocus].Blur()
		e.peerFocus++
		if e.peerFocus >= peerSubStepCount {
			e.peerFocus = 0
		}
		e.peerInputs[e.peerFocus].Focus()
		return a, nil

	case "shift+tab", "up":
		e.peerInputs[e.peerFocus].Blur()
		e.peerFocus--
		if e.peerFocus < 0 {
			e.peerFocus = peerSubStepCount - 1
		}
		e.peerInputs[e.peerFocus].Focus()
		return a, nil

	case "enter":
		if e.peerFocus == peerSubStepCount-1 {
			// On the last peer field: save the peer and exit peer edit mode
			return a.editorSavePeer()
		}
		// Move to the next peer field
		e.peerInputs[e.peerFocus].Blur()
		e.peerFocus++
		e.peerInputs[e.peerFocus].Focus()
		return a, nil
	}

	// Delegate to the focused peer text input
	var cmd tea.Cmd
	e.peerInputs[e.peerFocus], cmd = e.peerInputs[e.peerFocus].Update(msg)
	return a, cmd
}

// editorSavePeer saves the current peer inputs back into the peers slice.
func (a App) editorSavePeer() (App, tea.Cmd) {
	e := &a.editor

	peer := buildPeerFromInputs(e.peerInputs)

	// Validate public key is required
	if peer.PublicKey == "" {
		e.err = fmt.Errorf("public key is required")
		return a, nil
	}

	e.err = nil

	if e.peerIdx >= 0 && e.peerIdx < len(e.peers) {
		e.peers[e.peerIdx] = peer
	}

	e.peerInputs[e.peerFocus].Blur()
	e.editingPeer = false
	e.inputs[e.focusIndex].Focus()

	return a, nil
}

// editorSave builds the updated interface, validates, and saves.
func (a App) editorSave() (App, tea.Cmd) {
	e := &a.editor

	// Build updated interface from inputs
	address := strings.TrimSpace(e.inputs[editorFieldAddress].Value())
	portStr := strings.TrimSpace(e.inputs[editorFieldListenPort].Value())
	dns := strings.TrimSpace(e.inputs[editorFieldDNS].Value())
	mtuStr := strings.TrimSpace(e.inputs[editorFieldMTU].Value())

	// Validate
	if address == "" {
		e.err = fmt.Errorf("address is required")
		return a, nil
	}

	port := 0
	if portStr != "" {
		var err error
		port, err = strconv.Atoi(portStr)
		if err != nil {
			e.err = fmt.Errorf("listen port must be a number")
			return a, nil
		}
	}

	mtu := 0
	if mtuStr != "" {
		var err error
		mtu, err = strconv.Atoi(mtuStr)
		if err != nil {
			e.err = fmt.Errorf("MTU must be a number")
			return a, nil
		}
	}

	e.err = nil

	updated := &wg.Interface{
		Name:       e.profile.Name,
		PrivateKey: e.profile.PrivateKey,
		Address:    address,
		ListenPort: port,
		DNS:        dns,
		MTU:        mtu,
		Peers:      e.peers,
	}

	return a, func() tea.Msg {
		if err := wg.SaveConfig(configDir, updated); err != nil {
			return errMsg{err: err}
		}
		return editorSavedMsg{profile: updated}
	}
}

// view renders the editor UI.
func (e editorModel) view(width, height int) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Edit: " + e.profile.Name))
	b.WriteString("\n\n")

	// Read-only fields
	if e.profile.PrivateKey != "" {
		pubKey := truncateKey(e.profile.PrivateKey, 20)
		// Try to show the public key if we can derive it; otherwise show note
		b.WriteString("  " + labelStyle.Render("Private Key:") + descStyle.Render(pubKey+" (read-only)"))
		b.WriteString("\n")
	}
	b.WriteString("  " + labelStyle.Render("Name:") + descStyle.Render(e.profile.Name+" (read-only)"))
	b.WriteString("\n\n")

	if e.editingPeer {
		// Peer editing view
		b.WriteString(e.viewPeerEdit())
	} else {
		// Interface fields
		b.WriteString(e.viewInterfaceFields())

		// Peers list
		b.WriteString(e.viewPeersList())

		// Help bar
		b.WriteString("\n")
		help := helpKey("tab", "next field") + "  " +
			helpKey("ctrl+s", "save") + "  " +
			helpKey("esc", "cancel")
		b.WriteString(help)
	}

	// Error display
	if e.err != nil {
		b.WriteString("\n\n")
		b.WriteString("  " + errorStyle.Render(e.err.Error()))
	}

	return b.String()
}

// editorFieldLabels maps field indices to labels.
var editorFieldLabels = [editorFieldCount]string{
	"Address:",
	"Listen Port:",
	"DNS:",
	"MTU:",
}

// viewInterfaceFields renders the interface text input fields.
func (e editorModel) viewInterfaceFields() string {
	var b strings.Builder

	for i := 0; i < editorFieldCount; i++ {
		label := labelStyle.Render(editorFieldLabels[i])
		cursor := "  "
		if i == e.focusIndex {
			cursor = "> "
		}
		b.WriteString(cursor + label + e.inputs[i].View())
		b.WriteString("\n")
	}

	return b.String()
}

// viewPeersList renders the list of peers below the interface fields.
func (e editorModel) viewPeersList() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  " + labelStyle.Render("Peers:"))
	b.WriteString("\n")

	if len(e.peers) == 0 {
		b.WriteString("  " + descStyle.Render("No peers configured."))
		b.WriteString("\n")
	} else {
		for i, p := range e.peers {
			key := truncateKey(p.PublicKey, 12)
			line := fmt.Sprintf("  %d. %s  %s", i+1, key, p.AllowedIPs)
			if p.Endpoint != "" {
				line += "  " + p.Endpoint
			}
			if i == e.peerIdx {
				line = valueStyle.Render(line)
			} else {
				line = descStyle.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString("  " + helpKey("a", "add peer") + "  " +
		helpKey("d", "delete peer") + "  " +
		helpKey("1-9", "edit peer"))
	b.WriteString("\n")

	return b.String()
}

// viewPeerEdit renders the peer editing sub-form.
func (e editorModel) viewPeerEdit() string {
	var b strings.Builder

	peerNum := e.peerIdx + 1
	b.WriteString("  " + descStyle.Render(fmt.Sprintf("Editing Peer %d", peerNum)))
	b.WriteString("\n\n")

	for i := 0; i < peerSubStepCount; i++ {
		label := labelStyle.Render(peerStepLabels[i] + ":")
		cursor := "  "
		if i == e.peerFocus {
			cursor = "> "
		}
		b.WriteString(cursor + label + e.peerInputs[i].View())
		b.WriteString("\n")
	}

	b.WriteString("\n")
	help := helpKey("tab", "next field") + "  " +
		helpKey("enter", "save peer") + "  " +
		helpKey("ctrl+s", "save all") + "  " +
		helpKey("esc", "cancel peer edit")
	b.WriteString(help)

	return b.String()
}
