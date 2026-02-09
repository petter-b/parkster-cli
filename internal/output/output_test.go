package output

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// captureStdout runs fn and returns what it wrote to stdout
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

// --- Mode tests ---

func TestModeFromFlags_DefaultHuman(t *testing.T) {
	mode := ModeFromFlags(false, false)
	if mode != ModeHuman {
		t.Errorf("Expected ModeHuman, got %v", mode)
	}
}

func TestModeFromFlags_JSON(t *testing.T) {
	mode := ModeFromFlags(true, false)
	if mode != ModeJSON {
		t.Errorf("Expected ModeJSON, got %v", mode)
	}
}

func TestModeFromFlags_Plain(t *testing.T) {
	mode := ModeFromFlags(false, true)
	if mode != ModePlain {
		t.Errorf("Expected ModePlain, got %v", mode)
	}
}

func TestModeFromFlags_JSONTakesPrecedence(t *testing.T) {
	mode := ModeFromFlags(true, true)
	if mode != ModeJSON {
		t.Errorf("Expected ModeJSON when both set, got %v", mode)
	}
}

// --- JSON envelope tests ---

type testItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestPrintSuccess_JSON_Envelope(t *testing.T) {
	item := testItem{ID: 1, Name: "test"}

	out := captureStdout(t, func() {
		_ = PrintSuccess(item, ModeJSON)
	})

	var envelope Envelope
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("Failed to parse JSON envelope: %v\nOutput: %s", err, out)
	}

	if !envelope.Success {
		t.Error("Expected success=true")
	}
	if envelope.Error != nil {
		t.Error("Expected error=null")
	}
	if envelope.Data == nil {
		t.Error("Expected data to be non-null")
	}
}

func TestPrintSuccess_JSON_NilData(t *testing.T) {
	out := captureStdout(t, func() {
		_ = PrintSuccess(nil, ModeJSON)
	})

	var envelope Envelope
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	if !envelope.Success {
		t.Error("Expected success=true")
	}
}

func TestPrintSuccess_JSON_Slice(t *testing.T) {
	items := []testItem{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}

	out := captureStdout(t, func() {
		_ = PrintSuccess(items, ModeJSON)
	})

	var envelope Envelope
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("Failed to parse JSON: %v\nOutput: %s", err, out)
	}
	if !envelope.Success {
		t.Error("Expected success=true")
	}
}

func TestPrintError_JSON_Envelope(t *testing.T) {
	out := captureStdout(t, func() {
		PrintError("something went wrong", ModeJSON)
	})

	var envelope Envelope
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("Failed to parse JSON: %v\nOutput: %s", err, out)
	}

	if envelope.Success {
		t.Error("Expected success=false")
	}
	if envelope.Data != nil {
		t.Error("Expected data=null")
	}
	if envelope.Error == nil {
		t.Error("Expected non-null error")
	}

	errStr, ok := envelope.Error.(string)
	if !ok {
		t.Fatalf("Expected error to be string, got %T", envelope.Error)
	}
	if errStr != "something went wrong" {
		t.Errorf("Expected error 'something went wrong', got %s", errStr)
	}
}

func TestPrintSuccess_Human_Struct(t *testing.T) {
	item := testItem{ID: 42, Name: "hello"}

	out := captureStdout(t, func() {
		_ = PrintSuccess(item, ModeHuman)
	})

	// Human mode should print field: value lines
	if out == "" {
		t.Error("Expected non-empty output")
	}
}

func TestPrintSuccess_Plain_Struct(t *testing.T) {
	item := testItem{ID: 42, Name: "hello"}

	out := captureStdout(t, func() {
		_ = PrintSuccess(item, ModePlain)
	})

	// Plain/TSV mode should produce tab-separated values
	if out == "" {
		t.Error("Expected non-empty output")
	}
}

// captureStderr runs fn and returns what it wrote to stderr
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	fn()

	_ = w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

// --- Additional coverage tests ---

