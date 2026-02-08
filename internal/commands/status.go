package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/petter-b/parkster-cli/internal/auth"
	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/petter-b/parkster-cli/internal/parkster"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show active parking sessions",
	Long: `Display all active parking sessions.

Examples:
  parkster status
  parkster status --format json`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	email, err := auth.GetEmail(cmd)
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}
	password, err := auth.GetPassword(cmd)
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	client := parkster.NewClient(email, password)

	debugLog("fetching active parkings")

	parkings, err := client.GetActiveParkings()
	if err != nil {
		return fmt.Errorf("failed to get active parkings: %w", err)
	}

	debugLog("found %d active parkings", len(parkings))

	return output.Print(parkings, GetFormat())
}
