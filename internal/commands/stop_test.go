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

// --- Stop command tests ---

func TestStop_SingleActiveParking_Success(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 500},
			},
		},
		stopParkingResp: &parkster.Parking{
			ID:          500,
			ParkingZone: parkster.Zone{ZoneCode: "80500", Name: "Ericsson Kista"},
			Car:         parkster.Car{LicenseNbr: "ABC123"},
			Currency:    parkster.Currency{Code: "SEK"},
		},
	}
	withMockClient(t, mock)

	_, stderr, err := executeCommand("stop")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stderr, "Parking stopped") {
		t.Errorf("expected 'Parking stopped' in stderr, got: %q", stderr)
	}
}

func TestStop_NoActiveParkings_Message(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:                1,
			ShortTermParkings: []parkster.Parking{},
		},
	}
	withMockClient(t, mock)

	_, stderr, err := executeCommand("stop")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(stderr, "No active parkings") {
		t.Errorf("expected 'No active parkings' in stderr, got: %q", stderr)
	}
}

func TestStop_NoActiveParkings_JSON(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:                1,
			ShortTermParkings: []parkster.Parking{},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("stop", "--json")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected valid JSON, got: %v", err)
	}
	if !envelope.Success {
		t.Error("expected success=true")
	}
}

func TestStop_MultipleParkingsWithoutFlag_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 500},
				{ID: 501},
			},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("stop")
	if err == nil {
		t.Fatal("expected error for multiple parkings without flag, got nil")
	}
	if !strings.Contains(err.Error(), "multiple active parkings") {
		t.Errorf("expected 'multiple active parkings' in error, got: %v", err)
	}
}

func TestStop_ParkingIDFlagSelectsCorrect(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 500},
				{ID: 501},
			},
		},
		stopParkingResp: &parkster.Parking{ID: 501},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("stop", "--parking-id", "501")
	if err != nil {
		t.Fatalf("expected success with --parking-id flag, got: %v", err)
	}
}

func TestStop_ParkingIDNotFound_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 500},
			},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("stop", "--parking-id", "999")
	if err == nil {
		t.Fatal("expected error for parking ID not found, got nil")
	}
	if !strings.Contains(err.Error(), "parking session not found") {
		t.Errorf("expected 'parking session not found' in error, got: %v", err)
	}
}

func TestStop_StopParkingFails_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 500},
			},
		},
		stopParkingErr: errors.New("server error"),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("stop")
	if err == nil {
		t.Fatal("expected error when StopParking fails, got nil")
	}
	if !strings.Contains(err.Error(), "failed to stop parking") {
		t.Errorf("expected 'failed to stop parking' in error, got: %v", err)
	}
}

func TestStop_MultipleParkings_HumanOutput_NoRawStructs(t *testing.T) {
	setAuth(t)

	now := time.Now()
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{
					ID:          500,
					ParkingZone: parkster.Zone{ZoneCode: "80500", Name: "Ericsson"},
					Car:         parkster.Car{LicenseNbr: "ABC123", CarPersonalization: parkster.CarPersonalization{Name: "Volkswagen"}},
					CheckInTime: now.Add(-10 * time.Minute).UnixMilli(),
					TimeoutTime: now.Add(50 * time.Minute).UnixMilli(),
					Currency:    parkster.Currency{Code: "SEK"},
				},
				{
					ID:          501,
					ParkingZone: parkster.Zone{ZoneCode: "90100", Name: "Solna"},
					Car:         parkster.Car{LicenseNbr: "UPC304"},
					CheckInTime: now.Add(-5 * time.Minute).UnixMilli(),
					TimeoutTime: now.Add(25 * time.Minute).UnixMilli(),
					Currency:    parkster.Currency{Code: "SEK"},
				},
			},
		},
	}
	withMockClient(t, mock)

	stdout, _, _ := executeCommand("stop")
	// Should show zone codes, not raw structs
	if !strings.Contains(stdout, "80500") {
		t.Errorf("expected zone code in output, got: %q", stdout)
	}
	if strings.Contains(stdout, "{") {
		t.Errorf("should not show curly braces in output, got: %q", stdout)
	}
}

// --- Stop auth required ---

func TestStop_NotAuthenticated_Error(t *testing.T) {
	origGet := getCredentials
	getCredentials = func() (string, string, auth.CredentialSource, error) {
		return "", "", "", errors.New("no credentials found")
	}
	t.Cleanup(func() { getCredentials = origGet })

	_, stderr, err := executeCommand("stop")
	if err == nil {
		t.Fatal("expected error for stop without auth")
	}
	if !strings.Contains(stderr, "Not authenticated") {
		t.Errorf("expected 'Not authenticated' in stderr, got: %q", stderr)
	}
}

func TestStop_LoginFails_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginErr: errors.New("network timeout"),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("stop")
	if err == nil {
		t.Fatal("expected error when login fails")
	}
	if !strings.Contains(err.Error(), "failed to authenticate") {
		t.Errorf("expected 'failed to authenticate' in error, got: %v", err)
	}
}

// --- Multiple active parkings for stop ---

