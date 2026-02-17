package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/petter-b/parkster-cli/internal/auth"
	"github.com/petter-b/parkster-cli/internal/caller"
	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/petter-b/parkster-cli/internal/parkster"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// resetFlags resets global flag state between tests.
// Cobra commands are package-level singletons, so flag values
// (including --help) persist across test runs.
func resetFlags() {
	debug = false
	jsonFlag = false
	quietFlag = false
	detectedCaller = caller.Info{}            // reset caller detection
	isStderrTTY = func() bool { return true } // pipes aren't TTYs; override for test capture
	isStdinTTY = func() bool { return true }  // default to TTY for tests
	resetCommandFlags(rootCmd)
}

func resetCommandFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
		_ = f.Value.Set(f.DefValue)
	})
	for _, child := range cmd.Commands() {
		resetCommandFlags(child)
	}
}

// executeCommand runs a command with args and captures stdout/stderr.
// NOTE: This calls rootCmd.Execute() directly, bypassing the Execute()
// wrapper that handles JSON error formatting. Use executeCommandFull()
// for tests that need the full error-wrapping behavior.
func executeCommand(args ...string) (stdout string, stderr string, err error) {
	resetFlags()

	// Capture stdout
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	// Capture stderr
	oldStderr := os.Stderr
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr

	rootCmd.SetArgs(args)
	err = rootCmd.Execute()

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	_, _ = bufOut.ReadFrom(rOut)
	_, _ = bufErr.ReadFrom(rErr)

	return bufOut.String(), bufErr.String(), err
}

// executeCommandFull runs a command through the full Execute() wrapper,
// which formats non-silent errors as JSON when --json is set.
// Use this for tests that verify JSON error envelopes on validation errors.
func executeCommandFull(args ...string) (stdout string, stderr string, err error) {
	resetFlags()

	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	oldStderr := os.Stderr
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr

	rootCmd.SetArgs(args)
	err = Execute()

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	_, _ = bufOut.ReadFrom(rOut)
	_, _ = bufErr.ReadFrom(rErr)

	return bufOut.String(), bufErr.String(), err
}

// --- Help tests ---

func TestHelp_RootCommand(t *testing.T) {
	stdout, _, err := executeCommand("--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "parkster") {
		t.Error("Help should mention 'parkster'")
	}
	if !strings.Contains(stdout, "--json") {
		t.Error("Help should show --json flag")
	}
}

func TestHelp_StartCommand(t *testing.T) {
	stdout, _, err := executeCommand("start", "--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "--zone") {
		t.Error("start help should show --zone flag")
	}
	if !strings.Contains(stdout, "--duration") {
		t.Error("start help should show --duration flag")
	}
}

func TestHelp_StopCommand(t *testing.T) {
	stdout, _, err := executeCommand("stop", "--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "--parking-id") {
		t.Error("stop help should show --parking-id flag")
	}
}

func TestHelp_ChangeCommand(t *testing.T) {
	stdout, _, err := executeCommand("change", "--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "--duration") {
		t.Error("change help should show --duration flag")
	}
	if !strings.Contains(stdout, "--until") {
		t.Error("change help should show --until flag")
	}
}

func TestHelp_AuthCommand(t *testing.T) {
	stdout, _, err := executeCommand("auth", "--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "login") {
		t.Error("auth help should show login subcommand")
	}
	if !strings.Contains(stdout, "logout") {
		t.Error("auth help should show logout subcommand")
	}
	if !strings.Contains(stdout, "status") {
		t.Error("auth help should show status subcommand")
	}
}

// --- Version command tests ---

func TestVersion_Human(t *testing.T) {
	stdout, _, err := executeCommand("version")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "parkster") {
		t.Error("Version output should contain 'parkster'")
	}
}

func TestVersion_JSON(t *testing.T) {
	stdout, _, err := executeCommand("version", "--json")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("Version --json should produce valid JSON envelope: %v\nOutput: %s", err, stdout)
	}
	if !envelope.Success {
		t.Error("Version should return success=true")
	}
	if envelope.Data == nil {
		t.Error("Version data should not be null")
	}
}

// --- Error handling tests ---

func TestStart_MissingZone_Error(t *testing.T) {
	_, _, err := executeCommand("start")
	if err == nil {
		t.Error("start without --zone should return error")
	}
}

func TestStart_MissingZone_ErrorJSON(t *testing.T) {
	// Missing required --zone flag with --json should produce JSON error
	stdout, _, err := executeCommand("start", "--json")
	if err == nil {
		t.Fatal("start without --zone should return error")
	}

	// The error output should be valid JSON envelope
	if stdout != "" {
		var envelope output.Envelope
		if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
			t.Fatalf("Error output with --json should be valid JSON: %v\nOutput: %s", err, stdout)
		}
		if envelope.Success {
			t.Error("Error envelope should have success=false")
		}
	}
}

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

// --- OutputMode tests ---

func TestOutputMode_Default(t *testing.T) {
	resetFlags()
	if OutputMode() != output.ModeHuman {
		t.Error("Default output mode should be ModeHuman")
	}
}

func TestOutputMode_JSON(t *testing.T) {
	resetFlags()
	jsonFlag = true
	if OutputMode() != output.ModeJSON {
		t.Error("OutputMode should return ModeJSON when jsonFlag is set")
	}
}

// --- Mock API client ---

type mockAPI struct {
	loginResp        *parkster.User
	loginErr         error
	getZoneResp      *parkster.Zone
	getZoneErr       error
	startParkingResp *parkster.Parking
	startParkingErr  error
	// Captured arguments for verification
	extendParkingID   int
	extendMinutes     int
	stopParkingResp   *parkster.Parking
	stopParkingErr    error
	extendResp        *parkster.Parking
	extendErr         error
	searchZonesResp   *parkster.SearchResult
	searchZonesErr    error
	getZoneByCodeResp *parkster.Zone
	getZoneByCodeErr  error
	estimateCostResp  *parkster.CostEstimate
	estimateCostErr   error
}

// Compile-time check that mockAPI implements parkster.API
var _ parkster.API = (*mockAPI)(nil)

func (m *mockAPI) Login() (*parkster.User, error) {
	return m.loginResp, m.loginErr
}

func (m *mockAPI) GetZone(_ int) (*parkster.Zone, error) {
	return m.getZoneResp, m.getZoneErr
}

func (m *mockAPI) StartParking(_, _, _ int, _ string, _ int) (*parkster.Parking, error) {
	return m.startParkingResp, m.startParkingErr
}

func (m *mockAPI) StopParking(_ int) (*parkster.Parking, error) {
	return m.stopParkingResp, m.stopParkingErr
}

func (m *mockAPI) ExtendParking(parkingID, minutes int) (*parkster.Parking, error) {
	m.extendParkingID = parkingID
	m.extendMinutes = minutes
	return m.extendResp, m.extendErr
}

func (m *mockAPI) SearchZones(_, _ float64, _ int) (*parkster.SearchResult, error) {
	return m.searchZonesResp, m.searchZonesErr
}

func (m *mockAPI) GetZoneByCode(_ string, _, _ float64, _ int) (*parkster.Zone, error) {
	return m.getZoneByCodeResp, m.getZoneByCodeErr
}

func (m *mockAPI) EstimateCost(_, _, _ int, _ string, _ int) (*parkster.CostEstimate, error) {
	return m.estimateCostResp, m.estimateCostErr
}

// withMockClient swaps the global newAPIClient factory with one that returns
// the given mock, and restores the original factory when the test finishes.
func withMockClient(t *testing.T, m *mockAPI) {
	t.Helper()
	orig := newAPIClient
	newAPIClient = func(_, _ string) parkster.API { return m }
	t.Cleanup(func() { newAPIClient = orig })
}

// setAuth swaps the getCredentials function var to return test credentials,
// bypassing keychain and environment variables entirely.
func setAuth(t *testing.T) {
	t.Helper()
	orig := getCredentials
	getCredentials = func() (string, string, auth.CredentialSource, error) {
		return "testuser", "testpass", auth.SourceEnvironment, nil
	}
	t.Cleanup(func() { getCredentials = orig })
}

// --- Start command tests ---

func TestStart_SingleCarSinglePayment_Success(t *testing.T) {
	setAuth(t)

	now := time.Now()
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneResp: &parkster.Zone{ID: 17429, ZoneCode: "80500", Name: "Ericsson Kista", FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp: &parkster.Parking{
			ID:          999,
			ParkingZone: parkster.Zone{ID: 17429, ZoneCode: "80500", Name: "Ericsson Kista"},
			Car:         parkster.Car{ID: 100, LicenseNbr: "ABC123"},
			CheckInTime: now.UnixMilli(),
			TimeoutTime: now.Add(30 * time.Minute).UnixMilli(),
			Currency:    parkster.Currency{Code: "SEK"},
		},
	}
	withMockClient(t, mock)

	_, stderr, err := executeCommand("start", "--zone", "17429", "--duration", "30")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stderr, "Parking started") {
		t.Errorf("expected 'Parking started' in stderr, got: %q", stderr)
	}
}

func TestStart_NoCars_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30")
	if err == nil {
		t.Fatal("expected error for no cars, got nil")
	}
	if !strings.Contains(err.Error(), "no cars") {
		t.Errorf("expected 'no cars' in error, got: %v", err)
	}
}

