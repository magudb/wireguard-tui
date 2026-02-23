package teleport

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// LoadOrCreateUUID loads an existing client UUID or creates a new one.
// UUIDs are stored in dir/<name>_uuid.
func LoadOrCreateUUID(dir, name string) (string, error) {
	path := filepath.Join(dir, name+"_uuid")

	data, err := os.ReadFile(path)
	if err == nil {
		return strings.TrimSpace(string(data)), nil
	}

	if !os.IsNotExist(err) {
		return "", fmt.Errorf("reading UUID file: %w", err)
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("creating credential dir: %w", err)
	}

	id := strings.ToUpper(uuid.New().String())
	if err := os.WriteFile(path, []byte(id), 0600); err != nil {
		return "", fmt.Errorf("writing UUID file: %w", err)
	}

	return id, nil
}

// SaveToken saves a device token for the given profile name.
func SaveToken(dir, name, token string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating credential dir: %w", err)
	}
	path := filepath.Join(dir, name+"_token")
	if err := os.WriteFile(path, []byte(token), 0600); err != nil {
		return fmt.Errorf("writing token file: %w", err)
	}
	return nil
}

// LoadToken loads a saved device token for the given profile name.
func LoadToken(dir, name string) (string, error) {
	path := filepath.Join(dir, name+"_token")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading token file: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// HasToken returns true if a device token exists for the given profile name.
func HasToken(dir, name string) bool {
	path := filepath.Join(dir, name+"_token")
	_, err := os.Stat(path)
	return err == nil
}
