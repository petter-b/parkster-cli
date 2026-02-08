package commands

import (
	"fmt"
	"os"

	"github.com/petter-b/parkster-cli/internal/auth"
	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/petter-b/parkster-cli/internal/parkster"
	"github.com/spf13/cobra"
)

var extendCmd = &cobra.Command{
	Use:   "extend",
	Short: "Extend parking duration",
	Long: `Add more time to an active parking session.

Examples:
  parkster extend --minutes 30                      # Auto-selects if only one active
  parkster extend --minutes 30 --parking-id 123456  # Extend specific session`,
	RunE: runExtend,
}

func init() {
	rootCmd.AddCommand(extendCmd)
	extendCmd.Flags().Int("minutes", 0, "Minutes to add (required)")
	extendCmd.Flags().Int("parking-id", 0, "Parking session ID (auto-selects if only one active)")
	_ = extendCmd.MarkFlagRequired("minutes")
}

func runExtend(cmd *cobra.Command, args []string) error {
	minutes, _ := cmd.Flags().GetInt("minutes")
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
		_ = output.PrintSuccess(parkings, OutputMode())
		return fmt.Errorf("multiple active parkings found, use --parking-id flag to specify")
	}

	debugLog("extending parking session %d by %d minutes", parkingID, minutes)

	parking, err := client.ExtendParking(parkingID, minutes)
	if err != nil {
		return fmt.Errorf("failed to extend parking: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Parking extended successfully\n")

	return output.PrintSuccess(parking, OutputMode())
}
