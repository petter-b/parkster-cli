package commands

import (
	"fmt"
	"os"

	"github.com/petter-b/parkster-cli/internal/config"
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
- Start, stop, and extend parking sessions
- View active parking status
- JSON output for AI agent integration
- Secure credential storage via OS keychain`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		_, err := config.Load("")
		if err != nil {
			debugLog("config load warning: %v", err)
		}
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug output to stderr")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output as JSON with envelope")
	rootCmd.PersistentFlags().BoolVar(&plainFlag, "plain", false, "Output as tab-separated values")

	// Global credential flags
	rootCmd.PersistentFlags().String("username", "", "Parkster account username (email or phone number)")
	rootCmd.PersistentFlags().String("password", "", "Parkster account password")

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
