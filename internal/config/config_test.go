package config

import (
	"os"
	"strings"
	"testing"
)

func TestDefaultPath_ContainsParkster(t *testing.T) {
	// Clear XDG to test the home-based default
	os.Unsetenv("XDG_CONFIG_HOME")

	path := DefaultPath()
	if !strings.Contains(path, "parkster") {
		t.Errorf("DefaultPath should contain 'parkster', got: %s", path)
	}
	if strings.Contains(path, "mycli") {
		t.Errorf("DefaultPath should not contain 'mycli', got: %s", path)
	}
}

func TestDefaultPath_XDGOverride(t *testing.T) {
	os.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-test")
	defer os.Unsetenv("XDG_CONFIG_HOME")

	path := DefaultPath()
	if !strings.HasPrefix(path, "/tmp/xdg-test/parkster/") {
		t.Errorf("Expected path starting with /tmp/xdg-test/parkster/, got: %s", path)
	}
}

func TestDefault_ReturnsValidConfig(t *testing.T) {
	cfg := Default()
	if cfg.OutputFormat != "plain" {
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
	if cfg.OutputFormat != "plain" {
		t.Errorf("Expected default OutputFormat, got %s", cfg.OutputFormat)
	}
}
