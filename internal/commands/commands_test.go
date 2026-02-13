package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

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
	plainFlag = false
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

// executeCommand runs a command with args and captures stdout/stderr
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
	if !strings.Contains(stdout, "--plain") {
		t.Error("Help should show --plain flag")
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
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 500, TimeoutTime: time.Now().Add(30 * time.Minute).UnixMilli()},
			},
		},
	}
	withMockClient(t, mock)

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

func TestOutputMode_Plain(t *testing.T) {
	resetFlags()
	plainFlag = true
	if OutputMode() != output.ModePlain {
		t.Error("OutputMode should return ModePlain when plainFlag is set")
	}
}

// --- Mock API client ---

type mockAPI struct {
	loginResp         *parkster.User
	loginErr          error
	getZoneResp       *parkster.Zone
	getZoneErr        error
	startParkingResp  *parkster.Parking
	startParkingErr   error
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

func (m *mockAPI) ExtendParking(_, _ int) (*parkster.Parking, error) {
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

// setAuth sets the environment variables for authentication.
func setAuth(t *testing.T) {
	t.Helper()
	t.Setenv("PARKSTER_USERNAME", "testuser")
	t.Setenv("PARKSTER_PASSWORD", "testpass")
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

	stdout, _, err := executeCommand("start", "--zone", "17429", "--duration", "30")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stdout, "Parking started") {
		t.Errorf("expected 'Parking started' in output, got: %q", stdout)
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

	stdout, _, err := executeCommand("stop")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stdout, "Parking stopped") {
		t.Errorf("expected 'Parking stopped' in output, got: %q", stdout)
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

	stdout, _, err := executeCommand("stop")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(stdout, "No active parkings") {
		t.Errorf("expected 'No active parkings' in output, got: %q", stdout)
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
	stdout, _, err := executeCommand("change", "--duration", "60")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stdout, "Parking changed") {
		t.Errorf("expected 'Parking changed' in output, got: %q", stdout)
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

	stdout, _, err := executeCommand("change", "--duration", "30")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(stdout, "No active parkings") {
		t.Errorf("expected 'No active parkings' in output, got: %q", stdout)
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

	stdout, _, err := executeCommand("status")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stdout, "No active parkings") {
		t.Errorf("expected 'No active parkings' in output, got: %q", stdout)
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
	t.Setenv("PARKSTER_USERNAME", "testuser@example.com")
	t.Setenv("PARKSTER_PASSWORD", "testpass")

	stdout, _, err := executeCommand("auth", "status")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stdout, "Logged in as: testuser@example.com") {
		t.Errorf("expected 'Logged in as: testuser@example.com' in output, got: %q", stdout)
	}
}

func TestAuthStatus_WithoutCredentials_NotAuthenticated(t *testing.T) {
	// When no env vars are set, auth.GetCredentials falls through to keyring
	// which can block on macOS waiting for Keychain access prompt.
	if runtime.GOOS == "darwin" {
		t.Skip("skipping: macOS Keychain may block in test environment")
	}

	t.Setenv("PARKSTER_USERNAME", "")
	t.Setenv("PARKSTER_PASSWORD", "")

	stdout, _, err := executeCommand("auth", "status")
	if err != nil {
		t.Fatalf("expected success (not authenticated is not an error), got: %v", err)
	}
	if !strings.Contains(stdout, "Not authenticated") {
		t.Errorf("expected 'Not authenticated' in output, got: %q", stdout)
	}
}

func TestAuthStatus_JSON_Envelope(t *testing.T) {
	t.Setenv("PARKSTER_USERNAME", "testuser@example.com")
	t.Setenv("PARKSTER_PASSWORD", "testpass")

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

	stdout, _, err := executeCommand("zones", "search", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success with empty results, got: %v", err)
	}
	if !strings.Contains(stdout, "No zones found") {
		t.Errorf("expected 'No zones found' in output, got: %q", stdout)
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

func TestZonesInfo_Success_Plain(t *testing.T) {
	mock := &mockAPI{
		getZoneByCodeResp: &parkster.Zone{
			ID:       80500,
			Name:     "Ericsson Kista",
			ZoneCode: "80500",
			City:     parkster.City{Name: "Stockholm"},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("zones", "info", "80500", "--lat", "59.373", "--lon", "17.893", "--plain")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stdout, "80500") {
		t.Errorf("expected zone data in plain output, got: %q", stdout)
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

func TestStart_DryRun_Plain(t *testing.T) {
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
			FeeZone:  parkster.FeeZone{ID: 27545, Currency: parkster.Currency{Code: "SEK"}},
		},
		estimateCostResp: &parkster.CostEstimate{
			Amount:   15.0,
			Currency: "SEK",
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("start", "--zone", "17429", "--duration", "30", "--dry-run", "--plain")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	// Plain mode should contain tab-separated values with cost info
	if !strings.Contains(stdout, "15") {
		t.Errorf("expected cost in plain output, got: %q", stdout)
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
