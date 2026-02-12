package commands

import (
	"fmt"

	"github.com/petter-b/parkster-cli/internal/auth"
	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show active parking sessions",
	Long: `Display all active parking sessions.

Examples:
  parkster status
  parkster status --json`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	username, password, err := auth.GetCredentials(cmd)
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	client := newAPIClient(username, password)

	debugLog("fetching active parkings")

	user, err := client.Login()
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	parkings := user.ShortTermParkings

	debugLog("found %d active parkings", len(parkings))

	if len(parkings) == 0 {
		if OutputMode() == output.ModeJSON {
			return output.PrintSuccess([]any{}, OutputMode())
		}
		fmt.Println("No active parkings")
		return nil
	}

	mode := OutputMode()
	if mode != output.ModeHuman {
		return output.PrintSuccess(parkings, mode)
	}
	fmt.Println(output.FormatParkingList(parkings))
	return nil
}
