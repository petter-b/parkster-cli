package commands

import (
	"errors"
	"fmt"
	"testing"
)

func TestExitError_Code(t *testing.T) {
	err := &ExitError{Code: ExitAuth, Err: errors.New("not authenticated")}
	if err.Code != 3 {
		t.Errorf("expected code 3, got %d", err.Code)
	}
}

func TestExitError_Error(t *testing.T) {
	err := &ExitError{Code: ExitAuth, Err: errors.New("not authenticated")}
	if err.Error() != "not authenticated" {
		t.Errorf("expected 'not authenticated', got %q", err.Error())
	}
}

func TestExitError_Unwrap(t *testing.T) {
	inner := errors.New("inner")
	err := &ExitError{Code: ExitAuth, Err: inner}
	if !errors.Is(err, inner) {
		t.Error("expected Unwrap to return inner error")
	}
}

func TestExitError_Silent(t *testing.T) {
	err := &ExitError{Code: ExitAuth, Silent: true}
	if !err.Silent {
		t.Error("expected Silent to be true")
	}
}

func TestExitCode_ExtractsCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil error", nil, 0},
		{"plain error", errors.New("oops"), ExitGeneral},
		{"exit error", &ExitError{Code: ExitAuth, Err: errors.New("auth")}, ExitAuth},
		{"wrapped exit error", fmt.Errorf("wrapped: %w", &ExitError{Code: ExitAPI, Err: errors.New("api")}), ExitAPI},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExitCode(tt.err); got != tt.want {
				t.Errorf("ExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestAuthRequiredError_ExitCode(t *testing.T) {
	err := authRequiredError()
	if ExitCode(err) != ExitAuth {
		t.Errorf("expected exit code %d (auth), got %d", ExitAuth, ExitCode(err))
	}
}

func TestExecute_ExitError_PreservesCode(t *testing.T) {
	// Trigger a usage error (unknown flag) — Cobra returns these directly
	_, _, err := executeCommandFull("--bogus-flag")
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if ExitCode(err) != ExitUsage {
		t.Errorf("expected exit code %d (usage), got %d", ExitUsage, ExitCode(err))
	}
}
