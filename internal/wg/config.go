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

// sectionKind tracks which section we are currently parsing.
type sectionKind int

const (
	sectionNone sectionKind = iota
	sectionInterface
	sectionPeer
)

// ParseConfig reads a WireGuard .conf format from r and returns the parsed Interface.
// Lines starting with # are treated as comments and skipped.
// Empty lines are skipped. Parse errors include line number context.
func ParseConfig(r io.Reader) (*Interface, error) {
	iface := &Interface{}
	scanner := bufio.NewScanner(r)
	section := sectionNone
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for section headers
		if line == "[Interface]" {
			section = sectionInterface
			continue
		}
		if line == "[Peer]" {
			section = sectionPeer
			iface.Peers = append(iface.Peers, Peer{})
			continue
		}

		// Parse Key = Value pairs
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: expected Key = Value, got %q", lineNum, line)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch section {
		case sectionInterface:
			if err := setInterfaceField(iface, key, value, lineNum); err != nil {
				return nil, err
			}
		case sectionPeer:
			if len(iface.Peers) == 0 {
				return nil, fmt.Errorf("line %d: key %q outside of [Peer] section", lineNum, key)
			}
			if err := setPeerField(&iface.Peers[len(iface.Peers)-1], key, value, lineNum); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("line %d: key %q outside of any section", lineNum, key)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	return iface, nil
}

// setInterfaceField sets a field on the Interface from a key/value pair.
func setInterfaceField(iface *Interface, key, value string, lineNum int) error {
	switch key {
	case "PrivateKey":
		iface.PrivateKey = value
	case "Address":
		iface.Address = value
	case "ListenPort":
		port, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("line %d: invalid ListenPort %q: %w", lineNum, value, err)
		}
		iface.ListenPort = port
	case "DNS":
		iface.DNS = value
	case "MTU":
		mtu, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("line %d: invalid MTU %q: %w", lineNum, value, err)
		}
		iface.MTU = mtu
	default:
		// Ignore unknown keys for forward compatibility
	}
	return nil
}

// setPeerField sets a field on the Peer from a key/value pair.
func setPeerField(peer *Peer, key, value string, lineNum int) error {
	switch key {
	case "PublicKey":
		peer.PublicKey = value
	case "PresharedKey":
		peer.PresharedKey = value
	case "AllowedIPs":
		peer.AllowedIPs = value
	case "Endpoint":
		peer.Endpoint = value
	case "PersistentKeepalive":
		keepalive, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("line %d: invalid PersistentKeepalive %q: %w", lineNum, value, err)
		}
		peer.PersistentKeepalive = keepalive
	default:
		// Ignore unknown keys for forward compatibility
	}
	return nil
}

// MarshalConfig serializes an Interface back to WireGuard .conf format.
// Optional fields with zero/empty values are omitted.
// Peer sections are separated by blank lines.
func MarshalConfig(iface *Interface) string {
	var b strings.Builder

	b.WriteString("[Interface]\n")
	fmt.Fprintf(&b, "PrivateKey = %s\n", iface.PrivateKey)
	fmt.Fprintf(&b, "Address = %s\n", iface.Address)
	if iface.ListenPort != 0 {
		fmt.Fprintf(&b, "ListenPort = %d\n", iface.ListenPort)
	}
	if iface.DNS != "" {
		fmt.Fprintf(&b, "DNS = %s\n", iface.DNS)
	}
	if iface.MTU != 0 {
		fmt.Fprintf(&b, "MTU = %d\n", iface.MTU)
	}

	for _, peer := range iface.Peers {
		b.WriteString("\n[Peer]\n")
		fmt.Fprintf(&b, "PublicKey = %s\n", peer.PublicKey)
		if peer.PresharedKey != "" {
			fmt.Fprintf(&b, "PresharedKey = %s\n", peer.PresharedKey)
		}
		fmt.Fprintf(&b, "AllowedIPs = %s\n", peer.AllowedIPs)
		if peer.Endpoint != "" {
			fmt.Fprintf(&b, "Endpoint = %s\n", peer.Endpoint)
		}
		if peer.PersistentKeepalive != 0 {
			fmt.Fprintf(&b, "PersistentKeepalive = %d\n", peer.PersistentKeepalive)
		}
	}

	return b.String()
}

// LoadConfigsFromDir reads all .conf files from the given directory and returns
// the parsed Interface configurations. Each Interface's Name field is set from
// the filename (without the .conf extension). Non-.conf files are ignored.
func LoadConfigsFromDir(dir string) ([]*Interface, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	var configs []*Interface
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".conf") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("opening %s: %w", path, err)
		}

		iface, err := ParseConfig(f)
		_ = f.Close()
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}

		// Set Name from filename without .conf extension
		iface.Name = strings.TrimSuffix(entry.Name(), ".conf")
		configs = append(configs, iface)
	}

	return configs, nil
}

// SaveConfig writes the Interface configuration to dir/name.conf with 0600 permissions.
func SaveConfig(dir string, iface *Interface) error {
	path := filepath.Join(dir, iface.Name+".conf")
	content := MarshalConfig(iface)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("writing config %s: %w", path, err)
	}
	return nil
}

// DeleteConfig removes the configuration file dir/name.conf.
func DeleteConfig(dir string, name string) error {
	path := filepath.Join(dir, name+".conf")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("removing config %s: %w", path, err)
	}
	return nil
}
