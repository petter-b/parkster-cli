package commands

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/petter-b/parkster-cli/internal/output"
)

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
