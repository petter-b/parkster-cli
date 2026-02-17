package commands

import (
	"bytes"
	"os"
	"testing"

	"github.com/petter-b/parkster-cli/internal/auth"
	"github.com/petter-b/parkster-cli/internal/caller"
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
func executeCommand(args ...string) (stdout, stderr string, err error) {
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
func executeCommandFull(args ...string) (stdout, stderr string, err error) {
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
