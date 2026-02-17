package commands

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/petter-b/parkster-cli/internal/auth"
	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/petter-b/parkster-cli/internal/parkster"
)

// --- Change command error handling tests ---

func TestChange_NeitherDurationNorUntil_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 500, CheckInTime: time.Now().UnixMilli(), TimeoutTime: time.Now().Add(30 * time.Minute).UnixMilli()},
			},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("change")
	if err == nil {
		t.Fatal("expected error when neither --duration nor --until specified")
	}
}

func TestChange_BothDurationAndUntil_Error(t *testing.T) {
	setAuth(t)

	_, _, err := executeCommand("change", "--duration", "30", "--until", "23:00")
	if err == nil {
		t.Fatal("expected error when both --duration and --until specified")
	}
}

func TestChange_UntilInvalid_Error(t *testing.T) {
	// Invalid --until now fails before auth (no mock needed)
	_, _, err := executeCommand("change", "--until", "not-a-time")
	if err == nil {
		t.Fatal("expected error for invalid --until format")
	}
}

// --- Change command tests ---

func TestChange_Duration_Success(t *testing.T) {
	setAuth(t)

	now := time.Now()
	currentEnd := now.Add(30 * time.Minute)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 500, CheckInTime: now.Add(-10 * time.Minute).UnixMilli(), TimeoutTime: currentEnd.UnixMilli()},
			},
		},
		extendResp: &parkster.Parking{
			ID:          500,
			ParkingZone: parkster.Zone{ZoneCode: "80500", Name: "Ericsson Kista"},
			Car:         parkster.Car{LicenseNbr: "ABC123"},
			TimeoutTime: now.Add(60 * time.Minute).UnixMilli(),
			Currency:    parkster.Currency{Code: "SEK"},
		},
	}
	withMockClient(t, mock)

	// --duration 60 means "set end time to now + 60 minutes"
	_, stderr, err := executeCommand("change", "--duration", "60")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stderr, "Parking changed") {
		t.Errorf("expected 'Parking changed' in stderr, got: %q", stderr)
	}
}

func TestChange_Until_Success(t *testing.T) {
	setAuth(t)

	now := time.Now()
	currentEnd := now.Add(30 * time.Minute)
	// Use a time 2 hours from now to ensure it's in the future
	untilTime := now.Add(2 * time.Hour)
	untilStr := untilTime.Format("15:04")

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 500, CheckInTime: now.Add(-10 * time.Minute).UnixMilli(), TimeoutTime: currentEnd.UnixMilli()},
			},
		},
		extendResp: &parkster.Parking{ID: 500, TimeoutTime: untilTime.UnixMilli()},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("change", "--until", untilStr)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
}

func TestChange_NoParkings_Message(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:                1,
			ShortTermParkings: []parkster.Parking{},
		},
	}
	withMockClient(t, mock)

	_, stderr, err := executeCommand("change", "--duration", "30")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(stderr, "No active parkings") {
		t.Errorf("expected 'No active parkings' in stderr, got: %q", stderr)
	}
}

func TestChange_MultipleParkingsWithoutFlag_Error(t *testing.T) {
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

	_, _, err := executeCommand("change", "--duration", "30")
	if err == nil {
		t.Fatal("expected error for multiple parkings without flag, got nil")
	}
	if !strings.Contains(err.Error(), "multiple active parkings") {
		t.Errorf("expected 'multiple active parkings' in error, got: %v", err)
	}
}

func TestChange_ParkingIDNotFound_Error(t *testing.T) {
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

	_, _, err := executeCommand("change", "--duration", "30", "--parking-id", "999")
	if err == nil {
		t.Fatal("expected error for parking ID not found, got nil")
	}
	if !strings.Contains(err.Error(), "parking session not found") {
		t.Errorf("expected 'parking session not found' in error, got: %v", err)
	}
}

func TestChange_ChangeParkingFails_Error(t *testing.T) {
	setAuth(t)

	now := time.Now()
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 500, TimeoutTime: now.Add(30 * time.Minute).UnixMilli()},
			},
		},
		extendErr: errors.New("server error"),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("change", "--duration", "60")
	if err == nil {
		t.Fatal("expected error when ExtendParking fails, got nil")
	}
	if !strings.Contains(err.Error(), "failed to change parking") {
		t.Errorf("expected 'failed to change parking' in error, got: %v", err)
	}
}

