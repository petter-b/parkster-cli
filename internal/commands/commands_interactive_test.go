//go:build interactive

package commands

import (
	"strings"
	"testing"

	"github.com/petter-b/parkster-cli/internal/auth"
)

// These tests exercise the real OS keychain. They only run with:
//
//	go test -tags interactive ./internal/commands/
//
// They are excluded from normal `go test ./...` runs to avoid
// keychain prompts in CI or on developer machines.

func TestInteractive_AuthLoginLogoutCycle(t *testing.T) {
	// Ensure real auth functions are used (not test swaps)
	origGet := getCredentials
	origSave := saveCredentials
	origDelete := deleteCredentials
	getCredentials = auth.GetCredentials
	saveCredentials = auth.SaveCredentials
	deleteCredentials = auth.DeleteCredentials
	t.Cleanup(func() {
		getCredentials = origGet
		saveCredentials = origSave
		deleteCredentials = origDelete
	})

	// Save test credentials
	if err := saveCredentials("interactive-test@example.com", "test-password"); err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}
	t.Cleanup(func() {
		_ = deleteCredentials()
	})

	// Verify auth status shows authenticated
	_, stderr, err := executeCommand("auth", "status")
	if err != nil {
		t.Fatalf("auth status failed: %v", err)
	}
	if !strings.Contains(stderr, "interactive-test@example.com") {
		t.Errorf("auth status should show username, got: %q", stderr)
	}
	if !strings.Contains(stderr, "keyring") {
		t.Errorf("auth status should show source 'keyring', got: %q", stderr)
	}

	// Delete credentials
	if err := deleteCredentials(); err != nil {
		t.Fatalf("DeleteCredentials failed: %v", err)
	}

	// Verify auth status shows not authenticated
	_, stderr, err = executeCommand("auth", "status")
	if err != nil {
		t.Fatalf("auth status should not error, got: %v", err)
	}
	if !strings.Contains(stderr, "Not authenticated") {
		t.Errorf("after logout, should show 'Not authenticated', got: %q", stderr)
	}
}
