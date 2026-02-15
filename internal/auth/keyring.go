package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/99designs/keyring"
)

const serviceName = "parkster"

// ErrNoCredentials indicates no credentials were found to delete.
var ErrNoCredentials = fmt.Errorf("no credentials stored")

// CredentialSource indicates where credentials were found.
type CredentialSource string

const (
	SourceKeyring     CredentialSource = "keyring"
	SourceEnvironment CredentialSource = "environment"
)

// credentials holds username and password as a single JSON blob for keychain storage.
type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// KeyringStore abstracts keyring operations for testability.
type KeyringStore interface {
	Get(key string) (keyring.Item, error)
	Set(item keyring.Item) error
	Remove(key string) error
}

// OpenKeyring opens the OS keychain
func OpenKeyring() (keyring.Keyring, error) {
	// Determine file backend path for headless Linux
	fileDir := filepath.Join(configDir(), "credentials")

	ring, err := keyring.Open(keyring.Config{
		ServiceName: serviceName,

		// macOS - trust this app so reads don't prompt
		KeychainTrustApplication: true,

		// Linux - prefer secret service, fall back to encrypted file
		FileDir: fileDir,
		FilePasswordFunc: func(prompt string) (string, error) {
			fmt.Fprintf(os.Stderr, "%s: ", prompt)
			var password string
			_, err := fmt.Scanln(&password)
			return password, err
		},

		// Windows - uses Credential Manager automatically
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open keyring: %w", err)
	}

	return ring, nil
}

// credentialKey returns the keyring key for a service
func credentialKey(service string) string {
	return "apikey:" + service
}

// configDir returns the XDG config directory
func configDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, serviceName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return filepath.Join(home, ".config", serviceName)
}

// GetCredentials retrieves credentials.
// Priority: keyring > env vars (PARKSTER_USERNAME/PARKSTER_PASSWORD).
func GetCredentials() (username, password string, source CredentialSource, err error) {
	// 1. Try keyring first
	ring, kerr := OpenKeyring()
	if kerr == nil {
		username, password, err = getCredentialsFromKeyring(ring)
		if err == nil {
			return username, password, SourceKeyring, nil
		}
	}

	// 2. Fall back to env vars
	username = os.Getenv("PARKSTER_USERNAME")
	password = os.Getenv("PARKSTER_PASSWORD")
	if username != "" && password != "" {
		return username, password, SourceEnvironment, nil
	}

	return "", "", "", fmt.Errorf("no credentials found (use PARKSTER_USERNAME/PARKSTER_PASSWORD env vars, or 'parkster auth login')")
}

// getCredentialsWithKeyring is like GetCredentials but accepts a KeyringStore for testing.
func getCredentialsWithKeyring(ring KeyringStore) (username, password string, source CredentialSource, err error) {
	// 1. Try keyring first
	username, password, err = getCredentialsFromKeyring(ring)
	if err == nil {
		return username, password, SourceKeyring, nil
	}

	// 2. Fall back to env vars
	username = os.Getenv("PARKSTER_USERNAME")
	password = os.Getenv("PARKSTER_PASSWORD")
	if username != "" && password != "" {
		return username, password, SourceEnvironment, nil
	}

	return "", "", "", fmt.Errorf("no credentials found (use PARKSTER_USERNAME/PARKSTER_PASSWORD env vars, or 'parkster auth login')")
}

// getCredentialsFromKeyring reads credentials from the keyring.
func getCredentialsFromKeyring(ring KeyringStore) (string, string, error) {
	item, err := ring.Get(credentialKey("credentials"))
	if err != nil {
		return "", "", fmt.Errorf("no credentials in keyring")
	}
	var creds credentials
	if err := json.Unmarshal(item.Data, &creds); err != nil {
		return "", "", fmt.Errorf("corrupted credentials: run 'parkster auth login' to re-store")
	}
	if creds.Username == "" || creds.Password == "" {
		return "", "", fmt.Errorf("incomplete credentials in keyring")
	}
	return creds.Username, creds.Password, nil
}

// SaveCredentials stores username and password as a single JSON item in keyring
func SaveCredentials(username, password string) error {
	ring, err := OpenKeyring()
	if err != nil {
		return fmt.Errorf("failed to open keyring: %w", err)
	}
	return saveCredentialsTo(ring, username, password)
}

// saveCredentialsTo stores credentials using the provided KeyringStore.
func saveCredentialsTo(ring KeyringStore, username, password string) error {
	creds := credentials{Username: username, Password: password}
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to encode credentials: %w", err)
	}
	return ring.Set(keyring.Item{
		Key:         credentialKey("credentials"),
		Data:        data,
		Label:       "Parkster Credentials",
		Description: "Parkster CLI credential",
	})
}

// DeleteCredentials removes credentials from keyring
func DeleteCredentials() error {
	ring, err := OpenKeyring()
	if err != nil {
		return fmt.Errorf("failed to open keyring: %w", err)
	}
	return deleteCredentialsFrom(ring)
}

// deleteCredentialsFrom removes credentials using the provided KeyringStore.
func deleteCredentialsFrom(ring KeyringStore) error {
	err := ring.Remove(credentialKey("credentials"))
	if err != nil {
		return ErrNoCredentials
	}
	// Clean up legacy separate items (ignore errors)
	_ = ring.Remove(credentialKey("username"))
	_ = ring.Remove(credentialKey("password"))
	return nil
}
