package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	wg "github.com/mlu/wireguard-tui/internal/wg"
)

// wizardStepCount is the total number of main wizard steps.
const wizardStepCount = 6

// Peer sub-step indices within the peer sub-wizard.
const (
	peerStepPubKey     = 0
	peerStepAllowedIPs = 1
	peerStepEndpoint   = 2
	peerStepPSK        = 3
	peerStepKeepalive  = 4
	peerSubStepCount   = 5
)

// wizardModel holds the state for the multi-step profile creation wizard.
type wizardModel struct {
	step   int               // 0-5 main steps
	inputs []textinput.Model // one per main step (0-3)

	// Key generation for the interface
	privateKey string
	publicKey  string

	// Peer collection
	peers      []wg.Peer
	peerStep   int               // 0-4 within peer sub-wizard
	peerInputs []textinput.Model // 5 inputs for current peer
	addingPeer bool
	askingMore bool // asking "add another peer?"

	// For generated peer keys display
	generatedPeerPrivKey string
	generatedPeerPubKey  string

	// For generated preshared key display
	generatedPSK string

	err error
}

// isValidInterfaceName checks that a name is safe for use as a Linux
// network interface name: non-empty, at most 15 characters, and only
// containing alphanumeric characters, hyphens, and underscores.
func isValidInterfaceName(name string) bool {
	if len(name) == 0 || len(name) > 15 {
		return false
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// suggestInterfaceName returns the next available wgN name by checking
// which .conf files already exist in configDir.
func suggestInterfaceName() string {
	for i := 0; ; i++ {
		name := fmt.Sprintf("wg%d", i)
		path := filepath.Join(configDir, name+".conf")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return name
		}
	}
}

// newWizardModel creates a new wizard model with generated keys and
// initialized text inputs for all steps.
func newWizardModel() wizardModel {
	w := wizardModel{}

	// Generate interface keypair
	privKey, pubKey, err := wg.GenerateKeyPair()
	if err != nil {
		w.err = fmt.Errorf("generating keypair: %w", err)
	} else {
		w.privateKey = privKey
		w.publicKey = pubKey
	}

	// Initialize main step inputs (steps 0-3)
	w.inputs = make([]textinput.Model, 4)

	// Step 0: Interface Name
	w.inputs[0] = textinput.New()
	w.inputs[0].Placeholder = "wg0"
	w.inputs[0].SetValue(suggestInterfaceName())
	w.inputs[0].Focus()
	w.inputs[0].CharLimit = 15

	// Step 1: Address
	w.inputs[1] = textinput.New()
	w.inputs[1].Placeholder = "10.0.0.1/24"
	w.inputs[1].SetValue("10.0.0.1/24")
	w.inputs[1].CharLimit = 43

	// Step 2: Listen Port
	w.inputs[2] = textinput.New()
	w.inputs[2].Placeholder = "51820"
	w.inputs[2].SetValue("51820")
	w.inputs[2].CharLimit = 5

	// Step 3: DNS
	w.inputs[3] = textinput.New()
	w.inputs[3].Placeholder = "1.1.1.1, 8.8.8.8"
	w.inputs[3].SetValue("1.1.1.1, 8.8.8.8")
	w.inputs[3].CharLimit = 100

	// Initialize peer sub-step inputs
	w.peerInputs = makePeerInputs()

	return w
}

// makePeerInputs creates a fresh set of 5 text inputs for the peer sub-wizard.
func makePeerInputs() []textinput.Model {
	inputs := make([]textinput.Model, peerSubStepCount)

	// Peer step 0: Public Key
	inputs[peerStepPubKey] = textinput.New()
	inputs[peerStepPubKey].Placeholder = "base64 public key (or press 'g' to generate)"
	inputs[peerStepPubKey].CharLimit = 44

	// Peer step 1: Allowed IPs
	inputs[peerStepAllowedIPs] = textinput.New()
	inputs[peerStepAllowedIPs].Placeholder = "0.0.0.0/0, ::/0"
	inputs[peerStepAllowedIPs].SetValue("0.0.0.0/0, ::/0")
	inputs[peerStepAllowedIPs].CharLimit = 200

	// Peer step 2: Endpoint
	inputs[peerStepEndpoint] = textinput.New()
	inputs[peerStepEndpoint].Placeholder = "host:port (optional, enter to skip)"
	inputs[peerStepEndpoint].CharLimit = 100

	// Peer step 3: Preshared Key
	inputs[peerStepPSK] = textinput.New()
	inputs[peerStepPSK].Placeholder = "base64 preshared key (optional, 'g' to generate)"
	inputs[peerStepPSK].CharLimit = 44

	// Peer step 4: Persistent Keepalive
	inputs[peerStepKeepalive] = textinput.New()
	inputs[peerStepKeepalive].Placeholder = "25"
	inputs[peerStepKeepalive].SetValue("25")
	inputs[peerStepKeepalive].CharLimit = 5

	return inputs
}

// configSavedMsg is sent after a config has been successfully saved.
type configSavedMsg struct{ name string }

func (a App) updateWizard(msg tea.Msg) (App, tea.Cmd) {
	w := &a.wizard

	switch msg := msg.(type) {
	case configSavedMsg:
		a.message = fmt.Sprintf("Created profile %q", msg.name)
		a.currentView = viewList
		return a, tea.Batch(loadProfiles(), clearMessageAfter(3*time.Second))

	case tea.KeyMsg:
		key := msg.String()

		// Global: ctrl+c handled by app, esc goes back
		if key == "esc" {
			return a.wizardHandleEsc()
		}

		// Review step (step 5) has its own key handling
		if w.step == 5 {
			return a.wizardHandleReview(key)
		}

		// Asking "add another peer?" prompt
		if w.askingMore {
			return a.wizardHandleAskMore(key)
		}

		// Peer sub-wizard (step 4, addingPeer == true)
		if w.step == 4 && w.addingPeer {
			return a.wizardHandlePeerStep(msg)
		}

		// Main steps 0-3: text inputs
		if w.step >= 0 && w.step <= 3 {
			return a.wizardHandleMainStep(msg)
		}

		// Step 4 peer summary (not adding, not asking)
		if w.step == 4 && !w.addingPeer {
			switch key {
			case "enter":
				// Start adding a new peer
				w.addingPeer = true
				w.peerStep = 0
				w.peerInputs = makePeerInputs()
				w.peerInputs[0].Focus()
				w.generatedPeerPrivKey = ""
				w.generatedPeerPubKey = ""
				w.generatedPSK = ""
			case "n":
				// Continue to review (only if peers exist)
				if len(w.peers) > 0 {
					w.step = 5
				}
			}
			return a, nil
		}
	}

	return a, nil
}

// wizardHandleEsc handles the escape key at any wizard step.
func (a App) wizardHandleEsc() (App, tea.Cmd) {
	w := &a.wizard

	if w.askingMore {
		// Cancel asking, go back to last peer sub-step
		w.askingMore = false
		w.addingPeer = true
		w.peerStep = peerStepKeepalive
		w.peerInputs[w.peerStep].Focus()
		return a, nil
	}

	if w.step == 5 {
		// Review back to peer step
		w.step = 4
		w.addingPeer = false
		return a, nil
	}

	if w.step == 4 && w.addingPeer {
		if w.peerStep > 0 {
			w.peerInputs[w.peerStep].Blur()
			w.peerStep--
			w.peerInputs[w.peerStep].Focus()
			// Clear generated key display when going back to key step
			if w.peerStep == peerStepPubKey {
				w.generatedPeerPrivKey = ""
				w.generatedPeerPubKey = ""
			}
			if w.peerStep == peerStepPSK {
				w.generatedPSK = ""
			}
			return a, nil
		}
		// At first peer sub-step, go back to step 3
		w.addingPeer = false
		w.peerInputs[0].Blur()
		// If we have peers already, just go back to non-adding state
		if len(w.peers) > 0 {
			w.step = 4
			return a, nil
		}
		w.step = 3
		w.inputs[3].Focus()
		return a, nil
	}

	if w.step == 4 && !w.addingPeer {
		w.step = 3
		w.inputs[3].Focus()
		return a, nil
	}

	if w.step > 0 {
		w.inputs[w.step].Blur()
		w.step--
		w.inputs[w.step].Focus()
		return a, nil
	}

	// At step 0, go back to list
	a.currentView = viewList
	return a, nil
}

// wizardHandleMainStep handles key events for main steps 0-3.
func (a App) wizardHandleMainStep(msg tea.KeyMsg) (App, tea.Cmd) {
	w := &a.wizard

	if msg.String() == "enter" {
		val := strings.TrimSpace(w.inputs[w.step].Value())
		// Validate required fields
		if w.step == 0 {
			if val == "" {
				w.err = fmt.Errorf("interface name is required")
				return a, nil
			}
			if !isValidInterfaceName(val) {
				w.err = fmt.Errorf("invalid interface name: use only a-z, A-Z, 0-9, hyphen, underscore (max 15 chars)")
				return a, nil
			}
		}
		if w.step == 1 && val == "" {
			w.err = fmt.Errorf("address is required")
			return a, nil
		}
		if w.step == 2 && val == "" {
			w.err = fmt.Errorf("listen port is required")
			return a, nil
		}
		if w.step == 2 {
			if _, err := strconv.Atoi(val); err != nil {
				w.err = fmt.Errorf("listen port must be a number")
				return a, nil
			}
		}

		w.err = nil
		w.inputs[w.step].Blur()
		w.step++

		if w.step <= 3 {
			w.inputs[w.step].Focus()
		} else {
			// Moving to step 4: start peer sub-wizard
			w.addingPeer = true
			w.peerStep = 0
			w.peerInputs = makePeerInputs()
			w.peerInputs[0].Focus()
			w.generatedPeerPrivKey = ""
			w.generatedPeerPubKey = ""
			w.generatedPSK = ""
		}
		return a, nil
	}

	// Delegate to the text input
	var cmd tea.Cmd
	w.inputs[w.step], cmd = w.inputs[w.step].Update(msg)
	return a, cmd
}

// wizardHandlePeerStep handles key events for the peer sub-wizard.
func (a App) wizardHandlePeerStep(msg tea.KeyMsg) (App, tea.Cmd) {
	w := &a.wizard
	key := msg.String()

	// Handle 'g' for key generation at public key step
	if w.peerStep == peerStepPubKey && key == "g" && w.peerInputs[peerStepPubKey].Value() == "" {
		privKey, pubKey, err := wg.GenerateKeyPair()
		if err != nil {
			w.err = fmt.Errorf("generating peer keypair: %w", err)
			return a, nil
		}
		w.generatedPeerPrivKey = privKey
		w.generatedPeerPubKey = pubKey
		w.peerInputs[peerStepPubKey].SetValue(pubKey)
		return a, nil
	}

	// Handle 'g' for preshared key generation
	if w.peerStep == peerStepPSK && key == "g" && w.peerInputs[peerStepPSK].Value() == "" {
		psk, err := wg.GeneratePresharedKey()
		if err != nil {
			w.err = fmt.Errorf("generating preshared key: %w", err)
			return a, nil
		}
		w.generatedPSK = psk
		w.peerInputs[peerStepPSK].SetValue(psk)
		return a, nil
	}

	if key == "enter" {
		val := strings.TrimSpace(w.peerInputs[w.peerStep].Value())

		// Validate required peer fields
		if w.peerStep == peerStepPubKey && val == "" {
			w.err = fmt.Errorf("public key is required (enter a key or press 'g' to generate)")
			return a, nil
		}

		// Validate keepalive is a number if provided
		if w.peerStep == peerStepKeepalive && val != "" {
			if _, err := strconv.Atoi(val); err != nil {
				w.err = fmt.Errorf("persistent keepalive must be a number")
				return a, nil
			}
		}

		w.err = nil
		w.peerInputs[w.peerStep].Blur()

		if w.peerStep < peerSubStepCount-1 {
			w.peerStep++
			w.peerInputs[w.peerStep].Focus()
			return a, nil
		}

		// Completed all peer sub-steps: build the peer
		keepalive := 0
		if ka := strings.TrimSpace(w.peerInputs[peerStepKeepalive].Value()); ka != "" {
			keepalive, _ = strconv.Atoi(ka)
		}

		peer := wg.Peer{
			PublicKey:           strings.TrimSpace(w.peerInputs[peerStepPubKey].Value()),
			AllowedIPs:          strings.TrimSpace(w.peerInputs[peerStepAllowedIPs].Value()),
			Endpoint:            strings.TrimSpace(w.peerInputs[peerStepEndpoint].Value()),
			PresharedKey:        strings.TrimSpace(w.peerInputs[peerStepPSK].Value()),
			PersistentKeepalive: keepalive,
		}
		w.peers = append(w.peers, peer)
		w.addingPeer = false
		w.askingMore = true
		w.generatedPeerPrivKey = ""
		w.generatedPeerPubKey = ""
		w.generatedPSK = ""
		return a, nil
	}

	// Delegate to the text input
	var cmd tea.Cmd
	w.peerInputs[w.peerStep], cmd = w.peerInputs[w.peerStep].Update(msg)
	return a, cmd
}

// wizardHandleAskMore handles the "add another peer?" prompt.
func (a App) wizardHandleAskMore(key string) (App, tea.Cmd) {
	w := &a.wizard

	switch key {
	case "y":
		w.askingMore = false
		w.addingPeer = true
		w.peerStep = 0
		w.peerInputs = makePeerInputs()
		w.peerInputs[0].Focus()
		w.generatedPeerPrivKey = ""
		w.generatedPeerPubKey = ""
		w.generatedPSK = ""
	case "n":
		w.askingMore = false
		w.addingPeer = false
		w.step = 5 // Advance to review
	}

	return a, nil
}

// wizardHandleReview handles keys at the review/confirm step.
func (a App) wizardHandleReview(key string) (App, tea.Cmd) {
	w := &a.wizard

	switch key {
	case "c":
		// Build and save the config
		iface := a.wizardBuildInterface()
		name := iface.Name
		return a, func() tea.Msg {
			if err := wg.SaveConfig(configDir, iface); err != nil {
				return errMsg{err: err}
			}
			return configSavedMsg{name: name}
		}

	case "b":
		// Back to peer step
		w.step = 4
		w.addingPeer = false
		return a, nil

	case "a":
		// Abort back to list
		a.currentView = viewList
		return a, nil
	}

	return a, nil
}

// wizardBuildInterface constructs a wg.Interface from the current wizard state.
func (a App) wizardBuildInterface() *wg.Interface {
	w := &a.wizard

	port, _ := strconv.Atoi(strings.TrimSpace(w.inputs[2].Value()))

	return &wg.Interface{
		Name:       strings.TrimSpace(w.inputs[0].Value()),
		Address:    strings.TrimSpace(w.inputs[1].Value()),
		ListenPort: port,
		PrivateKey: w.privateKey,
		DNS:        strings.TrimSpace(w.inputs[3].Value()),
		Peers:      w.peers,
	}
}

// view renders the wizard UI based on the current step.
func (w wizardModel) view(width, height int) string {
	var b strings.Builder

	if w.err != nil {
		// Show key-gen error prominently if we failed to generate keys
		if w.privateKey == "" && w.publicKey == "" {
			b.WriteString(titleStyle.Render("New Profile"))
			b.WriteString("\n\n")
			b.WriteString(errorStyle.Render("Error: " + w.err.Error()))
			b.WriteString("\n\n")
			b.WriteString(helpKey("esc", "back"))
			return b.String()
		}
	}

	switch {
	case w.step <= 3:
		return w.viewMainStep(width)
	case w.step == 4 && w.askingMore:
		return w.viewAskMore()
	case w.step == 4 && w.addingPeer:
		return w.viewPeerStep(width)
	case w.step == 4 && !w.addingPeer:
		return w.viewPeerSummary()
	case w.step == 5:
		return w.viewReview(width)
	}

	return ""
}

// stepLabels returns a human-readable label for each main step.
var stepLabels = [wizardStepCount]string{
	"Interface Name",
	"Address",
	"Listen Port",
	"DNS",
	"Peers",
	"Review & Confirm",
}

// viewMainStep renders steps 0-3 (text input steps).
func (w wizardModel) viewMainStep(width int) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("New Profile"))
	b.WriteString("\n\n")

	// Step indicator
	indicator := fmt.Sprintf("Step %d/%d: %s", w.step+1, wizardStepCount, stepLabels[w.step])
	b.WriteString(descStyle.Render(indicator))
	b.WriteString("\n\n")

	// Public key info
	if w.publicKey != "" {
		b.WriteString("  " + labelStyle.Render("Your public key:") + valueStyle.Render(w.publicKey))
		b.WriteString("\n")
		b.WriteString("  " + descStyle.Render("(share this with peers)"))
		b.WriteString("\n\n")
	}

	// Context help for current step
	switch w.step {
	case 0:
		b.WriteString("  " + descStyle.Render("Name for the WireGuard interface (e.g., wg0, wg1)"))
	case 1:
		b.WriteString("  " + descStyle.Render("VPN address with CIDR notation (e.g., 10.0.0.1/24)"))
	case 2:
		b.WriteString("  " + descStyle.Render("UDP port to listen on"))
	case 3:
		b.WriteString("  " + descStyle.Render("DNS servers, comma-separated"))
	}
	b.WriteString("\n\n")

	// Render the input
	b.WriteString("  " + w.inputs[w.step].View())
	b.WriteString("\n\n")

	// Error
	if w.err != nil {
		b.WriteString("  " + errorStyle.Render(w.err.Error()))
		b.WriteString("\n\n")
	}

	// Help
	help := helpKey("enter", "next") + "  " + helpKey("esc", "back")
	b.WriteString(help)

	return b.String()
}