func TestPrintError_Human(t *testing.T) {
	// In human mode, PrintError should write to stderr, not stdout
	stdoutOut := captureStdout(t, func() {
		stderrOut := captureStderr(t, func() {
			PrintError("bad thing happened", ModeHuman)
		})
		if stderrOut == "" {
			t.Error("Expected output on stderr, got empty")
		}
		if stderrOut != "Error: bad thing happened\n" {
			t.Errorf("Expected 'Error: bad thing happened\\n', got %q", stderrOut)
		}
	})
	if stdoutOut != "" {
		t.Errorf("Expected no stdout output in human error mode, got %q", stdoutOut)
	}
}

func TestPrintSuccess_Human_Slice(t *testing.T) {
	items := []testItem{
		{ID: 1, Name: "alpha"},
		{ID: 2, Name: "beta"},
	}

	out := captureStdout(t, func() {
		_ = PrintSuccess(items, ModeHuman)
	})

	// Should contain both items
	if !strings.Contains(out, "alpha") {
		t.Error("Expected output to contain 'alpha'")
	}
	if !strings.Contains(out, "beta") {
		t.Error("Expected output to contain 'beta'")
	}
	// Items should be separated by a blank line
	if !strings.Contains(out, "\n\n") {
		t.Error("Expected blank line separating items in human slice output")
	}
}

func TestPrintSuccess_Plain_Slice(t *testing.T) {
	items := []testItem{
		{ID: 10, Name: "first"},
		{ID: 20, Name: "second"},
	}

	out := captureStdout(t, func() {
		_ = PrintSuccess(items, ModePlain)
	})

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 TSV rows, got %d: %q", len(lines), out)
	}

	// Each line should be tab-separated
	if !strings.Contains(lines[0], "\t") {
		t.Errorf("Expected tab-separated values in first row, got %q", lines[0])
	}
	if !strings.Contains(lines[0], "10") || !strings.Contains(lines[0], "first") {
		t.Errorf("First row should contain '10' and 'first', got %q", lines[0])
	}
	if !strings.Contains(lines[1], "20") || !strings.Contains(lines[1], "second") {
		t.Errorf("Second row should contain '20' and 'second', got %q", lines[1])
	}
}

func TestPrintSuccess_Human_EmptyFields(t *testing.T) {
	// Name is zero value (empty string), should be omitted
	item := testItem{ID: 7, Name: ""}

	out := captureStdout(t, func() {
		_ = PrintSuccess(item, ModeHuman)
	})

	if !strings.Contains(out, "id: 7") {
		t.Errorf("Expected 'id: 7' in output, got %q", out)
	}
	if strings.Contains(out, "name") {
		t.Errorf("Expected zero-value 'name' field to be omitted, got %q", out)
	}
}

func TestPrintSuccess_Human_Pointer(t *testing.T) {
	item := &testItem{ID: 99, Name: "pointer-test"}

	out := captureStdout(t, func() {
		_ = PrintSuccess(item, ModeHuman)
	})

	if !strings.Contains(out, "99") {
		t.Errorf("Expected output to contain '99', got %q", out)
	}
	if !strings.Contains(out, "pointer-test") {
		t.Errorf("Expected output to contain 'pointer-test', got %q", out)
	}
}

func TestPrintSuccess_Plain_NonStruct(t *testing.T) {
	out := captureStdout(t, func() {
		_ = PrintSuccess("just a string", ModePlain)
	})

	trimmed := strings.TrimSpace(out)
	if trimmed != "just a string" {
		t.Errorf("Expected 'just a string', got %q", trimmed)
	}
}

func TestPrintSuccess_Human_NonStruct(t *testing.T) {
	out := captureStdout(t, func() {
		_ = PrintSuccess("hello world", ModeHuman)
	})

	trimmed := strings.TrimSpace(out)
	if trimmed != "hello world" {
		t.Errorf("Expected 'hello world', got %q", trimmed)
	}
}
