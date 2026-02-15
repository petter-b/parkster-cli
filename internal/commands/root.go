package commands

import (
	"fmt"
	"os"

	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	debug     bool
	jsonFlag  bool
	plainFlag bool
)

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
	rootCmd.PersistentFlags().BoolVar(&plainFlag, "plain", false, "Output as tab-separated values")

	// Environment variable bindings
	if os.Getenv("PARKSTER_DEBUG") == "1" || os.Getenv("PARKSTER_DEBUG") == "true" {
		debug = true
	}
}

// Execute runs the root command, formatting errors based on output mode
func Execute() error {
	err := rootCmd.Execute()
	if err != nil {
		output.PrintError(err.Error(), OutputMode())
	}
	return err
}

// OutputMode returns the current output mode based on flags
func OutputMode() output.Mode {
	return output.ModeFromFlags(jsonFlag, plainFlag)
}

// debugLog prints to stderr if debug mode is enabled
func debugLog(fmt_ string, args ...any) {
	if debug {
		fmt.Fprintf(os.Stderr, "DEBUG: "+fmt_+"\n", args...)
	}
}
