package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/petter-b/parkster-cli/internal/auth"
	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/petter-b/parkster-cli/internal/parkster"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a parking session",
	Long: `Stop an active parking session.

Examples:
  parkster stop                      # Auto-stops if only one active
  parkster stop --parking-id 123456  # Stop specific session`,
	RunE: runStop,
}

func init() {
	rootCmd.AddCommand(stopCmd)
	stopCmd.Flags().Int("parking-id", 0, "Parking session ID (auto-selects if only one active)")
}

func runStop(cmd *cobra.Command, args []string) error {
	parkingIDFlag, _ := cmd.Flags().GetInt("parking-id")

	username, err := auth.GetUsername(cmd)
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}
	password, err := auth.GetPassword(cmd)
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	client := parkster.NewClient(username, password)

	debugLog("fetching active parkings")

	user, err := client.Login()
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	parkings := user.ShortTermParkings

	debugLog("found %d active parkings", len(parkings))

	var parkingID int
	if parkingIDFlag != 0 {
		parkingID = parkingIDFlag
		found := false
		for _, p := range parkings {
			if p.ID == parkingID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("parking session not found: %d", parkingID)
		}
	} else if len(parkings) == 0 {
		return fmt.Errorf("no active parking sessions")
	} else if len(parkings) == 1 {
		parkingID = parkings[0].ID
	} else {
		output.PrintSuccess(parkings, OutputMode())
		return fmt.Errorf("multiple active parkings found. Use --parking-id flag to specify")
	}

	debugLog("stopping parking session %d", parkingID)

	parking, err := client.StopParking(parkingID)
	if err != nil {
		return fmt.Errorf("failed to stop parking: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Parking stopped successfully\n")

	return output.PrintSuccess(parking, OutputMode())
}
