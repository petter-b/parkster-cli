package auth

import (
	"os"
	"runtime"
	"testing"

	"github.com/spf13/cobra"
)

// skipIfKeychainBlocks skips tests that fall through to the OS keychain,
// which can block on macOS (SecItemCopyMatching prompts for access).
func skipIfKeychainBlocks(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "darwin" {
		t.Skip("skipping: macOS Keychain may block in test environment")
	}
}

// --- GetUsername tests ---

func TestGetUsername_FlagPriority(t *testing.T) {
	// Flag should take priority over env var
	os.Setenv("PARKSTER_USERNAME", "envuser")
	defer os.Unsetenv("PARKSTER_USERNAME")

	cmd := &cobra.Command{}
	cmd.Flags().String("username", "", "")
	cmd.Flags().Set("username", "flaguser")

	username, err := GetUsername(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "flaguser" {
		t.Errorf("expected flaguser, got %s", username)
	}
}

func TestGetUsername_EnvFallback(t *testing.T) {
	// Env var should be used when flag is not set
	os.Setenv("PARKSTER_USERNAME", "envuser")
	defer os.Unsetenv("PARKSTER_USERNAME")

	cmd := &cobra.Command{}
	cmd.Flags().String("username", "", "")

	username, err := GetUsername(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "envuser" {
		t.Errorf("expected envuser, got %s", username)
	}
}

func TestGetUsername_NilCmd_EnvFallback(t *testing.T) {
	// When cmd is nil, should use env var
	os.Setenv("PARKSTER_USERNAME", "envuser")
	defer os.Unsetenv("PARKSTER_USERNAME")

	username, err := GetUsername(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "envuser" {
		t.Errorf("expected envuser, got %s", username)
	}
}

func TestGetUsername_EmptyFlag_UsesEnv(t *testing.T) {
	// Empty string flag should fall through to env
	os.Setenv("PARKSTER_USERNAME", "envuser")
	defer os.Unsetenv("PARKSTER_USERNAME")

	cmd := &cobra.Command{}
	cmd.Flags().String("username", "", "")

	username, err := GetUsername(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "envuser" {
		t.Errorf("expected envuser, got %s", username)
	}
}

func TestGetUsername_NoCredentials(t *testing.T) {
	skipIfKeychainBlocks(t)
	os.Unsetenv("PARKSTER_USERNAME")

	cmd := &cobra.Command{}
	cmd.Flags().String("username", "", "")

	_, err := GetUsername(cmd)
	if err == nil {
		t.Fatal("expected error when no credentials configured")
	}
}

func TestGetUsername_EmptyEnvVar_FallsThrough(t *testing.T) {
	skipIfKeychainBlocks(t)
	os.Setenv("PARKSTER_USERNAME", "")
	defer os.Unsetenv("PARKSTER_USERNAME")

	cmd := &cobra.Command{}
	cmd.Flags().String("username", "", "")

	_, err := GetUsername(cmd)
	if err == nil {
		t.Error("Expected error when PARKSTER_USERNAME is empty string")
	}
}

// --- GetPassword tests ---

func TestGetPassword_FlagPriority(t *testing.T) {
	os.Setenv("PARKSTER_PASSWORD", "envpass")
	defer os.Unsetenv("PARKSTER_PASSWORD")

	cmd := &cobra.Command{}
	cmd.Flags().String("password", "", "")
	cmd.Flags().Set("password", "flagpass")

	password, err := GetPassword(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if password != "flagpass" {
		t.Errorf("expected flagpass, got %s", password)
	}
}

func TestGetPassword_EnvFallback(t *testing.T) {
	os.Setenv("PARKSTER_PASSWORD", "envpass")
	defer os.Unsetenv("PARKSTER_PASSWORD")

	cmd := &cobra.Command{}
	cmd.Flags().String("password", "", "")

	password, err := GetPassword(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if password != "envpass" {
		t.Errorf("expected envpass, got %s", password)
	}
}

func TestGetPassword_NilCmd_EnvFallback(t *testing.T) {
	os.Setenv("PARKSTER_PASSWORD", "envpass")
	defer os.Unsetenv("PARKSTER_PASSWORD")

	password, err := GetPassword(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if password != "envpass" {
		t.Errorf("expected envpass, got %s", password)
	}
}

func TestGetPassword_NoCredentials(t *testing.T) {
	skipIfKeychainBlocks(t)
	os.Unsetenv("PARKSTER_PASSWORD")

	cmd := &cobra.Command{}
	cmd.Flags().String("password", "", "")

	_, err := GetPassword(cmd)
	if err == nil {
		t.Fatal("expected error when no credentials configured")
	}
}

func TestGetPassword_EmptyEnvVar_FallsThrough(t *testing.T) {
	skipIfKeychainBlocks(t)
	os.Setenv("PARKSTER_PASSWORD", "")
	defer os.Unsetenv("PARKSTER_PASSWORD")

	cmd := &cobra.Command{}
	cmd.Flags().String("password", "", "")

	_, err := GetPassword(cmd)
	if err == nil {
		t.Error("Expected error when PARKSTER_PASSWORD is empty string")
	}
}
