package commands

import (
	"fmt"

	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/spf13/cobra"
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

	username, password, _, err := getCredentials()
	if err != nil {
		return authRequiredError()
	}

	client := newAPIClient(username, password)

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
		if OutputMode() == output.ModeJSON {
			return output.PrintSuccess([]any{}, OutputMode())
		}
		statusMsg("No active parkings")
		return nil
	} else if len(parkings) == 1 {
		parkingID = parkings[0].ID
	} else {
		fmt.Println(output.FormatParkingList(parkings))
		return fmt.Errorf("multiple active parkings found, use --parking-id flag to specify")
	}

	debugLog("stopping parking session %d", parkingID)

	parking, err := client.StopParking(parkingID)
	if err != nil {
		return fmt.Errorf("failed to stop parking: %w", err)
	}

	mode := OutputMode()
	if mode != output.ModeHuman {
		return output.PrintSuccess(parking, mode)
	}
	statusMsg("Parking stopped")
	fmt.Println(output.FormatParkingStopped(*parking))
	return nil
}
