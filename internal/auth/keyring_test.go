package auth

import (
	"runtime"
	"strings"
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
	t.Setenv("PARKSTER_USERNAME", "envuser")

	cmd := &cobra.Command{}
	cmd.Flags().String("username", "", "")
	_ = cmd.Flags().Set("username", "flaguser")

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
	t.Setenv("PARKSTER_USERNAME", "envuser")

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
	t.Setenv("PARKSTER_USERNAME", "envuser")

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
	t.Setenv("PARKSTER_USERNAME", "envuser")

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
	t.Setenv("PARKSTER_USERNAME", "")

	cmd := &cobra.Command{}
	cmd.Flags().String("username", "", "")

	_, err := GetUsername(cmd)
	if err == nil {
		t.Fatal("expected error when no credentials configured")
	}
}

func TestGetUsername_EmptyEnvVar_FallsThrough(t *testing.T) {
	skipIfKeychainBlocks(t)
	t.Setenv("PARKSTER_USERNAME", "")

	cmd := &cobra.Command{}
	cmd.Flags().String("username", "", "")

	_, err := GetUsername(cmd)
	if err == nil {
		t.Error("Expected error when PARKSTER_USERNAME is empty string")
	}
}

// --- GetPassword tests ---

func TestGetPassword_FlagPriority(t *testing.T) {
	t.Setenv("PARKSTER_PASSWORD", "envpass")

	cmd := &cobra.Command{}
	cmd.Flags().String("password", "", "")
	_ = cmd.Flags().Set("password", "flagpass")

	password, err := GetPassword(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if password != "flagpass" {
		t.Errorf("expected flagpass, got %s", password)
	}
}

func TestGetPassword_EnvFallback(t *testing.T) {
	t.Setenv("PARKSTER_PASSWORD", "envpass")

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
	t.Setenv("PARKSTER_PASSWORD", "envpass")

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
	t.Setenv("PARKSTER_PASSWORD", "")

	cmd := &cobra.Command{}
	cmd.Flags().String("password", "", "")

	_, err := GetPassword(cmd)
	if err == nil {
		t.Fatal("expected error when no credentials configured")
	}
}

func TestGetPassword_EmptyEnvVar_FallsThrough(t *testing.T) {
	skipIfKeychainBlocks(t)
	t.Setenv("PARKSTER_PASSWORD", "")

	cmd := &cobra.Command{}
	cmd.Flags().String("password", "", "")

	_, err := GetPassword(cmd)
	if err == nil {
		t.Error("Expected error when PARKSTER_PASSWORD is empty string")
	}
}

// --- GetCredentials tests (combined username+password) ---

func TestGetCredentials_FromEnvVars(t *testing.T) {
	t.Setenv("PARKSTER_USERNAME", "envuser")
	t.Setenv("PARKSTER_PASSWORD", "envpass")

	username, password, err := GetCredentials(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "envuser" {
		t.Errorf("expected username 'envuser', got %q", username)
	}
	if password != "envpass" {
		t.Errorf("expected password 'envpass', got %q", password)
	}
}

func TestGetCredentials_FlagPriority(t *testing.T) {
	t.Setenv("PARKSTER_USERNAME", "envuser")
	t.Setenv("PARKSTER_PASSWORD", "envpass")

	cmd := &cobra.Command{}
	cmd.Flags().String("username", "", "")
	cmd.Flags().String("password", "", "")
	_ = cmd.Flags().Set("username", "flaguser")
	_ = cmd.Flags().Set("password", "flagpass")

	username, password, err := GetCredentials(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "flaguser" {
		t.Errorf("expected username 'flaguser', got %q", username)
	}
	if password != "flagpass" {
		t.Errorf("expected password 'flagpass', got %q", password)
	}
}

func TestGetCredentials_MixedFlagAndEnv(t *testing.T) {
	// Flag for username, env for password
	t.Setenv("PARKSTER_USERNAME", "envuser")
	t.Setenv("PARKSTER_PASSWORD", "envpass")

	cmd := &cobra.Command{}
	cmd.Flags().String("username", "", "")
	cmd.Flags().String("password", "", "")
	_ = cmd.Flags().Set("username", "flaguser")

	username, password, err := GetCredentials(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "flaguser" {
		t.Errorf("expected username 'flaguser', got %q", username)
	}
	if password != "envpass" {
		t.Errorf("expected password 'envpass', got %q", password)
	}
}

func TestGetCredentials_MissingUsername(t *testing.T) {
	t.Setenv("PARKSTER_USERNAME", "")
	t.Setenv("PARKSTER_PASSWORD", "somepass")

	if runtime.GOOS == "darwin" {
		t.Skip("skipping: macOS Keychain may block")
	}

	_, _, err := GetCredentials(nil)
	if err == nil {
		t.Error("expected error when username not available")
	}
}

func TestGetCredentials_MissingPassword(t *testing.T) {
	t.Setenv("PARKSTER_USERNAME", "user")
	t.Setenv("PARKSTER_PASSWORD", "")

	if runtime.GOOS == "darwin" {
		t.Skip("skipping: macOS Keychain may block")
	}

	_, _, err := GetCredentials(nil)
	if err == nil {
		t.Error("expected error when password not available")
	}
}

// --- GetCredential tests ---

func TestGetCredential_EnvVar(t *testing.T) {
	t.Setenv("PARKSTER_MY_SERVICE_API_KEY", "secret123")

	val, err := GetCredential("my-service")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "secret123" {
		t.Errorf("expected secret123, got %s", val)
	}
}

func TestGetCredential_NoEnvVar(t *testing.T) {
	skipIfKeychainBlocks(t)
	t.Setenv("PARKSTER_MISSING_API_KEY", "")

	_, err := GetCredential("missing")
	if err == nil {
		t.Fatal("expected error when no credential available")
	}
}

// --- credentialKey tests ---

func TestCredentialKey(t *testing.T) {
	key := credentialKey("myservice")
	if key != "apikey:myservice" {
		t.Errorf("expected apikey:myservice, got %s", key)
	}
}

// --- configDir tests ---

func TestConfigDir_XDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-test")
	dir := configDir()
	if dir != "/tmp/xdg-test/parkster" {
		t.Errorf("expected /tmp/xdg-test/parkster, got %s", dir)
	}
}

func TestConfigDir_Default(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	dir := configDir()
	if !strings.Contains(dir, "parkster") {
		t.Errorf("expected path containing 'parkster', got %s", dir)
	}
	if !strings.HasSuffix(dir, ".config/parkster") {
		t.Errorf("expected path ending in .config/parkster, got %s", dir)
	}
}

// --- itemForCredential tests ---

func TestItemForCredential_Username(t *testing.T) {
	item := itemForCredential("username", "user@example.com")
	if item.Label != "Parkster Username" {
		t.Errorf("expected Label 'Parkster Username', got %q", item.Label)
	}
	if item.Description != "Parkster CLI credential" {
		t.Errorf("expected Description 'Parkster CLI credential', got %q", item.Description)
	}
	if item.Key != "apikey:username" {
		t.Errorf("expected Key 'apikey:username', got %q", item.Key)
	}
	if string(item.Data) != "user@example.com" {
		t.Errorf("expected Data 'user@example.com', got %q", string(item.Data))
	}
}

func TestItemForCredential_Password(t *testing.T) {
	item := itemForCredential("password", "secret123")
	if item.Label != "Parkster Password" {
		t.Errorf("expected Label 'Parkster Password', got %q", item.Label)
	}
	if item.Description != "Parkster CLI credential" {
		t.Errorf("expected Description 'Parkster CLI credential', got %q", item.Description)
	}
	if item.Key != "apikey:password" {
		t.Errorf("expected Key 'apikey:password', got %q", item.Key)
	}
	if string(item.Data) != "secret123" {
		t.Errorf("expected Data 'secret123', got %q", string(item.Data))
	}
}

func TestItemForCredential_GenericService(t *testing.T) {
	item := itemForCredential("my-api", "key123")
	if item.Label != "Parkster My-api" {
		t.Errorf("expected Label 'Parkster My-api', got %q", item.Label)
	}
	if item.Description != "Parkster CLI credential" {
		t.Errorf("expected Description 'Parkster CLI credential', got %q", item.Description)
	}
	if item.Key != "apikey:my-api" {
		t.Errorf("expected Key 'apikey:my-api', got %q", item.Key)
	}
	if string(item.Data) != "key123" {
		t.Errorf("expected Data 'key123', got %q", string(item.Data))
	}
}
