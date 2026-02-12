package wg

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// InterfaceStatus represents the live runtime status of a WireGuard interface
// as reported by `wg show`.
type InterfaceStatus struct {
	PublicKey  string
	ListenPort int
	Peers      []PeerStatus
}

// PeerStatus represents the live runtime status of a single peer
// as reported by `wg show`.
type PeerStatus struct {
	PublicKey           string
	Endpoint            string
	AllowedIPs          string
	LatestHandshake     time.Duration
	TransferRx          string
	TransferTx          string
	PersistentKeepalive int
}

// durationPartRe matches a single component like "1 minute" or "30 seconds".
var durationPartRe = regexp.MustCompile(`(\d+)\s+(hour|minute|second)s?`)

// GetStatus runs `wg show <name>` and parses the output into an InterfaceStatus.
func GetStatus(name string) (*InterfaceStatus, error) {
	out, err := runWgCmd("show", name)
	if err != nil {
		return nil, fmt.Errorf("getting status for %s: %w", name, err)
	}
	return parseWgShow(out)
}

// parseWgShow parses the raw text output of `wg show <name>` into an
// InterfaceStatus. The output format uses indented key-value pairs grouped
// under `interface:` and `peer:` headers.
func parseWgShow(output string) (*InterfaceStatus, error) {
	status := &InterfaceStatus{}
	var currentPeer *PeerStatus

	for _, line := range strings.Split(output, "\n") {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check for section headers (not indented)
		if strings.HasPrefix(line, "interface:") {
			// Interface section — nothing to extract from header itself
			continue
		}
		if strings.HasPrefix(line, "peer:") {
			pubKey := strings.TrimSpace(strings.TrimPrefix(line, "peer:"))
			status.Peers = append(status.Peers, PeerStatus{PublicKey: pubKey})
			currentPeer = &status.Peers[len(status.Peers)-1]
			continue
		}

		// All key-value lines are indented with 2 spaces
		trimmed := strings.TrimSpace(line)
		sepIdx := strings.Index(trimmed, ": ")
		if sepIdx < 0 {
			continue
		}
		key := trimmed[:sepIdx]
		value := trimmed[sepIdx+2:]

		// Determine whether this belongs to the interface or current peer
		if currentPeer == nil {
			// Interface section
			switch key {
			case "public key":
				status.PublicKey = value
			case "listening port":
				port, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("parsing listening port %q: %w", value, err)
				}
				status.ListenPort = port
			}
		} else {
			// Peer section
			switch key {
			case "endpoint":
				currentPeer.Endpoint = value
			case "allowed ips":
				currentPeer.AllowedIPs = value
			case "latest handshake":
				d, err := parseHandshakeTime(value)
				if err != nil {
					return nil, fmt.Errorf("parsing handshake time %q: %w", value, err)
				}
				currentPeer.LatestHandshake = d
			case "transfer":
				rx, tx := parseTransfer(value)
				currentPeer.TransferRx = rx
				currentPeer.TransferTx = tx
			case "persistent keepalive":
				k, err := parseKeepalive(value)
				if err != nil {
					return nil, fmt.Errorf("parsing keepalive %q: %w", value, err)
				}
				currentPeer.PersistentKeepalive = k
			}
		}
	}

	return status, nil
}

// parseHandshakeTime parses strings like "1 minute, 30 seconds ago" into a
// time.Duration. It extracts all hour/minute/second components using regex.
func parseHandshakeTime(s string) (time.Duration, error) {
	matches := durationPartRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("no duration components found in %q", s)
	}

	var total time.Duration
	for _, match := range matches {
		n, err := strconv.Atoi(match[1])
		if err != nil {
			return 0, fmt.Errorf("parsing number %q: %w", match[1], err)
		}
		switch match[2] {
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

// parseTransfer splits a transfer line like "1.50 MiB received, 3.24 MiB sent"
// into separate rx and tx strings including units (e.g. "1.50 MiB", "3.24 MiB").
func parseTransfer(s string) (rx, tx string) {
	// Format: "X.XX UiB received, Y.YY UiB sent"
	parts := strings.SplitN(s, ", ", 2)
	if len(parts) == 2 {
		rx = strings.TrimSuffix(parts[0], " received")
		tx = strings.TrimSuffix(parts[1], " sent")
	}
	return rx, tx
}

// parseKeepalive extracts the number of seconds from a string like
// "every 25 seconds".
func parseKeepalive(s string) (int, error) {
	s = strings.TrimPrefix(s, "every ")
	s = strings.TrimSuffix(s, " seconds")
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("parsing keepalive interval %q: %w", s, err)
	}
	return n, nil
}