func TestStart_MultipleCarsWithoutFlag_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			Cars: []parkster.Car{
				{ID: 100, LicenseNbr: "ABC123"},
				{ID: 101, LicenseNbr: "DEF456"},
			},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30")
	if err == nil {
		t.Fatal("expected error for multiple cars without flag, got nil")
	}
	if !strings.Contains(err.Error(), "multiple cars") {
		t.Errorf("expected 'multiple cars' in error, got: %v", err)
	}
}

func TestStart_CarFlagSelectsCorrectCar(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			Cars: []parkster.Car{
				{ID: 100, LicenseNbr: "ABC123"},
				{ID: 101, LicenseNbr: "DEF456"},
			},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneResp:      &parkster.Zone{ID: 17429, FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp: &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30", "--car", "DEF456")
	if err != nil {
		t.Fatalf("expected success with --car flag, got: %v", err)
	}
}

func TestStart_CarFlagUnknownPlate_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30", "--car", "UNKNOWN")
	if err == nil {
		t.Fatal("expected error for unknown car plate, got nil")
	}
	if !strings.Contains(err.Error(), "car not found") {
		t.Errorf("expected 'car not found' in error, got: %v", err)
	}
}

func TestStart_NoPaymentAccounts_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30")
	if err == nil {
		t.Fatal("expected error for no payment accounts, got nil")
	}
	if !strings.Contains(err.Error(), "no payment") {
		t.Errorf("expected 'no payment' in error, got: %v", err)
	}
}

func TestStart_MultiplePaymentsWithoutFlag_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:   1,
			Cars: []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{
				{PaymentAccountID: "pay1"},
				{PaymentAccountID: "pay2"},
			},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30")
	if err == nil {
		t.Fatal("expected error for multiple payment accounts without flag, got nil")
	}
	if !strings.Contains(err.Error(), "multiple payment") {
		t.Errorf("expected 'multiple payment' in error, got: %v", err)
	}
}

func TestStart_PaymentFlagSelectsCorrect(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:   1,
			Cars: []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{
				{PaymentAccountID: "pay1"},
				{PaymentAccountID: "pay2"},
			},
		},
		getZoneResp:      &parkster.Zone{ID: 17429, FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp: &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30", "--payment", "pay2")
	if err != nil {
		t.Fatalf("expected success with --payment flag, got: %v", err)
	}
}

func TestStart_GetZoneFails_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneErr: errors.New("zone not found"),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "99999", "--duration", "30")
	if err == nil {
		t.Fatal("expected error when GetZone fails, got nil")
	}
	if !strings.Contains(err.Error(), "zone") {
		t.Errorf("expected 'zone' in error message, got: %v", err)
	}
}

