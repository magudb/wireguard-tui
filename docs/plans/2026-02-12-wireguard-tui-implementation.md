# WireGuard TUI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a full-lifecycle WireGuard VPN profile manager as a terminal UI in Go.

**Architecture:** Pure Bubbletea app (no Cobra). `internal/wg` package encapsulates all WireGuard operations behind a clean Go API. `internal/tui` package contains Bubbletea models for each view. Root model in `app.go` manages navigation via a view stack. Requires sudo.

**Tech Stack:** Go, Bubbletea, Lipgloss, Bubbles (textinput, list, table, spinner), go-qrcode, standard library os/exec.

---

### Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `main.go` (minimal placeholder)
- Create: `internal/wg/config.go` (empty package)
- Create: `internal/tui/app.go` (empty package)

**Step 1: Initialize Go module**

Run: `go mod init github.com/mlu/wireguard-tui`

**Step 2: Install dependencies**

Run:
```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/skip2/go-qrcode@latest
```

**Step 3: Create minimal main.go**

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "wireguard-tui must be run as root (use sudo)")
		os.Exit(1)
	}
	fmt.Println("wireguard-tui placeholder")
}
```

**Step 4: Create package stubs**

`internal/wg/config.go`:
```go
package wg
```

`internal/tui/app.go`:
```go
package tui
```

**Step 5: Verify it builds**

Run: `go build -o wireguard-tui .`
Expected: Binary builds without errors.

**Step 6: Commit**

```bash
git add go.mod go.sum main.go internal/
git commit -m "feat: scaffold project with Go module and dependencies"
```

---

### Task 2: WireGuard Data Model & Config Parser

**Files:**
- Create: `internal/wg/config.go`
- Create: `internal/wg/config_test.go`

**Step 1: Write failing tests for config parsing**

`internal/wg/config_test.go`:
```go
package wg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleConf = `[Interface]
PrivateKey = yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk=
Address = 10.0.0.1/24
ListenPort = 51820
DNS = 1.1.1.1, 8.8.8.8
MTU = 1420

[Peer]
PublicKey = xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=
PresharedKey = AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=
AllowedIPs = 0.0.0.0/0, ::/0
Endpoint = 203.0.113.1:51820
PersistentKeepalive = 25

[Peer]
PublicKey = TrMvSoP4jYQlY6RIzBgbssQqY3vxI2piVFBs2LR9PQc=
AllowedIPs = 10.0.0.2/32
`

func TestParseConfig(t *testing.T) {
	iface, err := ParseConfig(strings.NewReader(sampleConf))
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	if iface.PrivateKey != "yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk=" {
		t.Errorf("PrivateKey = %q, want yAnz5TF+...", iface.PrivateKey)
	}
	if iface.Address != "10.0.0.1/24" {
		t.Errorf("Address = %q, want 10.0.0.1/24", iface.Address)
	}
	if iface.ListenPort != 51820 {
		t.Errorf("ListenPort = %d, want 51820", iface.ListenPort)
	}
	if iface.DNS != "1.1.1.1, 8.8.8.8" {
		t.Errorf("DNS = %q, want 1.1.1.1, 8.8.8.8", iface.DNS)
	}
	if iface.MTU != 1420 {
		t.Errorf("MTU = %d, want 1420", iface.MTU)
	}
	if len(iface.Peers) != 2 {
		t.Fatalf("len(Peers) = %d, want 2", len(iface.Peers))
	}

	peer := iface.Peers[0]
	if peer.PublicKey != "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=" {
		t.Errorf("Peer[0].PublicKey = %q", peer.PublicKey)
	}
	if peer.PresharedKey != "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=" {
		t.Errorf("Peer[0].PresharedKey = %q", peer.PresharedKey)
	}
	if peer.AllowedIPs != "0.0.0.0/0, ::/0" {
		t.Errorf("Peer[0].AllowedIPs = %q", peer.AllowedIPs)
	}
	if peer.Endpoint != "203.0.113.1:51820" {
		t.Errorf("Peer[0].Endpoint = %q", peer.Endpoint)
	}
	if peer.PersistentKeepalive != 25 {
		t.Errorf("Peer[0].PersistentKeepalive = %d", peer.PersistentKeepalive)
	}

	peer2 := iface.Peers[1]
	if peer2.PublicKey != "TrMvSoP4jYQlY6RIzBgbssQqY3vxI2piVFBs2LR9PQc=" {
		t.Errorf("Peer[1].PublicKey = %q", peer2.PublicKey)
	}
	if peer2.PresharedKey != "" {
		t.Errorf("Peer[1].PresharedKey = %q, want empty", peer2.PresharedKey)
	}
}

func TestMarshalConfig(t *testing.T) {
	iface := &Interface{
		PrivateKey: "privkey123",
		Address:    "10.0.0.1/24",
		ListenPort: 51820,
		DNS:        "1.1.1.1",
		Peers: []Peer{
			{
				PublicKey:           "pubkey456",
				AllowedIPs:          "0.0.0.0/0",
				Endpoint:            "1.2.3.4:51820",
				PersistentKeepalive: 25,
			},
		},
	}

	out := MarshalConfig(iface)

	if !strings.Contains(out, "PrivateKey = privkey123") {
		t.Errorf("missing PrivateKey in output:\n%s", out)
	}
	if !strings.Contains(out, "Address = 10.0.0.1/24") {
		t.Errorf("missing Address in output:\n%s", out)
	}
	if !strings.Contains(out, "[Peer]") {
		t.Errorf("missing [Peer] section in output:\n%s", out)
	}
	if !strings.Contains(out, "PublicKey = pubkey456") {
		t.Errorf("missing peer PublicKey in output:\n%s", out)
	}
}

