package commands

import (
	"fmt"
	"os"

	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/petter-b/parkster-cli/internal/parkster"
)

// selectParking authenticates, fetches active parkings, and selects one.
// If parkingIDFlag is 0, auto-selects when exactly one parking is active.
// Returns (nil, client, nil) when no active parkings exist (output already handled).
// Returns (nil, nil, err) on auth or selection errors.
func selectParking(parkingIDFlag int) (*parkster.Parking, parkster.API, error) {
	username, password, _, err := getCredentials()
	if err != nil {
		return nil, nil, authRequiredError()
	}

	client := newAPIClient(username, password)

	debugLog("fetching active parkings")

	user, err := client.Login()
	if err != nil {
		return nil, nil, &ExitError{Code: ExitAPI, Err: fmt.Errorf("failed to authenticate: %w", err)}
	}

	parkings := user.ShortTermParkings

	debugLog("found %d active parkings", len(parkings))

	if len(parkings) == 0 {
		if OutputMode() == output.ModeJSON {
			_ = output.PrintSuccess([]any{}, OutputMode())
		} else {
			statusMsg("No active parkings")
		}
		return nil, client, nil
	}

	if parkingIDFlag != 0 {
		for i, p := range parkings {
			if p.ID == parkingIDFlag {
				return &parkings[i], client, nil
			}
		}
		return nil, nil, &ExitError{Code: ExitNotFound, Err: fmt.Errorf("parking session not found: %d", parkingIDFlag)}
	}

	if len(parkings) == 1 {
		return &parkings[0], client, nil
	}

	// Multiple parkings — show list and error
	msg := "multiple active parkings found, use --parking-id flag to specify"
	if OutputMode() != output.ModeHuman {
		fmt.Fprintln(os.Stderr, output.FormatParkingList(parkings))
		output.PrintError(msg, OutputMode())
		return nil, nil, &ExitError{Code: ExitUsage, Silent: true}
	}
	fmt.Println(output.FormatParkingList(parkings))
	return nil, nil, &ExitError{Code: ExitUsage, Err: fmt.Errorf("%s", msg)}
}
