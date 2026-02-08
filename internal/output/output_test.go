package output

import (
	"bytes"
	"encoding/json"
	"os"
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