// peerStepLabels returns labels for each peer sub-step.
var peerStepLabels = [peerSubStepCount]string{
	"Public Key",
	"Allowed IPs",
	"Endpoint",
	"Preshared Key",
	"Persistent Keepalive",
}

// viewPeerStep renders the peer sub-wizard.
func (w wizardModel) viewPeerStep(width int) string {
	var b strings.Builder

	peerNum := len(w.peers) + 1

	b.WriteString(titleStyle.Render("New Profile"))
	b.WriteString("\n\n")

	// Step indicator
	indicator := fmt.Sprintf("Step %d/%d: %s > Peer %d > %s",
		5, wizardStepCount, stepLabels[4], peerNum, peerStepLabels[w.peerStep])
	b.WriteString(descStyle.Render(indicator))
	b.WriteString("\n\n")

	// Context help for peer sub-step
	switch w.peerStep {
	case peerStepPubKey:
		b.WriteString("  " + descStyle.Render("Enter the peer's public key or press 'g' to generate a new keypair"))
		b.WriteString("\n")
		if w.generatedPeerPrivKey != "" {
			b.WriteString("\n")
			b.WriteString("  " + successStyle.Render("Generated keypair for peer:"))
			b.WriteString("\n")
			b.WriteString("  " + labelStyle.Render("Private key:") + valueStyle.Render(w.generatedPeerPrivKey))
			b.WriteString("\n")
			b.WriteString("  " + descStyle.Render("(give this private key to the peer)"))
			b.WriteString("\n")
			b.WriteString("  " + labelStyle.Render("Public key:") + valueStyle.Render(w.generatedPeerPubKey))
			b.WriteString("\n")
		}
	case peerStepAllowedIPs:
		b.WriteString("  " + descStyle.Render("IP ranges this peer is allowed to send traffic from"))
	case peerStepEndpoint:
		b.WriteString("  " + descStyle.Render("Remote endpoint address:port (optional, press enter to skip)"))
	case peerStepPSK:
		b.WriteString("  " + descStyle.Render("Optional preshared key for extra security (press 'g' to generate)"))
		if w.generatedPSK != "" {
			b.WriteString("\n\n")
			b.WriteString("  " + successStyle.Render("Generated preshared key:"))
			b.WriteString("\n")
			b.WriteString("  " + valueStyle.Render(w.generatedPSK))
			b.WriteString("\n")
			b.WriteString("  " + descStyle.Render("(share this same key with the peer)"))
		}
	case peerStepKeepalive:
		b.WriteString("  " + descStyle.Render("Seconds between keepalive packets (0 to disable)"))
	}
	b.WriteString("\n\n")

	// Render the input
	b.WriteString("  " + w.peerInputs[w.peerStep].View())
	b.WriteString("\n\n")

	// Error
	if w.err != nil {
		b.WriteString("  " + errorStyle.Render(w.err.Error()))
		b.WriteString("\n\n")
	}

	// Help
	helpItems := helpKey("enter", "next") + "  " + helpKey("esc", "back")
	if w.peerStep == peerStepPubKey || w.peerStep == peerStepPSK {
		helpItems = helpKey("g", "generate") + "  " + helpItems
	}
	b.WriteString(helpItems)

	return b.String()
}

