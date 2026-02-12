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
  transfer: 500.00 KiB received, 120.00 KiB sent`

func TestParseWgShow(t *testing.T) {
	status, err := parseWgShow(sampleWgShow)
	if err != nil {
		t.Fatalf("parseWgShow() returned error: %v", err)
	}

	// Verify interface fields
	if status.PublicKey != "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=" {
		t.Errorf("PublicKey = %q, want %q", status.PublicKey, "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=")
	}
	if status.ListenPort != 51820 {
		t.Errorf("ListenPort = %d, want %d", status.ListenPort, 51820)
	}

	// Verify peer count
	if len(status.Peers) != 2 {
		t.Fatalf("len(Peers) = %d, want 2", len(status.Peers))
	}

	// Verify first peer
	p0 := status.Peers[0]
	if p0.PublicKey != "TrMvSoP4jYQlY6RIzBgbssQqY3vxI2piVFBs2LR9PQc=" {
		t.Errorf("Peer[0].PublicKey = %q, want %q", p0.PublicKey, "TrMvSoP4jYQlY6RIzBgbssQqY3vxI2piVFBs2LR9PQc=")
	}
	if p0.Endpoint != "203.0.113.1:51820" {
		t.Errorf("Peer[0].Endpoint = %q, want %q", p0.Endpoint, "203.0.113.1:51820")
	}
	if p0.AllowedIPs != "10.0.0.2/32" {
		t.Errorf("Peer[0].AllowedIPs = %q, want %q", p0.AllowedIPs, "10.0.0.2/32")
	}
	if p0.LatestHandshake != 90*time.Second {
		t.Errorf("Peer[0].LatestHandshake = %v, want %v", p0.LatestHandshake, 90*time.Second)
	}
	if p0.TransferRx != "1.50 MiB" {
		t.Errorf("Peer[0].TransferRx = %q, want %q", p0.TransferRx, "1.50 MiB")
	}
	if p0.TransferTx != "3.24 MiB" {
		t.Errorf("Peer[0].TransferTx = %q, want %q", p0.TransferTx, "3.24 MiB")
	}
	if p0.PersistentKeepalive != 25 {
		t.Errorf("Peer[0].PersistentKeepalive = %d, want %d", p0.PersistentKeepalive, 25)
	}

	// Verify second peer
	p1 := status.Peers[1]
	if p1.PublicKey != "abc123publickey=" {
		t.Errorf("Peer[1].PublicKey = %q, want %q", p1.PublicKey, "abc123publickey=")
	}
	if p1.Endpoint != "198.51.100.1:51820" {
		t.Errorf("Peer[1].Endpoint = %q, want %q", p1.Endpoint, "198.51.100.1:51820")
	}
	if p1.AllowedIPs != "10.0.0.3/32" {
		t.Errorf("Peer[1].AllowedIPs = %q, want %q", p1.AllowedIPs, "10.0.0.3/32")
	}
	if p1.LatestHandshake != 45*time.Second {
		t.Errorf("Peer[1].LatestHandshake = %v, want %v", p1.LatestHandshake, 45*time.Second)
	}
	if p1.TransferRx != "500.00 KiB" {
		t.Errorf("Peer[1].TransferRx = %q, want %q", p1.TransferRx, "500.00 KiB")
	}
	if p1.TransferTx != "120.00 KiB" {
		t.Errorf("Peer[1].TransferTx = %q, want %q", p1.TransferTx, "120.00 KiB")
	}
	// Second peer has no persistent keepalive — should be zero value
	if p1.PersistentKeepalive != 0 {
		t.Errorf("Peer[1].PersistentKeepalive = %d, want 0", p1.PersistentKeepalive)
	}
}

func TestParseHandshakeTime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{
			name:  "minute and seconds",
			input: "1 minute, 30 seconds ago",
			want:  90 * time.Second,
		},
		{
			name:  "seconds only",
			input: "45 seconds ago",
			want:  45 * time.Second,
		},
		{
			name:  "hours minutes seconds",
			input: "2 hours, 5 minutes, 10 seconds ago",
			want:  2*time.Hour + 5*time.Minute + 10*time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseHandshakeTime(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseHandshakeTime(%q) expected error, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseHandshakeTime(%q) returned error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("parseHandshakeTime(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseTransfer(t *testing.T) {
	rx, tx := parseTransfer("1.50 MiB received, 3.24 MiB sent")
	if rx != "1.50 MiB" {
		t.Errorf("parseTransfer rx = %q, want %q", rx, "1.50 MiB")
	}
	if tx != "3.24 MiB" {
		t.Errorf("parseTransfer tx = %q, want %q", tx, "3.24 MiB")
	}

	rx2, tx2 := parseTransfer("500.00 KiB received, 120.00 KiB sent")
	if rx2 != "500.00 KiB" {
		t.Errorf("parseTransfer rx = %q, want %q", rx2, "500.00 KiB")
	}
	if tx2 != "120.00 KiB" {
		t.Errorf("parseTransfer tx = %q, want %q", tx2, "120.00 KiB")
	}
}

func TestParseKeepalive(t *testing.T) {
	got, err := parseKeepalive("every 25 seconds")
	if err != nil {
		t.Fatalf("parseKeepalive() returned error: %v", err)
	}
	if got != 25 {
		t.Errorf("parseKeepalive() = %d, want 25", got)
	}
}