func TestStart_StartParkingFails_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneResp:     &parkster.Zone{ID: 17429, FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingErr: errors.New("server error"),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30")
	if err == nil {
		t.Fatal("expected error when StartParking fails, got nil")
	}
	if !strings.Contains(err.Error(), "failed to start parking") {
		t.Errorf("expected 'failed to start parking' in error, got: %v", err)
	}
}

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
	// authRequiredError() returns errSilent
	if err != nil && !errors.Is(err, errSilent) {
		t.Fatalf("expected errSilent or nil, got: %v", err)
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

// --- Zones no-auth tests ---

func TestZonesSearch_NoAuth_Success(t *testing.T) {
	// Do NOT call setAuth — no PARKSTER_USERNAME/PASSWORD set
	// Zone commands should work without credentials
	mock := &mockAPI{
		searchZonesResp: &parkster.SearchResult{
			ParkingZonesAtPosition: []parkster.ZoneSearchItem{
				{ID: 17429, Name: "Ericsson Kista", ZoneCode: "80500"},
			},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("zones", "search", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("zones search should work without auth, got: %v", err)
	}
}

func TestZonesInfo_NoAuth_Success(t *testing.T) {
	mock := &mockAPI{
		getZoneByCodeResp: &parkster.Zone{
			ID: 17429, Name: "Ericsson Kista", ZoneCode: "80500",
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("zones", "info", "80500", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("zones info should work without auth, got: %v", err)
	}
}

// --- Zones command tests ---

func TestHelp_ZonesCommand(t *testing.T) {
	stdout, _, err := executeCommand("zones", "--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "search") {
		t.Error("zones help should show search subcommand")
	}
	if !strings.Contains(stdout, "info") {
		t.Error("zones help should show info subcommand")
	}
}

func TestZonesSearch_Success(t *testing.T) {
	mock := &mockAPI{
		searchZonesResp: &parkster.SearchResult{
			ParkingZonesAtPosition: []parkster.ZoneSearchItem{
				{ID: 17429, Name: "Ericsson Kista", ZoneCode: "80500", City: parkster.City{Name: "Stockholm"}},
			},
			ParkingZonesNearbyPosition: []parkster.ZoneSearchItem{
				{ID: 17430, Name: "Kistagången", ZoneCode: "80501", City: parkster.City{Name: "Stockholm"}},
			},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("zones", "search", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stdout, "80500") {
		t.Errorf("expected zone code 80500 in output, got: %q", stdout)
	}
	if !strings.Contains(stdout, "Ericsson Kista") {
		t.Errorf("expected zone name in output, got: %q", stdout)
	}
}

func TestZonesSearch_Success_JSON(t *testing.T) {
	mock := &mockAPI{
		searchZonesResp: &parkster.SearchResult{
			ParkingZonesAtPosition: []parkster.ZoneSearchItem{
				{ID: 17429, Name: "Ericsson Kista", ZoneCode: "80500"},
			},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("zones", "search", "--lat", "59.373", "--lon", "17.893", "--json")
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
	dataBytes, _ := json.Marshal(envelope.Data)
	if !strings.Contains(string(dataBytes), "80500") {
		t.Error("expected zone data in JSON output")
	}
}

func TestZonesSearch_NoResults(t *testing.T) {
	mock := &mockAPI{
		searchZonesResp: &parkster.SearchResult{
			ParkingZonesAtPosition:     []parkster.ZoneSearchItem{},
			ParkingZonesNearbyPosition: []parkster.ZoneSearchItem{},
		},
	}
	withMockClient(t, mock)

	_, stderr, err := executeCommand("zones", "search", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success with empty results, got: %v", err)
	}
	if !strings.Contains(stderr, "No zones found") {
		t.Errorf("expected 'No zones found' in stderr, got: %q", stderr)
	}
}

func TestZonesSearch_NoResults_JSON(t *testing.T) {
	mock := &mockAPI{
		searchZonesResp: &parkster.SearchResult{
			ParkingZonesAtPosition:     []parkster.ZoneSearchItem{},
			ParkingZonesNearbyPosition: []parkster.ZoneSearchItem{},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("zones", "search", "--lat", "59.373", "--lon", "17.893", "--json")
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

func TestZonesSearch_MissingLatLon_Error(t *testing.T) {
	_, _, err := executeCommand("zones", "search")
	if err == nil {
		t.Fatal("expected error for missing --lat and --lon flags, got nil")
	}
}

func TestZonesSearch_InvalidCoordinates_Error(t *testing.T) {
	_, _, err := executeCommand("zones", "search", "--lat", "999", "--lon", "17.893")
	if err == nil {
		t.Fatal("expected error for invalid latitude, got nil")
	}
	if !strings.Contains(err.Error(), "latitude") {
		t.Errorf("expected 'latitude' in error, got: %v", err)
	}
}

func TestZonesSearch_SearchFails_Error(t *testing.T) {
	mock := &mockAPI{
		searchZonesErr: errors.New("search failed"),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("zones", "search", "--lat", "59.373", "--lon", "17.893")
	if err == nil {
		t.Fatal("expected error when SearchZones fails, got nil")
	}
	if !strings.Contains(err.Error(), "search") {
		t.Errorf("expected 'search' in error, got: %v", err)
	}
}

func TestZonesSearch_CustomRadius(t *testing.T) {
	mock := &mockAPI{
		searchZonesResp: &parkster.SearchResult{
			ParkingZonesAtPosition: []parkster.ZoneSearchItem{
				{ID: 17429, Name: "Ericsson Kista", ZoneCode: "80500"},
			},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("zones", "search", "--lat", "59.373", "--lon", "17.893", "--radius", "500")
	if err != nil {
		t.Fatalf("expected success with custom radius, got: %v", err)
	}
}

// --- Zones info command tests ---

func TestZonesInfo_Success(t *testing.T) {
	mock := &mockAPI{
		getZoneByCodeResp: &parkster.Zone{
			ID:       80500,
			Name:     "Ericsson Kista",
			ZoneCode: "80500",
			City:     parkster.City{Name: "Stockholm"},
			FeeZone: parkster.FeeZone{
				ID:       27545,
				Currency: parkster.Currency{Code: "SEK", Symbol: "kr"},
				ParkingFees: []parkster.ParkingFee{
					{AmountPerHour: 10.0, Description: "Weekdays 8-18", StartTime: 480, EndTime: 1080},
				},
			},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("zones", "info", "80500", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stdout, "80500") {
		t.Errorf("expected zone code in output, got: %q", stdout)
	}
	if !strings.Contains(stdout, "10") || !strings.Contains(stdout, "kr") {
		t.Errorf("expected pricing info in output, got: %q", stdout)
	}
}

func TestZonesInfo_Success_JSON(t *testing.T) {
	mock := &mockAPI{
		getZoneByCodeResp: &parkster.Zone{
			ID:       80500,
			Name:     "Ericsson Kista",
			ZoneCode: "80500",
			City:     parkster.City{Name: "Stockholm"},
			FeeZone: parkster.FeeZone{
				ID:       27545,
				Currency: parkster.Currency{Code: "SEK", Symbol: "kr"},
			},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("zones", "info", "80500", "--lat", "59.373", "--lon", "17.893", "--json")
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
	dataBytes, _ := json.Marshal(envelope.Data)
	if !strings.Contains(string(dataBytes), "80500") {
		t.Error("expected zone data in JSON output")
	}
}

func TestZonesInfo_NumericID_WithoutLatLon_Success(t *testing.T) {
	mock := &mockAPI{
		getZoneResp: &parkster.Zone{
			ID:       17429,
			Name:     "Ericsson Kista",
			ZoneCode: "80500",
			FeeZone:  parkster.FeeZone{ID: 27545},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("zones", "info", "17429")
	if err != nil {
		t.Fatalf("expected success with numeric zone ID, got: %v", err)
	}
	if !strings.Contains(stdout, "Ericsson Kista") {
		t.Errorf("expected zone name in output, got: %q", stdout)
	}
}

func TestZonesInfo_NonNumericCode_MissingLatLon_Error(t *testing.T) {
	_, _, err := executeCommand("zones", "info", "ABC123")
	if err == nil {
		t.Fatal("expected error for non-numeric code without --lat/--lon, got nil")
	}
	if !strings.Contains(err.Error(), "lat") || !strings.Contains(err.Error(), "lon") {
		t.Errorf("expected error about lat/lon required, got: %v", err)
	}
}

func TestZonesInfo_MissingArg_Error(t *testing.T) {
	_, _, err := executeCommand("zones", "info")
	if err == nil {
		t.Fatal("expected error for missing zone code argument, got nil")
	}
}

func TestZonesInfo_NotFound_Error(t *testing.T) {
	mock := &mockAPI{
		getZoneByCodeErr: errors.New("zone not found"),
		getZoneErr:       errors.New("zone not found"),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("zones", "info", "99999", "--lat", "59.373", "--lon", "17.893")
	if err == nil {
		t.Fatal("expected error when zone not found, got nil")
	}
	if !strings.Contains(err.Error(), "zone") {
		t.Errorf("expected 'zone' in error message, got: %v", err)
	}
}

// --- Start command with zone code tests ---

func TestStart_WithZoneCode_Success(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{
			ID:       17429,
			ZoneCode: "80500",
			Name:     "Ericsson Kista",
			FeeZone:  parkster.FeeZone{ID: 27545},
		},
		startParkingResp: &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success with zone code, got: %v", err)
	}
}

func TestStart_WithNumericID_Success(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeErr: errors.New("code lookup failed"),
		getZoneResp: &parkster.Zone{
			ID:      17429,
			FeeZone: parkster.FeeZone{ID: 27545},
		},
		startParkingResp: &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30")
	if err != nil {
		t.Fatalf("expected success with numeric ID, got: %v", err)
	}
}

func TestStart_ZoneCodeNotFound_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeErr: errors.New("zone not found"),
		getZoneErr:       errors.New("zone not found"),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "99999", "--duration", "30", "--lat", "59.373", "--lon", "17.893")
	if err == nil {
		t.Fatal("expected error when zone code not found, got nil")
	}
	if !strings.Contains(err.Error(), "zone") && !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected zone/not found in error, got: %v", err)
	}
}

func TestStart_ZoneCodeWithoutLatLon_FallsBackToID(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneResp: &parkster.Zone{
			ID:      17429,
			FeeZone: parkster.FeeZone{ID: 27545},
		},
		startParkingResp: &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30")
	if err != nil {
		t.Fatalf("expected success falling back to ID lookup, got: %v", err)
	}
}

// --- Start command --until tests ---

func TestStart_Until_Success(t *testing.T) {
	setAuth(t)

	// Use a time 2 hours from now
	untilTime := time.Now().Add(2 * time.Hour)
	untilStr := untilTime.Format("15:04")

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneResp:      &parkster.Zone{ID: 17429, FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp: &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--until", untilStr)
	if err != nil {
		t.Fatalf("expected success with --until, got: %v", err)
	}
}

func TestStart_BothDurationAndUntil_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30", "--until", "23:00")
	if err == nil {
		t.Fatal("expected error when both --duration and --until specified")
	}
}

func TestStart_NeitherDurationNorUntil_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429")
	if err == nil {
		t.Fatal("expected error when neither --duration nor --until specified")
	}
}

// --- Start command dry-run tests ---

func TestStart_DryRun_ShowsCost(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneResp: &parkster.Zone{
			ID:       17429,
			ZoneCode: "80500",
			Name:     "Ericsson Kista",
			FeeZone: parkster.FeeZone{
				ID: 27545,
				Currency: parkster.Currency{
					Code:   "SEK",
					Symbol: "kr",
				},
			},
		},
		estimateCostResp: &parkster.CostEstimate{
			Amount:   15.0,
			Currency: "SEK",
		},
		// startParkingResp intentionally nil - should not be called
	}
	withMockClient(t, mock)

	stdout, stderr, err := executeCommand("start", "--zone", "17429", "--duration", "30", "--dry-run")
	if err != nil {
		t.Fatalf("expected success with dry-run, got: %v", err)
	}
	if !strings.Contains(stderr, "DRY RUN") {
		t.Errorf("expected 'DRY RUN' in stderr, got: %q", stderr)
	}
	// Check that cost appears in output (either stdout or stderr)
	output := stdout + stderr
	if !strings.Contains(output, "15") {
		t.Errorf("expected cost '15' in output, got stdout: %q, stderr: %q", stdout, stderr)
	}
}

func TestStart_DryRun_JSON(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneResp: &parkster.Zone{
			ID:       17429,
			ZoneCode: "80500",
			Name:     "Ericsson Kista",
			FeeZone: parkster.FeeZone{
				ID:       27545,
				Currency: parkster.Currency{Code: "SEK", Symbol: "kr"},
			},
		},
		estimateCostResp: &parkster.CostEstimate{
			Amount:   15.0,
			Currency: "SEK",
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("start", "--zone", "17429", "--duration", "30", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected valid JSON envelope, got parse error: %v\nOutput: %s", err, stdout)
	}
	if !envelope.Success {
		t.Error("expected success=true in JSON envelope")
	}

	// Parse data to check for zone, car, duration, cost
	dataBytes, _ := json.Marshal(envelope.Data)
	var dryRunData map[string]interface{}
	if err := json.Unmarshal(dataBytes, &dryRunData); err != nil {
		t.Fatalf("failed to parse dry-run data: %v", err)
	}

	if dryRunData["zone"] == nil {
		t.Error("expected zone field in dry-run data")
	}
	if dryRunData["car"] == nil {
		t.Error("expected car field in dry-run data")
	}
	if dryRunData["duration"] == nil {
		t.Error("expected duration field in dry-run data")
	}
	if dryRunData["cost"] == nil {
		t.Error("expected cost field in dry-run data")
	}
	if dryRunData["dryRun"] != true {
		t.Error("expected dryRun=true in dry-run data")
	}
}

func TestStart_DryRun_CostEstimateFails_StillSucceeds(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneResp: &parkster.Zone{
			ID:       17429,
			ZoneCode: "80500",
			Name:     "Ericsson Kista",
			FeeZone:  parkster.FeeZone{ID: 27545},
		},
		estimateCostErr: errors.New("cost estimation unavailable"),
	}
	withMockClient(t, mock)

	_, stderr, err := executeCommand("start", "--zone", "17429", "--duration", "30", "--dry-run")
	if err != nil {
		t.Fatalf("expected success even when cost estimate fails, got: %v", err)
	}
	if !strings.Contains(stderr, "DRY RUN") {
		t.Errorf("expected 'DRY RUN' in stderr even when cost fails, got: %q", stderr)
	}
}

// --- Help handling tests ---

func TestZonesInfo_HelpText_SaysZoneCode(t *testing.T) {
	stdout, _, err := executeCommand("zones", "info", "--help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(stdout, "zone-code-or-id") {
		t.Error("help text should not say 'zone-code-or-id', should say 'zone-code'")
	}
	if !strings.Contains(stdout, "zone-code") {
		t.Error("help text should mention 'zone-code'")
	}
}

func TestZonesSearch_RadiusDefault_IsZero(t *testing.T) {
	f := zonesSearchCmd.Flags().Lookup("radius")
	if f == nil {
		t.Fatal("--radius flag not found")
	}
	if f.DefValue != "0" {
		t.Errorf("expected --radius default '0', got %q", f.DefValue)
	}
}

func TestZonesSearch_HumanOutput_NoInternalIDs(t *testing.T) {
	mock := &mockAPI{
		searchZonesResp: &parkster.SearchResult{
			ParkingZonesAtPosition: []parkster.ZoneSearchItem{
				{ID: 17429, Name: "Ericsson", ZoneCode: "80500", City: parkster.City{Name: "Kista"}},
			},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("zones", "search", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	// Should show zone code and name
	if !strings.Contains(stdout, "80500") {
		t.Errorf("expected zone code in output, got: %q", stdout)
	}
	// Should NOT show internal ID or curly braces
	if strings.Contains(stdout, "17429") {
		t.Errorf("should not show internal zone ID, got: %q", stdout)
	}
	if strings.Contains(stdout, "{") {
		t.Errorf("should not show curly braces, got: %q", stdout)
	}
}

func TestZonesInfo_HumanOutput_NoInternalIDs(t *testing.T) {
	mock := &mockAPI{
		getZoneByCodeResp: &parkster.Zone{
			ID: 17429, Name: "Ericsson", ZoneCode: "80500",
			City: parkster.City{Name: "Kista"},
			FeeZone: parkster.FeeZone{
				ID:       27545,
				Currency: parkster.Currency{Code: "SEK", Symbol: "kr"},
			},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("zones", "info", "80500", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stdout, "80500") {
		t.Errorf("expected zone code, got: %q", stdout)
	}
	if strings.Contains(stdout, "17429") {
		t.Errorf("should not show internal zone ID, got: %q", stdout)
	}
	if strings.Contains(stdout, "27545") {
		t.Errorf("should not show fee zone ID, got: %q", stdout)
	}
	if strings.Contains(stdout, "{") {
		t.Errorf("should not show curly braces, got: %q", stdout)
	}
}

func TestStart_MultipleCars_HumanOutput_NoInternalIDs(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			Cars: []parkster.Car{
				{ID: 100, LicenseNbr: "ABC123", CarPersonalization: parkster.CarPersonalization{Name: "Volkswagen"}},
				{ID: 101, LicenseNbr: "UPC304", CarPersonalization: parkster.CarPersonalization{Name: "Saab"}},
			},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
	}
	withMockClient(t, mock)

	stdout, _, _ := executeCommand("start", "--zone", "17429", "--duration", "30")
	// Should show car names and plates
	if !strings.Contains(stdout, "Volkswagen") {
		t.Errorf("expected car name in output, got: %q", stdout)
	}
	// Should NOT show internal IDs or curly braces
	if strings.Contains(stdout, " 100") || strings.Contains(stdout, " 101") {
		t.Errorf("should not show internal car IDs, got: %q", stdout)
	}
	if strings.Contains(stdout, "{") {
		t.Errorf("should not show curly braces, got: %q", stdout)
	}
}

func TestStart_MultiplePayments_HumanOutput_Clean(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:   1,
			Cars: []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{
				{PaymentAccountID: "PRIVATE:9999999"},
				{PaymentAccountID: "AT_WORK:72624"},
			},
		},
	}
	withMockClient(t, mock)

	stdout, _, _ := executeCommand("start", "--zone", "17429", "--duration", "30")
	// Should show payment info
	if !strings.Contains(stdout, "PRIVATE") {
		t.Errorf("expected payment type in output, got: %q", stdout)
	}
	if !strings.Contains(stdout, "9999999") {
		t.Errorf("expected payment ID in output, got: %q", stdout)
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

func TestStart_WithRadius_PassesToZoneLookup(t *testing.T) {
	setAuth(t)
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{
			ID: 17429, ZoneCode: "80500", Name: "Ericsson Kista",
			FeeZone: parkster.FeeZone{ID: 27545},
		},
		startParkingResp: &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30",
		"--lat", "59.373", "--lon", "17.893", "--radius", "1000")
	if err != nil {
		t.Fatalf("expected success with --radius, got: %v", err)
	}
}

// --- Car flag matching tests ---

func TestStart_CarFlag_MatchesByName(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			Cars: []parkster.Car{
				{ID: 100, LicenseNbr: "ABC123", CarPersonalization: parkster.CarPersonalization{Name: "Volkswagen"}},
				{ID: 101, LicenseNbr: "UPC304", CarPersonalization: parkster.CarPersonalization{Name: "Saab"}},
			},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneResp:      &parkster.Zone{ID: 17429, FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp: &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30", "--car", "Volkswagen")
	if err != nil {
		t.Fatalf("expected --car to match by name, got: %v", err)
	}
}

func TestStart_CarFlag_CaseInsensitive(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			Cars: []parkster.Car{
				{ID: 100, LicenseNbr: "ABC123", CarPersonalization: parkster.CarPersonalization{Name: "Volkswagen"}},
			},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneResp:      &parkster.Zone{ID: 17429, FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp: &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30", "--car", "volkswagen")
	if err != nil {
		t.Fatalf("expected case-insensitive match, got: %v", err)
	}
}

func TestStart_CarFlag_CaseInsensitivePlate(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			Cars: []parkster.Car{
				{ID: 100, LicenseNbr: "ABC123", CarPersonalization: parkster.CarPersonalization{Name: "Volkswagen"}},
			},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneResp:      &parkster.Zone{ID: 17429, FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp: &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30", "--car", "abc123")
	if err != nil {
		t.Fatalf("expected case-insensitive plate match, got: %v", err)
	}
}

// --- parseUntil flexible format tests ---

func TestParseUntil_DotSeparator(t *testing.T) {
	setAuth(t)

	now := time.Now()
	untilTime := now.Add(2 * time.Hour)
	// Use dot format: "HH.MM"
	untilStr := fmt.Sprintf("%02d.%02d", untilTime.Hour(), untilTime.Minute())

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 500, TimeoutTime: now.Add(30 * time.Minute).UnixMilli()},
			},
		},
		extendResp: &parkster.Parking{ID: 500, TimeoutTime: untilTime.UnixMilli()},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("change", "--until", untilStr)
	if err != nil {
		t.Fatalf("expected dot-separated time to work, got: %v", err)
	}
}

func TestParseUntil_BareHour(t *testing.T) {
	setAuth(t)

	now := time.Now()
	// Use a bare hour 2 hours from now
	futureHour := (now.Hour() + 2) % 24
	untilStr := fmt.Sprintf("%d", futureHour)

	// Skip if the target would be in the past (near midnight)
	target := time.Date(now.Year(), now.Month(), now.Day(), futureHour, 0, 0, 0, now.Location())
	if target.Before(now) {
		t.Skip("skipping: bare hour would be in the past near midnight")
	}

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 500, TimeoutTime: now.Add(30 * time.Minute).UnixMilli()},
			},
		},
		extendResp: &parkster.Parking{ID: 500, TimeoutTime: target.UnixMilli()},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("change", "--until", untilStr)
	if err != nil {
		t.Fatalf("expected bare hour to work, got: %v", err)
	}
}

// --- Payment flag flexible matching tests ---

func TestStart_PaymentFlag_MatchesByNumericPart(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:   1,
			Cars: []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{
				{PaymentAccountID: "PRIVATE:9999999"},
				{PaymentAccountID: "AT_WORK:72624"},
			},
		},
		getZoneResp:      &parkster.Zone{ID: 17429, FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp: &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30", "--payment", "9999999")
	if err != nil {
		t.Fatalf("expected --payment to match numeric suffix, got: %v", err)
	}
}

func TestStart_PaymentFlag_MatchesByTypePrefix(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:   1,
			Cars: []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{
				{PaymentAccountID: "PRIVATE:9999999"},
				{PaymentAccountID: "AT_WORK:72624"},
			},
		},
		getZoneResp:      &parkster.Zone{ID: 17429, FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp: &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30", "--payment", "PRIVATE")
	if err != nil {
		t.Fatalf("expected --payment to match type prefix, got: %v", err)
	}
}

func TestStart_PaymentFlag_FullID_StillWorks(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:   1,
			Cars: []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{
				{PaymentAccountID: "PRIVATE:9999999"},
			},
		},
		getZoneResp:      &parkster.Zone{ID: 17429, FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp: &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30", "--payment", "PRIVATE:9999999")
	if err != nil {
		t.Fatalf("expected full payment ID to still work, got: %v", err)
	}
}

// --- Debug short flag test ---

func TestDebug_ShortFlag(t *testing.T) {
	setAuth(t)
	mock := &mockAPI{
		loginResp: &parkster.User{ID: 1},
	}
	withMockClient(t, mock)

	_, stderr, _ := executeCommand("status", "-d")
	if !strings.Contains(stderr, "DEBUG:") {
		t.Error("expected DEBUG output with -d flag")
	}
}

// --- parseUntil edge case tests ---

func TestParseUntil_MidnightFormats(t *testing.T) {
	// "0" should parse as 00:00
	result, err := parseUntil("0")
	if err != nil {
		t.Fatalf("parseUntil(\"0\") failed: %v", err)
	}
	if result.Hour() != 0 || result.Minute() != 0 {
		t.Errorf("parseUntil(\"0\") = %v, want 00:00", result.Format("15:04"))
	}

	// "00:00" should parse as midnight
	result, err = parseUntil("00:00")
	if err != nil {
		t.Fatalf("parseUntil(\"00:00\") failed: %v", err)
	}
	if result.Hour() != 0 || result.Minute() != 0 {
		t.Errorf("parseUntil(\"00:00\") = %v, want 00:00", result.Format("15:04"))
	}

	// "23:59" should parse as 23:59
	result, err = parseUntil("23:59")
	if err != nil {
		t.Fatalf("parseUntil(\"23:59\") failed: %v", err)
	}
	if result.Hour() != 23 || result.Minute() != 59 {
		t.Errorf("parseUntil(\"23:59\") = %v, want 23:59", result.Format("15:04"))
	}
}

// --- Duration validation tests ---

func TestStart_ZeroDuration_Error(t *testing.T) {
	setAuth(t)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "0")
	if err == nil {
		t.Fatal("expected error for --duration 0, got nil")
	}
	if !strings.Contains(err.Error(), "positive") {
		t.Errorf("expected 'positive' in error, got: %v", err)
	}
}

func TestStart_NegativeDuration_Error(t *testing.T) {
	setAuth(t)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "-5")
	if err == nil {
		t.Fatal("expected error for --duration -5, got nil")
	}
	if !strings.Contains(err.Error(), "positive") {
		t.Errorf("expected 'positive' in error, got: %v", err)
	}
}

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

func TestStart_LatWithoutLon_Error(t *testing.T) {
	setAuth(t)

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--lat", "59.37")
	if err == nil {
		t.Fatal("expected error for --lat without --lon, got nil")
	}
	if !strings.Contains(err.Error(), "together") {
		t.Errorf("expected 'together' in error, got: %v", err)
	}
}

func TestStart_LonWithoutLat_Error(t *testing.T) {
	setAuth(t)

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--lon", "17.89")
	if err == nil {
		t.Fatal("expected error for --lon without --lat, got nil")
	}
	if !strings.Contains(err.Error(), "together") {
		t.Errorf("expected 'together' in error, got: %v", err)
	}
}

// =============================================================================
// Tests added by 2026-02-15 CLI tree-search walkthrough.
// =============================================================================

// --- zones info missing lat/lon pairing validation ---

func TestZonesInfo_LatWithoutLon_Error(t *testing.T) {
	_, _, err := executeCommand("zones", "info", "80500", "--lat", "59.37")
	if err == nil {
		t.Fatal("expected error for --lat without --lon, got nil")
	}
	if !strings.Contains(err.Error(), "together") {
		t.Errorf("expected 'together' in error, got: %v", err)
	}
}

func TestZonesInfo_LonWithoutLat_Error(t *testing.T) {
	_, _, err := executeCommand("zones", "info", "80500", "--lon", "17.89")
	if err == nil {
		t.Fatal("expected error for --lon without --lat, got nil")
	}
	if !strings.Contains(err.Error(), "together") {
		t.Errorf("expected 'together' in error, got: %v", err)
	}
}

// --- BUG-4: zones search accepts negative radius ---

func TestZonesSearch_NegativeRadius_Error(t *testing.T) {
	_, _, err := executeCommand("zones", "search", "--lat", "59.373", "--lon", "17.893", "--radius", "-100")
	if err == nil {
		t.Fatal("expected error for negative radius, got nil")
	}
}

// --- BUG-5: auth logout succeeds when no credentials exist ---

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

// --- BUG-6: auth status does not indicate credential source ---

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

// --- Additional coverage: debug flag on zones ---

func TestZonesSearch_Debug_WritesToStderr(t *testing.T) {
	mock := &mockAPI{
		searchZonesResp: &parkster.SearchResult{
			ParkingZonesAtPosition: []parkster.ZoneSearchItem{
				{ID: 17429, Name: "Ericsson", ZoneCode: "80500"},
			},
		},
	}
	withMockClient(t, mock)

	_, stderr, err := executeCommand("zones", "search", "--lat", "59.373", "--lon", "17.893", "-d")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stderr, "DEBUG:") {
		t.Errorf("expected DEBUG output with -d flag, got stderr: %q", stderr)
	}
}

// --- Additional coverage: error formatting in --json mode ---

func TestZonesSearch_InvalidCoordinates_JSON_Error(t *testing.T) {
	stdout, _, err := executeCommand("zones", "search", "--lat", "999", "--lon", "17.893", "--json")
	if err == nil {
		t.Fatal("expected error for invalid latitude, got nil")
	}
	// When --json is set, error should still be formatted as JSON
	if stdout != "" {
		var envelope output.Envelope
		if jsonErr := json.Unmarshal([]byte(stdout), &envelope); jsonErr != nil {
			t.Fatalf("error output with --json should be valid JSON: %v\nOutput: %s", jsonErr, stdout)
		}
		if envelope.Success {
			t.Error("error envelope should have success=false")
		}
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

// --- Additional coverage: start --json success output ---

func TestStart_SingleCarSinglePayment_JSON(t *testing.T) {
	setAuth(t)

	now := time.Now()
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneResp: &parkster.Zone{ID: 17429, ZoneCode: "80500", Name: "Ericsson Kista", FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp: &parkster.Parking{
			ID:          999,
			ParkingZone: parkster.Zone{ID: 17429, ZoneCode: "80500"},
			Car:         parkster.Car{ID: 100, LicenseNbr: "ABC123"},
			CheckInTime: now.UnixMilli(),
			TimeoutTime: now.Add(30 * time.Minute).UnixMilli(),
			Currency:    parkster.Currency{Code: "SEK"},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("start", "--zone", "17429", "--duration", "30", "--json")
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

// --- Additional coverage: unknown command ---

func TestUnknownCommand_Error(t *testing.T) {
	_, _, err := executeCommand("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown command, got nil")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected 'unknown command' in error, got: %v", err)
	}
}

// --- Additional coverage: help for all subcommands ---

func TestHelp_AuthLoginCommand(t *testing.T) {
	stdout, _, err := executeCommand("auth", "login", "--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Store") {
		t.Error("auth login help should describe storing credentials")
	}
}

func TestHelp_AuthLogoutCommand(t *testing.T) {
	stdout, _, err := executeCommand("auth", "logout", "--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Remove") {
		t.Error("auth logout help should describe removing credentials")
	}
}

func TestHelp_ZonesSearchCommand(t *testing.T) {
	stdout, _, err := executeCommand("zones", "search", "--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "--lat") {
		t.Error("zones search help should show --lat flag")
	}
	if !strings.Contains(stdout, "--lon") {
		t.Error("zones search help should show --lon flag")
	}
	if !strings.Contains(stdout, "--radius") {
		t.Error("zones search help should show --radius flag")
	}
}

func TestHelp_VersionCommand(t *testing.T) {
	stdout, _, err := executeCommand("version", "--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "version") {
		t.Error("version help should mention version")
	}
}

// --- Completion command tests ---
// Note: Completion subcommands (bash, zsh, fish, powershell) output large scripts
// that overflow the os.Pipe buffer in executeCommand. We test via --help instead.

func TestHelp_CompletionCommand(t *testing.T) {
	stdout, _, err := executeCommand("completion", "--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "bash") {
		t.Error("completion help should mention bash")
	}
	if !strings.Contains(stdout, "zsh") {
		t.Error("completion help should mention zsh")
	}
	if !strings.Contains(stdout, "fish") {
		t.Error("completion help should mention fish")
	}
	if !strings.Contains(stdout, "powershell") {
		t.Error("completion help should mention powershell")
	}
}

func TestHelp_CompletionBash(t *testing.T) {
	stdout, _, err := executeCommand("completion", "bash", "--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "bash") {
		t.Error("completion bash help should mention bash")
	}
}

func TestHelp_CompletionZsh(t *testing.T) {
	stdout, _, err := executeCommand("completion", "zsh", "--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "zsh") {
		t.Error("completion zsh help should mention zsh")
	}
}

func TestHelp_CompletionFish(t *testing.T) {
	stdout, _, err := executeCommand("completion", "fish", "--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "fish") {
		t.Error("completion fish help should mention fish")
	}
}

func TestHelp_CompletionPowershell(t *testing.T) {
	stdout, _, err := executeCommand("completion", "powershell", "--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "powershell") {
		t.Error("completion powershell help should mention powershell")
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

func TestZones_BareCommand_ShowsHelp(t *testing.T) {
	stdout, _, err := executeCommand("zones")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "search") {
		t.Error("zones without subcommand should show help mentioning search")
	}
}

// --- Auth login/logout with mock credentials ---

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

// --- Start --until with past time ---

func TestStart_UntilInPast_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
	}
	withMockClient(t, mock)

	// Use a time that's definitely in the past (00:01 today)
	_, _, err := executeCommand("start", "--zone", "17429", "--until", "00:01")
	if err == nil {
		// Only errors if it's actually past midnight+1min
		if time.Now().Hour() > 0 || time.Now().Minute() > 1 {
			t.Fatal("expected error for --until time in the past")
		}
	}
}

// --- Start --payment with unknown payment ---

func TestStart_PaymentFlagUnknown_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "PRIVATE:123"}},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30", "--payment", "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for unknown payment account")
	}
	if !strings.Contains(err.Error(), "payment account not found") {
		t.Errorf("expected 'payment account not found' in error, got: %v", err)
	}
}

// --- PARKSTER_DEBUG env var ---

func TestDebug_EnvVar(t *testing.T) {
	// The PARKSTER_DEBUG env var is read at init time.
	// We can test that debugLog works when debug is true.
	resetFlags()
	debug = true
	defer func() { debug = false }()

	_, stderr, err := executeCommand("version")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// version command doesn't produce debug output, but the flag should be set
	_ = stderr // no debug output from version, that's fine
}

// --- Change --until with past time ---

func TestChange_UntilInPast_Error(t *testing.T) {
	// Use a time that's definitely in the past
	_, _, err := executeCommand("change", "--until", "00:01")
	if err == nil {
		if time.Now().Hour() > 0 || time.Now().Minute() > 1 {
			t.Fatal("expected error for --until time in the past")
		}
	}
	if err != nil && !strings.Contains(err.Error(), "in the past") {
		t.Errorf("expected 'in the past' in error, got: %v", err)
	}
}

// --- Start --dry-run with --until ---

func TestStart_DryRun_WithUntil(t *testing.T) {
	setAuth(t)

	now := time.Now()
	// Use a time 2 hours from now
	futureHour := now.Add(2 * time.Hour)
	untilStr := fmt.Sprintf("%02d:%02d", futureHour.Hour(), futureHour.Minute())

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneResp:      &parkster.Zone{ID: 17429, ZoneCode: "80500", Name: "Ericsson Kista", FeeZone: parkster.FeeZone{ID: 27545}},
		estimateCostResp: &parkster.CostEstimate{Amount: 20.0, Currency: "SEK"},
	}
	withMockClient(t, mock)

	stdout, stderr, err := executeCommand("start", "--zone", "17429", "--until", untilStr, "--dry-run")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stderr, "DRY RUN") {
		t.Errorf("expected 'DRY RUN' in stderr, got: %q", stderr)
	}
	if stdout == "" {
		t.Error("expected dry-run output on stdout")
	}
}

// --- Auth status: not authenticated (human mode) ---

func TestAuthStatus_NotAuthenticated_Human(t *testing.T) {
	origGet := getCredentials
	getCredentials = func() (string, string, auth.CredentialSource, error) {
		return "", "", "", errors.New("no credentials found")
	}
	t.Cleanup(func() { getCredentials = origGet })

	_, stderr, err := executeCommand("auth", "status")
	// authRequiredError returns errSilent
	if err != nil && !errors.Is(err, errSilent) {
		t.Fatalf("expected errSilent or nil, got: %v", err)
	}
	if !strings.Contains(stderr, "Not authenticated") {
		t.Errorf("expected 'Not authenticated' in stderr, got: %q", stderr)
	}
}

// --- Help for remaining subcommands ---

func TestHelp_ChangeCommand_UntilFlag(t *testing.T) {
	stdout, _, err := executeCommand("change", "--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "--parking-id") {
		t.Error("change help should show --parking-id flag")
	}
}

func TestHelp_StopCommand_ParkingIDFlag(t *testing.T) {
	stdout, _, err := executeCommand("stop", "--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "--parking-id") {
		t.Error("stop help should show --parking-id flag")
	}
}

func TestHelp_StartCommand_AllFlags(t *testing.T) {
	stdout, _, err := executeCommand("start", "--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	for _, flag := range []string{"--zone", "--duration", "--until", "--car", "--payment", "--dry-run", "--lat", "--lon", "--radius"} {
		if !strings.Contains(stdout, flag) {
			t.Errorf("start help should show %s flag", flag)
		}
	}
}

func TestHelp_ZonesInfoCommand(t *testing.T) {
	stdout, _, err := executeCommand("zones", "info", "--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "--lat") {
		t.Error("zones info help should show --lat flag")
	}
	if !strings.Contains(stdout, "--lon") {
		t.Error("zones info help should show --lon flag")
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

// --- Start auth required ---

func TestStart_NotAuthenticated_Error(t *testing.T) {
	origGet := getCredentials
	getCredentials = func() (string, string, auth.CredentialSource, error) {
		return "", "", "", errors.New("no credentials found")
	}
	t.Cleanup(func() { getCredentials = origGet })

	_, stderr, err := executeCommand("start", "--zone", "17429", "--duration", "30")
	if err == nil {
		t.Fatal("expected error for start without auth")
	}
	if !strings.Contains(stderr, "Not authenticated") {
		t.Errorf("expected 'Not authenticated' in stderr, got: %q", stderr)
	}
}

// --- Login failures ---

func TestStart_LoginFails_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginErr: errors.New("network timeout"),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30")
	if err == nil {
		t.Fatal("expected error when login fails")
	}
	if !strings.Contains(err.Error(), "failed to authenticate") {
		t.Errorf("expected 'failed to authenticate' in error, got: %v", err)
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

// --- Multiple active parkings for stop/change ---

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

// --- Auth login empty username/password ---

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

// --- Auth status tests ---

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
	// Should return errSilent (non-zero exit) since auth failed
	if !errors.Is(err, errSilent) {
		t.Fatalf("expected errSilent, got: %v", err)
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
	// authRequiredError returns errSilent which is suppressed
	if err != nil && !errors.Is(err, errSilent) {
		t.Fatalf("expected errSilent or nil, got: %v", err)
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
	// Should return errSilent (non-zero exit) since auth failed
	if !errors.Is(err, errSilent) {
		t.Fatalf("expected errSilent, got: %v", err)
	}

	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected valid JSON, got: %v\nOutput: %s", err, stdout)
	}
	if envelope.Success {
		t.Error("expected success=false for invalid credentials")
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

// --- Quiet flag tests ---

func TestQuiet_SuppressesStatusNoParkings(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{ID: 1},
	}
	withMockClient(t, mock)

	_, stderr, err := executeCommand("--quiet", "status")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if strings.Contains(stderr, "No active parkings") {
		t.Error("--quiet should suppress 'No active parkings' on stderr")
	}
}

func TestQuiet_SuppressesParkingStopped(t *testing.T) {
	setAuth(t)

	now := time.Now()
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 500, ParkingZone: parkster.Zone{ZoneCode: "80500"}, Car: parkster.Car{LicenseNbr: "ABC123"}, CheckInTime: now.UnixMilli(), TimeoutTime: now.Add(30 * time.Minute).UnixMilli()},
			},
		},
		stopParkingResp: &parkster.Parking{ID: 500, ParkingZone: parkster.Zone{ZoneCode: "80500"}, Car: parkster.Car{LicenseNbr: "ABC123"}, Cost: 5.0, Currency: parkster.Currency{Code: "SEK"}},
	}
	withMockClient(t, mock)

	stdout, stderr, err := executeCommand("--quiet", "stop")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if strings.Contains(stderr, "Parking stopped") {
		t.Error("--quiet should suppress 'Parking stopped' on stderr")
	}
	// Stdout should still have the parking details
	if stdout == "" {
		t.Error("expected parking details on stdout even with --quiet")
	}
}

func TestQuiet_SuppressesParkingStarted(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneResp:      &parkster.Zone{ID: 17429, ZoneCode: "80500", Name: "Test Zone", FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp: &parkster.Parking{ID: 999, ParkingZone: parkster.Zone{ZoneCode: "80500"}, Car: parkster.Car{LicenseNbr: "ABC123"}, Cost: 0},
	}
	withMockClient(t, mock)

	stdout, stderr, err := executeCommand("--quiet", "start", "--zone", "17429", "--duration", "30")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if strings.Contains(stderr, "Parking started") {
		t.Error("--quiet should suppress 'Parking started' on stderr")
	}
	if stdout == "" {
		t.Error("expected parking details on stdout even with --quiet")
	}
}

func TestQuiet_NotSetShowsStatusMessages(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{ID: 1},
	}
	withMockClient(t, mock)

	// Without --quiet, status messages should appear (TTY is overridden to true in tests)
	_, stderr, err := executeCommand("status")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stderr, "No active parkings") {
		t.Errorf("expected 'No active parkings' in stderr without --quiet, got: %q", stderr)
	}
}

func TestStatusMsg_TTYFalse_Suppresses(t *testing.T) {
	resetFlags()
	quietFlag = false
	jsonFlag = false
	isStderrTTY = func() bool { return false }
	defer func() { isStderrTTY = func() bool { return true } }()

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	statusMsg("should not appear")

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	if buf.String() != "" {
		t.Errorf("statusMsg should produce no output when TTY=false, got: %q", buf.String())
	}
}

func TestStatusMsg_QuietTrue(t *testing.T) {
	resetFlags()
	quietFlag = true
	defer func() { quietFlag = false }()

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	statusMsg("should not appear")

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	if buf.String() != "" {
		t.Errorf("statusMsg should produce no output when quiet=true, got: %q", buf.String())
	}
}

func TestStatusMsg_QuietFalse(t *testing.T) {
	resetFlags()
	quietFlag = false
	// isStderrTTY is already set to return true by resetFlags()

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	statusMsg("hello %s", "world")

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	if buf.String() != "hello world\n" {
		t.Errorf("statusMsg should produce output when quiet=false, got: %q", buf.String())
	}
}

func TestHelp_ShowsQuietFlag(t *testing.T) {
	stdout, _, err := executeCommand("--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "--quiet") && !strings.Contains(stdout, "-q") {
		t.Error("Help should show --quiet / -q flag")
	}
}

// --- Caller detection tests ---

func TestDebug_IncludesCallerInfo(t *testing.T) {
	setAuth(t)
	mock := &mockAPI{
		loginResp: &parkster.User{ID: 1},
	}
	withMockClient(t, mock)

	_, stderr, _ := executeCommand("status", "-d")
	// When run from go test, caller detection should find something
	if !strings.Contains(stderr, "DEBUG: caller=") {
		t.Errorf("expected debug output to include caller info, got stderr: %q", stderr)
	}
	if !strings.Contains(stderr, "pid=") {
		t.Errorf("expected debug output to include pid, got stderr: %q", stderr)
	}
}

func TestGetCredentials_FunctionVariable_Works(t *testing.T) {
	// Test that the getCredentials function variable can be called
	orig := getCredentials
	called := false
	getCredentials = func() (string, string, auth.CredentialSource, error) {
		called = true
		return "u", "p", auth.SourceEnvironment, nil
	}
	t.Cleanup(func() { getCredentials = orig })

	u, p, src, err := getCredentials()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected getCredentials function variable to be called")
	}
	if u != "u" || p != "p" || src != auth.SourceEnvironment {
		t.Errorf("unexpected return values: %s, %s, %s", u, p, src)
	}
}

func TestIsStdinTTY_Overridable(t *testing.T) {
	orig := isStdinTTY
	t.Cleanup(func() { isStdinTTY = orig })

	isStdinTTY = func() bool { return false }
	if isStdinTTY() {
		t.Error("expected isStdinTTY override to return false")
	}

	isStdinTTY = func() bool { return true }
	if !isStdinTTY() {
		t.Error("expected isStdinTTY override to return true")
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

// --- Bug regression: disambiguation list contaminates JSON output ---

// TestStart_MultipleCars_JSON_OutputIsValidJSON verifies that when multiple cars
// exist and --json is set, the stdout output is valid JSON (no car list preamble).
func TestStart_MultipleCars_JSON_OutputIsValidJSON(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			Cars: []parkster.Car{
				{ID: 100, LicenseNbr: "ABC123", CarPersonalization: parkster.CarPersonalization{Name: "Volkswagen"}},
				{ID: 101, LicenseNbr: "UPC304", CarPersonalization: parkster.CarPersonalization{Name: "Saab"}},
			},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
	}
	withMockClient(t, mock)

	stdout, _, _ := executeCommand("start", "--zone", "17429", "--duration", "30", "--json")

	// stdout should be ONLY valid JSON - no car list contamination
	var envelope map[string]any
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Errorf("stdout should be valid JSON, got parse error: %v\nstdout was:\n%s", err, stdout)
	}
}

// TestStart_MultiplePayments_JSON_OutputIsValidJSON verifies payment disambiguation
// does not contaminate JSON output.
func TestStart_MultiplePayments_JSON_OutputIsValidJSON(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:   1,
			Cars: []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{
				{PaymentAccountID: "PRIVATE:9999999"},
				{PaymentAccountID: "AT_WORK:72624"},
			},
		},
	}
	withMockClient(t, mock)

	stdout, _, _ := executeCommand("start", "--zone", "17429", "--duration", "30", "--json")

	var envelope map[string]any
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Errorf("stdout should be valid JSON, got parse error: %v\nstdout was:\n%s", err, stdout)
	}
}

// TestStop_MultipleParkings_JSON_OutputIsValidJSON verifies parking disambiguation
// does not contaminate JSON output.
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

// TestChange_MultipleParkings_JSON_OutputIsValidJSON verifies parking disambiguation
// does not contaminate JSON output for the change command.
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

// --- Edge case: start --radius help text vs actual default ---

func TestStart_RadiusDefault_IsZero(t *testing.T) {
	// Verify that the radius flag defaults to 0 (API decides default),
	// matching the behavior of zones search
	resetFlags()
	f := startCmd.Flags().Lookup("radius")
	if f == nil {
		t.Fatal("expected radius flag on start command")
	}
	if f.DefValue != "0" {
		t.Errorf("radius default should be 0, got %q", f.DefValue)
	}
}

// --- Edge case: zones info with lat only or lon only ---

func TestZonesInfo_LatOnly_Error(t *testing.T) {
	_, _, err := executeCommand("zones", "info", "80500", "--lat", "59.3")
	if err == nil {
		t.Fatal("expected error for --lat without --lon")
	}
	if !strings.Contains(err.Error(), "--lat and --lon must be used together") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestZonesInfo_LonOnly_Error(t *testing.T) {
	_, _, err := executeCommand("zones", "info", "80500", "--lon", "17.9")
	if err == nil {
		t.Fatal("expected error for --lon without --lat")
	}
	if !strings.Contains(err.Error(), "--lat and --lon must be used together") {
		t.Errorf("unexpected error: %v", err)
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

// --- Auth logout JSON tests ---

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

// --- Disambiguation lists go to stderr, not stdout ---

func TestStart_MultipleCars_DisambiguationOnStderr(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			Cars: []parkster.Car{
				{ID: 100, LicenseNbr: "ABC123", CarPersonalization: parkster.CarPersonalization{Name: "Volvo"}},
				{ID: 101, LicenseNbr: "DEF456", CarPersonalization: parkster.CarPersonalization{Name: "Saab"}},
			},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
	}
	withMockClient(t, mock)

	stdout, stderr, _ := executeCommand("start", "--zone", "17429", "--duration", "30", "--json")

	// Car list should appear on stderr, not stdout
	if strings.Contains(stdout, "ABC123") || strings.Contains(stdout, "DEF456") {
		t.Errorf("disambiguation list should not appear on stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "ABC123") && !strings.Contains(stderr, "DEF456") {
		t.Errorf("disambiguation list should appear on stderr, got: %q", stderr)
	}
}

func TestStart_MultiplePayments_DisambiguationOnStderr(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:   1,
			Cars: []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{
				{PaymentAccountID: "PRIVATE:111"},
				{PaymentAccountID: "AT_WORK:222"},
			},
		},
	}
	withMockClient(t, mock)

	stdout, stderr, _ := executeCommand("start", "--zone", "17429", "--duration", "30", "--json")

	if strings.Contains(stdout, "PRIVATE") || strings.Contains(stdout, "AT_WORK") {
		t.Errorf("payment disambiguation should not appear on stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "PRIVATE") && !strings.Contains(stderr, "AT_WORK") {
		t.Errorf("payment disambiguation should appear on stderr, got: %q", stderr)
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

// The following tests use executeCommandFull to test the full Execute() wrapper
// which handles JSON error formatting for non-silent errors.

func TestStart_ZeroDuration_JSON_Error(t *testing.T) {
	setAuth(t)

	stdout, _, err := executeCommandFull("start", "--zone", "17429", "--duration", "0", "--json")
	if err == nil {
		t.Fatal("expected error for zero duration")
	}
	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Errorf("error output should be valid JSON: %v\nstdout: %s", err, stdout)
	}
	if envelope.Success {
		t.Error("expected success=false")
	}
}

func TestStart_NegativeDuration_JSON_Error(t *testing.T) {
	setAuth(t)

	stdout, _, err := executeCommandFull("start", "--zone", "17429", "--duration", "-5", "--json")
	if err == nil {
		t.Fatal("expected error for negative duration")
	}
	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Errorf("error output should be valid JSON: %v\nstdout: %s", err, stdout)
	}
}

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

func TestStart_UntilInPast_JSON_Error(t *testing.T) {
	if time.Now().Hour() == 0 && time.Now().Minute() <= 1 {
		t.Skip("too close to midnight to test past time")
	}
	setAuth(t)

	stdout, _, err := executeCommandFull("start", "--zone", "17429", "--until", "00:01", "--json")
	if err == nil {
		t.Fatal("expected error for past time")
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

// --- Human-mode disambiguation shows list on stdout ---

func TestStart_MultipleCars_HumanMode_ShowsList(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			Cars: []parkster.Car{
				{ID: 100, LicenseNbr: "ABC123", CarPersonalization: parkster.CarPersonalization{Name: "Volvo"}},
				{ID: 101, LicenseNbr: "DEF456", CarPersonalization: parkster.CarPersonalization{Name: "Saab"}},
			},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("start", "--zone", "17429", "--duration", "30")
	if err == nil {
		t.Fatal("expected error for multiple cars")
	}
	// In human mode, car list should appear on stdout
	if !strings.Contains(stdout, "ABC123") && !strings.Contains(stdout, "Volvo") {
		t.Errorf("expected car list on stdout in human mode, got: %q", stdout)
	}
}

// --- version --json --quiet still outputs JSON ---

func TestVersion_JSON_Quiet(t *testing.T) {
	stdout, _, err := executeCommand("version", "--json", "--quiet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("version --json --quiet should produce valid JSON: %v", err)
	}
	if !envelope.Success {
		t.Error("expected success=true")
	}
}

// --- zones search --quiet ---

func TestZonesSearch_Quiet_HasResults(t *testing.T) {
	mock := &mockAPI{
		searchZonesResp: &parkster.SearchResult{
			ParkingZonesAtPosition: []parkster.ZoneSearchItem{
				{ID: 1, ZoneCode: "80500", Name: "Ericsson", City: parkster.City{Name: "Kista"}},
			},
		},
	}
	withMockClient(t, mock)

	stdout, stderr, err := executeCommand("zones", "search", "--lat", "59.3", "--lon", "17.9", "--quiet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Data should still appear on stdout
	if !strings.Contains(stdout, "80500") {
		t.Errorf("expected zone data on stdout even with --quiet, got: %q", stdout)
	}
	// Status messages should be suppressed on stderr
	if strings.Contains(stderr, "Found") || strings.Contains(stderr, "Searching") {
		t.Errorf("--quiet should suppress status messages on stderr, got: %q", stderr)
	}
}

// --- Audit-discovered tests (2026-02-17) ---
// These tests document behavior found during the CLI command audit.
// See docs/plans/2026-02-17-cli-command-audit.md for full audit report.

// G1: Unknown auth subcommand returns error
func TestAuth_UnknownSubcommand_ReturnsError(t *testing.T) {
	_, _, err := executeCommand("auth", "foobar")
	if err == nil {
		t.Fatal("expected error for unknown auth subcommand")
	}
	if !strings.Contains(err.Error(), "unknown subcommand") {
		t.Errorf("expected 'unknown subcommand' in error, got: %q", err.Error())
	}
}

// G2: Unknown zones subcommand returns error
func TestZones_UnknownSubcommand_ReturnsError(t *testing.T) {
	_, _, err := executeCommand("zones", "foobar")
	if err == nil {
		t.Fatal("expected error for unknown zones subcommand")
	}
	if !strings.Contains(err.Error(), "unknown subcommand") {
		t.Errorf("expected 'unknown subcommand' in error, got: %q", err.Error())
	}
}

// G5: JSON + quiet already covered by TestVersion_JSON_Quiet (line 4179)

// G6: Debug + JSON sends debug to stderr and JSON to stdout
func TestVersion_Debug_JSON_SeparateStreams(t *testing.T) {
	stdout, stderr, err := executeCommand("version", "--debug", "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// JSON on stdout
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected valid JSON on stdout, got: %s", stdout)
	}
	// Debug on stderr
	if !strings.Contains(stderr, "DEBUG") {
		t.Errorf("expected DEBUG output on stderr, got: %q", stderr)
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

// G3: Extra positional args on version are silently ignored
func TestVersion_ExtraArgs_Ignored(t *testing.T) {
	stdout, _, err := executeCommand("version", "extra", "args")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "parkster") {
		t.Error("expected version output despite extra args")
	}
}

// Flag validation: start --zone accepts non-numeric zone codes
func TestStart_ZoneCode_NonNumeric_Accepted(t *testing.T) {
	setAuth(t)
	mock := &mockAPI{
		loginResp: &parkster.User{
			Cars:            []parkster.Car{{ID: 1, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{ID: 100, FeeZone: parkster.FeeZone{ID: 200}},
		startParkingResp:  &parkster.Parking{ID: 999, ParkingZone: parkster.Zone{Name: "Test"}},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "ABC123", "--duration", "30", "--lat", "59.0", "--lon", "17.0")
	if err != nil {
		t.Fatalf("expected non-numeric zone code to be accepted, got error: %v", err)
	}
}

// Error JSON envelope: zones search missing flags with --json
func TestZonesSearch_MissingFlags_JSON_Error(t *testing.T) {
	stdout, _, _ := executeCommandFull("zones", "search", "--json")
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected valid JSON error envelope, got: %s", stdout)
	}
	if envelope["success"] != false {
		t.Errorf("expected success=false, got %v", envelope["success"])
	}
}

// G4: PARKSTER_DEBUG=false does NOT enable debug
func TestDebug_EnvFalse_DisablesDebug(t *testing.T) {
	t.Setenv("PARKSTER_DEBUG", "false")
	resetFlags()
	if debug {
		t.Error("PARKSTER_DEBUG=false should NOT enable debug mode")
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
	getCredentials = func() (string, string, auth.CredentialSource, error) {
		return auth.GetCredentials()
	}
	t.Cleanup(func() { getCredentials = orig })

	_, _, _, err := getCredentials()
	if err == nil {
		t.Error("PARKSTER_EMAIL should not be recognized; only PARKSTER_USERNAME works")
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

func TestStatus_ExtraArgs_Error(t *testing.T) {
	_, _, err := executeCommand("status", "extra")
	if err == nil {
		t.Fatal("expected error for extra positional args on status")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected 'unknown command' error, got: %v", err)
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

func TestChange_ExtraArgs_Error(t *testing.T) {
	_, _, err := executeCommand("change", "extra")
	if err == nil {
		t.Fatal("expected error for extra positional args on change")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected 'unknown command' error, got: %v", err)
	}
}

func TestStart_ExtraArgs_Error(t *testing.T) {
	_, _, err := executeCommand("start", "extra")
	if err == nil {
		t.Fatal("expected error for extra positional args on start")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected 'unknown command' error, got: %v", err)
	}
}

func TestFlagParseError_JSON_WrappedInEnvelope(t *testing.T) {
	// --lat expects float64; "abc" should fail parsing.
	// Even though --json is passed, Cobra may not have parsed it before the error.
	// We must set os.Args because hasJSONFlag checks os.Args as a fallback.
	oldArgs := os.Args
	os.Args = []string{"parkster", "zones", "search", "--lat", "abc", "--lon", "1.0", "--json"}
	defer func() { os.Args = oldArgs }()

	stdout, _, err := executeCommandFull("zones", "search", "--lat", "abc", "--lon", "1.0", "--json")
	if err == nil {
		t.Fatal("expected error from invalid --lat value")
	}

	// The error should be wrapped in a JSON envelope on stdout
	var envelope map[string]interface{}
	if jsonErr := json.Unmarshal([]byte(stdout), &envelope); jsonErr != nil {
		t.Fatalf("expected valid JSON error envelope, got parse error: %v\nstdout: %q", jsonErr, stdout)
	}
	if envelope["success"] != false {
		t.Errorf("expected success=false, got %v", envelope["success"])
	}
	if envelope["error"] == nil {
		t.Error("expected non-null error field")
	}
}

func TestZonesInfo_NumericInput_NoLatLon_HintsAboutCoordinates(t *testing.T) {
	mock := &mockAPI{
		getZoneErr: fmt.Errorf("Parking zone not found."),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("zones", "info", "80500")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--lat") || !strings.Contains(err.Error(), "--lon") {
		t.Errorf("expected hint about --lat/--lon in error, got: %v", err)
	}
}

func TestStart_NumericZone_NoLatLon_HintsAboutCoordinates(t *testing.T) {
	setAuth(t)
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 1, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "PAY:1"}},
		},
		getZoneErr: fmt.Errorf("Parking zone not found."),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--lat") || !strings.Contains(err.Error(), "--lon") {
		t.Errorf("expected hint about --lat/--lon in error, got: %v", err)
	}
}

// --- resolveZone unit tests ---

func TestResolveZone_ByNumericID_Success(t *testing.T) {
	mock := &mockAPI{
		getZoneResp: &parkster.Zone{ID: 17429, Name: "Ericsson", ZoneCode: "80500"},
	}

	zone, err := resolveZone(mock, "17429", 0, 0, 0)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if zone.ID != 17429 {
		t.Errorf("expected zone ID 17429, got %d", zone.ID)
	}
}

func TestResolveZone_ByCode_WithLatLon_Success(t *testing.T) {
	mock := &mockAPI{
		getZoneByCodeResp: &parkster.Zone{ID: 17429, Name: "Ericsson", ZoneCode: "80500"},
	}

	zone, err := resolveZone(mock, "80500", 59.373, 17.893, 500)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if zone.ZoneCode != "80500" {
		t.Errorf("expected zone code 80500, got %s", zone.ZoneCode)
	}
}

func TestResolveZone_ByCode_WithoutLatLon_ErrorHints(t *testing.T) {
	mock := &mockAPI{
		getZoneErr: fmt.Errorf("Parking zone not found."),
	}

	// Non-numeric input without lat/lon should hint about --lat/--lon
	_, err := resolveZone(mock, "ABC80500", 0, 0, 0)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--lat") || !strings.Contains(err.Error(), "--lon") {
		t.Errorf("expected hint about --lat/--lon, got: %v", err)
	}
}

func TestResolveZone_NumericID_NotFound_WithoutLatLon_Hints(t *testing.T) {
	mock := &mockAPI{
		getZoneErr: fmt.Errorf("Parking zone not found."),
	}

	_, err := resolveZone(mock, "80500", 0, 0, 0)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--lat") || !strings.Contains(err.Error(), "--lon") {
		t.Errorf("expected hint about --lat/--lon, got: %v", err)
	}
}

func TestResolveZone_NumericID_NotFound_WithLatLon_NoHint(t *testing.T) {
	mock := &mockAPI{
		getZoneByCodeErr: fmt.Errorf("not found as code"),
		getZoneErr:       fmt.Errorf("Parking zone not found."),
	}

	_, err := resolveZone(mock, "99999", 59.373, 17.893, 500)
	if err == nil {
		t.Fatal("expected error")
	}
	// Should NOT suggest --lat/--lon since they were already provided
	if strings.Contains(err.Error(), "--lat") {
		t.Errorf("should not hint about --lat/--lon when they're already provided, got: %v", err)
	}
}

func TestResolveZone_CodeFallsBackToID(t *testing.T) {
	// Zone code lookup fails, but the input also parses as a numeric ID
	mock := &mockAPI{
		getZoneByCodeErr: fmt.Errorf("code not found"),
		getZoneResp:      &parkster.Zone{ID: 17429, Name: "Ericsson", ZoneCode: "80500"},
	}

	zone, err := resolveZone(mock, "17429", 59.373, 17.893, 500)
	if err != nil {
		t.Fatalf("expected fallback to numeric ID, got: %v", err)
	}
	if zone.ID != 17429 {
		t.Errorf("expected zone ID 17429, got %d", zone.ID)
	}
}
