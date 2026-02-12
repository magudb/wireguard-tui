package wg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleConfig = `[Interface]
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
	iface, err := ParseConfig(strings.NewReader(sampleConfig))
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	// Verify Interface fields
	if iface.PrivateKey != "yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk=" {
		t.Errorf("PrivateKey = %q, want %q", iface.PrivateKey, "yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk=")
	}
	if iface.Address != "10.0.0.1/24" {
		t.Errorf("Address = %q, want %q", iface.Address, "10.0.0.1/24")
	}
	if iface.ListenPort != 51820 {
		t.Errorf("ListenPort = %d, want %d", iface.ListenPort, 51820)
	}
	if iface.DNS != "1.1.1.1, 8.8.8.8" {
		t.Errorf("DNS = %q, want %q", iface.DNS, "1.1.1.1, 8.8.8.8")
	}
	if iface.MTU != 1420 {
		t.Errorf("MTU = %d, want %d", iface.MTU, 1420)
	}

	// Verify peers
	if len(iface.Peers) != 2 {
		t.Fatalf("len(Peers) = %d, want 2", len(iface.Peers))
	}

	// First peer
	p0 := iface.Peers[0]
	if p0.PublicKey != "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=" {
		t.Errorf("Peer[0].PublicKey = %q, want %q", p0.PublicKey, "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=")
	}
	if p0.PresharedKey != "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=" {
		t.Errorf("Peer[0].PresharedKey = %q, want %q", p0.PresharedKey, "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	}
	if p0.AllowedIPs != "0.0.0.0/0, ::/0" {
		t.Errorf("Peer[0].AllowedIPs = %q, want %q", p0.AllowedIPs, "0.0.0.0/0, ::/0")
	}
	if p0.Endpoint != "203.0.113.1:51820" {
		t.Errorf("Peer[0].Endpoint = %q, want %q", p0.Endpoint, "203.0.113.1:51820")
	}
	if p0.PersistentKeepalive != 25 {
		t.Errorf("Peer[0].PersistentKeepalive = %d, want %d", p0.PersistentKeepalive, 25)
	}

	// Second peer
	p1 := iface.Peers[1]
	if p1.PublicKey != "TrMvSoP4jYQlY6RIzBgbssQqY3vxI2piVFBs2LR9PQc=" {
		t.Errorf("Peer[1].PublicKey = %q, want %q", p1.PublicKey, "TrMvSoP4jYQlY6RIzBgbssQqY3vxI2piVFBs2LR9PQc=")
	}
	if p1.AllowedIPs != "10.0.0.2/32" {
		t.Errorf("Peer[1].AllowedIPs = %q, want %q", p1.AllowedIPs, "10.0.0.2/32")
	}
	// These should be zero-values for the second peer
	if p1.PresharedKey != "" {
		t.Errorf("Peer[1].PresharedKey = %q, want empty", p1.PresharedKey)
	}
	if p1.Endpoint != "" {
		t.Errorf("Peer[1].Endpoint = %q, want empty", p1.Endpoint)
	}
	if p1.PersistentKeepalive != 0 {
		t.Errorf("Peer[1].PersistentKeepalive = %d, want 0", p1.PersistentKeepalive)
	}
}

func TestParseConfigComments(t *testing.T) {
	input := `# This is a comment
[Interface]
PrivateKey = abc123=
Address = 10.0.0.1/24
# Another comment
`
	iface, err := ParseConfig(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}
	if iface.PrivateKey != "abc123=" {
		t.Errorf("PrivateKey = %q, want %q", iface.PrivateKey, "abc123=")
	}
	if iface.Address != "10.0.0.1/24" {
		t.Errorf("Address = %q, want %q", iface.Address, "10.0.0.1/24")
	}
}

func TestParseConfigErrorOnInvalidInt(t *testing.T) {
	input := `[Interface]