func TestChange_MultipleParkings_HumanOutput_NoRawStructs(t *testing.T) {
	setAuth(t)

	now := time.Now()
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{
					ID:          500,
					ParkingZone: parkster.Zone{ZoneCode: "80500", Name: "Ericsson"},
					Car:         parkster.Car{LicenseNbr: "ABC123"},
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

	stdout, _, _ := executeCommand("change", "--duration", "60")
	if !strings.Contains(stdout, "80500") {
		t.Errorf("expected zone code in output, got: %q", stdout)
	}
	if strings.Contains(stdout, "{") {
		t.Errorf("should not show curly braces in output, got: %q", stdout)
	}
}

// --- Duration validation tests ---

func TestChange_ZeroDuration_Error(t *testing.T) {
	setAuth(t)

	_, _, err := executeCommand("change", "--duration", "0")
	if err == nil {
		t.Fatal("expected error for --duration 0, got nil")
	}
	if !strings.Contains(err.Error(), "positive") {
		t.Errorf("expected 'positive' in error, got: %v", err)
	}
}

func TestChange_NegativeDuration_Error(t *testing.T) {
	setAuth(t)

	_, _, err := executeCommand("change", "--duration", "-5")
	if err == nil {
		t.Fatal("expected error for --duration -5, got nil")
	}
	if !strings.Contains(err.Error(), "positive") {
		t.Errorf("expected 'positive' in error, got: %v", err)
	}
}

func TestChange_UntilInvalid_NoAuthNeeded(t *testing.T) {
	// Invalid --until should fail BEFORE auth, so no setAuth or mock needed
	_, _, err := executeCommand("change", "--until", "99:99")
	if err == nil {
		t.Fatal("expected error for invalid --until format")
	}
	if !strings.Contains(err.Error(), "invalid time") {
		t.Errorf("expected 'invalid time' in error, got: %v", err)
	}
}

// --- Change auth required ---

func TestChange_NotAuthenticated_Error(t *testing.T) {
	origGet := getCredentials
	getCredentials = func() (string, string, auth.CredentialSource, error) {
		return "", "", "", errors.New("no credentials found")
	}
	t.Cleanup(func() { getCredentials = origGet })

	_, stderr, err := executeCommand("change", "--duration", "30")
	if err == nil {
		t.Fatal("expected error for change without auth")
	}
	if !strings.Contains(stderr, "Not authenticated") {
		t.Errorf("expected 'Not authenticated' in stderr, got: %q", stderr)
	}
}

func TestChange_LoginFails_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginErr: errors.New("network timeout"),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("change", "--duration", "30")
	if err == nil {
		t.Fatal("expected error when login fails")
	}
	if !strings.Contains(err.Error(), "failed to authenticate") {
		t.Errorf("expected 'failed to authenticate' in error, got: %v", err)
	}
}

// --- Additional coverage: change --json success output ---

func TestChange_Duration_JSON(t *testing.T) {
	setAuth(t)

	now := time.Now()
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 500, CheckInTime: now.Add(-10 * time.Minute).UnixMilli(), TimeoutTime: now.Add(30 * time.Minute).UnixMilli()},
			},
		},
		extendResp: &parkster.Parking{
			ID:          500,
			TimeoutTime: now.Add(60 * time.Minute).UnixMilli(),
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("change", "--duration", "60", "--json")
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
}

func TestChange_Duration_OutputShowsZoneAndCar(t *testing.T) {
	setAuth(t)

	now := time.Now()
	currentEnd := now.Add(30 * time.Minute)
	newEnd := now.Add(60 * time.Minute)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{
					ID:          500,
					ParkingZone: parkster.Zone{ZoneCode: "80500", Name: "Ericsson Kista"},
					Car:         parkster.Car{LicenseNbr: "ABC123", CarPersonalization: parkster.CarPersonalization{Name: "Volkswagen"}},
					CheckInTime: now.Add(-10 * time.Minute).UnixMilli(),
					TimeoutTime: currentEnd.UnixMilli(),
					Currency:    parkster.Currency{Code: "SEK"},
				},
			},
		},
		// Extend API returns only timeoutTime, cost, currency (no zone/car)
		extendResp: &parkster.Parking{
			TimeoutTime: newEnd.UnixMilli(),
			Cost:        15.0,
			TotalCost:   15.0,
			Currency:    parkster.Currency{Code: "SEK"},
		},
	}
	withMockClient(t, mock)

	stdout, stderr, err := executeCommand("change", "--duration", "60")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stderr, "Parking changed") {
		t.Errorf("expected 'Parking changed' in stderr, got: %q", stderr)
	}
	// Zone and car should come from the login response, not the extend response
	if !strings.Contains(stdout, "80500") {
		t.Errorf("expected zone code '80500' in output, got: %q", stdout)
	}
	if !strings.Contains(stdout, "Volkswagen") {
		t.Errorf("expected car name 'Volkswagen' in output, got: %q", stdout)
	}
	if !strings.Contains(stdout, "15.00 SEK") {
		t.Errorf("expected cost '15.00 SEK' in output, got: %q", stdout)
	}
}

