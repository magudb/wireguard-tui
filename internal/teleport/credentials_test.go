package teleport

import (
	"testing"
)

func TestLoadOrCreateUUID(t *testing.T) {
	dir := t.TempDir()
	uuid1, err := LoadOrCreateUUID(dir, "test")
	if err != nil {
		t.Fatalf("LoadOrCreateUUID() error: %v", err)
	}
	if len(uuid1) != 36 {
		t.Errorf("UUID length = %d, want 36", len(uuid1))
	}

	// Second call should return same UUID
	uuid2, err := LoadOrCreateUUID(dir, "test")
	if err != nil {
		t.Fatalf("LoadOrCreateUUID() second call error: %v", err)
	}
	if uuid1 != uuid2 {
		t.Errorf("UUID changed: %q != %q", uuid1, uuid2)
	}
}

func TestSaveAndLoadToken(t *testing.T) {
	dir := t.TempDir()
	token := "fTpHzN4q0DktZupldxN5KR0eEtsvwcJL26c1n7z7LVc="

	if err := SaveToken(dir, "myrouter", token); err != nil {
		t.Fatalf("SaveToken() error: %v", err)
	}

	loaded, err := LoadToken(dir, "myrouter")
	if err != nil {
		t.Fatalf("LoadToken() error: %v", err)
	}
	if loaded != token {
		t.Errorf("LoadToken() = %q, want %q", loaded, token)
	}
}

func TestHasToken(t *testing.T) {
	dir := t.TempDir()
	if HasToken(dir, "missing") {
		t.Error("HasToken() = true for missing token")
	}

	_ = SaveToken(dir, "exists", "tok123")
	if !HasToken(dir, "exists") {
		t.Error("HasToken() = false for existing token")
	}
}

func TestLoadTokenMissing(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadToken(dir, "missing")
	if err == nil {
		t.Error("LoadToken() should error for missing token")
	}
}
