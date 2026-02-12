package wg

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Up brings up the named WireGuard interface by running `wg-quick up <name>`.
// On failure the combined stdout/stderr output is included in the error.
func Up(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wg-quick", "up", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wg-quick up %s: %w: %s", name, err, strings.TrimSpace(string(output)))
	}
	return nil
}

// Down brings down the named WireGuard interface by running `wg-quick down <name>`.
// On failure the combined stdout/stderr output is included in the error.
func Down(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wg-quick", "down", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wg-quick down %s: %w: %s", name, err, strings.TrimSpace(string(output)))
	}
	return nil
}

// IsUp reports whether the named WireGuard interface is currently active.
// It runs `wg show <name>` and returns true if the command exits 0,
// false if the command exits with a non-zero status (interface not found/down),
// and a non-nil error only for unexpected failures (e.g. wg binary not found).
func IsUp(name string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wg", "show", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Distinguish between "interface not found" (exit code 1) and
		// unexpected errors (binary missing, timeout, etc.).
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Non-zero exit from wg means the interface is not up.
			_ = exitErr
			_ = output
			return false, nil
		}
		return false, fmt.Errorf("wg show %s: %w: %s", name, err, strings.TrimSpace(string(output)))
	}
	return true, nil
}

// Toggle flips the state of the named WireGuard interface: if it is currently
// up it is brought down, and vice versa. It returns the new state (true = up).
func Toggle(name string) (nowUp bool, err error) {
	up, err := IsUp(name)
	if err != nil {
		return false, fmt.Errorf("checking interface state: %w", err)
	}

	if up {
		if err := Down(name); err != nil {
			return false, err
		}
		return false, nil
	}

	if err := Up(name); err != nil {
		return false, err
	}
	return true, nil
}

// ListInterfaces returns the names of all active WireGuard interfaces by
// running `wg show interfaces` and splitting the output on whitespace.
// An empty slice is returned when no interfaces are active.
func ListInterfaces() ([]string, error) {
	out, err := runWgCmd("show", "interfaces")
	if err != nil {
		return nil, fmt.Errorf("listing interfaces: %w", err)
	}

	if out == "" {
		return []string{}, nil
	}
	return strings.Fields(out), nil
}
