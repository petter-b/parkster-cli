package auth

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
)

// --- GetEmail tests ---

func TestGetEmail_FlagPriority(t *testing.T) {
	// Flag should take priority over env var
	os.Setenv("PARKSTER_EMAIL", "env@example.com")
	defer os.Unsetenv("PARKSTER_EMAIL")

	cmd := &cobra.Command{}
	cmd.Flags().String("email", "", "")
	cmd.Flags().Set("email", "flag@example.com")

	email, err := GetEmail(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if email != "flag@example.com" {
		t.Errorf("expected flag@example.com, got %s", email)
	}
}

func TestGetEmail_EnvFallback(t *testing.T) {
	// Env var should be used when flag is not set
	os.Setenv("PARKSTER_EMAIL", "env@example.com")
	defer os.Unsetenv("PARKSTER_EMAIL")

	cmd := &cobra.Command{}
	cmd.Flags().String("email", "", "")

	email, err := GetEmail(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if email != "env@example.com" {
		t.Errorf("expected env@example.com, got %s", email)
	}
}

func TestGetEmail_NilCmd_EnvFallback(t *testing.T) {
	// When cmd is nil, should use env var
	os.Setenv("PARKSTER_EMAIL", "env@example.com")
	defer os.Unsetenv("PARKSTER_EMAIL")

	email, err := GetEmail(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if email != "env@example.com" {
		t.Errorf("expected env@example.com, got %s", email)
	}
}

func TestGetEmail_NoCredentials(t *testing.T) {
	// Should return error when nothing configured
	os.Unsetenv("PARKSTER_EMAIL")

	cmd := &cobra.Command{}
	cmd.Flags().String("email", "", "")

	_, err := GetEmail(cmd)
	if err == nil {
		t.Fatal("expected error when no credentials configured")
	}
}

func TestGetEmail_EmptyFlag_UsesEnv(t *testing.T) {
	// Empty string flag should fall through to env
	os.Setenv("PARKSTER_EMAIL", "env@example.com")
	defer os.Unsetenv("PARKSTER_EMAIL")

	cmd := &cobra.Command{}
	cmd.Flags().String("email", "", "")
	// Don't set flag value - default is ""

	email, err := GetEmail(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if email != "env@example.com" {
		t.Errorf("expected env@example.com, got %s", email)
	}
}

// --- GetPassword tests ---

func TestGetPassword_FlagPriority(t *testing.T) {
	os.Setenv("PARKSTER_PASSWORD", "envpass")
	defer os.Unsetenv("PARKSTER_PASSWORD")

	cmd := &cobra.Command{}
	cmd.Flags().String("password", "", "")
	cmd.Flags().Set("password", "flagpass")

	password, err := GetPassword(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if password != "flagpass" {
		t.Errorf("expected flagpass, got %s", password)
	}
}

func TestGetPassword_EnvFallback(t *testing.T) {
	os.Setenv("PARKSTER_PASSWORD", "envpass")
	defer os.Unsetenv("PARKSTER_PASSWORD")

	cmd := &cobra.Command{}
	cmd.Flags().String("password", "", "")

	password, err := GetPassword(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if password != "envpass" {
		t.Errorf("expected envpass, got %s", password)
	}
}

func TestGetPassword_NilCmd_EnvFallback(t *testing.T) {
	os.Setenv("PARKSTER_PASSWORD", "envpass")
	defer os.Unsetenv("PARKSTER_PASSWORD")

	password, err := GetPassword(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if password != "envpass" {
		t.Errorf("expected envpass, got %s", password)
	}
}

func TestGetPassword_NoCredentials(t *testing.T) {
	os.Unsetenv("PARKSTER_PASSWORD")

	cmd := &cobra.Command{}
	cmd.Flags().String("password", "", "")

	_, err := GetPassword(cmd)
	if err == nil {
		t.Fatal("expected error when no credentials configured")
	}
}
