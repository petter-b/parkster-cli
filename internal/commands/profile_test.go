package commands

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/petter-b/parkster-cli/internal/auth"
	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/petter-b/parkster-cli/internal/parkster"
)

// --- Profile command tests ---

func TestProfile_Human_ShowsAccountInfo(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:          1,
			Email:       "+46700000000",
			AccountType: "NEUTRAL",
			Cars: []parkster.Car{
				{ID: 100, LicenseNbr: "ABC123", CountryCode: "SE", CarPersonalization: parkster.CarPersonalization{Name: "Volkswagen"}},
			},
			PaymentAccounts: []parkster.PaymentAccount{
				{PaymentAccountID: "PRIVATE:9999999"},
			},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("profile")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stdout, "Account:   +46700000000") {
		t.Errorf("expected account line, got: %q", stdout)
	}
	if !strings.Contains(stdout, "Volkswagen - ABC123") {
		t.Errorf("expected car in output, got: %q", stdout)
	}
	if !strings.Contains(stdout, "PRIVATE") {
		t.Errorf("expected payment in output, got: %q", stdout)
	}
}

func TestProfile_JSON_ReturnsEnvelope(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:          1,
			Email:       "+46700000000",
			AccountType: "NEUTRAL",
			Cars: []parkster.Car{
				{ID: 100, LicenseNbr: "ABC123", CountryCode: "SE", CarPersonalization: parkster.CarPersonalization{Name: "Volkswagen"}},
			},
			PaymentAccounts: []parkster.PaymentAccount{
				{PaymentAccountID: "PRIVATE:9999999"},
			},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("profile", "--json")
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

	// Verify data contains expected fields
	dataBytes, _ := json.Marshal(envelope.Data)
	dataStr := string(dataBytes)
	if !strings.Contains(dataStr, "+46700000000") {
		t.Errorf("expected username in JSON data, got: %s", dataStr)
	}
	if !strings.Contains(dataStr, "ABC123") {
		t.Errorf("expected car license in JSON data, got: %s", dataStr)
	}
}

func TestProfile_NotAuthenticated(t *testing.T) {
	origGet := getCredentials
	getCredentials = func() (string, string, auth.CredentialSource, error) {
		return "", "", "", errors.New("no credentials found")
	}
	t.Cleanup(func() { getCredentials = origGet })

	_, stderr, err := executeCommand("profile")
	if err == nil {
		t.Fatal("expected error for profile without auth")
	}
	if !strings.Contains(stderr, "Not authenticated") {
		t.Errorf("expected 'Not authenticated' in stderr, got: %q", stderr)
	}
}

func TestProfile_LoginFails_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{loginErr: errors.New("auth failed")}
	withMockClient(t, mock)

	_, _, err := executeCommand("profile")
	if err == nil {
		t.Fatal("expected error when Login fails")
	}
	if !strings.Contains(err.Error(), "failed to fetch profile") {
		t.Errorf("expected 'failed to fetch profile' in error, got: %v", err)
	}
}

func TestProfile_ExtraArgs_Error(t *testing.T) {
	_, _, err := executeCommand("profile", "extra")
	if err == nil {
		t.Fatal("expected error for extra positional args on profile")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected 'unknown command' error, got: %v", err)
	}
}
