package commands

import (
	"fmt"
	"os"
	"strconv"

	"github.com/petter-b/parkster-cli/internal/auth"
	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/petter-b/parkster-cli/internal/parkster"
	"github.com/spf13/cobra"
)

// DryRunResult holds the simulation output for --dry-run mode
type DryRunResult struct {
	Zone     string  `json:"zone"`
	ZoneCode string  `json:"zoneCode"`
	ZoneName string  `json:"zoneName"`
	Car      string  `json:"car"`
	Payment  string  `json:"payment"`
	Duration int     `json:"duration"`
	Cost     float64 `json:"cost,omitempty"`
	Currency string  `json:"currency,omitempty"`
	DryRun   bool    `json:"dryRun"`
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a parking session",
	Long: `Start a new parking session in a specific zone.

Examples:
  parkster start --zone 80500 --duration 30 --lat 59.373 --lon 17.893
  parkster start --zone 17429 --duration 30
  parkster start --dry-run --zone 80500 --duration 30 --lat 59.373 --lon 17.893`,
	RunE: runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().String("zone", "", "Parking zone code or ID (e.g., 80500, 17429)")
	startCmd.Flags().Int("duration", 30, "Parking duration in minutes")
	startCmd.Flags().String("car", "", "License plate (auto-selects if only one car)")
	startCmd.Flags().String("payment", "", "Payment account ID (auto-selects if only one)")
	startCmd.Flags().Bool("dry-run", false, "Simulate parking flow without starting (shows cost estimate)")
	startCmd.Flags().Float64("lat", 0, "Latitude for zone code lookup")
	startCmd.Flags().Float64("lon", 0, "Longitude for zone code lookup")
	_ = startCmd.MarkFlagRequired("zone")
}

// resolveZone attempts to resolve a zone from either a zone code or numeric ID.
// If lat/lon are provided, it tries zone code lookup first via GetZoneByCode.
// Otherwise, it tries to parse the input as a numeric ID and calls GetZone.
func resolveZone(client parkster.API, zoneInput string, lat, lon float64) (*parkster.Zone, error) {
	// If lat/lon provided, try zone code lookup first
	if lat != 0 && lon != 0 {
		zone, err := client.GetZoneByCode(zoneInput, lat, lon)
		if err == nil {
			return zone, nil
		}
		debugLog("zone code lookup failed: %v, trying as numeric ID", err)
	}

	// Fallback: try parsing as numeric ID
	zoneID, parseErr := strconv.Atoi(zoneInput)
	if parseErr != nil {
		if lat == 0 && lon == 0 {
			return nil, fmt.Errorf("zone code %q requires --lat and --lon flags for lookup", zoneInput)
		}
		return nil, fmt.Errorf("zone %q not found as code or ID", zoneInput)
	}

	zone, err := client.GetZone(zoneID)
	if err != nil {
		return nil, fmt.Errorf("zone %d not found: %w", zoneID, err)
	}

	return zone, nil
}

func runStart(cmd *cobra.Command, args []string) error {
	zoneInput, _ := cmd.Flags().GetString("zone")
	duration, _ := cmd.Flags().GetInt("duration")
	carFlag, _ := cmd.Flags().GetString("car")
	paymentFlag, _ := cmd.Flags().GetString("payment")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	lat, _ := cmd.Flags().GetFloat64("lat")
	lon, _ := cmd.Flags().GetFloat64("lon")

	username, err := auth.GetUsername(cmd)
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}
	password, err := auth.GetPassword(cmd)
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	client := newAPIClient(username, password)

	debugLog("authenticating as %s", username)

	user, err := client.Login()
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	debugLog("found %d cars and %d payment accounts", len(user.Cars), len(user.PaymentAccounts))

	// Select car
	var selectedCar *parkster.Car
	if carFlag != "" {
		for i := range user.Cars {
			if user.Cars[i].LicenseNbr == carFlag {
				selectedCar = &user.Cars[i]
				break
			}
		}
		if selectedCar == nil {
			return fmt.Errorf("car not found: %s", carFlag)
		}
	} else if len(user.Cars) == 1 {
		selectedCar = &user.Cars[0]
	} else if len(user.Cars) == 0 {
		return fmt.Errorf("no cars registered, add a car first")
	} else {
		_ = output.PrintSuccess(user.Cars, OutputMode())
		return fmt.Errorf("multiple cars found, use --car flag to specify license plate")
	}

	debugLog("selected car: %s", selectedCar.LicenseNbr)

	// Select payment
	var selectedPayment *parkster.PaymentAccount
	if paymentFlag != "" {
		for i := range user.PaymentAccounts {
			if user.PaymentAccounts[i].PaymentAccountID == paymentFlag {
				selectedPayment = &user.PaymentAccounts[i]
				break
			}
		}
		if selectedPayment == nil {
			return fmt.Errorf("payment account not found: %s", paymentFlag)
		}
	} else if len(user.PaymentAccounts) == 1 {
		selectedPayment = &user.PaymentAccounts[0]
	} else if len(user.PaymentAccounts) == 0 {
		return fmt.Errorf("no payment methods configured, add a payment method first")
	} else {
		_ = output.PrintSuccess(user.PaymentAccounts, OutputMode())
		return fmt.Errorf("multiple payment accounts found, use --payment flag to specify payment account ID")
	}

	debugLog("selected payment: %s", selectedPayment.PaymentAccountID)

	// Resolve zone (by code or ID)
	debugLog("resolving zone: %s", zoneInput)
	zone, err := resolveZone(client, zoneInput, lat, lon)
	if err != nil {
		return err
	}

	debugLog("resolved to zone %d (fee zone ID %d)", zone.ID, zone.FeeZone.ID)

	// Dry-run mode: simulate without starting
	if dryRun {
		debugLog("dry-run mode: estimating cost")

		result := DryRunResult{
			Zone:     fmt.Sprintf("%d", zone.ID),
			ZoneCode: zone.ZoneCode,
			ZoneName: zone.Name,
			Car:      selectedCar.LicenseNbr,
			Payment:  selectedPayment.PaymentAccountID,
			Duration: duration,
			DryRun:   true,
		}

		// Try to get cost estimate (graceful failure)
		estimate, err := client.EstimateCost(zone.ID, zone.FeeZone.ID, selectedCar.ID, selectedPayment.PaymentAccountID, duration)
		if err != nil {
			debugLog("cost estimation failed: %v", err)
		} else {
			result.Cost = estimate.Amount
			result.Currency = estimate.Currency
		}

		fmt.Fprintf(os.Stderr, "DRY RUN: Would start parking\n")
		fmt.Fprintf(os.Stderr, "  Zone: %s %s (%s)\n", zone.ZoneCode, zone.Name, fmt.Sprintf("%d", zone.ID))
		fmt.Fprintf(os.Stderr, "  Car: %s\n", selectedCar.LicenseNbr)
		fmt.Fprintf(os.Stderr, "  Duration: %d minutes\n", duration)
		if result.Cost > 0 {
			fmt.Fprintf(os.Stderr, "  Estimated cost: %.2f %s\n", result.Cost, result.Currency)
		}

		return output.PrintSuccess(result, OutputMode())
	}

	// Start parking
	debugLog("starting parking for %d minutes", duration)
	parking, err := client.StartParking(zone.ID, zone.FeeZone.ID, selectedCar.ID, selectedPayment.PaymentAccountID, duration)
	if err != nil {
		return fmt.Errorf("failed to start parking: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Parking started successfully\n")

	return output.PrintSuccess(parking, OutputMode())
}
