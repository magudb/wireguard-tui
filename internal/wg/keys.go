package wg

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// cmdTimeout is the maximum duration for any wg subprocess invocation.
const cmdTimeout = 5 * time.Second

// runWgCmd executes the wg binary with the given arguments and returns its
// trimmed stdout. A context timeout of cmdTimeout is applied.
func runWgCmd(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wg", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("wg %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}

	return strings.TrimSpace(stdout.String()), nil
}

// runWgCmdWithInput executes the wg binary with the given arguments, piping
// input to its stdin, and returns trimmed stdout. A context timeout of
// cmdTimeout is applied.
func runWgCmdWithInput(input string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wg", args...)
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("wg %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GeneratePrivateKey generates a new WireGuard private key by calling `wg genkey`.
// The returned key is a base64-encoded 32-byte Curve25519 private key.
func GeneratePrivateKey() (string, error) {
	key, err := runWgCmd("genkey")
	if err != nil {
		return "", fmt.Errorf("generating private key: %w", err)
	}
	return key, nil
}

// DerivePublicKey derives the public key from a WireGuard private key by
// piping it to `wg pubkey`. The derivation is deterministic.
func DerivePublicKey(privateKey string) (string, error) {
	key, err := runWgCmdWithInput(privateKey, "pubkey")
	if err != nil {
		return "", fmt.Errorf("deriving public key: %w", err)
	}
	return key, nil
}

// GeneratePresharedKey generates a new WireGuard preshared key by calling
// `wg genpsk`. The returned key is a base64-encoded 32-byte random value.
func GeneratePresharedKey() (string, error) {
	key, err := runWgCmd("genpsk")
	if err != nil {
		return "", fmt.Errorf("generating preshared key: %w", err)
	}
	return key, nil
}

// GenerateKeyPair generates a new WireGuard private key and derives its
// corresponding public key. Both are returned as base64-encoded strings.
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
