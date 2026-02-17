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

// openKeyring opens the OS keyring. Replaced in tests to simulate unavailable keyring.
var openKeyring = OpenKeyring

// CredentialSource indicates where credentials were found.
type CredentialSource string

const (
	SourceKeyring     CredentialSource = "keyring"
	SourceEnvironment CredentialSource = "environment"
	SourceFile        CredentialSource = "file"
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
	ring, err := keyring.Open(keyring.Config{
		ServiceName: serviceName,

		// macOS - trust this app so reads don't prompt
		KeychainTrustApplication: true,

		// No FileDir/FilePasswordFunc — we handle file storage ourselves
		// to avoid passphrase prompts on headless Linux.
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

// CredentialsFilePath returns the path to the plaintext credentials file.
func CredentialsFilePath() string {
	return filepath.Join(configDir(), "credentials.json")
}

// writeFileCredentials stores credentials as plaintext JSON in the config directory.
func writeFileCredentials(username, password string) error {
	creds := credentials{Username: username, Password: password}
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to encode credentials: %w", err)
	}
	path := CredentialsFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}
	return nil
}

// readFileCredentials reads credentials from the plaintext JSON file.
func readFileCredentials() (username, password string, err error) {
	data, err := os.ReadFile(CredentialsFilePath())
	if err != nil {
		return "", "", fmt.Errorf("no credentials file found")
	}
	var creds credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return "", "", fmt.Errorf("corrupted credentials file: run 'parkster auth login' to re-store")
	}
	if creds.Username == "" || creds.Password == "" {
		return "", "", fmt.Errorf("incomplete credentials in file")
	}
	return creds.Username, creds.Password, nil
}

// deleteFileCredentials removes the plaintext credentials file.
func deleteFileCredentials() error {
	err := os.Remove(CredentialsFilePath())
	if err != nil {
		return fmt.Errorf("no credentials file to remove")
	}
	return nil
}

// credentialOpts configures credential retrieval behavior.
type credentialOpts struct {
	ring   KeyringStore // nil = use default OS keyring
	caller string       // non-empty = update keychain description after retrieval
}

// errNoCredentials is the shared error for all credential retrieval failures.
var errNoCredentials = fmt.Errorf("no credentials found (use PARKSTER_USERNAME/PARKSTER_PASSWORD env vars, or 'parkster auth login')")

// getCredentialsInternal is the unified credential retrieval implementation.
// Priority: keyring > file > env vars.
// When opts.ring is nil, it opens the default OS keyring (and skips on error).
// When opts.ring is non-nil (tests), it uses the injected store and skips file fallback.
func getCredentialsInternal(opts credentialOpts) (username, password string, source CredentialSource, err error) {
	if opts.ring != nil {
		// Injected keyring (test path): skip file fallback
		username, password, err = getCredentialsFromKeyring(opts.ring)
		if err == nil {
			if opts.caller != "" {
				updateKeychainDescription(opts.ring, username, password, opts.caller)
			}
			return username, password, SourceKeyring, nil
		}
	} else {
		// Production path: try OS keyring, skip on open error
		ring, kerr := openKeyring()
		if kerr == nil {
			username, password, err = getCredentialsFromKeyring(ring)
			if err == nil {
				if opts.caller != "" {
					updateKeychainDescription(ring, username, password, opts.caller)
				}
				return username, password, SourceKeyring, nil
			}
		}

		// Try plaintext file (only in production path)
		username, password, err = readFileCredentials()
		if err == nil {
			return username, password, SourceFile, nil
		}
	}

	// Fall back to env vars
	username = os.Getenv("PARKSTER_USERNAME")
	password = os.Getenv("PARKSTER_PASSWORD")
	if username != "" && password != "" {
		return username, password, SourceEnvironment, nil
	}

	return "", "", "", errNoCredentials
}

// GetCredentials retrieves credentials.
// Priority: OS keyring > plaintext file > env vars (PARKSTER_USERNAME/PARKSTER_PASSWORD).
func GetCredentials() (username, password string, source CredentialSource, err error) {
	return getCredentialsInternal(credentialOpts{})
}

// GetCredentialsWithCaller retrieves credentials and updates the keychain
// item description with the caller name (for agent identification).
func GetCredentialsWithCaller(callerName string) (username, password string, source CredentialSource, err error) {
	return getCredentialsInternal(credentialOpts{caller: callerName})
}

// getCredentialsWithKeyring is like GetCredentials but accepts a KeyringStore for testing.
func getCredentialsWithKeyring(ring KeyringStore) (username, password string, source CredentialSource, err error) {
	return getCredentialsInternal(credentialOpts{ring: ring})
}

// getCredentialsWithCallerKeyring is like GetCredentialsWithCaller but accepts a KeyringStore for testing.
func getCredentialsWithCallerKeyring(ring KeyringStore, callerName string) (username, password string, source CredentialSource, err error) {
	return getCredentialsInternal(credentialOpts{ring: ring, caller: callerName})
}

func updateKeychainDescription(ring KeyringStore, username, password, callerName string) {
	creds := credentials{Username: username, Password: password}
	data, _ := json.Marshal(creds)
	description := fmt.Sprintf("Parkster CLI credential (via %s)", callerName)
	_ = ring.Set(keyring.Item{
		Key:         credentialKey("credentials"),
		Data:        data,
		Label:       "Parkster Credentials",
		Description: description,
	})
}

// getCredentialsFromKeyring reads credentials from the keyring.
func getCredentialsFromKeyring(ring KeyringStore) (username, password string, err error) {
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

// SaveCredentials stores credentials. Tries OS keyring first, falls back to plaintext file.
func SaveCredentials(username, password string) (CredentialSource, error) {
	ring, err := openKeyring()
	if err == nil {
		if err := saveCredentialsTo(ring, username, password); err == nil {
			return SourceKeyring, nil
		}
	}
	// Fall back to plaintext file
	if err := writeFileCredentials(username, password); err != nil {
		return "", fmt.Errorf("failed to store credentials: %w", err)
	}
	return SourceFile, nil
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

// DeleteCredentials removes credentials from both keyring and file.
func DeleteCredentials() error {
	keyringDeleted := false
	ring, err := openKeyring()
	if err == nil {
		if err := deleteCredentialsFrom(ring); err == nil {
			keyringDeleted = true
		}
	}
	fileDeleted := deleteFileCredentials() == nil
	if !keyringDeleted && !fileDeleted {
		return ErrNoCredentials
	}
	return nil
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
