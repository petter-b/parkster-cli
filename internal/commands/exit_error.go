package commands

import "errors"

// Exit codes for agent automation.
const (
	ExitGeneral  = 1 // General/unknown error
	ExitUsage    = 2 // Invalid flags, missing args, ambiguous selection
	ExitAuth     = 3 // No credentials, invalid credentials
	ExitAPI      = 4 // Network failure, HTTP 5xx, timeout
	ExitNotFound = 5 // Zone/parking not found
)

// ExitError wraps an error with an exit code.
// When Silent is true, Execute() skips printing (message was already output).
type ExitError struct {
	Code   int
	Err    error
	Silent bool
}

func (e *ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return ""
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

// ExitCode extracts the exit code from an error.
// Returns 0 for nil, the code for *ExitError, or ExitGeneral for plain errors.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}
	return ExitGeneral
}
