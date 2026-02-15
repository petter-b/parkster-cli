package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/99designs/keyring"
)

// --- GetCredentials tests (combined username+password) ---

func TestGetCredentials_FromEnvVars(t *testing.T) {
	t.Setenv("PARKSTER_USERNAME", "envuser")
	t.Setenv("PARKSTER_PASSWORD", "envpass")

	username, password, err := GetCredentials()
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

func TestGetCredentials_KeyringOverridesEnv(t *testing.T) {
	t.Setenv("PARKSTER_USERNAME", "env-user")
	t.Setenv("PARKSTER_PASSWORD", "env-pass")

	ring := newMockKeyring()
	creds := credentials{Username: "keyring-user", Password: "keyring-pass"}
	data, _ := json.Marshal(creds)
	_ = ring.Set(keyring.Item{Key: credentialKey("credentials"), Data: data})

	username, password, err := getCredentialsWithKeyring(ring)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "keyring-user" {
		t.Errorf("expected 'keyring-user', got %q", username)
	}
	if password != "keyring-pass" {
		t.Errorf("expected 'keyring-pass', got %q", password)
	}
}

func TestGetCredentials_MissingUsername(t *testing.T) {
	t.Setenv("PARKSTER_USERNAME", "")
	t.Setenv("PARKSTER_PASSWORD", "somepass")

	if runtime.GOOS == "darwin" {
		t.Skip("skipping: macOS Keychain may block")
	}

	_, _, err := GetCredentials()
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

	_, _, err := GetCredentials()
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

// --- mockKeyringStore ---

type mockKeyringStore struct {
	items map[string]keyring.Item
	err   error // if set, all operations return this error
}

func newMockKeyring() *mockKeyringStore {
	return &mockKeyringStore{items: make(map[string]keyring.Item)}
}

func (m *mockKeyringStore) Get(key string) (keyring.Item, error) {
	if m.err != nil {
		return keyring.Item{}, m.err
	}
	item, ok := m.items[key]
	if !ok {
		return keyring.Item{}, keyring.ErrKeyNotFound
	}
	return item, nil
}

func (m *mockKeyringStore) Set(item keyring.Item) error {
	if m.err != nil {
		return m.err
	}
	m.items[item.Key] = item
	return nil
}

func (m *mockKeyringStore) Remove(key string) error {
	if m.err != nil {
		return m.err
	}
	delete(m.items, key)
	return nil
}

// --- SaveCredentials tests ---

func TestSaveCredentials_StoresJSONBlob(t *testing.T) {
	ring := newMockKeyring()
	err := saveCredentialsTo(ring, "user@test.com", "secret123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	item, err := ring.Get(credentialKey("credentials"))
	if err != nil {
		t.Fatalf("expected item in keyring: %v", err)
	}

	var creds credentials
	if err := json.Unmarshal(item.Data, &creds); err != nil {
		t.Fatalf("stored data is not valid JSON: %v", err)
	}
	if creds.Username != "user@test.com" {
		t.Errorf("expected username 'user@test.com', got %q", creds.Username)
	}
	if creds.Password != "secret123" {
		t.Errorf("expected password 'secret123', got %q", creds.Password)
	}
}

func TestSaveCredentials_KeyringError(t *testing.T) {
	ring := &mockKeyringStore{items: make(map[string]keyring.Item), err: fmt.Errorf("keyring locked")}
	err := saveCredentialsTo(ring, "user", "pass")
	if err == nil {
		t.Fatal("expected error when keyring fails")
	}
}

// --- DeleteCredentials tests ---

func TestDeleteCredentials_RemovesKey(t *testing.T) {
	ring := newMockKeyring()
	// Pre-populate
	_ = ring.Set(keyring.Item{Key: credentialKey("credentials"), Data: []byte(`{}`)})
	_ = ring.Set(keyring.Item{Key: credentialKey("username"), Data: []byte(`old`)})
	_ = ring.Set(keyring.Item{Key: credentialKey("password"), Data: []byte(`old`)})

	err := deleteCredentialsFrom(ring)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := ring.Get(credentialKey("credentials")); err == nil {
		t.Error("expected credentials to be removed")
	}
	if _, err := ring.Get(credentialKey("username")); err == nil {
		t.Error("expected legacy username to be removed")
	}
	if _, err := ring.Get(credentialKey("password")); err == nil {
		t.Error("expected legacy password to be removed")
	}
}

func TestDeleteCredentials_EmptyKeyring_ReturnsError(t *testing.T) {
	ring := newMockKeyring() // empty, nothing stored

	err := deleteCredentialsFrom(ring)
	if err == nil {
		t.Fatal("expected error when no credentials to delete")
	}
	if !errors.Is(err, ErrNoCredentials) {
		t.Errorf("expected ErrNoCredentials, got: %v", err)
	}
}

// --- GetCredentials keyring path tests ---

func TestGetCredentials_FallsBackToKeyring(t *testing.T) {
	t.Setenv("PARKSTER_USERNAME", "")
	t.Setenv("PARKSTER_PASSWORD", "")

	ring := newMockKeyring()
	creds := credentials{Username: "keyring-user", Password: "keyring-pass"}
	data, _ := json.Marshal(creds)
	_ = ring.Set(keyring.Item{Key: credentialKey("credentials"), Data: data})

	username, password, err := getCredentialsWithKeyring(ring)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "keyring-user" {
		t.Errorf("expected 'keyring-user', got %q", username)
	}
	if password != "keyring-pass" {
		t.Errorf("expected 'keyring-pass', got %q", password)
	}
}

func TestGetCredentials_CorruptedKeyringJSON_FallsThrough(t *testing.T) {
	t.Setenv("PARKSTER_USERNAME", "")
	t.Setenv("PARKSTER_PASSWORD", "")

	ring := newMockKeyring()
	_ = ring.Set(keyring.Item{Key: credentialKey("credentials"), Data: []byte(`not-json`)})

	// Corrupted keyring falls through to env vars; both empty → error
	_, _, err := getCredentialsWithKeyring(ring)
	if err == nil {
		t.Fatal("expected error when keyring corrupted and no env vars")
	}
	if !strings.Contains(err.Error(), "no credentials found") {
		t.Errorf("expected 'no credentials found' in error, got: %v", err)
	}
}

func TestGetCredentials_CorruptedKeyringJSON_EnvFallback(t *testing.T) {
	t.Setenv("PARKSTER_USERNAME", "env-user")
	t.Setenv("PARKSTER_PASSWORD", "env-pass")

	ring := newMockKeyring()
	_ = ring.Set(keyring.Item{Key: credentialKey("credentials"), Data: []byte(`not-json`)})

	// Corrupted keyring falls through to env vars
	username, password, err := getCredentialsWithKeyring(ring)
	if err != nil {
		t.Fatalf("expected env fallback to succeed, got: %v", err)
	}
	if username != "env-user" {
		t.Errorf("expected 'env-user', got %q", username)
	}
	if password != "env-pass" {
		t.Errorf("expected 'env-pass', got %q", password)
	}
}

func TestGetCredentials_KeyringNotFound(t *testing.T) {
	t.Setenv("PARKSTER_USERNAME", "")
	t.Setenv("PARKSTER_PASSWORD", "")

	ring := newMockKeyring() // empty

	_, _, err := getCredentialsWithKeyring(ring)
	if err == nil {
		t.Fatal("expected error when no credentials found")
	}
}
