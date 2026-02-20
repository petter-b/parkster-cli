package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/petter-b/parkster-cli/internal/auth"
	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/petter-b/parkster-cli/internal/parkster"
)

// --- Auth status command tests ---

func TestAuthStatus_WithEnvCredentials_Authenticated(t *testing.T) {
	orig := getCredentials
	getCredentials = func() (string, string, auth.CredentialSource, error) {
		return "testuser@example.com", "testpass", auth.SourceEnvironment, nil
	}
	t.Cleanup(func() { getCredentials = orig })

	mock := &mockAPI{loginResp: &parkster.User{ID: 1, Email: "testuser@example.com"}}
	withMockClient(t, mock)

	_, stderr, err := executeCommand("auth", "status")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stderr, "Logged in as: testuser@example.com") {
		t.Errorf("expected 'Logged in as: testuser@example.com' in stderr, got: %q", stderr)
	}
}

func TestAuthStatus_WithoutCredentials_NotAuthenticated(t *testing.T) {
	orig := getCredentials
	getCredentials = func() (string, string, auth.CredentialSource, error) {
		return "", "", "", fmt.Errorf("no credentials found")
	}
	t.Cleanup(func() { getCredentials = orig })

	_, stderr, err := executeCommand("auth", "status")
	// authRequiredError() returns ExitAuth
	if err != nil && ExitCode(err) != ExitAuth {
		t.Fatalf("expected ExitAuth or nil, got: %v (code=%d)", err, ExitCode(err))
	}
	if !strings.Contains(stderr, "Not authenticated") {
		t.Errorf("expected 'Not authenticated' in stderr, got: %q", stderr)
	}
}

func TestAuthStatus_JSON_Envelope(t *testing.T) {
	orig := getCredentials
	getCredentials = func() (string, string, auth.CredentialSource, error) {
		return "testuser@example.com", "testpass", auth.SourceEnvironment, nil
	}
	t.Cleanup(func() { getCredentials = orig })

	mock := &mockAPI{loginResp: &parkster.User{ID: 1, Email: "testuser@example.com"}}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("auth", "status", "--json")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected valid JSON, got parse error: %v\nOutput: %s", err, stdout)
	}
	if !envelope.Success {
		t.Error("expected success=true in JSON envelope")
	}

	// Parse data to check authenticated and username fields
	dataBytes, _ := json.Marshal(envelope.Data)
	var authData struct {
		Authenticated bool   `json:"authenticated"`
		Username      string `json:"username"`
	}
	if err := json.Unmarshal(dataBytes, &authData); err != nil {
		t.Fatalf("failed to parse auth status data: %v", err)
	}
	if !authData.Authenticated {
		t.Error("expected authenticated=true")
	}
	if authData.Username != "testuser@example.com" {
		t.Errorf("expected username 'testuser@example.com', got: %q", authData.Username)
	}
}

func TestAuthStatus_EnvSource_ShouldIndicateSource(t *testing.T) {
	orig := getCredentials
	getCredentials = func() (string, string, auth.CredentialSource, error) {
		return "testuser@example.com", "testpass", auth.SourceEnvironment, nil
	}
	t.Cleanup(func() { getCredentials = orig })

	mock := &mockAPI{loginResp: &parkster.User{ID: 1, Email: "testuser@example.com"}}
	withMockClient(t, mock)

	_, stderr, err := executeCommand("auth", "status")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stderr, "env") && !strings.Contains(stderr, "environment") {
		t.Errorf("auth status should indicate env var source, got: %q", stderr)
	}
}

func TestAuthStatus_EnvSource_JSON_ShouldIncludeSource(t *testing.T) {
	orig := getCredentials
	getCredentials = func() (string, string, auth.CredentialSource, error) {
		return "testuser@example.com", "testpass", auth.SourceEnvironment, nil
	}
	t.Cleanup(func() { getCredentials = orig })

	mock := &mockAPI{loginResp: &parkster.User{ID: 1, Email: "testuser@example.com"}}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("auth", "status", "--json")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stdout, "source") {
		t.Errorf("auth status JSON should include 'source' field, got: %s", stdout)
	}
}