func TestStop_MultipleParkings_NoParkingID_Error(t *testing.T) {
	setAuth(t)

	now := time.Now()
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 500, ParkingZone: parkster.Zone{ZoneCode: "80500"}, Car: parkster.Car{LicenseNbr: "ABC123"}, CheckInTime: now.UnixMilli(), TimeoutTime: now.Add(30 * time.Minute).UnixMilli()},
				{ID: 501, ParkingZone: parkster.Zone{ZoneCode: "90100"}, Car: parkster.Car{LicenseNbr: "DEF456"}, CheckInTime: now.UnixMilli(), TimeoutTime: now.Add(60 * time.Minute).UnixMilli()},
			},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("stop")
	if err == nil {
		t.Fatal("expected error for multiple active parkings without --parking-id")
	}
	if !strings.Contains(err.Error(), "multiple active parkings") {
		t.Errorf("expected 'multiple active parkings' in error, got: %v", err)
	}
}

// --- Additional coverage: stop --json success output ---

func TestStop_SingleActiveParking_JSON(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:                1,
			ShortTermParkings: []parkster.Parking{{ID: 500}},
		},
		stopParkingResp: &parkster.Parking{
			ID:          500,
			ParkingZone: parkster.Zone{ZoneCode: "80500", Name: "Ericsson Kista"},
			Car:         parkster.Car{LicenseNbr: "ABC123"},
			Currency:    parkster.Currency{Code: "SEK"},
			Cost:        12.50,
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("stop", "--json")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected valid JSON envelope, got: %v\nOutput: %s", err, stdout)
	}
	if !envelope.Success {
		t.Error("expected success=true")
	}
	dataBytes, _ := json.Marshal(envelope.Data)
	if !strings.Contains(string(dataBytes), "12.5") {
		t.Errorf("expected cost in JSON output, got: %s", string(dataBytes))
	}
}

// --- Bug regression: disambiguation list contaminates JSON output ---

func TestStop_MultipleParkings_JSON_OutputIsValidJSON(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 500, ParkingZone: parkster.Zone{ZoneCode: "80500"}},
				{ID: 501, ParkingZone: parkster.Zone{ZoneCode: "17429"}},
			},
		},
	}
	withMockClient(t, mock)

	stdout, _, _ := executeCommand("stop", "--json")

	var envelope map[string]any
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Errorf("stdout should be valid JSON, got parse error: %v\nstdout was:\n%s", err, stdout)
	}
}

func TestStop_MultipleParkings_DisambiguationOnStderr(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 500, ParkingZone: parkster.Zone{ZoneCode: "80500", Name: "Zone A"}},
				{ID: 501, ParkingZone: parkster.Zone{ZoneCode: "17429", Name: "Zone B"}},
			},
		},
	}
	withMockClient(t, mock)

	stdout, stderr, _ := executeCommand("stop", "--json")

	if strings.Contains(stdout, "80500") || strings.Contains(stdout, "Zone A") {
		t.Errorf("parking disambiguation should not appear on stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "80500") && !strings.Contains(stderr, "Zone A") {
		t.Errorf("parking disambiguation should appear on stderr, got: %q", stderr)
	}
}

// --- Edge case: stop with --parking-id 0 ---

func TestStop_ParkingID_Zero_AutoSelects(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 500, ParkingZone: parkster.Zone{ZoneCode: "80500"}},
			},
		},
		stopParkingResp: &parkster.Parking{ID: 500, Cost: 10},
	}
	withMockClient(t, mock)

	// parking-id 0 is the default/zero value — should auto-select the single parking
	_, _, err := executeCommand("stop", "--parking-id", "0")
	if err != nil {
		t.Fatalf("parking-id 0 should auto-select, got: %v", err)
	}
}

// --- Edge case: stop --quiet with no active parkings ---

func TestStop_Quiet_NoParkings_NoOutput(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:                1,
			ShortTermParkings: []parkster.Parking{},
		},
	}
	withMockClient(t, mock)

	stdout, stderr, err := executeCommand("stop", "--quiet")
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

// G7: Negative parking-id is accepted by flag parser but rejected as not found
func TestStop_NegativeParkingID_Error(t *testing.T) {
	setAuth(t)
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 100, ParkingZone: parkster.Zone{Name: "Test"}},
				{ID: 200, ParkingZone: parkster.Zone{Name: "Other"}},
			},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("stop", "--parking-id", "-1")
	if err == nil {
		t.Error("expected error for negative parking ID")
	}
}

func TestStop_ExtraArgs_Error(t *testing.T) {
	_, _, err := executeCommand("stop", "extra")
	if err == nil {
		t.Fatal("expected error for extra positional args on stop")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected 'unknown command' error, got: %v", err)
	}
}

func TestStopCommand_NoParkings_JSON_EmptyArray(t *testing.T) {
	setAuth(t)
	mock := &mockAPI{
		loginResp: &parkster.User{ID: 1},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("stop", "--json")
	if err != nil {
		t.Fatalf("expected no error for empty parkings, got: %v", err)
	}

	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected valid JSON envelope, got: %v\nstdout: %q", err, stdout)
	}
	if !envelope.Success {
		t.Error("expected success=true for empty list")
	}
}