func TestMarshalConfigOmitsEmptyOptionalFields(t *testing.T) {
	iface := &Interface{
		PrivateKey: "key",
		Address:    "10.0.0.1/24",
		Peers:      []Peer{{PublicKey: "pkey", AllowedIPs: "0.0.0.0/0"}},
	}

	out := MarshalConfig(iface)

	if strings.Contains(out, "ListenPort") {
		t.Errorf("should omit ListenPort when 0:\n%s", out)
	}
	if strings.Contains(out, "DNS") {
		t.Errorf("should omit DNS when empty:\n%s", out)
	}
	if strings.Contains(out, "MTU") {
		t.Errorf("should omit MTU when 0:\n%s", out)
	}
	if strings.Contains(out, "PresharedKey") {
		t.Errorf("should omit PresharedKey when empty:\n%s", out)
	}
	if strings.Contains(out, "Endpoint") {
		t.Errorf("should omit Endpoint when empty:\n%s", out)
	}
	if strings.Contains(out, "PersistentKeepalive") {
		t.Errorf("should omit PersistentKeepalive when 0:\n%s", out)
	}
}

func TestLoadConfigsFromDir(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "wg0.conf"), []byte(sampleConf), 0600)
	if err != nil {
		t.Fatal(err)
	}
	// Non-conf file should be ignored
	err = os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignore me"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	interfaces, err := LoadConfigsFromDir(dir)
	if err != nil {
		t.Fatalf("LoadConfigsFromDir failed: %v", err)
	}
	if len(interfaces) != 1 {
		t.Fatalf("len(interfaces) = %d, want 1", len(interfaces))
	}
	if interfaces[0].Name != "wg0" {
		t.Errorf("Name = %q, want wg0", interfaces[0].Name)
	}
}

func TestRoundTrip(t *testing.T) {
	iface, err := ParseConfig(strings.NewReader(sampleConf))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	iface.Name = "wg0"

	out := MarshalConfig(iface)
	iface2, err := ParseConfig(strings.NewReader(out))
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}

	if iface.Address != iface2.Address {
		t.Errorf("Address mismatch: %q vs %q", iface.Address, iface2.Address)
	}
	if len(iface.Peers) != len(iface2.Peers) {
		t.Errorf("Peer count mismatch: %d vs %d", len(iface.Peers), len(iface2.Peers))
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/wg/ -v`
Expected: FAIL — types and functions not defined.

**Step 3: Implement data model and parser**

`internal/wg/config.go`:
```go
package wg

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Interface represents a WireGuard interface configuration.
type Interface struct {
	Name       string
	Address    string
	ListenPort int
	PrivateKey string
	DNS        string
	MTU        int
	Peers      []Peer
}

// Peer represents a WireGuard peer configuration.
type Peer struct {
	PublicKey           string
	PresharedKey        string
	AllowedIPs          string
	Endpoint            string
	PersistentKeepalive int
}

// ParseConfig parses a WireGuard .conf file from a reader.
func ParseConfig(r io.Reader) (*Interface, error) {
	iface := &Interface{}
	var currentPeer *Peer
	inPeer := false

	scanner := bufio.NewScanner(r)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if line == "[Interface]" {
			inPeer = false
			continue
		}
		if line == "[Peer]" {
			if currentPeer != nil {
				iface.Peers = append(iface.Peers, *currentPeer)
			}
			currentPeer = &Peer{}
			inPeer = true
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("line %d: invalid format: %s", lineNum, line)
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		if inPeer {
			switch key {
			case "PublicKey":
				currentPeer.PublicKey = val
			case "PresharedKey":
				currentPeer.PresharedKey = val
			case "AllowedIPs":
				currentPeer.AllowedIPs = val
			case "Endpoint":
				currentPeer.Endpoint = val
			case "PersistentKeepalive":
				n, err := strconv.Atoi(val)
				if err != nil {
					return nil, fmt.Errorf("line %d: invalid PersistentKeepalive: %v", lineNum, err)
				}
				currentPeer.PersistentKeepalive = n
			}
		} else {
			switch key {
			case "PrivateKey":
				iface.PrivateKey = val
			case "Address":
				iface.Address = val
			case "ListenPort":
				n, err := strconv.Atoi(val)
				if err != nil {
					return nil, fmt.Errorf("line %d: invalid ListenPort: %v", lineNum, err)
				}
				iface.ListenPort = n
			case "DNS":
				iface.DNS = val
			case "MTU":
				n, err := strconv.Atoi(val)
				if err != nil {
					return nil, fmt.Errorf("line %d: invalid MTU: %v", lineNum, err)
				}
				iface.MTU = n
			}
		}
	}

	if currentPeer != nil {
		iface.Peers = append(iface.Peers, *currentPeer)
	}

	return iface, scanner.Err()
}

// MarshalConfig serializes an Interface to WireGuard .conf format.
func MarshalConfig(iface *Interface) string {
	var b strings.Builder

	b.WriteString("[Interface]\n")
	b.WriteString(fmt.Sprintf("PrivateKey = %s\n", iface.PrivateKey))
	b.WriteString(fmt.Sprintf("Address = %s\n", iface.Address))
	if iface.ListenPort != 0 {
		b.WriteString(fmt.Sprintf("ListenPort = %d\n", iface.ListenPort))
	}
	if iface.DNS != "" {
		b.WriteString(fmt.Sprintf("DNS = %s\n", iface.DNS))
	}
	if iface.MTU != 0 {
		b.WriteString(fmt.Sprintf("MTU = %d\n", iface.MTU))
	}

	for _, peer := range iface.Peers {
		b.WriteString("\n[Peer]\n")
		b.WriteString(fmt.Sprintf("PublicKey = %s\n", peer.PublicKey))
		if peer.PresharedKey != "" {
			b.WriteString(fmt.Sprintf("PresharedKey = %s\n", peer.PresharedKey))
		}
		b.WriteString(fmt.Sprintf("AllowedIPs = %s\n", peer.AllowedIPs))
		if peer.Endpoint != "" {
			b.WriteString(fmt.Sprintf("Endpoint = %s\n", peer.Endpoint))
		}
		if peer.PersistentKeepalive != 0 {
			b.WriteString(fmt.Sprintf("PersistentKeepalive = %d\n", peer.PersistentKeepalive))
		}
	}

	return b.String()
}

// LoadConfigsFromDir reads all .conf files from a directory.
func LoadConfigsFromDir(dir string) ([]*Interface, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading config dir: %w", err)
	}

	var interfaces []*Interface
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".conf") {
			continue
		}

		f, err := os.Open(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("opening %s: %w", entry.Name(), err)
		}

		iface, err := ParseConfig(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}

		iface.Name = strings.TrimSuffix(entry.Name(), ".conf")
		interfaces = append(interfaces, iface)
	}

	return interfaces, nil
}

