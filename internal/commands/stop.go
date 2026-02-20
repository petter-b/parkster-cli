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
	Args: cobra.NoArgs,
	RunE: runStop,
}

func init() {
	rootCmd.AddCommand(stopCmd)
	stopCmd.Flags().Int("parking-id", 0, "Parking session ID (auto-selects if only one active)")
}

func runStop(cmd *cobra.Command, args []string) error {
	parkingIDFlag, _ := cmd.Flags().GetInt("parking-id")

	selected, client, err := selectParking(parkingIDFlag)
	if err != nil {
		return err
	}
	if selected == nil {
		return nil // no active parkings (already handled)
	}

	debugLog("stopping parking session %d", selected.ID)

	parking, err := client.StopParking(selected.ID)
	if err != nil {
		return &ExitError{Code: ExitAPI, Err: fmt.Errorf("failed to stop parking: %w", err)}
	}

	mode := OutputMode()
	if mode != output.ModeHuman {
		return output.PrintSuccess(parking, mode)
	}
	statusMsg("Parking stopped")
	fmt.Println(output.FormatParkingStopped(*parking))
	return nil
}
