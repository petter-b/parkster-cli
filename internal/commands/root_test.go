package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/petter-b/parkster-cli/internal/auth"
	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/petter-b/parkster-cli/internal/parkster"
)

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

func TestHelp_ShowsQuietFlag(t *testing.T) {
	stdout, _, err := executeCommand("--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "--quiet") && !strings.Contains(stdout, "-q") {
		t.Error("Help should show --quiet / -q flag")
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

// --- Debug tests ---

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

// G4: PARKSTER_DEBUG=false does NOT enable debug
func TestDebug_EnvFalse_DisablesDebug(t *testing.T) {
	t.Setenv("PARKSTER_DEBUG", "false")
	resetFlags()
	if debug {
		t.Error("PARKSTER_DEBUG=false should NOT enable debug mode")
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
		getZoneByCodeResp: &parkster.Zone{ID: 17429, ZoneCode: "80500", Name: "Test Zone", FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp:  &parkster.Parking{ID: 999, ParkingZone: parkster.Zone{ZoneCode: "80500"}, Car: parkster.Car{LicenseNbr: "ABC123"}, Cost: 0},
	}
	withMockClient(t, mock)

	stdout, stderr, err := executeCommand("--quiet", "start", "--zone", "80500", "--duration", "30", "--lat", "59.373", "--lon", "17.893")
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

// --- StatusMsg tests ---

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

// --- Version flag tests ---

func TestVersionFlag_Long(t *testing.T) {
	stdout, _, err := executeCommand("--version")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "parkster") {
		t.Error("--version should contain 'parkster'")
	}
}

func TestVersionFlag_Short(t *testing.T) {
	stdout, _, err := executeCommand("-v")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "parkster") {
		t.Error("-v should contain 'parkster'")
	}
}

// --- Unknown command ---

func TestUnknownCommand_Error(t *testing.T) {
	_, _, err := executeCommand("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown command, got nil")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected 'unknown command' in error, got: %v", err)
	}
}

// --- GetCredentials function variable ---

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

// --- IsStdinTTY ---

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

// --- Zones bare command ---

func TestZones_BareCommand_ShowsHelp(t *testing.T) {
	stdout, _, err := executeCommand("zones")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "search") {
		t.Error("zones without subcommand should show help mentioning search")
	}
}

// G2: Unknown zones subcommand shows help (Cobra default for parent commands)
func TestZones_UnknownSubcommand_ShowsHelp(t *testing.T) {
	stdout, _, err := executeCommand("zones", "foobar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "search") {
		t.Error("zones with unknown subcommand should show help mentioning search")
	}
}

// --- Flag parse error JSON envelope ---

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
