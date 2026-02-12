package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/99designs/keyring"
	"github.com/spf13/cobra"
)

const serviceName = "parkster"

// Store wraps keyring operations
type Store struct {
	ring keyring.Keyring
}

// OpenKeyring opens the OS keychain
func OpenKeyring() (*Store, error) {
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

	return &Store{ring: ring}, nil
}

// Set stores a credential
func (s *Store) Set(service, secret string) error {
	return s.ring.Set(itemForCredential(service, secret))
}

// itemForCredential creates a keyring.Item with proper Label and Description
func itemForCredential(service, secret string) keyring.Item {
	// Capitalize first letter for human-readable labels
	label := "Parkster "
	switch service {
	case "username":
		label += "Username"
	case "password":
		label += "Password"
	default:
		// Capitalize first letter: "my-api" -> "My-api"
		if len(service) > 0 {
			label += strings.ToUpper(service[:1]) + service[1:]
		} else {
			label += service
		}
	}

	return keyring.Item{
		Key:         credentialKey(service),
		Data:        []byte(secret),
		Label:       label,
		Description: "Parkster CLI credential",
	}
}

// Get retrieves a credential
func (s *Store) Get(service string) (string, error) {
	item, err := s.ring.Get(credentialKey(service))
	if err != nil {
		return "", err
	}
	return string(item.Data), nil
}

// Delete removes a credential
func (s *Store) Delete(service string) error {
	return s.ring.Remove(credentialKey(service))
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

	// 3. If either still missing, try keyring (open once)
	if username == "" || password == "" {
		store, kerr := OpenKeyring()
		if kerr != nil {
			if username == "" {
				return "", "", fmt.Errorf("no credentials found (use --username/--password flags, PARKSTER_USERNAME/PARKSTER_PASSWORD env vars, or 'parkster auth login')")
			}
			return "", "", fmt.Errorf("no credentials found (use --password flag, PARKSTER_PASSWORD env var, or 'parkster auth login')")
		}
		if username == "" {
			username, err = store.Get("username")
			if err != nil {
				return "", "", fmt.Errorf("no credentials found (use --username flag, PARKSTER_USERNAME env var, or 'parkster auth login')")
			}
		}
		if password == "" {
			password, err = store.Get("password")
			if err != nil {
				return "", "", fmt.Errorf("no credentials found (use --password flag, PARKSTER_PASSWORD env var, or 'parkster auth login')")
			}
		}
	}

	return username, password, nil
}

// SaveCredentials stores username and password in keyring
func SaveCredentials(username, password string) error {
	store, err := OpenKeyring()
	if err != nil {
		return fmt.Errorf("failed to open keyring: %w", err)
	}

	if err := store.Set("username", username); err != nil {
		return fmt.Errorf("failed to store username: %w", err)
	}

	if err := store.Set("password", password); err != nil {
		return fmt.Errorf("failed to store password: %w", err)
	}

	return nil
}

// DeleteCredentials removes username and password from keyring
func DeleteCredentials() error {
	store, err := OpenKeyring()
	if err != nil {
		return fmt.Errorf("failed to open keyring: %w", err)
	}

	_ = store.Delete("username") // Ignore errors
	_ = store.Delete("password") // Ignore errors

	return nil
}
