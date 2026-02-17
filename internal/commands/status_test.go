package commands

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/petter-b/parkster-cli/internal/auth"
	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/petter-b/parkster-cli/internal/parkster"
)

// --- Status command tests ---

func TestStatus_NoParkings_PrintsMessage(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:                1,
			ShortTermParkings: []parkster.Parking{},
		},
	}
	withMockClient(t, mock)

	_, stderr, err := executeCommand("status")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stderr, "No active parkings") {
		t.Errorf("expected 'No active parkings' in stderr, got: %q", stderr)
	}
}

func TestStatus_NoParkings_JSON_EmptyArray(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:                1,
			ShortTermParkings: []parkster.Parking{},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("status", "--json")
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
	// Data should be an empty array
	dataBytes, _ := json.Marshal(envelope.Data)
	if string(dataBytes) != "[]" {
		t.Errorf("expected empty array in data, got: %s", string(dataBytes))
	}
}

func TestStatus_HasParkings_PrintsThem(t *testing.T) {
	setAuth(t)

	now := time.Now()
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{
					ID: 500,
					Car: parkster.Car{
						ID:                 100,
						LicenseNbr:         "ABC123",
						CarPersonalization: parkster.CarPersonalization{Name: "Volvo"},
					},
					ParkingZone: parkster.Zone{
						ID:       17429,
						Name:     "Ericsson Kista",
						ZoneCode: "80500",
					},
					CheckInTime: now.Add(-30 * time.Minute).UnixMilli(),
					TimeoutTime: now.Add(60 * time.Minute).UnixMilli(),
					Currency:    parkster.Currency{Code: "SEK", Symbol: "kr"},
				},
			},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("status")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stdout, "80500") {
		t.Errorf("expected zone code 80500 in output, got: %q", stdout)
	}
	if !strings.Contains(stdout, "ABC123") {
		t.Errorf("expected license plate in output, got: %q", stdout)
	}
	// Should NOT show internal IDs
	if strings.Contains(stdout, "17429") {
		t.Errorf("should not show zone ID in human output, got: %q", stdout)
	}
}

func TestStatus_LoginFails_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginErr: errors.New("auth failed"),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("status")
	if err == nil {
		t.Fatal("expected error when Login fails, got nil")
	}
	if !strings.Contains(err.Error(), "failed to authenticate") {
		t.Errorf("expected 'failed to authenticate' in error, got: %v", err)
	}
}

// --- Status auth required error ---

func TestStatus_NotAuthenticated_Human(t *testing.T) {
	origGet := getCredentials
	getCredentials = func() (string, string, auth.CredentialSource, error) {
		return "", "", "", errors.New("no credentials found")
	}
	t.Cleanup(func() { getCredentials = origGet })

	_, stderr, err := executeCommand("status")
	if err == nil {
		t.Fatal("expected error for status without auth")
	}
	if !strings.Contains(stderr, "Not authenticated") {
		t.Errorf("expected 'Not authenticated' in stderr, got: %q", stderr)
	}
}

func TestStatus_NotAuthenticated_JSON(t *testing.T) {
	origGet := getCredentials
	getCredentials = func() (string, string, auth.CredentialSource, error) {
		return "", "", "", errors.New("no credentials found")
	}
	t.Cleanup(func() { getCredentials = origGet })

	stdout, _, err := executeCommand("status", "--json")
	if err == nil {
		t.Fatal("expected error for status without auth")
	}
	// JSON mode should still produce JSON error
	if stdout != "" {
		var envelope output.Envelope
		if jsonErr := json.Unmarshal([]byte(stdout), &envelope); jsonErr != nil {
			t.Fatalf("expected JSON output, got: %q", stdout)
		}
		if envelope.Success {
			t.Error("expected success=false")
		}
	}
}

// --- Edge case: status --quiet with no active parkings ---

func TestStatus_Quiet_NoParkings_NoOutput(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:                1,
			ShortTermParkings: []parkster.Parking{},
		},
	}
	withMockClient(t, mock)

	stdout, stderr, err := executeCommand("status", "--quiet")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if stdout != "" {
		t.Errorf("expected no stdout, got: %q", stdout)
	}
	if stderr != "" {
		t.Errorf("expected no stderr with --quiet, got: %q", stderr)
	}
}

func TestStatus_ExtraArgs_Error(t *testing.T) {
	_, _, err := executeCommand("status", "extra")
	if err == nil {
		t.Fatal("expected error for extra positional args on status")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected 'unknown command' error, got: %v", err)
	}
}
