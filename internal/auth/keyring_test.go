package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/99designs/keyring"
)

// --- GetCredentials tests (combined username+password) ---

func TestGetCredentials_FromEnvVars(t *testing.T) {
	t.Setenv("PARKSTER_USERNAME", "envuser")
	t.Setenv("PARKSTER_PASSWORD", "envpass")

	ring := newMockKeyring()
	username, password, _, err := getCredentialsWithKeyring(ring)
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

	username, password, _, err := getCredentialsWithKeyring(ring)
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

	ring := newMockKeyring()
	_, _, _, err := getCredentialsWithKeyring(ring)
	if err == nil {
		t.Error("expected error when username not available")
	}
}

func TestGetCredentials_MissingPassword(t *testing.T) {
	t.Setenv("PARKSTER_USERNAME", "user")
	t.Setenv("PARKSTER_PASSWORD", "")

	ring := newMockKeyring()
	_, _, _, err := getCredentialsWithKeyring(ring)
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

func TestSourceFile_Value(t *testing.T) {
	if SourceFile != "file" {
		t.Errorf("expected SourceFile to be \"file\", got %q", SourceFile)
	}
}

func TestCredentialsFilePath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/test-xdg")
	path := CredentialsFilePath()
	expected := "/tmp/test-xdg/parkster/credentials.json"
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
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
	if _, ok := m.items[key]; !ok {
		return keyring.ErrKeyNotFound
	}
	delete(m.items, key)
	return nil
}

func (m *mockKeyringStore) GetMetadata(key string) (keyring.Metadata, error) {
	if m.err != nil {
		return keyring.Metadata{}, m.err
	}
	if _, ok := m.items[key]; !ok {
		return keyring.Metadata{}, keyring.ErrKeyNotFound
	}
	return keyring.Metadata{}, nil
}

func (m *mockKeyringStore) Keys() ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	keys := make([]string, 0, len(m.items))
	for k := range m.items {
		keys = append(keys, k)
	}
	return keys, nil
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

	username, password, _, err := getCredentialsWithKeyring(ring)
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
	_, _, _, err := getCredentialsWithKeyring(ring)
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
	username, password, _, err := getCredentialsWithKeyring(ring)
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

func TestGetCredentials_ReturnsKeyringSource(t *testing.T) {
	t.Setenv("PARKSTER_USERNAME", "")
	t.Setenv("PARKSTER_PASSWORD", "")

	ring := newMockKeyring()
	creds := credentials{Username: "kr-user", Password: "kr-pass"}
	data, _ := json.Marshal(creds)
	_ = ring.Set(keyring.Item{Key: credentialKey("credentials"), Data: data})

	_, _, source, err := getCredentialsWithKeyring(ring)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != SourceKeyring {
		t.Errorf("expected source %q, got %q", SourceKeyring, source)
	}
}

func TestGetCredentials_ReturnsEnvSource(t *testing.T) {
	t.Setenv("PARKSTER_USERNAME", "envuser")
	t.Setenv("PARKSTER_PASSWORD", "envpass")

	ring := newMockKeyring() // empty keyring

	_, _, source, err := getCredentialsWithKeyring(ring)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != SourceEnvironment {
		t.Errorf("expected source %q, got %q", SourceEnvironment, source)
	}
}

func TestGetCredentials_KeyringNotFound(t *testing.T) {
	t.Setenv("PARKSTER_USERNAME", "")
	t.Setenv("PARKSTER_PASSWORD", "")

	ring := newMockKeyring() // empty

	_, _, _, err := getCredentialsWithKeyring(ring)
	if err == nil {
		t.Fatal("expected error when no credentials found")
	}
}

// --- GetCredentialsWithCaller tests ---

func TestGetCredentialsWithCaller_UpdatesDescription(t *testing.T) {
	t.Setenv("PARKSTER_USERNAME", "")
	t.Setenv("PARKSTER_PASSWORD", "")

	ring := newMockKeyring()
	creds := credentials{Username: "kr-user", Password: "kr-pass"}
	data, _ := json.Marshal(creds)
	_ = ring.Set(keyring.Item{Key: credentialKey("credentials"), Data: data})

	username, password, source, err := getCredentialsWithCallerKeyring(ring, "claude")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "kr-user" || password != "kr-pass" {
		t.Errorf("expected kr-user/kr-pass, got %s/%s", username, password)
	}
	if source != SourceKeyring {
		t.Errorf("expected source %q, got %q", SourceKeyring, source)
	}

	// Verify the item description was updated
	item, err := ring.Get(credentialKey("credentials"))
	if err != nil {
		t.Fatalf("expected item in keyring: %v", err)
	}
	expectedDesc := "Parkster CLI credential (via claude)"
	if item.Description != expectedDesc {
		t.Errorf("expected description %q, got %q", expectedDesc, item.Description)
	}
}