// SaveConfig writes an interface config to a file in the given directory.
func SaveConfig(dir string, iface *Interface) error {
	path := filepath.Join(dir, iface.Name+".conf")
	return os.WriteFile(path, []byte(MarshalConfig(iface)), 0600)
}

// DeleteConfig removes a config file from the given directory.
func DeleteConfig(dir string, name string) error {
	path := filepath.Join(dir, name+".conf")
	return os.Remove(path)
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/wg/ -v`
Expected: All PASS.

**Step 5: Commit**

```bash
git add internal/wg/config.go internal/wg/config_test.go
git commit -m "feat: add WireGuard config parser with round-trip support"
```

---

### Task 3: Key Generation

**Files:**
- Create: `internal/wg/keys.go`
- Create: `internal/wg/keys_test.go`

**Step 1: Write failing tests**

`internal/wg/keys_test.go`:
```go
package wg

import (
	"encoding/base64"
	"testing"
)

func TestGeneratePrivateKey(t *testing.T) {
	key, err := GeneratePrivateKey()
	if err != nil {
		t.Fatalf("GeneratePrivateKey: %v", err)
	}
	// WireGuard keys are 32 bytes, base64 encoded = 44 chars
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		t.Fatalf("invalid base64: %v", err)
	}
	if len(decoded) != 32 {
		t.Errorf("key length = %d bytes, want 32", len(decoded))
	}
}

