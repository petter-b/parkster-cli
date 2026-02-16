package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultPath_ContainsParkster(t *testing.T) {
	// Clear XDG to test the home-based default
	t.Setenv("XDG_CONFIG_HOME", "")

	path := DefaultPath()
	if !strings.Contains(path, "parkster") {
		t.Errorf("DefaultPath should contain 'parkster', got: %s", path)
	}
	if strings.Contains(path, "mycli") {
		t.Errorf("DefaultPath should not contain 'mycli', got: %s", path)
	}
}

func TestDefaultPath_XDGOverride(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-test")

	path := DefaultPath()
	if !strings.HasPrefix(path, "/tmp/xdg-test/parkster/") {
		t.Errorf("Expected path starting with /tmp/xdg-test/parkster/, got: %s", path)
	}
}

func TestDefault_ReturnsValidConfig(t *testing.T) {
	cfg := Default()
	if cfg.OutputFormat != "human" {
		t.Errorf("Expected default OutputFormat 'plain', got %s", cfg.OutputFormat)
	}
	if cfg.Debug != false {
		t.Error("Expected default Debug false")
	}
	if cfg.Services == nil {
		t.Error("Expected non-nil Services map")
	}
}

func TestLoad_NonExistentFile_ReturnsDefault(t *testing.T) {
	cfg, err := Load("/tmp/nonexistent-parkster-config-test.yaml")
	if err != nil {
		t.Fatalf("Expected no error for missing file, got: %v", err)
	}
	if cfg.OutputFormat != "human" {
		t.Errorf("Expected default OutputFormat, got %s", cfg.OutputFormat)
	}
}

func TestLoad_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	yamlContent := `output_format: json
timeout: 1m
debug: true
default_account: myaccount
services:
  parkster:
    base_url: https://api.example.com
    timeout: 45s
`
	if err := os.WriteFile(path, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("Failed to write test YAML: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.OutputFormat != "json" {
		t.Errorf("Expected OutputFormat 'json', got %q", cfg.OutputFormat)
	}
	if cfg.Timeout.Duration() != 1*time.Minute {
		t.Errorf("Expected Timeout 1m, got %v", cfg.Timeout.Duration())
	}
	if !cfg.Debug {
		t.Error("Expected Debug true")
	}
	if cfg.DefaultAccount != "myaccount" {
		t.Errorf("Expected DefaultAccount 'myaccount', got %q", cfg.DefaultAccount)
	}

	svc, ok := cfg.Services["parkster"]
	if !ok {
		t.Fatal("Expected 'parkster' service config to exist")
	}
	if svc.BaseURL != "https://api.example.com" {
		t.Errorf("Expected BaseURL 'https://api.example.com', got %q", svc.BaseURL)
	}
	if svc.Timeout.Duration() != 45*time.Second {
		t.Errorf("Expected service Timeout 45s, got %v", svc.Timeout.Duration())
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	invalidYAML := `output_format: json
timeout: [this is not valid
  {broken yaml
`
	if err := os.WriteFile(path, []byte(invalidYAML), 0600); err != nil {
		t.Fatalf("Failed to write test YAML: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Expected error for invalid YAML, got nil")
	}
}

func TestSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	original := &Config{
		OutputFormat:   "json",
		Timeout:        Duration(2 * time.Minute),
		Debug:          true,
		DefaultAccount: "testaccount",
		Services: map[string]ServiceConfig{
			"parkster": {
				BaseURL: "https://api.parkster.se",
				Timeout: Duration(10 * time.Second),
			},
		},
	}

	if err := original.Save(path); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if loaded.OutputFormat != original.OutputFormat {
		t.Errorf("OutputFormat: got %q, want %q", loaded.OutputFormat, original.OutputFormat)
	}
	if loaded.Timeout.Duration() != original.Timeout.Duration() {
		t.Errorf("Timeout: got %v, want %v", loaded.Timeout.Duration(), original.Timeout.Duration())
	}
	if loaded.Debug != original.Debug {
		t.Errorf("Debug: got %v, want %v", loaded.Debug, original.Debug)
	}
	if loaded.DefaultAccount != original.DefaultAccount {
		t.Errorf("DefaultAccount: got %q, want %q", loaded.DefaultAccount, original.DefaultAccount)
	}

	svc, ok := loaded.Services["parkster"]
	if !ok {
		t.Fatal("Expected 'parkster' service config after round-trip")
	}
	if svc.BaseURL != "https://api.parkster.se" {
		t.Errorf("Service BaseURL: got %q, want %q", svc.BaseURL, "https://api.parkster.se")
	}
	if svc.Timeout.Duration() != 10*time.Second {
		t.Errorf("Service Timeout: got %v, want %v", svc.Timeout.Duration(), 10*time.Second)
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	nestedPath := filepath.Join(dir, "nested", "deep", "config.yaml")

	cfg := Default()
	if err := cfg.Save(nestedPath); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	// Verify the file exists
	if _, err := os.Stat(nestedPath); os.IsNotExist(err) {
		t.Fatal("Expected config file to be created, but it does not exist")
	}

	// Verify the file can be loaded
	loaded, err := Load(nestedPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.OutputFormat != "human" {
		t.Errorf("Expected OutputFormat 'plain', got %q", loaded.OutputFormat)
	}
}

func TestDuration_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected time.Duration
	}{
		{"seconds", "timeout: 30s", 30 * time.Second},
		{"minutes", "timeout: 5m", 5 * time.Minute},
		{"hours", "timeout: 1h", 1 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")

			if err := os.WriteFile(path, []byte(tt.yaml), 0600); err != nil {
				t.Fatalf("Failed to write YAML: %v", err)
			}

			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("Load returned error: %v", err)
			}

			if cfg.Timeout.Duration() != tt.expected {
				t.Errorf("Expected timeout %v, got %v", tt.expected, cfg.Timeout.Duration())
			}
		})
	}
}

