package commands

import (
	"fmt"
	"time"

	"github.com/petter-b/parkster-cli/internal/auth"
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
	RunE: runChange,
}

func init() {
	rootCmd.AddCommand(changeCmd)
	changeCmd.Flags().Int("duration", 0, "Set end time to now + N minutes")
	changeCmd.Flags().String("until", "", "Set end time to HH:MM today")
	changeCmd.Flags().Int("parking-id", 0, "Parking session ID (auto-selects if only one active)")
}

// parseUntil parses "HH:MM" and returns the target time today in local timezone.
func parseUntil(s string) (time.Time, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time format %q (expected HH:MM)", s)
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

	username, password, err := auth.GetCredentials(cmd)
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	client := newAPIClient(username, password)

	user, err := client.Login()
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	parkings := user.ShortTermParkings

	if len(parkings) == 0 {
		if OutputMode() == output.ModeJSON {
			return output.PrintSuccess([]any{}, OutputMode())
		}
		fmt.Println("No active parkings")
		return nil
	}

	// Select parking
	var parkingIdx int
	if parkingIDFlag != 0 {
		found := false
		for i, p := range parkings {
			if p.ID == parkingIDFlag {
				parkingIdx = i
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("parking session not found: %d", parkingIDFlag)
		}
	} else if len(parkings) == 1 {
		parkingIdx = 0
	} else {
		fmt.Println(output.FormatParkingList(parkings))
		return fmt.Errorf("multiple active parkings found, use --parking-id flag to specify")
	}

	selected := parkings[parkingIdx]

	// Compute desired end time
	var desiredEnd time.Time
	now := time.Now()
	if hasDuration {
		desiredEnd = now.Add(time.Duration(duration) * time.Minute)
	} else {
		desiredEnd, err = parseUntil(until)
		if err != nil {
			return err
		}
		if desiredEnd.Before(now) {
			return fmt.Errorf("--until time %s is in the past", until)
		}
	}

	// Compute offset in minutes: desired_end - current_end
	currentEnd := time.UnixMilli(selected.TimeoutTime)
	offsetMinutes := int(desiredEnd.Sub(currentEnd).Minutes())

	debugLog("changing parking %d end time by %d minutes", selected.ID, offsetMinutes)

	parking, err := client.ExtendParking(selected.ID, offsetMinutes)
	if err != nil {
		return fmt.Errorf("failed to change parking: %w", err)
	}

	mode := OutputMode()
	if mode != output.ModeHuman {
		return output.PrintSuccess(parking, mode)
	}
	fmt.Println("Parking changed")
	fmt.Println(output.FormatParkingChanged(*parking))
	return nil
}