func TestGetCredentialsWithCaller_EmptyCallerName_NoUpdate(t *testing.T) {
	t.Setenv("PARKSTER_USERNAME", "")
	t.Setenv("PARKSTER_PASSWORD", "")

	ring := newMockKeyring()
	creds := credentials{Username: "kr-user", Password: "kr-pass"}
	data, _ := json.Marshal(creds)
	_ = ring.Set(keyring.Item{
		Key:         credentialKey("credentials"),
		Data:        data,
		Description: "original description",
	})

	_, _, _, err := getCredentialsWithCallerKeyring(ring, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Description should remain unchanged when callerName is empty
	item, _ := ring.Get(credentialKey("credentials"))
	if item.Description != "original description" {
		t.Errorf("expected description to remain 'original description', got %q", item.Description)
	}
}

func TestGetCredentialsWithCaller_EnvFallback_NoKeychainUpdate(t *testing.T) {
	t.Setenv("PARKSTER_USERNAME", "env-user")
	t.Setenv("PARKSTER_PASSWORD", "env-pass")

	ring := newMockKeyring() // empty keyring

	username, password, source, err := getCredentialsWithCallerKeyring(ring, "claude")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "env-user" || password != "env-pass" {
		t.Errorf("expected env-user/env-pass, got %s/%s", username, password)
	}
	if source != SourceEnvironment {
		t.Errorf("expected source %q, got %q", SourceEnvironment, source)
	}

	// No keychain item should exist (env vars used, not keyring)
	_, err = ring.Get(credentialKey("credentials"))
	if err == nil {
		t.Error("expected no credentials in keyring when using env fallback")
	}
}

func TestGetCredentialsWithCaller_NoCredentials_ReturnsError(t *testing.T) {
	t.Setenv("PARKSTER_USERNAME", "")
	t.Setenv("PARKSTER_PASSWORD", "")

	ring := newMockKeyring() // empty

	_, _, _, err := getCredentialsWithCallerKeyring(ring, "claude")
	if err == nil {
		t.Fatal("expected error when no credentials found")
	}
}

func TestUpdateKeychainDescription(t *testing.T) {
	ring := newMockKeyring()
	creds := credentials{Username: "user", Password: "pass"}
	data, _ := json.Marshal(creds)
	_ = ring.Set(keyring.Item{Key: credentialKey("credentials"), Data: data})

	updateKeychainDescription(ring, "user", "pass", "openclaw-gateway")

	item, _ := ring.Get(credentialKey("credentials"))
	expectedDesc := "Parkster CLI credential (via openclaw-gateway)"
	if item.Description != expectedDesc {
		t.Errorf("expected description %q, got %q", expectedDesc, item.Description)
	}
	if item.Label != "Parkster Credentials" {
		t.Errorf("expected label 'Parkster Credentials', got %q", item.Label)
	}

	// Verify credentials data is preserved
	var got credentials
	_ = json.Unmarshal(item.Data, &got)
	if got.Username != "user" || got.Password != "pass" {
		t.Errorf("credentials should be preserved, got %+v", got)
	}
}

// --- File credential tests ---

func TestWriteAndReadFileCredentials(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	err := writeFileCredentials("user@test.com", "secret123")
	if err != nil {
		t.Fatalf("writeFileCredentials failed: %v", err)
	}

	username, password, err := readFileCredentials()
	if err != nil {
		t.Fatalf("readFileCredentials failed: %v", err)
	}
	if username != "user@test.com" {
		t.Errorf("expected username 'user@test.com', got %q", username)
	}
	if password != "secret123" {
		t.Errorf("expected password 'secret123', got %q", password)
	}
}

func TestWriteFileCredentials_Permissions(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	err := writeFileCredentials("user", "pass")
	if err != nil {
		t.Fatalf("writeFileCredentials failed: %v", err)
	}

	info, err := os.Stat(CredentialsFilePath())
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("expected permissions 0o600, got %04o", perm)
	}
}

func TestReadFileCredentials_MissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	_, _, err := readFileCredentials()
	if err == nil {
		t.Fatal("expected error when file doesn't exist")
	}
}

func TestReadFileCredentials_CorruptedJSON(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	path := filepath.Join(dir, "parkster", "credentials.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(path, []byte("not-json"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	_, _, err := readFileCredentials()
	if err == nil {
		t.Fatal("expected error for corrupted JSON")
	}
}

func TestReadFileCredentials_EmptyFields(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	path := filepath.Join(dir, "parkster", "credentials.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"username":"","password":""}`), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	_, _, err := readFileCredentials()
	if err == nil {
		t.Fatal("expected error for empty credentials")
	}
}

func TestDeleteFileCredentials_RemovesFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	err := writeFileCredentials("user", "pass")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	err = deleteFileCredentials()
	if err != nil {
		t.Fatalf("deleteFileCredentials failed: %v", err)
	}

	if _, err := os.Stat(CredentialsFilePath()); !os.IsNotExist(err) {
		t.Error("expected credentials file to be deleted")
	}
}

func TestDeleteFileCredentials_MissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	err := deleteFileCredentials()
	if err == nil {
		t.Fatal("expected error when file doesn't exist")
	}
}