func TestDuration_MarshalYAML(t *testing.T) {
	tests := []struct {
		name     string
		duration Duration
		expected string
	}{
		{"seconds", Duration(30 * time.Second), "30s"},
		{"minutes", Duration(5 * time.Minute), "5m0s"},
		{"hours", Duration(1 * time.Hour), "1h0m0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := tt.duration.MarshalYAML()
			if err != nil {
				t.Fatalf("MarshalYAML returned error: %v", err)
			}
			str, ok := val.(string)
			if !ok {
				t.Fatalf("MarshalYAML returned %T, want string", val)
			}
			if str != tt.expected {
				t.Errorf("MarshalYAML: got %q, want %q", str, tt.expected)
			}
		})
	}
}

func TestDuration_Duration(t *testing.T) {
	d := Duration(42 * time.Second)
	if d.Duration() != 42*time.Second {
		t.Errorf("Duration(): got %v, want %v", d.Duration(), 42*time.Second)
	}

	d = Duration(0)
	if d.Duration() != 0 {
		t.Errorf("Duration() for zero: got %v, want 0", d.Duration())
	}
}

func TestGetServiceConfig_Exists(t *testing.T) {
	cfg := &Config{
		Timeout: Duration(30 * time.Second),
		Services: map[string]ServiceConfig{
			"parkster": {
				BaseURL: "https://api.parkster.se",
				Timeout: Duration(45 * time.Second),
			},
		},
	}

	svc := cfg.GetServiceConfig("parkster")
	if svc.BaseURL != "https://api.parkster.se" {
		t.Errorf("BaseURL: got %q, want %q", svc.BaseURL, "https://api.parkster.se")
	}
	if svc.Timeout.Duration() != 45*time.Second {
		t.Errorf("Timeout: got %v, want %v", svc.Timeout.Duration(), 45*time.Second)
	}
}

func TestGetServiceConfig_Missing(t *testing.T) {
	cfg := &Config{
		Timeout:  Duration(30 * time.Second),
		Services: map[string]ServiceConfig{},
	}

	svc := cfg.GetServiceConfig("nonexistent")
	if svc.BaseURL != "" {
		t.Errorf("Expected empty BaseURL for missing service, got %q", svc.BaseURL)
	}
	if svc.Timeout.Duration() != 30*time.Second {
		t.Errorf("Expected global timeout 30s for missing service, got %v", svc.Timeout.Duration())
	}
}
