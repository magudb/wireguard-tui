# Agent Guide

## Build and Test

```bash
# Build
go build -o wireguard-tui .

# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/wg/

# If Go proxy is unreachable, use direct fetching
GOPROXY=direct go test ./...
```

The binary must be run as root (`sudo ./wireguard-tui`). Tests for `internal/wg` use mocks and temp directories, so they run without root.

## Architecture

Pure Bubbletea application (no Cobra). Single entry point in `main.go` that checks for root and required binaries (`wg`, `wg-quick`), then launches a fullscreen `tea.Program`.

### Backend (`internal/wg/`)

Thin wrappers around WireGuard CLI tools:

- **config.go** — Parse and serialize `/etc/wireguard/*.conf` files. INI-like format with `[Interface]` and `[Peer]` sections. The `Interface` struct is the core data model shared across all views.
- **keys.go** — Key generation via `wg genkey`, `wg pubkey`, `wg genpsk`. All commands have a 5-second timeout.
- **interface.go** — Interface control via `wg-quick up/down`. `Toggle()` checks current state with `wg show` and flips it.
- **status.go** — Parses `wg show <name>` output into `InterfaceStatus`/`PeerStatus` structs with transfer bytes, handshake times, keepalive intervals.
- **qr.go** — QR code generation from config text using `go-qrcode`.

### TUI (`internal/tui/`)

Model-View-Update with view routing via `viewType` enum in `app.go`. The `App` struct holds all sub-models and delegates `Update`/`View` calls to the active view.

Navigation: views return `navigateMsg` or set `a.currentView` directly. Most views return to the previous view on `esc`.

Shared patterns:
- `errMsg` — Set `a.err`, auto-cleared after 3 seconds via `clearMessages()`
- `refreshMsg` — Triggers `loadProfiles()` to reload config directory
- `toggledMsg` — Interface toggled, updates status in list and detail views

### Config directory

All profiles read from and written to `/etc/wireguard/`. The `configDir` constant is in `app.go`.

## Code Style

- No Cobra, no external CLI framework — just Bubbletea
- Each view is a separate file with its own model struct and `update`/`view` methods
- Update methods are on `App` (not on the sub-model) so they can modify navigation and cross-view state
- Backend functions return errors, never panic
- Tests use `t.TempDir()` for file operations
- Interface names validated: alphanumeric, hyphens, underscores, max 15 chars

## Key Files

| File | What to Know |
|------|--------------|
| `app.go` | View routing and shared message types — start here |
| `styles.go` | All colors and styles — change appearance here |
| `config.go` | Data model (`Interface`, `Peer`) — the core types |
| `list.go` | Entry point view, `loadProfiles()` function |
| `wizard.go` | Largest file (~756 lines), 6-step creation flow with peer sub-wizard |
| `editor.go` | ~512 lines, text input focus management for action keys vs typing |
