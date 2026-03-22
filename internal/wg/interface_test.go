package wg

import (
	"os/exec"
	"testing"
)

func TestListInterfacesDoesNotError(t *testing.T) {
	if err := exec.Command("sudo", "-n", "true").Run(); err != nil {
		t.Skip("skipping: passwordless sudo not available")
	}
	// ListInterfaces should not error even if no interfaces are up.
	_, err := ListInterfaces()
	if err != nil {
		t.Fatalf("ListInterfaces() returned error: %v", err)
	}
}

func TestIsUpReturnsFalseForNonexistent(t *testing.T) {
	if err := exec.Command("sudo", "-n", "true").Run(); err != nil {
		t.Skip("skipping: passwordless sudo not available")
	}
	up, err := IsUp("wg_nonexistent_test_12345")
	if err != nil {
		t.Fatalf("IsUp() returned unexpected error: %v", err)
	}
	if up {
		t.Error("IsUp() returned true for nonexistent interface, want false")
	}
}