// --- Change --until with past time ---

// --- Bug regression: disambiguation list contaminates JSON output ---

func TestChange_MultipleParkings_JSON_OutputIsValidJSON(t *testing.T) {
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

	stdout, _, _ := executeCommand("change", "--duration", "30", "--json")

	var envelope map[string]any
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Errorf("stdout should be valid JSON, got parse error: %v\nstdout was:\n%s", err, stdout)
	}
}

func TestChange_MultipleParkings_DisambiguationOnStderr(t *testing.T) {
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

	stdout, stderr, _ := executeCommand("change", "--duration", "30", "--json")

	if strings.Contains(stdout, "80500") || strings.Contains(stdout, "Zone A") {
		t.Errorf("parking disambiguation should not appear on stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "80500") && !strings.Contains(stderr, "Zone A") {
		t.Errorf("parking disambiguation should appear on stderr, got: %q", stderr)
	}
}

// --- JSON error envelopes for validation errors ---

func TestChange_ZeroDuration_JSON_Error(t *testing.T) {
	setAuth(t)

	stdout, _, err := executeCommandFull("change", "--duration", "0", "--json")
	if err == nil {
		t.Fatal("expected error for zero duration")
	}
	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Errorf("error output should be valid JSON: %v\nstdout: %s", err, stdout)
	}
}

func TestChange_NegativeDuration_JSON_Error(t *testing.T) {
	setAuth(t)

	stdout, _, err := executeCommandFull("change", "--duration", "-5", "--json")
	if err == nil {
		t.Fatal("expected error for negative duration")
	}
	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Errorf("error output should be valid JSON: %v\nstdout: %s", err, stdout)
	}
}

func TestChange_UntilInPast_JSON_Error(t *testing.T) {
	if time.Now().Hour() == 0 && time.Now().Minute() <= 1 {
		t.Skip("too close to midnight to test past time")
	}

	stdout, _, err := executeCommandFull("change", "--until", "00:01", "--json")
	if err == nil {
		t.Fatal("expected error for past time")
	}
	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Errorf("error output should be valid JSON: %v\nstdout: %s", err, stdout)
	}
}

// --- Edge case: change --until with past time ---

func TestChange_UntilInPast_NoAuthNeeded(t *testing.T) {
	// The past-time check should happen before authentication
	env_u := os.Getenv("PARKSTER_USERNAME")
	env_p := os.Getenv("PARKSTER_PASSWORD")
	t.Setenv("PARKSTER_USERNAME", "")
	t.Setenv("PARKSTER_PASSWORD", "")
	t.Cleanup(func() {
		_ = os.Setenv("PARKSTER_USERNAME", env_u)
		_ = os.Setenv("PARKSTER_PASSWORD", env_p)
	})

	_, _, err := executeCommand("change", "--until", "00:01")
	if err == nil {
		t.Fatal("expected error for past time")
	}
	// Should be a time error, not an auth error
	if strings.Contains(err.Error(), "authenticated") {
		t.Errorf("expected time validation error before auth, got: %v", err)
	}
}

// --- Edge case: change --quiet with no active parkings ---

func TestChange_Quiet_NoParkings_NoOutput(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:                1,
			ShortTermParkings: []parkster.Parking{},
		},
	}
	withMockClient(t, mock)

	stdout, stderr, err := executeCommand("change", "--duration", "30", "--quiet")
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

// --- Bug regression: change command offset calculation ---

// TestChange_Duration_CorrectOffset verifies that `change --duration N` computes
// the offset as total minutes from parking start to desired end time,
// not as the difference between desired end and current end.
func TestChange_Duration_CorrectOffset(t *testing.T) {
	setAuth(t)

	now := time.Now()
	checkIn := now.Add(-60 * time.Minute)    // started 60 min ago
	currentEnd := now.Add(120 * time.Minute) // ends in 120 min (180 min total)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{
					ID:          500,
					CheckInTime: checkIn.UnixMilli(),
					TimeoutTime: currentEnd.UnixMilli(),
				},
			},
		},
		extendResp: &parkster.Parking{
			ID:          500,
			TimeoutTime: now.Add(30 * time.Minute).UnixMilli(),
		},
	}
	withMockClient(t, mock)

	// "change --duration 30" means end 30 min from now.
	// Parking started 60 min ago, so total duration = 60 + 30 = 90 min.
	// The API offset should be 90 (total minutes from start), not -90 (desired - current end).
	_, _, err := executeCommand("change", "--duration", "30")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	// The offset sent to the API should represent total duration from start to desired end.
	// Expected: ~90 minutes (60 min elapsed + 30 min from now)
	// BUG: current code sends desiredEnd - currentEnd = (now+30) - (now+120) = -90
	expectedOffset := 90
	tolerance := 2 // allow 2 min clock drift
	if mock.extendMinutes < expectedOffset-tolerance || mock.extendMinutes > expectedOffset+tolerance {
		t.Errorf("expected offset ~%d minutes (total from start), got %d (current code computes desiredEnd - currentEnd)", expectedOffset, mock.extendMinutes)
	}
}

