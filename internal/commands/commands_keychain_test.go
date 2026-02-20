//go:build interactive

package commands

import (
	"strings"
	"testing"

	"github.com/petter-b/parkster-cli/internal/auth"
	"github.com/petter-b/parkster-cli/internal/parkster"
)

// TestKeychain_AuthLoginLogoutCycle exercises the real OS keychain
// through the full auth login → status → logout → status cycle.
// The API client is mocked since we only test keychain storage here.
func TestKeychain_AuthLoginLogoutCycle(t *testing.T) {
	// Isolate file storage so file-fallback never touches the real config dir.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

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

	// Mock the API client — this test only exercises keychain, not the API
	mock := &mockAPI{
		loginResp: &parkster.User{ID: 1, Email: "interactive-test@example.com"},
	}
	withMockClient(t, mock)

	// Save test credentials — skip if keychain access is denied or unavailable.
	// Register cleanup BEFORE skip checks so credentials are always removed,
	// even when the test skips (e.g. file-fallback in keyring-less environments).
	source, err := saveCredentials("interactive-test@example.com", "test-password")
	t.Cleanup(func() {
		_ = deleteCredentials()
	})
	if err != nil {
		t.Skipf("keychain write denied or unavailable: %v", err)
	}
	if source != auth.SourceKeyring {
		t.Skipf("keychain not available, credentials stored via %s", source)
	}

	// Verify keychain read also works — skip if read access is denied
	// (macOS may prompt for keychain access that can't be answered in tests)
	if _, _, _, err := getCredentials(); err != nil {
		t.Skipf("keychain read denied or unavailable: %v", err)
	}

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
	// authRequiredError returns ExitAuth
	if err != nil && ExitCode(err) != ExitAuth {
		t.Fatalf("expected ExitAuth or nil, got: %v (code=%d)", err, ExitCode(err))
	}
	if !strings.Contains(stderr, "Not authenticated") {
		t.Errorf("after logout, should show 'Not authenticated', got: %q", stderr)
	}
}