func TestDeleteCredentials_RemovesFile_WhenKeyringUnavailable(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	// Make keyring unavailable
	orig := openKeyring
	openKeyring = func() (keyring.Keyring, error) {
		return nil, fmt.Errorf("no keyring available")
	}
	t.Cleanup(func() { openKeyring = orig })

	// Store credentials in file
	err := writeFileCredentials("user", "pass")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	err = DeleteCredentials()
	if err != nil {
		t.Fatalf("DeleteCredentials failed: %v", err)
	}

	// File should be gone
	if _, err := os.Stat(CredentialsFilePath()); !os.IsNotExist(err) {
		t.Error("expected credentials file to be deleted")
	}
}

func TestDeleteCredentials_NothingStored_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	// Make keyring unavailable, no file exists
	orig := openKeyring
	openKeyring = func() (keyring.Keyring, error) {
		return nil, fmt.Errorf("no keyring available")
	}
	t.Cleanup(func() { openKeyring = orig })

	err := DeleteCredentials()
	if !errors.Is(err, ErrNoCredentials) {
		t.Errorf("expected ErrNoCredentials, got: %v", err)
	}
}

// --- SaveCredentials file fallback tests ---

func TestSaveCredentials_FallsBackToFile_WhenKeyringUnavailable(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	// Make keyring unavailable
	orig := openKeyring
	openKeyring = func() (keyring.Keyring, error) {
		return nil, fmt.Errorf("no keyring available")
	}
	t.Cleanup(func() { openKeyring = orig })

	source, err := SaveCredentials("file-user", "file-pass")
	if err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}
	if source != SourceFile {
		t.Errorf("expected source %q, got %q", SourceFile, source)
	}

	// Verify we can read them back
	username, password, readErr := readFileCredentials()
	if readErr != nil {
		t.Fatalf("readFileCredentials failed: %v", readErr)
	}
	if username != "file-user" || password != "file-pass" {
		t.Errorf("expected file-user/file-pass, got %s/%s", username, password)
	}
}

func TestSaveCredentials_ReturnsKeyringSource_WhenKeyringAvailable(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	ring := newMockKeyring()

	// Make openKeyring return our mock
	orig := openKeyring
	openKeyring = func() (keyring.Keyring, error) {
		return ring, nil
	}
	t.Cleanup(func() { openKeyring = orig })

	source, err := SaveCredentials("kr-user", "kr-pass")
	if err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}
	if source != SourceKeyring {
		t.Errorf("expected source %q, got %q", SourceKeyring, source)
	}

	// Verify credentials were stored in keyring
	item, getErr := ring.Get(credentialKey("credentials"))
	if getErr != nil {
		t.Fatalf("expected item in keyring: %v", getErr)
	}
	var creds credentials
	if err := json.Unmarshal(item.Data, &creds); err != nil {
		t.Fatalf("stored data is not valid JSON: %v", err)
	}
	if creds.Username != "kr-user" || creds.Password != "kr-pass" {
		t.Errorf("expected kr-user/kr-pass, got %s/%s", creds.Username, creds.Password)
	}
}

// --- GetCredentials file fallback tests ---

func TestGetCredentials_FileFallback_WhenKeyringUnavailable(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("PARKSTER_USERNAME", "")
	t.Setenv("PARKSTER_PASSWORD", "")

	// Make keyring unavailable
	orig := openKeyring
	openKeyring = func() (keyring.Keyring, error) {
		return nil, fmt.Errorf("no keyring available")
	}
	t.Cleanup(func() { openKeyring = orig })

	// Store credentials in file
	err := writeFileCredentials("file-user", "file-pass")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	username, password, source, err := GetCredentials()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "file-user" {
		t.Errorf("expected 'file-user', got %q", username)
	}
	if password != "file-pass" {
		t.Errorf("expected 'file-pass', got %q", password)
	}
	if source != SourceFile {
		t.Errorf("expected source %q, got %q", SourceFile, source)
	}
}

func TestGetCredentials_FileTakesPriorityOverEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("PARKSTER_USERNAME", "env-user")
	t.Setenv("PARKSTER_PASSWORD", "env-pass")

	// Make keyring unavailable
	orig := openKeyring
	openKeyring = func() (keyring.Keyring, error) {
		return nil, fmt.Errorf("no keyring available")
	}
	t.Cleanup(func() { openKeyring = orig })

	// Store different credentials in file
	err := writeFileCredentials("file-user", "file-pass")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	username, _, source, err := GetCredentials()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "file-user" {
		t.Errorf("expected 'file-user' (file takes priority over env), got %q", username)
	}
	if source != SourceFile {
		t.Errorf("expected source %q, got %q", SourceFile, source)
	}
}

func TestGetCredentials_EnvFallback_WhenKeyringAndFileUnavailable(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("PARKSTER_USERNAME", "env-user")
	t.Setenv("PARKSTER_PASSWORD", "env-pass")

	// Make keyring unavailable, no file exists
	orig := openKeyring
	openKeyring = func() (keyring.Keyring, error) {
		return nil, fmt.Errorf("no keyring available")
	}
	t.Cleanup(func() { openKeyring = orig })

	username, _, source, err := GetCredentials()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "env-user" {
		t.Errorf("expected 'env-user', got %q", username)
	}
	if source != SourceEnvironment {
		t.Errorf("expected source %q, got %q", SourceEnvironment, source)
	}
}
