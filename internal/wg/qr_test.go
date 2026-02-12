package wg

import "testing"

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
		t.Fatalf("GenerateQRString returned error: %v", err)
	}
	if len(qr) == 0 {
		t.Fatal("GenerateQRString returned empty string")
	}
	if len(qr) <= 100 {
		t.Errorf("QR string too short (%d chars), expected substantial ASCII art (>100 chars)", len(qr))
	}
}

func TestGenerateQRStringNoPeers(t *testing.T) {
	iface := &Interface{
		PrivateKey: "testkey123",
		Address:    "10.0.0.1/24",
	}

	qr, err := GenerateQRString(iface)
	if err != nil {
		t.Fatalf("GenerateQRString returned error: %v", err)
	}
	if len(qr) == 0 {
		t.Fatal("GenerateQRString returned empty string")
	}
}