func TestAuthStatus_NotAuthenticated_Human(t *testing.T) {
	origGet := getCredentials
	getCredentials = func() (string, string, auth.CredentialSource, error) {
		return "", "", "", errors.New("no credentials found")
	}
	t.Cleanup(func() { getCredentials = origGet })

	_, stderr, err := executeCommand("auth", "status")
	// authRequiredError returns ExitAuth
	if err != nil && ExitCode(err) != ExitAuth {
		t.Fatalf("expected ExitAuth or nil, got: %v (code=%d)", err, ExitCode(err))
	}
	if !strings.Contains(stderr, "Not authenticated") {
		t.Errorf("expected 'Not authenticated' in stderr, got: %q", stderr)
	}
}

func TestAuthStatus_ValidCredentials(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{ID: 1, Email: "test@example.com"},
	}
	withMockClient(t, mock)

	_, stderr, err := executeCommand("auth", "status")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stderr, "Logged in as: testuser") {
		t.Errorf("expected 'Logged in as: testuser' in stderr, got: %q", stderr)
	}
}

func TestAuthStatus_InvalidCredentials(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginErr: fmt.Errorf("authentication failed (status 401)"),
	}
	withMockClient(t, mock)

	_, stderr, err := executeCommand("auth", "status")
	// Should return ExitAuth (non-zero exit) since auth failed
	if err == nil || ExitCode(err) != ExitAuth {
		t.Fatalf("expected ExitAuth, got: %v", err)
	}
	if !strings.Contains(stderr, "Credentials found but authentication failed") {
		t.Errorf("expected auth failure message in stderr, got: %q", stderr)
	}
}

func TestAuthStatus_NoCredentials(t *testing.T) {
	orig := getCredentials
	getCredentials = func() (string, string, auth.CredentialSource, error) {
		return "", "", "", fmt.Errorf("no credentials")
	}
	t.Cleanup(func() { getCredentials = orig })

	_, stderr, err := executeCommand("auth", "status")
	// authRequiredError returns ExitAuth
	if err != nil && ExitCode(err) != ExitAuth {
		t.Fatalf("expected ExitAuth or nil, got: %v (code=%d)", err, ExitCode(err))
	}
	if !strings.Contains(stderr, "Not authenticated") {
		t.Errorf("expected 'Not authenticated' in stderr, got: %q", stderr)
	}
	if !strings.Contains(stderr, "parkster auth login") {
		t.Errorf("expected login hint in stderr, got: %q", stderr)
	}
}

func TestAuthStatus_ValidCredentials_JSON(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{ID: 1, Email: "test@example.com"},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("auth", "status", "--json")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected valid JSON, got: %v\nOutput: %s", err, stdout)
	}
	if !envelope.Success {
		t.Error("expected success=true")
	}
}

func TestAuthStatus_InvalidCredentials_JSON(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginErr: fmt.Errorf("authentication failed (status 401)"),
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("auth", "status", "--json")
	// Should return ExitAuth (non-zero exit) since auth failed
	if err == nil || ExitCode(err) != ExitAuth {
		t.Fatalf("expected ExitAuth, got: %v", err)
	}

	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected valid JSON, got: %v\nOutput: %s", err, stdout)
	}
	if envelope.Success {
		t.Error("expected success=false for invalid credentials")
	}
}

// --- Auth login tests ---

func TestAuthLogin_InvalidCredentials_Error(t *testing.T) {
	mock := &mockAPI{
		loginErr: errors.New("authentication failed (status 401)"),
	}
	withMockClient(t, mock)

	// Simulate stdin input for interactive prompts
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	_, _ = w.WriteString("baduser\nbadpass\n")
	_ = w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	_, _, err := executeCommand("auth", "login")
	if err == nil {
		t.Fatal("expected error for invalid credentials, got nil")
	}
	if !strings.Contains(err.Error(), "invalid credentials") {
		t.Errorf("expected 'invalid credentials' in error, got: %v", err)
	}
}

