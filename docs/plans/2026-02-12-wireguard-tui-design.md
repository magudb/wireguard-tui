# WireGuard TUI Design

## Overview

A terminal user interface for managing WireGuard VPN profiles. Full lifecycle management: create, read, update, delete configs, control interfaces (up/down), view live connection stats, and import/export profiles including QR codes.

Built in Go with Bubbletea (TUI) and Lipgloss (styling). No Cobra — pure Bubbletea application. Requires sudo to run.

## Project Structure

```
wireguard-tui/
├── main.go                  # Entry point, launches Bubbletea program
├── go.mod / go.sum
├── internal/
│   ├── tui/
│   │   ├── app.go           # Root model, view routing
│   │   ├── list.go          # Profile list view (main screen)
│   │   ├── detail.go        # Profile detail/actions view
│   │   ├── wizard.go        # New profile creation wizard
│   │   ├── editor.go        # Edit existing profile
│   │   ├── status.go        # Live interface stats view
│   │   ├── import.go        # Import profile view
│   │   ├── export.go        # Export / QR display view
│   │   ├── confirm.go       # Confirmation dialog component
│   │   └── styles.go        # Lipgloss styles
│   └── wg/
│       ├── config.go        # Parse/write WireGuard .conf files
│       ├── keys.go          # Key generation (wg genkey/pubkey/preshared)
│       ├── interface.go     # Interface up/down via wg-quick
│       ├── status.go        # Read interface status via wg show
│       └── qr.go            # QR code generation
└── docs/
    └── plans/
```

## Navigation & View Flow

```
Profile List (home) ──Enter──> Profile Detail ──> Editor / Status / Export
       │                              │
    [n]ew ──> Wizard              [t]oggle up/down
    [i]mport ──> Import           [d]elete (with confirm)
    [q]uit                        [esc] back to list
```

- Profile List is the home screen. Esc always returns here.
- Profile Detail shows config summary and action hotkeys.
- Wizard is a multi-step guided flow.
- Status view refreshes every 1-2s via tea.Tick.
- q quits from any view (with confirmation if interfaces are active).

## Data Model

```go
type Interface struct {
    Name       string
    Address    string
    ListenPort int
    PrivateKey string
    DNS        string
    MTU        int
    Peers      []Peer
}

type Peer struct {
    PublicKey           string
    PresharedKey        string
    AllowedIPs          string
    Endpoint            string
    PersistentKeepalive int
}
```

## WireGuard Backend (internal/wg)

### config.go
- Parse .conf files from /etc/wireguard/ into Go structs.
- Write structs back to .conf format.

### keys.go
- Call wg genkey, wg pubkey, wg genpsk via exec.Command.
- Return key pairs as strings.

### interface.go
- Up(name): wg-quick up <name>
- Down(name): wg-quick down <name>
- IsUp(name): check via wg show <name>

### status.go
- Parse wg show <name> for: latest handshake, transfer rx/tx, endpoint.
- Return structured status per peer.

### qr.go
- Use github.com/skip2/go-qrcode to render config as terminal QR.
- Import: read from .conf file or pasted config text.

## Creation Wizard Flow

1. **Interface Name** — text input, default "wg0" (auto-increment if taken)
2. **Address** — text input, suggest "10.0.0.1/24" (scan existing for conflicts)
3. **Listen Port** — text input, default 51820 (scan for available)
4. **DNS** — text input, default "1.1.1.1, 8.8.8.8"
5. **Add Peers** (repeatable):
   - Public key (paste or generate keypair)
   - Allowed IPs (default "0.0.0.0/0, ::/0")
   - Endpoint (optional)
   - Preshared key (optional, auto-generate toggle)
   - Persistent keepalive (default 25)
   - "Add another peer?" or "Done"
6. **Review & Confirm** — full config preview, confirm/back/abort

Auto-generates private key for interface, shows derived public key.

## Dependencies

- github.com/charmbracelet/bubbletea — TUI framework
- github.com/charmbracelet/lipgloss — Styling
- github.com/charmbracelet/bubbles — Pre-built components (text input, list, spinner, table)
- github.com/skip2/go-qrcode — QR code generation
- Standard library for exec, templates, file I/O

## Error Handling

- Check for wg and wg-quick binaries on startup. Clear error if missing.
- Check for root/sudo on startup. Refuse to run without it.
- exec.Command calls wrapped with 5s timeout context.
- Config parse errors show the problematic line with context.
- Failed interface operations show stderr from wg-quick.

## Styling

- Lipgloss for consistent colors and borders.
- UP interfaces in green, DOWN in dim/gray.
- Errors in red, success in green.
- Keyboard shortcuts highlighted in accent color.
