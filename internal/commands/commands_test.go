package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/petter-b/parkster-cli/internal/output"
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

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	bufOut.ReadFrom(rOut)
	bufErr.ReadFrom(rErr)

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

func TestHelp_ExtendCommand(t *testing.T) {
	stdout, _, err := executeCommand("extend", "--help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "--minutes") {
		t.Error("extend help should show --minutes flag")
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

func TestExtend_MissingMinutes_Error(t *testing.T) {
	_, _, err := executeCommand("extend")
	if err == nil {
		t.Error("extend without --minutes should return error")
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