PrivateKey = abc123=
Address = 10.0.0.1/24
ListenPort = notanumber
`
	_, err := ParseConfig(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for invalid ListenPort, got nil")
	}
	// Error should contain line number context
	if !strings.Contains(err.Error(), "4") {
		t.Errorf("error %q should contain line number 4", err.Error())
	}
}

func TestMarshalConfig(t *testing.T) {
	iface := &Interface{
		PrivateKey: "yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk=",
		Address:    "10.0.0.1/24",
		ListenPort: 51820,
		DNS:        "1.1.1.1, 8.8.8.8",
		MTU:        1420,
		Peers: []Peer{
			{
				PublicKey:           "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=",
				PresharedKey:        "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
				AllowedIPs:          "0.0.0.0/0, ::/0",
				Endpoint:            "203.0.113.1:51820",
				PersistentKeepalive: 25,
			},
		},
	}

	output := MarshalConfig(iface)

	expectedLines := []string{
		"[Interface]",
		"PrivateKey = yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk=",
		"Address = 10.0.0.1/24",
		"ListenPort = 51820",
		"DNS = 1.1.1.1, 8.8.8.8",
		"MTU = 1420",
		"[Peer]",
		"PublicKey = xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=",
		"PresharedKey = AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"AllowedIPs = 0.0.0.0/0, ::/0",
		"Endpoint = 203.0.113.1:51820",
		"PersistentKeepalive = 25",
	}

	for _, line := range expectedLines {
		if !strings.Contains(output, line) {
			t.Errorf("output missing expected line %q", line)
		}
	}
}

func TestMarshalConfigOmitsEmptyOptionalFields(t *testing.T) {
	iface := &Interface{
		PrivateKey: "abc123=",
		Address:    "10.0.0.1/24",
		// ListenPort: 0 (zero value, should be omitted)
		// DNS: "" (empty, should be omitted)
		// MTU: 0 (zero value, should be omitted)
		Peers: []Peer{
			{
				PublicKey:  "def456=",
				AllowedIPs: "10.0.0.2/32",
				// PresharedKey: "" (empty, should be omitted)
				// Endpoint: "" (empty, should be omitted)
				// PersistentKeepalive: 0 (zero value, should be omitted)
			},
		},
	}

	output := MarshalConfig(iface)

	// These required fields should be present
	if !strings.Contains(output, "PrivateKey = abc123=") {
		t.Error("output should contain PrivateKey")
	}
	if !strings.Contains(output, "Address = 10.0.0.1/24") {
		t.Error("output should contain Address")
	}
	if !strings.Contains(output, "PublicKey = def456=") {
		t.Error("output should contain PublicKey")
	}
	if !strings.Contains(output, "AllowedIPs = 10.0.0.2/32") {
		t.Error("output should contain AllowedIPs")
	}

	// These optional zero-value fields should NOT be present
	omittedKeys := []string{
		"ListenPort",
		"DNS",
		"MTU",
		"PresharedKey",
		"Endpoint",
		"PersistentKeepalive",
	}
	for _, key := range omittedKeys {
		if strings.Contains(output, key) {
			t.Errorf("output should NOT contain %q when it is zero/empty", key)
		}
	}
}

func TestMarshalConfigPeerSeparation(t *testing.T) {
	iface := &Interface{
		PrivateKey: "abc123=",
		Address:    "10.0.0.1/24",
		Peers: []Peer{
			{PublicKey: "peer1=", AllowedIPs: "10.0.0.2/32"},
			{PublicKey: "peer2=", AllowedIPs: "10.0.0.3/32"},
		},
	}

	output := MarshalConfig(iface)

	// There should be a blank line between [Peer] sections
	// The output should have two [Peer] headers
	count := strings.Count(output, "[Peer]")
	if count != 2 {
		t.Errorf("[Peer] count = %d, want 2", count)
	}
}

func TestLoadConfigsFromDir(t *testing.T) {
	dir := t.TempDir()

	// Write two .conf files
	conf1 := `[Interface]
PrivateKey = key1=
Address = 10.0.0.1/24
`
	conf2 := `[Interface]
