package commands

import (
	"fmt"
	"os"
	"strconv"

	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/petter-b/parkster-cli/internal/parkster"
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

var zonesInfoCmd = &cobra.Command{
	Use:   "info <zone-code>",
	Short: "Show details for a parking zone by sign code",
	Long: `Look up a parking zone by its sign code (the code on the parking sign).

Requires --lat and --lon flags to search for the zone code near your location.
Numeric zone IDs are also accepted as a fallback but are deprecated.

Examples:
  parkster zones info 80500 --lat 59.373 --lon 17.893
  parkster zones info 100028 --lat 52.52 --lon 13.40 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runZonesInfo,
}

func init() {
	rootCmd.AddCommand(zonesCmd)
	zonesCmd.AddCommand(zonesSearchCmd)
	zonesCmd.AddCommand(zonesInfoCmd)

	// Flags for zones search
	zonesSearchCmd.Flags().Float64("lat", 0, "Latitude (required)")
	zonesSearchCmd.Flags().Float64("lon", 0, "Longitude (required)")
	zonesSearchCmd.Flags().Int("radius", 0, "Search radius in meters (0 = API default)")
	_ = zonesSearchCmd.MarkFlagRequired("lat")
	_ = zonesSearchCmd.MarkFlagRequired("lon")

	// Flags for zones info
	zonesInfoCmd.Flags().Float64("lat", 0, "Latitude (required for sign code lookup)")
	zonesInfoCmd.Flags().Float64("lon", 0, "Longitude (required for sign code lookup)")
}

func runZonesSearch(cmd *cobra.Command, args []string) error {
	lat, _ := cmd.Flags().GetFloat64("lat")
	lon, _ := cmd.Flags().GetFloat64("lon")
	radius, _ := cmd.Flags().GetInt("radius")

	if radius < 0 {
		return fmt.Errorf("--radius must be non-negative")
	}

	// Validate GPS coordinates
	if lat < -90 || lat > 90 {
		return fmt.Errorf("invalid latitude: must be between -90 and 90")
	}
	if lon < -180 || lon > 180 {
		return fmt.Errorf("invalid longitude: must be between -180 and 180")
	}

	client := newAPIClient("", "")

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
		fmt.Fprintln(os.Stderr, "No zones found")
		return nil
	}

	mode := OutputMode()
	if mode != output.ModeHuman {
		return output.PrintSuccess(allZones, mode)
	}
	fmt.Println(output.FormatZoneSearchList(allZones))
	return nil
}

func runZonesInfo(cmd *cobra.Command, args []string) error {
	zoneInput := args[0]
	lat, _ := cmd.Flags().GetFloat64("lat")
	lon, _ := cmd.Flags().GetFloat64("lon")

	client := newAPIClient("", "")

	// If lat/lon provided, try zone code lookup first
	if lat != 0 && lon != 0 {
		debugLog("looking up zone code %q near %.6f,%.6f", zoneInput, lat, lon)
		zone, err := client.GetZoneByCode(zoneInput, lat, lon, 0)
		if err == nil {
			debugLog("found zone %d: %s", zone.ID, zone.Name)
			return printZoneInfo(zone)
		}
		debugLog("zone code lookup failed: %v, trying as numeric ID", err)
	}

	// Fallback: try parsing as numeric zone ID
	zoneID, parseErr := strconv.Atoi(zoneInput)
	if parseErr != nil {
		if lat == 0 && lon == 0 {
			return fmt.Errorf("zone code %q requires --lat and --lon flags for lookup", zoneInput)
		}
		return fmt.Errorf("zone %q not found as code or ID", zoneInput)
	}

	debugLog("looking up zone by ID %d", zoneID)
	zone, err := client.GetZone(zoneID)
	if err != nil {
		return fmt.Errorf("zone not found: %w", err)
	}

	debugLog("found zone %d: %s", zone.ID, zone.Name)
	return printZoneInfo(zone)
}

// printZoneInfo outputs zone details using the custom formatter for human mode
func printZoneInfo(zone *parkster.Zone) error {
	mode := OutputMode()
	if mode != output.ModeHuman {
		return output.PrintSuccess(zone, mode)
	}
	fmt.Println(output.FormatZoneInfo(*zone))
	return nil
}