func TestAuthLogin_ValidCredentials_Success(t *testing.T) {
	mock := &mockAPI{
		loginResp: &parkster.User{ID: 1, Email: "test@example.com"},
	}
	withMockClient(t, mock)

	// Mock saveCredentials to avoid real keychain
	origSave := saveCredentials
	var savedUser, savedPass string
	saveCredentials = func(u, p string) (auth.CredentialSource, error) {
		savedUser = u
		savedPass = p
		return auth.SourceKeyring, nil
	}
	t.Cleanup(func() { saveCredentials = origSave })

	// Pipe stdin for username and password
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	_, _ = w.WriteString("testuser\ntestpass\n")
	_ = w.Close()
	t.Cleanup(func() { os.Stdin = oldStdin })

	_, stderr, err := executeCommand("auth", "login")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stderr, "Credentials stored") {
		t.Errorf("expected 'Credentials stored' in stderr, got: %q", stderr)
	}
	if savedUser != "testuser" {
		t.Errorf("expected saved username 'testuser', got %q", savedUser)
	}
	if savedPass != "testpass" {
		t.Errorf("expected saved password 'testpass', got %q", savedPass)
	}
}

func TestAuthLogin_EmptyUsername_Error(t *testing.T) {
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	_, _ = w.WriteString("\ntestpass\n")
	_ = w.Close()
	t.Cleanup(func() { os.Stdin = oldStdin })

	_, _, err := executeCommand("auth", "login")
	if err == nil {
		t.Fatal("expected error for empty username")
	}
	if !strings.Contains(err.Error(), "username cannot be empty") {
		t.Errorf("expected 'username cannot be empty' in error, got: %v", err)
	}
}

func TestAuthLogin_EmptyPassword_Error(t *testing.T) {
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	_, _ = w.WriteString("testuser\n\n")
	_ = w.Close()
	t.Cleanup(func() { os.Stdin = oldStdin })

	_, _, err := executeCommand("auth", "login")
	if err == nil {
		t.Fatal("expected error for empty password")
	}
	if !strings.Contains(err.Error(), "password cannot be empty") {
		t.Errorf("expected 'password cannot be empty' in error, got: %v", err)
	}
}

func TestAuthLogin_SaveFails_Error(t *testing.T) {
	mock := &mockAPI{
		loginResp: &parkster.User{ID: 1},
	}
	withMockClient(t, mock)

	origSave := saveCredentials
	saveCredentials = func(u, p string) (auth.CredentialSource, error) {
		return "", fmt.Errorf("keyring locked")
	}
	t.Cleanup(func() { saveCredentials = origSave })

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	_, _ = w.WriteString("testuser\ntestpass\n")
	_ = w.Close()
	t.Cleanup(func() { os.Stdin = oldStdin })

	_, _, err := executeCommand("auth", "login")
	if err == nil {
		t.Fatal("expected error when save fails")
	}
	if !strings.Contains(err.Error(), "failed to store credentials") {
		t.Errorf("expected 'failed to store credentials' in error, got: %v", err)
	}
}

func TestAuthLogin_Success_JSON_Envelope(t *testing.T) {
	// Override stdin to provide credentials
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	_, _ = w.WriteString("testuser@example.com\ntestpass\n")
	_ = w.Close()
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	mock := &mockAPI{loginResp: &parkster.User{ID: 1, Email: "testuser@example.com"}}
	withMockClient(t, mock)

	origSave := saveCredentials
	saveCredentials = func(_, _ string) (auth.CredentialSource, error) {
		return auth.SourceKeyring, nil
	}
	t.Cleanup(func() { saveCredentials = origSave })

	stdout, _, err := executeCommand("auth", "login", "--json")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected valid JSON envelope, got parse error: %v\nstdout: %s", err, stdout)
	}
	if !envelope.Success {
		t.Error("expected success=true")
	}
}

// --- Auth logout tests ---

