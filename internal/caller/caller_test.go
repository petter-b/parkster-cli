package caller

import (
	"os"
	"testing"
)

func TestDetect_ReturnsNonEmpty(t *testing.T) {
	info := Detect()
	if info.Name == "" {
		t.Error("expected Detect() to return a non-empty Name when run from go test")
	}
	if info.PID == 0 {
		t.Error("expected Detect() to return a non-zero PID")
	}
}

func TestDetect_PIDIsValid(t *testing.T) {
	info := Detect()
	// The PID should be a positive number
	if info.PID < 1 {
		t.Errorf("expected positive PID, got %d", info.PID)
	}
}

func TestSkipNames_ContainsShells(t *testing.T) {
	expected := []string{"zsh", "bash", "sh", "/bin/zsh", "/bin/bash", "/bin/sh"}
	for _, name := range expected {
		if !skipNames[name] {
			t.Errorf("expected skipNames to contain %q", name)
		}
	}
}

func TestSkipNames_ContainsGo(t *testing.T) {
	if !skipNames["go"] {
		t.Error("expected skipNames to contain 'go'")
	}
}

func TestSkipNames_ContainsNode(t *testing.T) {
	if !skipNames["node"] {
		t.Error("expected skipNames to contain 'node'")
	}
}

func TestProcessName_CurrentPID(t *testing.T) {
	name := processName(os.Getpid())
	if name == "" {
		t.Error("expected processName for current PID to return non-empty")
	}
}

func TestProcessName_InvalidPID(t *testing.T) {
	// PID 0 or negative should return empty
	name := processName(0)
	if name != "" {
		t.Errorf("expected empty name for PID 0, got %q", name)
	}
}

func TestParentPID_CurrentProcess(t *testing.T) {
	ppid := parentPID(os.Getpid())
	if ppid == 0 {
		t.Error("expected parentPID for current process to return non-zero")
	}
	// Should match os.Getppid()
	if ppid != os.Getppid() {
		t.Errorf("expected parentPID=%d, got %d", os.Getppid(), ppid)
	}
}

func TestParentPID_InvalidPID(t *testing.T) {
	ppid := parentPID(0)
	if ppid != 0 {
		t.Errorf("expected 0 for invalid PID, got %d", ppid)
	}
}

func TestInfo_ZeroValue(t *testing.T) {
	var info Info
	if info.Name != "" {
		t.Error("zero Info should have empty Name")
	}
	if info.PID != 0 {
		t.Error("zero Info should have zero PID")
	}
}
