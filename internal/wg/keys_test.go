package wg

import (
	"encoding/base64"
	"testing"
)

func TestGeneratePrivateKey(t *testing.T) {
	key, err := GeneratePrivateKey()
	if err != nil {
		t.Fatalf("GeneratePrivateKey() returned error: %v", err)
	}

	// WireGuard keys are base64-encoded 32-byte values (44 chars with padding)
	if len(key) != 44 {
		t.Errorf("GeneratePrivateKey() key length = %d, want 44", len(key))
	}

	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		t.Fatalf("GeneratePrivateKey() returned invalid base64: %v", err)
	}
	if len(decoded) != 32 {
		t.Errorf("GeneratePrivateKey() decoded length = %d, want 32 bytes", len(decoded))
	}

	// Generate a second key and verify it's different (randomness check)
	key2, err := GeneratePrivateKey()
	if err != nil {
		t.Fatalf("second GeneratePrivateKey() returned error: %v", err)
	}
	if key == key2 {
		t.Error("GeneratePrivateKey() generated two identical keys; expected unique keys")
	}
}

func TestDerivePublicKey(t *testing.T) {
	privKey, err := GeneratePrivateKey()
	if err != nil {
		t.Fatalf("GeneratePrivateKey() returned error: %v", err)
	}

	pubKey, err := DerivePublicKey(privKey)
	if err != nil {
		t.Fatalf("DerivePublicKey() returned error: %v", err)
	}

	// Verify base64-encoded 32-byte value
	if len(pubKey) != 44 {
		t.Errorf("DerivePublicKey() key length = %d, want 44", len(pubKey))
	}

	decoded, err := base64.StdEncoding.DecodeString(pubKey)
	if err != nil {
		t.Fatalf("DerivePublicKey() returned invalid base64: %v", err)
	}
	if len(decoded) != 32 {
		t.Errorf("DerivePublicKey() decoded length = %d, want 32 bytes", len(decoded))
	}

	// Derive again — must be deterministic
	pubKey2, err := DerivePublicKey(privKey)
	if err != nil {
		t.Fatalf("second DerivePublicKey() returned error: %v", err)
	}
	if pubKey != pubKey2 {
		t.Errorf("DerivePublicKey() not deterministic: %q != %q", pubKey, pubKey2)
	}

	// Public key must differ from private key
	if pubKey == privKey {
		t.Error("DerivePublicKey() returned same value as private key")
	}
}

func TestGeneratePresharedKey(t *testing.T) {
	psk, err := GeneratePresharedKey()
	if err != nil {
		t.Fatalf("GeneratePresharedKey() returned error: %v", err)
	}

	// Verify base64-encoded 32-byte value
	if len(psk) != 44 {
		t.Errorf("GeneratePresharedKey() key length = %d, want 44", len(psk))
	}

	decoded, err := base64.StdEncoding.DecodeString(psk)
	if err != nil {
		t.Fatalf("GeneratePresharedKey() returned invalid base64: %v", err)
	}
	if len(decoded) != 32 {
		t.Errorf("GeneratePresharedKey() decoded length = %d, want 32 bytes", len(decoded))
	}

	// Generate a second PSK and verify it's different
	psk2, err := GeneratePresharedKey()
	if err != nil {
		t.Fatalf("second GeneratePresharedKey() returned error: %v", err)
	}
	if psk == psk2 {
		t.Error("GeneratePresharedKey() generated two identical PSKs; expected unique keys")
	}
}

func TestGenerateKeyPair(t *testing.T) {
	privKey, pubKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() returned error: %v", err)
	}

	// Both keys should be valid base64-encoded 32-byte values
	for name, key := range map[string]string{"private": privKey, "public": pubKey} {
		if len(key) != 44 {
			t.Errorf("GenerateKeyPair() %s key length = %d, want 44", name, len(key))
		}
		decoded, err := base64.StdEncoding.DecodeString(key)
		if err != nil {
			t.Fatalf("GenerateKeyPair() %s key invalid base64: %v", name, err)
		}
		if len(decoded) != 32 {
			t.Errorf("GenerateKeyPair() %s key decoded length = %d, want 32", name, len(decoded))
		}
	}

	// Public key should match what DerivePublicKey returns for the same private key
	derivedPub, err := DerivePublicKey(privKey)
	if err != nil {
		t.Fatalf("DerivePublicKey() returned error: %v", err)
	}
	if pubKey != derivedPub {
		t.Errorf("GenerateKeyPair() public key %q != DerivePublicKey() %q", pubKey, derivedPub)
	}

	// Private and public should differ
	if privKey == pubKey {
		t.Error("GenerateKeyPair() private and public keys are identical")
	}
}

func TestDerivePublicKeyInvalidInput(t *testing.T) {
	_, err := DerivePublicKey("not-a-valid-key")
	if err == nil {
		t.Error("DerivePublicKey() with invalid input should return error, got nil")
	}
}