// TestChange_Until_CorrectOffset verifies that `change --until HH:MM` sends
// the right offset to the API (total minutes from parking start to target time).
func TestChange_Until_CorrectOffset(t *testing.T) {
	setAuth(t)

	now := time.Now()
	checkIn := now.Add(-30 * time.Minute)   // started 30 min ago
	currentEnd := now.Add(90 * time.Minute) // currently ends in 90 min (120 min total)

	// Target: 60 minutes from now => total duration = 30 + 60 = 90 min
	target := now.Add(60 * time.Minute)
	untilStr := target.Format("15:04")

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{
					ID:          500,
					CheckInTime: checkIn.UnixMilli(),
					TimeoutTime: currentEnd.UnixMilli(),
				},
			},
		},
		extendResp: &parkster.Parking{
			ID:          500,
			TimeoutTime: target.UnixMilli(),
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("change", "--until", untilStr)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	// Expected offset: total minutes from parking start to target time = 90 min
	// BUG: current code sends desiredEnd - currentEnd = (now+60) - (now+90) = -30
	expectedOffset := 90
	tolerance := 2
	if mock.extendMinutes < expectedOffset-tolerance || mock.extendMinutes > expectedOffset+tolerance {
		t.Errorf("expected offset ~%d minutes (total from start), got %d", expectedOffset, mock.extendMinutes)
	}
}

// TestChange_Until_RoundsUp verifies that `change --until` rounds the offset
// minutes instead of truncating. Without rounding, a fractional minute > 0.5
// gets floored, causing the parking to end 1 minute earlier than requested.
func TestChange_Until_RoundsUp(t *testing.T) {
	setAuth(t)

	now := time.Now()

	// Build a target time 30 minutes from now (as HH:MM, seconds stripped by parseUntil).
	// parseUntil always returns seconds=0, so desiredEnd has 0 seconds.
	targetRaw := now.Add(30 * time.Minute)
	untilStr := targetRaw.Format("15:04")
	// parseUntil will produce: today at HH:MM:00
	desiredEnd := time.Date(now.Year(), now.Month(), now.Day(),
		targetRaw.Hour(), targetRaw.Minute(), 0, 0, now.Location())

	// Set startTime so that its seconds component is 50.
	// desiredEnd has 0 seconds, so:
	//   desiredEnd - startTime seconds part = 60 - 50 = 10 seconds? No:
	// Actually, if startTime seconds = 50 and desiredEnd seconds = 0,
	// the sub is (desiredEnd - startTime). Let's make the total gap
	// be N minutes and 50 seconds, so we need:
	//   desiredEnd - startTime = Xm 50s
	// That means fractional = 50/60 = 0.833, int() = X, round() = X+1
	//
	// Set startTime = desiredEnd - 60m50s so gap = 60.833... min
	// int() => 60, math.Round() => 61
	startTime := desiredEnd.Add(-60*time.Minute - 50*time.Second)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{
					ID:          100,
					CheckInTime: startTime.UnixMilli(),
					TimeoutTime: now.Add(30 * time.Minute).UnixMilli(),
					ParkingZone: parkster.Zone{ZoneCode: "80500", Name: "Test"},
					Car:         parkster.Car{LicenseNbr: "ABC123"},
					Currency:    parkster.Currency{Code: "SEK"},
				},
			},
		},
		extendResp: &parkster.Parking{
			TimeoutTime: desiredEnd.UnixMilli(),
			Cost:        5.0,
			Currency:    parkster.Currency{Code: "SEK"},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("change", "--until", untilStr)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	// The offset should be rounded, not truncated.
	// 60m50s => 60.833... minutes => round => 61, truncate => 60
	if mock.extendMinutes != 61 {
		t.Errorf("expected offset 61 minutes (rounded), got %d", mock.extendMinutes)
	}
}

func TestChange_ExtraArgs_Error(t *testing.T) {
	_, _, err := executeCommand("change", "extra")
	if err == nil {
		t.Fatal("expected error for extra positional args on change")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected 'unknown command' error, got: %v", err)
	}
}