// viewAskMore renders the "add another peer?" prompt.
func (w wizardModel) viewAskMore() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("New Profile"))
	b.WriteString("\n\n")

	indicator := fmt.Sprintf("Step %d/%d: %s", 5, wizardStepCount, stepLabels[4])
	b.WriteString(descStyle.Render(indicator))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("  %d peer(s) added.\n\n", len(w.peers)))
	b.WriteString("  Add another peer?")
	b.WriteString("\n\n")

	help := helpKey("y", "yes") + "  " + helpKey("n", "no") + "  " + helpKey("esc", "back")
	b.WriteString(help)

	return b.String()
}

// viewPeerSummary shows the current peers and allows adding more or continuing.
func (w wizardModel) viewPeerSummary() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("New Profile"))
	b.WriteString("\n\n")

	indicator := fmt.Sprintf("Step %d/%d: %s", 5, wizardStepCount, stepLabels[4])
	b.WriteString(descStyle.Render(indicator))
	b.WriteString("\n\n")

	if len(w.peers) == 0 {
		b.WriteString("  " + descStyle.Render("No peers added yet."))
		b.WriteString("\n\n")
	} else {
		b.WriteString(fmt.Sprintf("  %d peer(s) configured:\n\n", len(w.peers)))
		for i, p := range w.peers {
			b.WriteString(fmt.Sprintf("  Peer %d: %s\n", i+1, truncateKey(p.PublicKey, 20)))
		}
		b.WriteString("\n")
	}

	help := helpKey("enter", "add peer") + "  " + helpKey("esc", "back")
	if len(w.peers) > 0 {
		help = helpKey("enter", "add peer") + "  " + helpKey("n", "continue to review") + "  " + helpKey("esc", "back")
	}
	b.WriteString(help)

	return b.String()
}