PrivateKey = key2=
Address = 10.0.0.2/24
`
	// Write a .txt file that should be ignored
	txtContent := `This is not a config file`

	if err := os.WriteFile(filepath.Join(dir, "wg0.conf"), []byte(conf1), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "wg1.conf"), []byte(conf2), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte(txtContent), 0644); err != nil {
		t.Fatal(err)
	}

	configs, err := LoadConfigsFromDir(dir)
	if err != nil {
		t.Fatalf("LoadConfigsFromDir returned error: %v", err)
	}

	if len(configs) != 2 {
		t.Fatalf("len(configs) = %d, want 2", len(configs))
	}

	// Check that names were set from filenames (without .conf extension)
	names := map[string]bool{}
	for _, c := range configs {
		names[c.Name] = true
	}
	if !names["wg0"] {
		t.Error("expected config with Name=wg0")
	}
	if !names["wg1"] {
		t.Error("expected config with Name=wg1")
	}
}

func TestLoadConfigsFromDirEmpty(t *testing.T) {
	dir := t.TempDir()

	configs, err := LoadConfigsFromDir(dir)
	if err != nil {
		t.Fatalf("LoadConfigsFromDir returned error: %v", err)
	}
	if len(configs) != 0 {
		t.Errorf("len(configs) = %d, want 0 for empty directory", len(configs))
	}
}

func TestSaveConfig(t *testing.T) {
	dir := t.TempDir()

	iface := &Interface{
		Name:       "wg0",
		PrivateKey: "testkey=",
		Address:    "10.0.0.1/24",
	}

	if err := SaveConfig(dir, iface); err != nil {
		t.Fatalf("SaveConfig returned error: %v", err)
	}

	path := filepath.Join(dir, "wg0.conf")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("config file not found: %v", err)
	}

	// Check permissions are 0600
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}

	// Verify the file content is valid
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}
	if !strings.Contains(string(data), "PrivateKey = testkey=") {
		t.Error("saved config missing PrivateKey")
	}
}

func TestDeleteConfig(t *testing.T) {
	dir := t.TempDir()

	// Create a config file to delete
	path := filepath.Join(dir, "wg0.conf")
	if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := DeleteConfig(dir, "wg0"); err != nil {
		t.Fatalf("DeleteConfig returned error: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("config file still exists after DeleteConfig")
	}
}

func TestDeleteConfigNotFound(t *testing.T) {
	dir := t.TempDir()

	err := DeleteConfig(dir, "nonexistent")
	if err == nil {
		t.Error("expected error when deleting nonexistent config, got nil")
	}
}

func TestRoundTrip(t *testing.T) {
	// Parse the sample config
	iface1, err := ParseConfig(strings.NewReader(sampleConfig))
	if err != nil {
		t.Fatalf("first ParseConfig returned error: %v", err)
	}

	// Marshal it back
	output := MarshalConfig(iface1)

	// Parse the marshaled output
	iface2, err := ParseConfig(strings.NewReader(output))
	if err != nil {
		t.Fatalf("second ParseConfig returned error: %v", err)
	}

	// Verify all fields match
	if iface1.PrivateKey != iface2.PrivateKey {
		t.Errorf("PrivateKey mismatch: %q vs %q", iface1.PrivateKey, iface2.PrivateKey)
	}
	if iface1.Address != iface2.Address {
		t.Errorf("Address mismatch: %q vs %q", iface1.Address, iface2.Address)
	}
	if iface1.ListenPort != iface2.ListenPort {
		t.Errorf("ListenPort mismatch: %d vs %d", iface1.ListenPort, iface2.ListenPort)
	}
	if iface1.DNS != iface2.DNS {
		t.Errorf("DNS mismatch: %q vs %q", iface1.DNS, iface2.DNS)
	}
	if iface1.MTU != iface2.MTU {
		t.Errorf("MTU mismatch: %d vs %d", iface1.MTU, iface2.MTU)
	}

	if len(iface1.Peers) != len(iface2.Peers) {
		t.Fatalf("Peers length mismatch: %d vs %d", len(iface1.Peers), len(iface2.Peers))
	}

	for i := range iface1.Peers {
		p1 := iface1.Peers[i]
		p2 := iface2.Peers[i]
		if p1.PublicKey != p2.PublicKey {
			t.Errorf("Peer[%d].PublicKey mismatch: %q vs %q", i, p1.PublicKey, p2.PublicKey)
		}
		if p1.PresharedKey != p2.PresharedKey {
			t.Errorf("Peer[%d].PresharedKey mismatch: %q vs %q", i, p1.PresharedKey, p2.PresharedKey)
		}
		if p1.AllowedIPs != p2.AllowedIPs {
			t.Errorf("Peer[%d].AllowedIPs mismatch: %q vs %q", i, p1.AllowedIPs, p2.AllowedIPs)
		}
		if p1.Endpoint != p2.Endpoint {
			t.Errorf("Peer[%d].Endpoint mismatch: %q vs %q", i, p1.Endpoint, p2.Endpoint)
		}
		if p1.PersistentKeepalive != p2.PersistentKeepalive {
			t.Errorf("Peer[%d].PersistentKeepalive mismatch: %d vs %d", i, p1.PersistentKeepalive, p2.PersistentKeepalive)
		}
	}
}