func TestDerivePublicKey(t *testing.T) {
	privKey, err := GeneratePrivateKey()
	if err != nil {
		t.Fatalf("GeneratePrivateKey: %v", err)
	}

	pubKey, err := DerivePublicKey(privKey)
	if err != nil {
		t.Fatalf("DerivePublicKey: %v", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(pubKey)
	if err != nil {
		t.Fatalf("invalid base64: %v", err)
	}
	if len(decoded) != 32 {
		t.Errorf("pubkey length = %d bytes, want 32", len(decoded))
	}

	// Deriving again from same private key should give same result
	pubKey2, err := DerivePublicKey(privKey)
	if err != nil {
		t.Fatalf("DerivePublicKey second call: %v", err)
	}
	if pubKey != pubKey2 {
		t.Error("DerivePublicKey not deterministic")
	}
}

func TestGeneratePresharedKey(t *testing.T) {
	psk, err := GeneratePresharedKey()
	if err != nil {
		t.Fatalf("GeneratePresharedKey: %v", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(psk)
	if err != nil {
		t.Fatalf("invalid base64: %v", err)
	}
	if len(decoded) != 32 {
		t.Errorf("psk length = %d bytes, want 32", len(decoded))
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/wg/ -run TestGenerate -v`
Expected: FAIL — functions not defined.

**Step 3: Implement key generation**

`internal/wg/keys.go`:
```go
package wg

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const cmdTimeout = 5 * time.Second

func runWgCmd(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wg", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("wg %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func runWgCmdWithInput(input string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wg", args...)
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("wg %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// GeneratePrivateKey generates a new WireGuard private key.
func GeneratePrivateKey() (string, error) {
	return runWgCmd("genkey")
}

// DerivePublicKey derives a public key from a private key.
func DerivePublicKey(privateKey string) (string, error) {
	return runWgCmdWithInput(privateKey, "pubkey")
}

// GeneratePresharedKey generates a new preshared key.
func GeneratePresharedKey() (string, error) {
	return runWgCmd("genpsk")
}

// GenerateKeyPair generates a private key and derives its public key.
func GenerateKeyPair() (privateKey, publicKey string, err error) {
	privateKey, err = GeneratePrivateKey()
	if err != nil {
		return "", "", err
	}
	publicKey, err = DerivePublicKey(privateKey)
	if err != nil {
		return "", "", err
	}
	return privateKey, publicKey, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/wg/ -run "TestGenerate|TestDerive" -v`
Expected: All PASS (requires `wg` binary installed).

**Step 5: Commit**

```bash
git add internal/wg/keys.go internal/wg/keys_test.go
git commit -m "feat: add WireGuard key generation (genkey, pubkey, genpsk)"
```

---

### Task 4: Interface Control

**Files:**
- Create: `internal/wg/interface.go`
- Create: `internal/wg/interface_test.go`

**Step 1: Write failing tests**

`internal/wg/interface_test.go`:
```go
package wg

import (
	"testing"
)

func TestListInterfacesDoesNotError(t *testing.T) {
	// Should not error even if no interfaces are up
	_, err := ListInterfaces()
	if err != nil {
		t.Fatalf("ListInterfaces: %v", err)
	}
}

func TestIsUpReturnsFalseForNonexistent(t *testing.T) {
	up, err := IsUp("wg_nonexistent_test_12345")
	if err != nil {
		t.Fatalf("IsUp: %v", err)
	}
	if up {
		t.Error("IsUp returned true for nonexistent interface")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/wg/ -run "TestListInterfaces|TestIsUp" -v`
Expected: FAIL — functions not defined.

**Step 3: Implement interface control**

`internal/wg/interface.go`:
```go
package wg

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Up brings a WireGuard interface up via wg-quick.
func Up(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wg-quick", "up", name)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("wg-quick up %s: %w\n%s", name, err, string(out))
	}
	return nil
}

// Down brings a WireGuard interface down via wg-quick.
func Down(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wg-quick", "down", name)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("wg-quick down %s: %w\n%s", name, err, string(out))
	}
	return nil
}

// IsUp checks whether a WireGuard interface is currently active.
func IsUp(name string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wg", "show", name)
	err := cmd.Run()
	if err != nil {
		// Exit code 1 means interface doesn't exist / is down
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("wg show %s: %w", name, err)
	}
	return true, nil
}

// Toggle brings an interface up if down, or down if up.
func Toggle(name string) (nowUp bool, err error) {
	up, err := IsUp(name)
	if err != nil {
		return false, err
	}
	if up {
		return false, Down(name)
	}
	return true, Up(name)
}

// ListInterfaces returns the names of all active WireGuard interfaces.
func ListInterfaces() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wg", "show", "interfaces")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("wg show interfaces: %w", err)
	}

	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}
	return strings.Fields(raw), nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/wg/ -run "TestListInterfaces|TestIsUp" -v`
Expected: All PASS.

**Step 5: Commit**

```bash
git add internal/wg/interface.go internal/wg/interface_test.go
git commit -m "feat: add interface control (up, down, toggle, list)"
```

---

### Task 5: Status Parsing

**Files:**
- Create: `internal/wg/status.go`
- Create: `internal/wg/status_test.go`

**Step 1: Write failing tests**

`internal/wg/status_test.go`:
```go
package wg

import (
	"testing"
	"time"
)

const sampleWgShow = `interface: wg0
  public key: xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=
  private key: (hidden)
  listening port: 51820

peer: TrMvSoP4jYQlY6RIzBgbssQqY3vxI2piVFBs2LR9PQc=
  endpoint: 203.0.113.1:51820
  allowed ips: 10.0.0.2/32
  latest handshake: 1 minute, 30 seconds ago
  transfer: 1.50 MiB received, 3.24 MiB sent
  persistent keepalive: every 25 seconds

peer: abc123publickey=
  endpoint: 198.51.100.1:51820
  allowed ips: 10.0.0.3/32
  latest handshake: 45 seconds ago
  transfer: 500.00 KiB received, 120.00 KiB sent
`

func TestParseWgShow(t *testing.T) {
	status, err := parseWgShow(sampleWgShow)
	if err != nil {
		t.Fatalf("parseWgShow: %v", err)
	}

	if status.PublicKey != "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=" {
		t.Errorf("PublicKey = %q", status.PublicKey)
	}
	if status.ListenPort != 51820 {
		t.Errorf("ListenPort = %d", status.ListenPort)
	}
	if len(status.Peers) != 2 {
		t.Fatalf("len(Peers) = %d, want 2", len(status.Peers))
	}

	p := status.Peers[0]
	if p.PublicKey != "TrMvSoP4jYQlY6RIzBgbssQqY3vxI2piVFBs2LR9PQc=" {
		t.Errorf("Peer[0].PublicKey = %q", p.PublicKey)
	}
	if p.Endpoint != "203.0.113.1:51820" {
		t.Errorf("Peer[0].Endpoint = %q", p.Endpoint)
	}
	if p.TransferRx != "1.50 MiB" {
		t.Errorf("Peer[0].TransferRx = %q", p.TransferRx)
	}
	if p.TransferTx != "3.24 MiB" {
		t.Errorf("Peer[0].TransferTx = %q", p.TransferTx)
	}
}

func TestParseHandshakeTime(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"1 minute, 30 seconds ago", 90 * time.Second},
		{"45 seconds ago", 45 * time.Second},
		{"2 hours, 5 minutes, 10 seconds ago", 2*time.Hour + 5*time.Minute + 10*time.Second},
	}

	for _, tt := range tests {
		got, err := parseHandshakeTime(tt.input)
		if err != nil {
			t.Errorf("parseHandshakeTime(%q): %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseHandshakeTime(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/wg/ -run "TestParseWgShow|TestParseHandshake" -v`
Expected: FAIL — types and functions not defined.

**Step 3: Implement status parser**

`internal/wg/status.go`:
```go
package wg

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// InterfaceStatus holds live status from `wg show`.
type InterfaceStatus struct {
	PublicKey   string
	ListenPort int
	Peers      []PeerStatus
}

// PeerStatus holds live per-peer status.
type PeerStatus struct {
	PublicKey          string
	Endpoint          string
	AllowedIPs        string
	LatestHandshake   time.Duration
	TransferRx        string
	TransferTx        string
	PersistentKeepalive int
}

// GetStatus retrieves the live status of a WireGuard interface.
func GetStatus(name string) (*InterfaceStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wg", "show", name)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("wg show %s: %w", name, err)
	}

	return parseWgShow(string(out))
}

func parseWgShow(output string) (*InterfaceStatus, error) {
	status := &InterfaceStatus{}
	var currentPeer *PeerStatus
	inPeer := false

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "interface:") {
			inPeer = false
			continue
		}

		if strings.HasPrefix(line, "peer:") {
			if currentPeer != nil {
				status.Peers = append(status.Peers, *currentPeer)
			}
			currentPeer = &PeerStatus{
				PublicKey: strings.TrimSpace(strings.TrimPrefix(line, "peer:")),
			}
			inPeer = true
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		if inPeer && currentPeer != nil {
			switch key {
			case "endpoint":
				currentPeer.Endpoint = val
			case "allowed ips":
				currentPeer.AllowedIPs = val
			case "latest handshake":
				d, err := parseHandshakeTime(val)
				if err == nil {
					currentPeer.LatestHandshake = d
				}
			case "transfer":
				rx, tx := parseTransfer(val)
				currentPeer.TransferRx = rx
				currentPeer.TransferTx = tx
			case "persistent keepalive":
				if n, err := parseKeepalive(val); err == nil {
					currentPeer.PersistentKeepalive = n
				}
			}
		} else {
			switch key {
			case "public key":
				status.PublicKey = val
			case "listening port":
				if n, err := strconv.Atoi(val); err == nil {
					status.ListenPort = n
				}
			}
		}
	}

	if currentPeer != nil {
		status.Peers = append(status.Peers, *currentPeer)
	}

	return status, nil
}

var durationRe = regexp.MustCompile(`(\d+)\s+(hour|minute|second)s?`)

func parseHandshakeTime(s string) (time.Duration, error) {
	matches := durationRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("no duration found in %q", s)
	}

	var total time.Duration
	for _, m := range matches {
		n, _ := strconv.Atoi(m[1])
		switch m[2] {
		case "hour":
			total += time.Duration(n) * time.Hour
		case "minute":
			total += time.Duration(n) * time.Minute
		case "second":
			total += time.Duration(n) * time.Second
		}
	}
	return total, nil
}

func parseTransfer(s string) (rx, tx string) {
	// "1.50 MiB received, 3.24 MiB sent"
	parts := strings.Split(s, ",")
	if len(parts) >= 1 {
		rx = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(parts[0]), "received"))
		rx = strings.TrimSpace(rx)
	}
	if len(parts) >= 2 {
		tx = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(parts[1]), "sent"))
		tx = strings.TrimSpace(tx)
	}
	return rx, tx
}

func parseKeepalive(s string) (int, error) {
	// "every 25 seconds"
	matches := regexp.MustCompile(`(\d+)`).FindStringSubmatch(s)
	if len(matches) < 2 {
		return 0, fmt.Errorf("no number in %q", s)
	}
	return strconv.Atoi(matches[1])
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/wg/ -run "TestParseWgShow|TestParseHandshake" -v`
Expected: All PASS.

**Step 5: Commit**

```bash
git add internal/wg/status.go internal/wg/status_test.go
git commit -m "feat: add wg show output parser for live status"
```

---

### Task 6: QR Code Generation

**Files:**
- Create: `internal/wg/qr.go`
- Create: `internal/wg/qr_test.go`

**Step 1: Write failing tests**

`internal/wg/qr_test.go`:
```go
package wg

import (
	"testing"
)

func TestGenerateQRString(t *testing.T) {
	iface := &Interface{
		PrivateKey: "testkey123",
		Address:    "10.0.0.1/24",
		DNS:        "1.1.1.1",
		Peers: []Peer{
			{
				PublicKey:  "peerpubkey",
				AllowedIPs: "0.0.0.0/0",
				Endpoint:   "1.2.3.4:51820",
			},
		},
	}

	qr, err := GenerateQRString(iface)
	if err != nil {
		t.Fatalf("GenerateQRString: %v", err)
	}
	if len(qr) == 0 {
		t.Error("QR string is empty")
	}
	// QR ASCII art should contain block characters
	if len(qr) < 100 {
		t.Errorf("QR string too short (%d chars), likely not valid", len(qr))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/wg/ -run TestGenerateQR -v`
Expected: FAIL — function not defined.

**Step 3: Implement QR generation**

`internal/wg/qr.go`:
```go
package wg

import (
	qrcode "github.com/skip2/go-qrcode"
)

// GenerateQRString generates a terminal-printable QR code from an interface config.
func GenerateQRString(iface *Interface) (string, error) {
	conf := MarshalConfig(iface)
	qr, err := qrcode.New(conf, qrcode.Medium)
	if err != nil {
		return "", err
	}
	return qr.ToSmallString(false), nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/wg/ -run TestGenerateQR -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/wg/qr.go internal/wg/qr_test.go
git commit -m "feat: add QR code generation for config export"
```

---

### Task 7: TUI Styles

**Files:**
- Create: `internal/tui/styles.go`

**Step 1: Implement shared styles**

`internal/tui/styles.go`:
```go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	colorGreen  = lipgloss.Color("42")
	colorRed    = lipgloss.Color("196")
	colorDim    = lipgloss.Color("240")
	colorAccent = lipgloss.Color("63")
	colorWhite  = lipgloss.Color("255")

	// Title
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			MarginBottom(1)

	// Status indicators
	statusUp = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true).
			Render("● UP")

	statusDown = lipgloss.NewStyle().
			Foreground(colorDim).
			Render("○ DOWN")

	// Key hints
	keyStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	descStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	// Error / success messages
	errorStyle = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(colorGreen)

	// Detail labels
	labelStyle = lipgloss.NewStyle().
			Foreground(colorDim).
			Width(20)

	valueStyle = lipgloss.NewStyle().
			Foreground(colorWhite)

	// Border box
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(1, 2)
)

func helpKey(key, desc string) string {
	return keyStyle.Render("["+key+"]") + " " + descStyle.Render(desc)
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/tui/`
Expected: Builds without errors.

**Step 3: Commit**

```bash
git add internal/tui/styles.go
git commit -m "feat: add Lipgloss styles for TUI"
```

---

### Task 8: Root App Model & Navigation

**Files:**
- Create: `internal/tui/app.go`

**Step 1: Implement the root model with view routing**

`internal/tui/app.go`:
```go
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// viewType identifies which view is currently active.
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

// configDir is the directory containing WireGuard configs.
const configDir = "/etc/wireguard"

// App is the root Bubbletea model.
type App struct {
	currentView viewType
	list        listModel
	detail      detailModel
	wizard      wizardModel
	editor      editorModel
	status      statusModel
	importView  importModel
	exportView  exportModel
	confirm     confirmModel

	width  int
	height int

	err     error
	message string
}

// NewApp creates a new App instance.
func NewApp() App {
	return App{
		currentView: viewList,
		list:        newListModel(),
	}
}

func (a App) Init() tea.Cmd {
	return a.list.loadProfiles()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
	case errMsg:
		a.err = msg.err
		return a, nil
	case clearErrMsg:
		a.err = nil
		a.message = ""
		return a, nil
	}

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
		content += "\n" + errorStyle.Render(fmt.Sprintf("Error: %v", a.err))
	}
	if a.message != "" {
		content += "\n" + successStyle.Render(a.message)
	}

	return content
}

// Custom messages
type errMsg struct{ err error }
type clearErrMsg struct{}
type navigateMsg struct {
	view viewType
}
type refreshMsg struct{}
```

**Step 2: Verify it compiles (will fail — sub-models not yet defined)**

This file won't compile until we add the view-specific models. That's expected. Proceed to next tasks.

**Step 3: Commit (once all sub-models have stubs)**

This will be committed together with Task 9.

---

### Task 9: Profile List View (Main Screen)

**Files:**
- Create: `internal/tui/list.go`
- Modify: `internal/tui/app.go` (add updateList method)

**Step 1: Implement the list model**

`internal/tui/list.go`:
```go
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mlu/wireguard-tui/internal/wg"
)

type listModel struct {
	profiles []*wg.Interface
	active   map[string]bool // interface name -> is up
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

func (l listModel) loadProfiles() tea.Cmd {
	return func() tea.Msg {
		profiles, err := wg.LoadConfigsFromDir(configDir)
		if err != nil {
			return errMsg{err}
		}

		activeIfaces, err := wg.ListInterfaces()
		if err != nil {
			return errMsg{err}
		}

		active := make(map[string]bool)
		for _, name := range activeIfaces {
			active[name] = true
		}

		return profilesLoadedMsg{profiles: profiles, active: active}
	}
}

func (a App) updateList(msg tea.Msg) (App, tea.Cmd) {
	switch msg := msg.(type) {
	case profilesLoadedMsg:
		a.list.profiles = msg.profiles
		a.list.active = msg.active
		if a.list.cursor >= len(a.list.profiles) {
			a.list.cursor = max(0, len(a.list.profiles)-1)
		}
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
				selected := a.list.profiles[a.list.cursor]
				isUp := a.list.active[selected.Name]
				a.detail = newDetailModel(selected, isUp)
				a.currentView = viewDetail
			}
		case "n":
			a.wizard = newWizardModel()
			a.currentView = viewWizard
		case "i":
			a.importView = newImportModel()
			a.currentView = viewImport
		}
	case refreshMsg:
		return a, a.list.loadProfiles()
	}
	return a, nil
}

func (l listModel) view(width, height int) string {
	var b strings.Builder

	title := titleStyle.Render("WireGuard TUI")
	b.WriteString(title + "\n\n")

	if len(l.profiles) == 0 {
		b.WriteString(descStyle.Render("  No profiles found in " + configDir + "\n"))
		b.WriteString(descStyle.Render("  Press [n] to create one.\n"))
	} else {
		for i, p := range l.profiles {
			cursor := "  "
			if i == l.cursor {
				cursor = "▸ "
			}

			status := statusDown
			if l.active[p.Name] {
				status = statusUp
			}

			name := lipgloss.NewStyle().Bold(true).Width(15).Render(p.Name)
			addr := lipgloss.NewStyle().Width(20).Foreground(colorDim).Render(p.Address)
			peers := descStyle.Render(fmt.Sprintf("%d peers", len(p.Peers)))

			line := fmt.Sprintf("%s%s %s %s  %s", cursor, name, addr, status, peers)
			if i == l.cursor {
				line = lipgloss.NewStyle().Bold(true).Render(line)
			}
			b.WriteString(line + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(helpKey("n", "new") + "  " + helpKey("i", "import") + "  " + helpKey("q", "quit") + "\n")

	return b.String()
}
```

**Step 2: Create stubs for all other view models so app.go compiles**

Create stub files for each view. Each stub defines just the type and required methods with placeholder implementations. These will be filled in by subsequent tasks.

Create `internal/tui/detail.go`, `internal/tui/wizard.go`, `internal/tui/editor.go`, `internal/tui/status.go`, `internal/tui/import.go`, `internal/tui/export.go`, `internal/tui/confirm.go` — each as a minimal struct with `view()` returning a placeholder string and a constructor, plus the `update*` method on App.

**Step 3: Wire up main.go**

Update `main.go`:
```go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mlu/wireguard-tui/internal/tui"
)

func main() {
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "wireguard-tui must be run as root (use sudo)")
		os.Exit(1)
	}

	// Check for wg and wg-quick
	for _, bin := range []string{"wg", "wg-quick"} {
		if _, err := exec.LookPath(bin); err != nil {
			fmt.Fprintf(os.Stderr, "Required binary not found: %s\n", bin)
			os.Exit(1)
		}
	}

	p := tea.NewProgram(tui.NewApp(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 4: Verify it builds and runs**

Run: `go build -o wireguard-tui . && sudo ./wireguard-tui`
Expected: Shows profile list (empty if no configs in /etc/wireguard).

**Step 5: Commit**

```bash
git add internal/tui/ main.go
git commit -m "feat: add profile list view with navigation"
```

---

### Task 10: Profile Detail View

**Files:**
- Modify: `internal/tui/detail.go`

**Step 1: Implement detail model**

`internal/tui/detail.go`:
```go
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mlu/wireguard-tui/internal/wg"
)

type detailModel struct {
	profile *wg.Interface
	isUp    bool
}

func newDetailModel(profile *wg.Interface, isUp bool) detailModel {
	return detailModel{profile: profile, isUp: isUp}
}

func (a App) updateDetail(msg tea.Msg) (App, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			a.currentView = viewList
			return a, a.list.loadProfiles()
		case "e":
			a.editor = newEditorModel(a.detail.profile)
			a.currentView = viewEditor
		case "s":
			a.status = newStatusModel(a.detail.profile.Name)
			a.currentView = viewStatus
			return a, a.status.init()
		case "t":
			name := a.detail.profile.Name
			return a, func() tea.Msg {
				nowUp, err := wg.Toggle(name)
				if err != nil {
					return errMsg{err}
				}
				return toggledMsg{name: name, nowUp: nowUp}
			}
		case "x":
			a.exportView = newExportModel(a.detail.profile)
			a.currentView = viewExport
		case "d":
			a.confirm = newConfirmModel(
				fmt.Sprintf("Delete profile %q?", a.detail.profile.Name),
				deleteAction{name: a.detail.profile.Name},
			)
			a.currentView = viewConfirm
		}
	case toggledMsg:
		a.detail.isUp = msg.nowUp
		if msg.nowUp {
			a.message = fmt.Sprintf("%s is now UP", msg.name)
		} else {
			a.message = fmt.Sprintf("%s is now DOWN", msg.name)
		}
	}
	return a, nil
}

type toggledMsg struct {
	name  string
	nowUp bool
}

func (d detailModel) view(width, height int) string {
	var b strings.Builder
	p := d.profile

	title := titleStyle.Render(fmt.Sprintf("Profile: %s", p.Name))
	b.WriteString(title + "\n\n")

	status := statusDown
	if d.isUp {
		status = statusUp
	}
	b.WriteString(fmt.Sprintf("  %s  %s\n\n", labelStyle.Render("Status:"), status))

	fields := []struct{ label, value string }{
		{"Address", p.Address},
		{"Listen Port", fmt.Sprintf("%d", p.ListenPort)},
		{"DNS", p.DNS},
	}
	if p.MTU != 0 {
		fields = append(fields, struct{ label, value string }{"MTU", fmt.Sprintf("%d", p.MTU)})
	}

	for _, f := range fields {
		if f.value == "" || f.value == "0" {
			continue
		}
		b.WriteString(fmt.Sprintf("  %s  %s\n", labelStyle.Render(f.label+":"), valueStyle.Render(f.value)))
	}

	b.WriteString(fmt.Sprintf("\n  %s  %s\n", labelStyle.Render("Peers:"), valueStyle.Render(fmt.Sprintf("%d", len(p.Peers)))))

	for i, peer := range p.Peers {
		b.WriteString(fmt.Sprintf("\n  Peer %d:\n", i+1))
		b.WriteString(fmt.Sprintf("    %s  %s\n", labelStyle.Render("Public Key:"), descStyle.Render(peer.PublicKey[:20]+"...")))
		if peer.Endpoint != "" {
			b.WriteString(fmt.Sprintf("    %s  %s\n", labelStyle.Render("Endpoint:"), valueStyle.Render(peer.Endpoint)))
		}
		b.WriteString(fmt.Sprintf("    %s  %s\n", labelStyle.Render("Allowed IPs:"), valueStyle.Render(peer.AllowedIPs)))
	}

	b.WriteString("\n")
	b.WriteString(helpKey("e", "edit") + "  " + helpKey("s", "status") + "  " + helpKey("t", "toggle") + "  ")
	b.WriteString(helpKey("x", "export") + "  " + helpKey("d", "delete") + "  " + helpKey("esc", "back") + "\n")

	return b.String()
}
```

**Step 2: Verify it builds**

Run: `go build ./internal/tui/`
Expected: Builds.

**Step 3: Commit**

```bash
git add internal/tui/detail.go
git commit -m "feat: add profile detail view with actions"
```

---

### Task 11: Confirmation Dialog

**Files:**
- Modify: `internal/tui/confirm.go`

**Step 1: Implement confirmation dialog**

`internal/tui/confirm.go`:
```go
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mlu/wireguard-tui/internal/wg"
)

type confirmAction interface {
	execute() tea.Msg
}

type deleteAction struct {
	name string
}

func (d deleteAction) execute() tea.Msg {
	// Bring down if up, then delete config
	up, _ := wg.IsUp(d.name)
	if up {
		if err := wg.Down(d.name); err != nil {
			return errMsg{err}
		}
	}
	if err := wg.DeleteConfig(configDir, d.name); err != nil {
		return errMsg{err}
	}
	return deletedMsg{name: d.name}
}

type deletedMsg struct{ name string }

type confirmModel struct {
	message  string
	action   confirmAction
	selected int // 0 = yes, 1 = no
}

func newConfirmModel(message string, action confirmAction) confirmModel {
	return confirmModel{
		message:  message,
		action:   action,
		selected: 1, // default to No
	}
}

func (a App) updateConfirm(msg tea.Msg) (App, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			a.confirm.selected = 0
		case "right", "l":
			a.confirm.selected = 1
		case "y":
			a.confirm.selected = 0
			action := a.confirm.action
			a.currentView = viewList
			return a, func() tea.Msg { return action.execute() }
		case "n", "esc":
			a.currentView = viewDetail
		case "enter":
			if a.confirm.selected == 0 {
				action := a.confirm.action
				a.currentView = viewList
				return a, tea.Batch(
					func() tea.Msg { return action.execute() },
				)
			}
			a.currentView = viewDetail
		}
	case deletedMsg:
		a.message = fmt.Sprintf("Deleted %s", msg.name)
		a.currentView = viewList
		return a, a.list.loadProfiles()
	}
	return a, nil
}

func (c confirmModel) view(width, height int) string {
	title := titleStyle.Render("Confirm")
	msg := errorStyle.Render(c.message)

	yes := "  Yes  "
	no := "  No  "
	if c.selected == 0 {
		yes = keyStyle.Render("[ Yes ]")
		no = descStyle.Render("  No  ")
	} else {
		yes = descStyle.Render("  Yes  ")
		no = keyStyle.Render("[ No ]")
	}

	buttons := yes + "    " + no

	return fmt.Sprintf("%s\n\n  %s\n\n  %s\n\n  %s\n",
		title, msg, buttons, helpKey("y", "yes")+"  "+helpKey("n", "no"))
}
```

**Step 2: Verify it builds**

Run: `go build ./internal/tui/`

**Step 3: Commit**

```bash
git add internal/tui/confirm.go
git commit -m "feat: add confirmation dialog for destructive actions"
```

---

### Task 12: Creation Wizard

**Files:**
- Modify: `internal/tui/wizard.go`

**Step 1: Implement multi-step wizard**

This is the most complex view. The wizard has 6 steps, each with a text input. It uses `bubbles/textinput` for each field and manages step transitions.

`internal/tui/wizard.go` — implement the following:

- `wizardModel` struct with: `step int`, `inputs []textinput.Model`, `peers []wg.Peer`, `currentPeer` fields, `addingPeer bool`, `peerStep int`, `preview string`
- Steps: 0=name, 1=address, 2=port, 3=dns, 4=peers, 5=review
- Peer sub-steps: 0=pubkey, 1=allowedIPs, 2=endpoint, 3=presharedKey, 4=keepalive
- `newWizardModel()` — initializes with default values, auto-generates keys
- `updateWizard()` on App — handles Enter to advance steps, Esc to go back, tab for peer fields
- `view()` — shows current step with input field and navigation hints
- On confirm at review step: call `wg.SaveConfig()` and return to list

Key behaviors:
- Auto-generate private key on wizard start, show derived public key
- Suggest default interface name (scan existing, auto-increment)
- Suggest default address avoiding conflicts
- Each peer step collects one field at a time
- Review step shows full config preview using `wg.MarshalConfig()`

**Step 2: Verify it builds and runs**

Run: `go build -o wireguard-tui . && sudo ./wireguard-tui`
Expected: Press `n` from list → wizard starts, can step through fields.

**Step 3: Commit**

```bash
git add internal/tui/wizard.go
git commit -m "feat: add profile creation wizard with key generation"
```

---

### Task 13: Profile Editor

**Files:**
- Modify: `internal/tui/editor.go`

**Step 1: Implement editor model**

The editor is similar to the wizard but pre-populated with existing values. It shows each field of the Interface as an editable text input. Navigate fields with Tab/Shift-Tab, save with Ctrl+S or Enter on last field, cancel with Esc.

`internal/tui/editor.go` — implement:

- `editorModel` struct with: `profile *wg.Interface`, `inputs []textinput.Model`, `focusIndex int`
- Fields: Address, ListenPort, DNS, MTU (PrivateKey shown but not directly editable without confirmation)
- Peer editing: list peers, select to edit individual peer fields
- `newEditorModel(profile)` — populates inputs from profile
- `updateEditor()` on App — Tab cycles focus, Enter saves, Esc cancels
- On save: update the profile struct, call `wg.SaveConfig()`, return to detail view

**Step 2: Verify it builds**

Run: `go build ./internal/tui/`

**Step 3: Commit**

```bash
git add internal/tui/editor.go
git commit -m "feat: add profile editor with field navigation"
```

---

### Task 14: Live Status View

**Files:**
- Modify: `internal/tui/status.go`

**Step 1: Implement live status model**

`internal/tui/status.go` — implement:

- `statusModel` struct with: `name string`, `status *wg.InterfaceStatus`, `loading bool`
- `init()` returns a `tea.Cmd` that fetches initial status
- Uses `tea.Tick` every 2 seconds to refresh
- `updateStatus()` on App — handles tick refresh, Esc to go back
- `view()` — shows interface public key, listen port, then table of peers with: public key (truncated), endpoint, latest handshake, transfer rx/tx, keepalive

Use a `tea.Every(2*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })` pattern for auto-refresh.

**Step 2: Verify it builds**

Run: `go build ./internal/tui/`

**Step 3: Commit**

```bash
git add internal/tui/status.go
git commit -m "feat: add live status view with auto-refresh"
```

---

### Task 15: Import View

**Files:**
- Modify: `internal/tui/import.go`

**Step 1: Implement import model**

`internal/tui/import.go` — implement:

- `importModel` struct with: `pathInput textinput.Model`, `preview string`, `parsed *wg.Interface`
- User types a file path to a .conf file
- On Enter: parse the file, show preview
- On confirm: copy to /etc/wireguard/ with appropriate name, return to list
- Esc cancels

**Step 2: Verify it builds**

Run: `go build ./internal/tui/`

**Step 3: Commit**

```bash
git add internal/tui/import.go
git commit -m "feat: add config import from file path"
```

---

### Task 16: Export View

**Files:**
- Modify: `internal/tui/export.go`

**Step 1: Implement export model**

`internal/tui/export.go` — implement:

- `exportModel` struct with: `profile *wg.Interface`, `showQR bool`, `qrString string`
- Default view: shows config text
- Press `q` for QR: generates QR code string via `wg.GenerateQRString()` and displays it
- Press `c` for config text: switches back
- Press `s` to save to a custom path (text input for path)
- Esc goes back to detail

**Step 2: Verify it builds**

Run: `go build ./internal/tui/`

**Step 3: Commit**

```bash
git add internal/tui/export.go
git commit -m "feat: add export view with QR code display"
```

---

### Task 17: Final Integration & Polish

**Files:**
- Modify: `main.go`
- All `internal/tui/*.go` files (minor fixes)

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All PASS.

**Step 2: Build and test full flow manually**

Run: `go build -o wireguard-tui . && sudo ./wireguard-tui`
Test:
1. Empty state — shows "No profiles" message
2. Press `n` — wizard starts, step through all fields
3. Confirm — profile appears in list
4. Enter on profile — detail view with correct data
5. `t` to toggle — interface comes up/down
6. `s` for status — shows live stats
7. `x` for export — shows config text and QR
8. `e` to edit — can modify fields
9. `d` to delete — confirmation dialog, then deletes

**Step 3: Fix any issues discovered in manual testing**

**Step 4: Final commit**

```bash
git add -A
git commit -m "feat: complete WireGuard TUI with full lifecycle management"
```

---

## Summary

| Task | Component | Est. Complexity |
|------|-----------|----------------|
| 1 | Project scaffolding | Low |
| 2 | Config parser + tests | Medium |
| 3 | Key generation + tests | Low |
| 4 | Interface control + tests | Low |
| 5 | Status parser + tests | Medium |
| 6 | QR generation + tests | Low |
| 7 | TUI styles | Low |
| 8 | Root app model | Medium |
| 9 | Profile list view | Medium |
| 10 | Profile detail view | Medium |
| 11 | Confirmation dialog | Low |
| 12 | Creation wizard | High |
| 13 | Profile editor | Medium |
| 14 | Live status view | Medium |
| 15 | Import view | Low |
| 16 | Export view | Low |
| 17 | Integration & polish | Medium |
