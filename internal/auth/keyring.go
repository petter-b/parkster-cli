package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/99designs/keyring"
	"github.com/spf13/cobra"
)

const serviceName = "parkster"

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

		// macOS - use default login keychain
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
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", serviceName)
}

// GetCredentials retrieves both username and password, opening the keyring at most once.
// Priority: CLI flags > env vars > keyring.
func GetCredentials(cmd *cobra.Command) (username, password string, err error) {
	// 1. Check CLI flags
	if cmd != nil {
		username, _ = cmd.Flags().GetString("username")
		password, _ = cmd.Flags().GetString("password")
	}

	// 2. Fill from env vars
	if username == "" {
		username = os.Getenv("PARKSTER_USERNAME")
	}
	if password == "" {
		password = os.Getenv("PARKSTER_PASSWORD")
	}

	// 3. If both set, done
	if username != "" && password != "" {
		return username, password, nil
	}

	// 4. Try keyring
	ring, kerr := OpenKeyring()
	if kerr != nil {
		return "", "", fmt.Errorf("no credentials found (use --username/--password flags, PARKSTER_USERNAME/PARKSTER_PASSWORD env vars, or 'parkster auth login')")
	}
	return getCredentialsFromKeyring(username, password, ring)
}

// getCredentialsWithKeyring is like GetCredentials but accepts a KeyringStore for testing.
func getCredentialsWithKeyring(cmd *cobra.Command, ring KeyringStore) (username, password string, err error) {
	if cmd != nil {
		username, _ = cmd.Flags().GetString("username")
		password, _ = cmd.Flags().GetString("password")
	}
	if username == "" {
		username = os.Getenv("PARKSTER_USERNAME")
	}
	if password == "" {
		password = os.Getenv("PARKSTER_PASSWORD")
	}
	if username != "" && password != "" {
		return username, password, nil
	}
	return getCredentialsFromKeyring(username, password, ring)
}

// getCredentialsFromKeyring fills missing credentials from the keyring.
func getCredentialsFromKeyring(username, password string, ring KeyringStore) (string, string, error) {
	item, err := ring.Get(credentialKey("credentials"))
	if err != nil {
		return "", "", fmt.Errorf("no credentials found (use --username/--password flags, PARKSTER_USERNAME/PARKSTER_PASSWORD env vars, or 'parkster auth login')")
	}
	var creds credentials
	if err := json.Unmarshal(item.Data, &creds); err != nil {
		return "", "", fmt.Errorf("corrupted credentials: run 'parkster auth login' to re-store")
	}
	if username == "" {
		username = creds.Username
	}
	if password == "" {
		password = creds.Password
	}
	return username, password, nil
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
	_ = ring.Remove(credentialKey("credentials"))
	// Clean up legacy separate items
	_ = ring.Remove(credentialKey("username"))
	_ = ring.Remove(credentialKey("password"))
	return nil
}
