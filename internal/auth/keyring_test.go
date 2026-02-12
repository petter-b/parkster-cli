package auth

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

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

// --- credentials JSON round-trip test ---

func TestCredentialsJSON_RoundTrip(t *testing.T) {
	creds := credentials{Username: "user@test.com", Password: "secret123"}
	data, err := json.Marshal(creds)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var got credentials
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got.Username != "user@test.com" || got.Password != "secret123" {
		t.Errorf("roundtrip mismatch: got %+v", got)
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
