package auth

import (
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
