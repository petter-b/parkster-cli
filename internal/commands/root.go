package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yourorg/mycli/internal/config"
)

var (
	// Global flags
	debug      bool
	format     string
	configPath string

	// Config
	cfg *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "mycli",
	Short: "A CLI tool following steipete patterns",
	Long: `mycli is a command-line tool built with Go and Cobra.

It demonstrates best practices from github.com/steipete's CLI ecosystem:
- Secure credential storage via OS keychain
- JSON output for AI agent integration
- OAuth2 browser flow authentication
- XDG-compliant configuration`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Load config before any command runs
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			debugLog("config load warning: %v", err)
			cfg = config.Default()
		}

		// Apply config defaults if flags not explicitly set
		if !cmd.Flags().Changed("format") && cfg.OutputFormat != "" {
			format = cfg.OutputFormat
		}

		return nil
	},
}

func init() {
	// Global flags available to all commands
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug output to stderr")
	rootCmd.PersistentFlags().StringVar(&format, "format", "plain", "Output format: plain|json|tsv")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to config file")

	// Environment variable bindings
	if val := os.Getenv("MYCLI_FORMAT"); val != "" && format == "plain" {
		format = val
	}
	if os.Getenv("MYCLI_DEBUG") == "1" || os.Getenv("MYCLI_DEBUG") == "true" {
		debug = true
	}
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// debugLog prints to stderr if debug mode is enabled
func debugLog(format string, args ...any) {
	if debug {
		fmt.Fprintf(os.Stderr, "DEBUG: "+format+"\n", args...)
	}
}

// GetFormat returns the current output format
func GetFormat() string {
	return format
}

// GetDebug returns whether debug mode is enabled
func GetDebug() bool {
	return debug
}

// GetConfig returns the loaded configuration
func GetConfig() *config.Config {
	return cfg
}