func TestAuthLogout_WithCredentials_Success(t *testing.T) {
	// Mock deleteCredentials to avoid real keychain
	origDelete := deleteCredentials
	deleteCalled := false
	deleteCredentials = func() error {
		deleteCalled = true
		return nil
	}
	t.Cleanup(func() { deleteCredentials = origDelete })

	_, stderr, err := executeCommand("auth", "logout")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !deleteCalled {
		t.Error("expected deleteCredentials to be called")
	}
	if !strings.Contains(stderr, "Credentials removed") {
		t.Errorf("expected 'Credentials removed' in stderr, got: %q", stderr)
	}
}

func TestAuthLogout_KeyringError_Error(t *testing.T) {
	origDelete := deleteCredentials
	deleteCredentials = func() error {
		return fmt.Errorf("keyring locked")
	}
	t.Cleanup(func() { deleteCredentials = origDelete })

	_, _, err := executeCommand("auth", "logout")
	if err == nil {
		t.Fatal("expected error when keyring fails")
	}
	if !strings.Contains(err.Error(), "failed to remove credentials") {
		t.Errorf("expected 'failed to remove credentials' in error, got: %v", err)
	}
}

func TestAuthLogout_NoCredentials_ShouldIndicate(t *testing.T) {
	orig := deleteCredentials
	deleteCredentials = func() error {
		return auth.ErrNoCredentials
	}
	t.Cleanup(func() { deleteCredentials = orig })

	_, stderr, err := executeCommand("auth", "logout")
	if err != nil {
		t.Fatalf("logout should not return error, got: %v", err)
	}
	if strings.Contains(stderr, "Credentials removed") {
		t.Error("logout should not say 'Credentials removed' when no credentials existed")
	}
}

func TestAuthLogout_NoCredentials_JSON(t *testing.T) {
	orig := deleteCredentials
	deleteCredentials = func() error {
		return auth.ErrNoCredentials
	}
	t.Cleanup(func() { deleteCredentials = orig })

	stdout, _, err := executeCommand("auth", "logout", "--json")
	if err != nil {
		t.Fatalf("logout should not error, got: %v", err)
	}

	var env output.Envelope
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope from stdout: %v\nstdout was: %q", err, stdout)
	}
	if !env.Success {
		t.Errorf("expected success=true, got false")
	}
}

func TestAuthLogout_WithCredentials_JSON(t *testing.T) {
	origDelete := deleteCredentials
	deleteCredentials = func() error { return nil }
	t.Cleanup(func() { deleteCredentials = origDelete })

	stdout, _, err := executeCommand("auth", "logout", "--json")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	var env output.Envelope
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope from stdout: %v\nstdout was: %q", err, stdout)
	}
	if !env.Success {
		t.Errorf("expected success=true, got false")
	}
}

// --- Bare command tests (no subcommand) ---

func TestAuth_BareCommand_ShowsHelp(t *testing.T) {
	stdout, _, err := executeCommand("auth")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "login") {
		t.Error("auth without subcommand should show help mentioning login")
	}
}

// G1: Unknown auth subcommand shows help (Cobra default for parent commands)
func TestAuth_UnknownSubcommand_ShowsHelp(t *testing.T) {
	stdout, _, err := executeCommand("auth", "foobar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "login") {
		t.Error("auth with unknown subcommand should show help mentioning login")
	}
}

// G8: PARKSTER_EMAIL is not a recognized env var (only PARKSTER_USERNAME works)
func TestAuth_EnvParksterEmail_NotRecognized(t *testing.T) {
	t.Setenv("PARKSTER_EMAIL", "test@example.com")
	t.Setenv("PARKSTER_PASSWORD", "testpass")
	t.Setenv("PARKSTER_USERNAME", "")

	// Isolate from keyring and file-based credentials
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	orig := getCredentials
	getCredentials = auth.GetCredentials
	t.Cleanup(func() { getCredentials = orig })

	_, _, _, err := getCredentials()
	if err == nil {
		t.Error("PARKSTER_EMAIL should not be recognized; only PARKSTER_USERNAME works")
	}
}
