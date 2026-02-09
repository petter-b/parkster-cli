package commands

import (
	"fmt"

	"github.com/petter-b/parkster-cli/internal/auth"
	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/spf13/cobra"
)

var zonesCmd = &cobra.Command{
	Use:   "zones",
	Short: "Manage parking zones",
	Long:  "Search for and view parking zone information.",
}

var zonesSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search for parking zones near a location",
	Long: `Search for parking zones near GPS coordinates.

Returns zones at your exact position and nearby zones within the search radius.

Examples:
  parkster zones search --lat 59.373 --lon 17.893
  parkster zones search --lat 52.52 --lon 13.40 --radius 500 --json`,
	RunE: runZonesSearch,
}

func init() {
	rootCmd.AddCommand(zonesCmd)
	zonesCmd.AddCommand(zonesSearchCmd)

	// Flags for zones search
	zonesSearchCmd.Flags().Float64("lat", 0, "Latitude (required)")
	zonesSearchCmd.Flags().Float64("lon", 0, "Longitude (required)")
	zonesSearchCmd.Flags().Int("radius", 250, "Search radius in meters")
	_ = zonesSearchCmd.MarkFlagRequired("lat")
	_ = zonesSearchCmd.MarkFlagRequired("lon")
}

func runZonesSearch(cmd *cobra.Command, args []string) error {
	lat, _ := cmd.Flags().GetFloat64("lat")
	lon, _ := cmd.Flags().GetFloat64("lon")
	radius, _ := cmd.Flags().GetInt("radius")

	// Validate GPS coordinates
	if lat < -90 || lat > 90 {
		return fmt.Errorf("invalid latitude: must be between -90 and 90")
	}
	if lon < -180 || lon > 180 {
		return fmt.Errorf("invalid longitude: must be between -180 and 180")
	}

	// Auth
	username, err := auth.GetUsername(cmd)
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}
	password, err := auth.GetPassword(cmd)
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	client := newAPIClient(username, password)

	debugLog("searching zones at %.6f,%.6f with radius %dm", lat, lon, radius)

	result, err := client.SearchZones(lat, lon, radius)
	if err != nil {
		return fmt.Errorf("zone search failed: %w", err)
	}

	// Merge both arrays
	allZones := append(result.ParkingZonesAtPosition, result.ParkingZonesNearbyPosition...)

	debugLog("found %d zones", len(allZones))

	// Handle empty results
	if len(allZones) == 0 {
		if OutputMode() == output.ModeJSON {
			return output.PrintSuccess([]any{}, OutputMode())
		}
		fmt.Println("No zones found")
		return nil
	}

	return output.PrintSuccess(allZones, OutputMode())
}
