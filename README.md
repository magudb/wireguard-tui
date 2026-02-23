# wireguard-tui

A terminal UI for managing WireGuard VPN profiles. Create, edit, toggle, import, export, and monitor connections — all from a single interface.

Requires `sudo` because WireGuard configuration lives in `/etc/wireguard/` and interface control needs root.

## Features

- **Profile list** with up/down status, peer counts, and quick toggle
- **Creation wizard** with auto key generation and smart defaults
- **Profile editor** with inline field editing and peer management
- **Live status view** with auto-refreshing transfer stats, handshake times, and keepalive
- **Import** from `.conf` files with preview before saving
- **Export** as config text or QR code, with save-to-file
- **Delete** with confirmation dialog
- **Amplifi Teleport** — create and refresh VPN profiles for Ubiquiti Amplifi routers via WebRTC signaling

## Requirements

- Go 1.25+
- `wg` and `wg-quick` (wireguard-tools)
- Linux (uses `/etc/wireguard/` and `ip` commands)

## Build

```bash
go build -o wireguard-tui .
```

## Run

```bash
sudo ./wireguard-tui
```

## Install

Copy the binary somewhere on your PATH:

```bash
sudo cp wireguard-tui /usr/local/bin/
```

## Keybindings

### List view

| Key       | Action                  |
|-----------|-------------------------|
| `j` / `k` | Move cursor up/down    |
| `enter`   | Open profile detail     |
| `n`       | New profile (wizard)    |
| `a`       | Amplifi Teleport setup  |
| `t`       | Toggle selected profile |
| `i`       | Import profile          |
| `q`       | Quit                    |

### Detail view

| Key   | Action          |
|-------|-----------------|
| `e`   | Edit profile              |
| `s`   | Live status               |
| `t`   | Toggle up/down            |
| `r`   | Reconnect Teleport        |
| `x`   | Export profile            |
| `d`   | Delete profile            |
| `esc` | Back to list              |

`r` only appears for profiles with a saved Teleport token.

## Amplifi Teleport

Native support for [Ubiquiti Amplifi](https://amplifi.com/) Teleport VPN. Create WireGuard profiles that connect through your Amplifi router without manually configuring anything.

### Initial setup

1. Open the AmpliFi app on your phone
2. Go to **Teleport** → **Add Device** to get a PIN
3. In wireguard-tui, press `a` from the list view
4. Enter a profile name and the PIN, then press Enter
5. The TUI connects via WebRTC signaling, negotiates an ICE endpoint, and generates a WireGuard config automatically

### Reconnecting

Teleport endpoints change when your network conditions change. To refresh:

1. Select the Teleport profile and press Enter to open it
2. Press `r` to reconnect — a new endpoint is negotiated using the saved device token (no PIN needed)

Tokens are stored in `/etc/wireguard/.teleport/`.

## Waybar Integration

A status script is included for showing WireGuard connection state in [Waybar](https://github.com/Alexays/Waybar). It uses `ip link show type wireguard` (no root needed) to detect active tunnels.

### Setup

1. Copy the status script:

```bash
cp examples/waybar-wireguard ~/.local/bin/waybar-wireguard
chmod +x ~/.local/bin/waybar-wireguard
```

2. Add the module to your `~/.config/waybar/config.jsonc`:

```jsonc
// Add to modules-right (or wherever you prefer)
"modules-right": [
    "custom/wireguard",
    "network",
    // ...
],

// Module definition
"custom/wireguard": {
    "exec": "waybar-wireguard",
    "return-type": "json",
    "interval": 5,
    "on-click": "wireguard-tui-launch",
    "tooltip": true
}
```

3. Add styles to your `~/.config/waybar/style.css`:

```css
#custom-wireguard {
    background-color: @background;
    color: @foreground;
    padding: 0 10px;
    margin: 5px 0;
}

#custom-wireguard.connected {
    color: #8bc34a;
}

#custom-wireguard.disconnected {
    color: #888;
}
```

4. Restart Waybar to apply changes.

### Launcher script (optional)

If you want the on-click to open the TUI as a floating terminal window, create `~/.local/bin/wireguard-tui-launch`:

```bash
#!/bin/bash
# Launch wireguard-tui as a floating dialog with sudo for privilege escalation
exec setsid uwsm-app -- xdg-terminal-exec --app-id=org.omarchy.wireguard-tui -e sudo wireguard-tui
```

This works with Hyprland window rules to float and center the terminal. Add to your Hyprland config:

```
windowrule = tag +floating-window, match:class org.omarchy.wireguard-tui
```

## Project Structure

```
.
├── main.go                     Entry point (root check, binary check, tea.Program)
├── internal/
│   ├── wg/                     WireGuard backend
│   │   ├── config.go           Config parsing and serialization
│   │   ├── keys.go             Key generation (wg genkey/pubkey/genpsk)
│   │   ├── interface.go        Interface control (up/down/toggle/status)
│   │   ├── status.go           Status parsing (wg show output)
│   │   ├── qr.go               QR code generation
│   │   └── *_test.go           Tests for each module
│   ├── teleport/               Amplifi Teleport backend
│   │   ├── credentials.go      Token and UUID persistence
│   │   ├── client.go           Amplifi API client
│   │   ├── sdp.go              SDP attribute injection/parsing
│   │   ├── connect.go          WebRTC orchestration and config generation
│   │   └── *_test.go           Tests for each module
│   └── tui/                    Terminal UI (Bubbletea)
│       ├── app.go              Root model, view routing, shared messages
│       ├── styles.go           Lipgloss styles and colors
│       ├── list.go             Profile list view
│       ├── detail.go           Profile detail view
│       ├── wizard.go           Creation wizard (6-step)
│       ├── editor.go           Profile editor with peer management
│       ├── status.go           Live status with auto-refresh
│       ├── importview.go       Import from .conf file
│       ├── export.go           Export as text/QR with save
│       ├── confirm.go          Confirmation dialog
│       └── teleportview.go     Amplifi Teleport setup/reconnect
└── examples/
    └── waybar-wireguard        Waybar status script
```

## Dependencies

- [bubbletea](https://github.com/charmbracelet/bubbletea) — Terminal UI framework
- [bubbles](https://github.com/charmbracelet/bubbles) — Text input components
- [lipgloss](https://github.com/charmbracelet/lipgloss) — Terminal styling
- [go-qrcode](https://github.com/skip2/go-qrcode) — QR code generation
- [pion/webrtc](https://github.com/pion/webrtc) — WebRTC for Amplifi Teleport
