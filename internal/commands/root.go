package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/petter-b/parkster-cli/internal/caller"
	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	// Global flags
	debug     bool
	jsonFlag  bool
	quietFlag bool

	// detectedCaller holds the detected calling process info (set in PersistentPreRunE).
	detectedCaller caller.Info
)

// errSilent indicates the error message was already printed.
// Execute() will skip printing but still return non-nil for os.Exit(1).
var errSilent = errors.New("")

var rootCmd = &cobra.Command{
	Use:   "parkster",
	Short: "Manage Parkster parking sessions from the terminal",
	Long: `parkster is a command-line tool for managing Parkster parking sessions.

Features:
- Start, stop, and change parking sessions
- View active parking status
- JSON output for AI agent integration
- Secure credential storage via OS keychain`,
	Example: `  parkster start --zone 80500 --duration 30
  parkster status --json
  parkster zones search --lat 59.37 --lon 17.89`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug output to stderr")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output as JSON with envelope")
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress status messages on stderr")

	// Environment variable bindings
	if os.Getenv("PARKSTER_DEBUG") == "1" || os.Getenv("PARKSTER_DEBUG") == "true" {
		debug = true
	}

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		detectedCaller = caller.Detect()
		if debug && detectedCaller.Name != "" {
			debugLog("caller=%s pid=%d", detectedCaller.Name, detectedCaller.PID)
		}
		return nil
	}
}

// Execute runs the root command, formatting errors based on output mode
func Execute() error {
	err := rootCmd.Execute()
	if err != nil && !errors.Is(err, errSilent) {
		mode := OutputMode()
		// If flag parsing failed, jsonFlag may not be set even though
		// the user passed --json. Check os.Args as a fallback.
		if mode == output.ModeHuman && hasJSONFlag(os.Args) {
			mode = output.ModeJSON
		}
		output.PrintError(err.Error(), mode)
	}
	return err
}

// hasJSONFlag checks os.Args for --json. Used as a fallback when Cobra flag
// parsing fails before it can set jsonFlag.
func hasJSONFlag(args []string) bool {
	for _, a := range args {
		if a == "--json" {
			return true
		}
	}
	return false
}

// OutputMode returns the current output mode based on flags
func OutputMode() output.Mode {
	return output.ModeFromFlags(jsonFlag)
}

// debugLog prints to stderr if debug mode is enabled
func debugLog(format string, args ...any) {
	if debug {
		fmt.Fprintf(os.Stderr, "DEBUG: "+format+"\n", args...)
	}
}

// isStderrTTY reports whether stderr is connected to a terminal.
// Declared as a variable so tests can override it.
var isStderrTTY = func() bool {
	return term.IsTerminal(int(os.Stderr.Fd()))
}

// isStdinTTY reports whether stdin is connected to a terminal.
// Declared as a variable so tests can override it.
var isStdinTTY = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// statusMsg prints a status message to stderr in human mode.
// Suppressed when: --quiet, --json, or stderr is not a TTY.
func statusMsg(format string, args ...any) {
	if quietFlag || jsonFlag || !isStderrTTY() {
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// authRequiredError prints a friendly no-auth message and returns errSilent.
// Used by commands that require credentials.
func authRequiredError() error {
	mode := OutputMode()
	if mode != output.ModeHuman {
		output.PrintError("not authenticated", mode)
	} else {
		fmt.Fprintln(os.Stderr, "Not authenticated. Use 'parkster auth login' or set PARKSTER_USERNAME/PARKSTER_PASSWORD.")
	}
	return errSilent
}