// viewReview renders the review/confirm step with the full config preview.
func (w wizardModel) viewReview(width int) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("New Profile"))
	b.WriteString("\n\n")

	indicator := fmt.Sprintf("Step %d/%d: %s", 6, wizardStepCount, stepLabels[5])
	b.WriteString(descStyle.Render(indicator))
	b.WriteString("\n\n")

	// Build the interface to marshal for preview
	port, _ := strconv.Atoi(strings.TrimSpace(w.inputs[2].Value()))
	iface := &wg.Interface{
		Name:       strings.TrimSpace(w.inputs[0].Value()),
		Address:    strings.TrimSpace(w.inputs[1].Value()),
		ListenPort: port,
		PrivateKey: w.privateKey,
		DNS:        strings.TrimSpace(w.inputs[3].Value()),
		Peers:      w.peers,
	}

	// Public key info
	if w.publicKey != "" {
		b.WriteString("  " + labelStyle.Render("Public key:") + valueStyle.Render(w.publicKey))
		b.WriteString("\n")
		b.WriteString("  " + descStyle.Render("(share this with peers)"))
		b.WriteString("\n\n")
	}

	// Config preview in a box
	config := wg.MarshalConfig(iface)
	configBox := boxStyle.Render(config)
	b.WriteString(configBox)
	b.WriteString("\n\n")

	// File path
	b.WriteString("  " + labelStyle.Render("Will save to:") + valueStyle.Render(filepath.Join(configDir, iface.Name+".conf")))
	b.WriteString("\n\n")

	// Help
	help := helpKey("c", "confirm & save") + "  " +
		helpKey("b", "back to peers") + "  " +
		helpKey("a", "abort")
	b.WriteString(help)

	return b.String()
}
