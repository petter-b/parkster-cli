package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/99designs/keyring"
)

const serviceName = "mycli"

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

		// macOS
		KeychainName:             serviceName,
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
	return s.ring.Set(keyring.Item{
		Key:  credentialKey(service),
		Data: []byte(secret),
	})
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

// List returns all stored service names
func (s *Store) List() ([]string, error) {
	keys, err := s.ring.Keys()
	if err != nil {
		return nil, err
	}

	var services []string
	prefix := "apikey:"
	for _, k := range keys {
		if strings.HasPrefix(k, prefix) {
			services = append(services, strings.TrimPrefix(k, prefix))
		}
	}
	return services, nil
}

// GetCredential retrieves a credential with env var fallback
// Priority: env var > keyring
func GetCredential(service string) (string, error) {
	// Check environment variable first
	envKey := fmt.Sprintf("MYCLI_%s_API_KEY", strings.ToUpper(strings.ReplaceAll(service, "-", "_")))
	if val := os.Getenv(envKey); val != "" {
		return val, nil
	}

	// Fall back to keyring
	store, err := OpenKeyring()
	if err != nil {
		return "", fmt.Errorf("no credential for %s: set %s or run 'mycli auth add %s'", service, envKey, service)
	}

	secret, err := store.Get(service)
	if err != nil {
		return "", fmt.Errorf("no credential for %s: set %s or run 'mycli auth add %s'", service, envKey, service)
	}

	return secret, nil
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
