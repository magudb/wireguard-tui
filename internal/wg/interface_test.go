package wg

import (
	"testing"
)

func TestListInterfacesDoesNotError(t *testing.T) {
	// ListInterfaces should not error even if no interfaces are up.
	_, err := ListInterfaces()
	if err != nil {
		t.Fatalf("ListInterfaces() returned error: %v", err)
	}
}

func TestIsUpReturnsFalseForNonexistent(t *testing.T) {
	up, err := IsUp("wg_nonexistent_test_12345")
	if err != nil {
		t.Fatalf("IsUp() returned unexpected error: %v", err)
	}
	if up {
		t.Error("IsUp() returned true for nonexistent interface, want false")
	}
}
