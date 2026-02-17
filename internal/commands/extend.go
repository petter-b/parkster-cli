package commands

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/spf13/cobra"
)

var changeCmd = &cobra.Command{
	Use:   "change",
	Short: "Change parking end time",
	Long: `Change the end time of an active parking session.

Specify the new end time using --duration (minutes from now) or --until (HH:MM today).
One of --duration or --until is required.

Examples:
  parkster change --duration 60                      # End 60 min from now
  parkster change --until 18:30                      # End at 18:30 today
  parkster change --duration 60 --parking-id 123456  # Change specific session`,
	Args: cobra.NoArgs,
	RunE: runChange,
}

func init() {
	rootCmd.AddCommand(changeCmd)
	changeCmd.Flags().Int("duration", 0, "Set end time to now + N minutes")
	changeCmd.Flags().String("until", "", "Set end time to HH:MM today")
	changeCmd.Flags().Int("parking-id", 0, "Parking session ID (auto-selects if only one active)")
}

// parseUntil parses time in "HH:MM", "HH.MM", or bare "HH" format.
// Returns the target time today in local timezone.
func parseUntil(s string) (time.Time, error) {
	// Normalize dot separator to colon
	normalized := strings.ReplaceAll(s, ".", ":")

	// Try HH:MM
	t, err := time.Parse("15:04", normalized)
	if err != nil {
		// Try bare hour (e.g. "17" or "9")
		t, err = time.Parse("15", s)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid time format %q (expected HH:MM, HH.MM, or HH)", s)
		}
	}

	now := time.Now()
	target := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
	return target, nil
}

func runChange(cmd *cobra.Command, args []string) error {
	duration, _ := cmd.Flags().GetInt("duration")
	until, _ := cmd.Flags().GetString("until")
	parkingIDFlag, _ := cmd.Flags().GetInt("parking-id")

	// Validate: exactly one of --duration or --until
	hasDuration := cmd.Flags().Changed("duration")
	hasUntil := until != ""
	if hasDuration && hasUntil {
		return fmt.Errorf("--duration and --until are mutually exclusive")
	}
	if !hasDuration && !hasUntil {
		return fmt.Errorf("one of --duration or --until is required")
	}
	if hasDuration && duration <= 0 {
		return fmt.Errorf("--duration must be a positive number of minutes")
	}

	// Parse --until early (before auth) to fail fast on invalid input
	var desiredEnd time.Time
	if hasDuration {
		desiredEnd = time.Now().Add(time.Duration(duration) * time.Minute)
	} else {
		var parseErr error
		desiredEnd, parseErr = parseUntil(until)
		if parseErr != nil {
			return parseErr
		}
		if desiredEnd.Before(time.Now()) {
			return fmt.Errorf("--until time %s is in the past", until)
		}
	}

	selected, client, err := selectParking(parkingIDFlag)
	if err != nil {
		return err
	}
	if selected == nil {
		return nil // no active parkings (already handled)
	}

	// Compute offset in minutes: total duration from parking start to desired end
	startTime := time.UnixMilli(selected.CheckInTime)
	offsetMinutes := int(math.Round(desiredEnd.Sub(startTime).Minutes()))

	debugLog("changing parking %d total duration to %d minutes", selected.ID, offsetMinutes)

	parking, err := client.ExtendParking(selected.ID, offsetMinutes)
	if err != nil {
		return fmt.Errorf("failed to change parking: %w", err)
	}

	// Merge updated fields from extend response into the original parking
	// (extend API only returns timeoutTime, cost, currency — not zone/car)
	selected.TimeoutTime = parking.TimeoutTime
	selected.Cost = parking.Cost
	selected.TotalCost = parking.TotalCost
	selected.Currency = parking.Currency

	mode := OutputMode()
	if mode != output.ModeHuman {
		return output.PrintSuccess(selected, mode)
	}
	statusMsg("Parking changed")
	fmt.Println(output.FormatParkingChanged(*selected))
	return nil
}
