package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/petter-b/parkster-cli/internal/auth"
	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/petter-b/parkster-cli/internal/parkster"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a parking session",
	Long: `Start a new parking session in a specific zone.

Examples:
  parkster start --zone 17429 --duration 30
  parkster start --zone 17429 --duration 30 --car ABC123
  parkster start --zone 17429 --duration 30 --email user@example.com --password secret`,
	RunE: runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().Int("zone", 0, "Parking zone ID (required)")
	startCmd.Flags().Int("duration", 30, "Parking duration in minutes")
	startCmd.Flags().String("car", "", "License plate (auto-selects if only one car)")
	startCmd.Flags().String("payment", "", "Payment account ID (auto-selects if only one)")
	startCmd.MarkFlagRequired("zone")
}

func runStart(cmd *cobra.Command, args []string) error {
	zoneID, _ := cmd.Flags().GetInt("zone")
	duration, _ := cmd.Flags().GetInt("duration")
	carFlag, _ := cmd.Flags().GetString("car")
	paymentFlag, _ := cmd.Flags().GetString("payment")

	email, err := auth.GetEmail(cmd)
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}
	password, err := auth.GetPassword(cmd)
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	client := parkster.NewClient(email, password)

	debugLog("authenticating as %s", email)

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
		return fmt.Errorf("no cars registered. Add a car first.")
	} else {
		output.Print(user.Cars, GetFormat())
		return fmt.Errorf("multiple cars found. Use --car flag to specify license plate")
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
		return fmt.Errorf("no payment methods configured. Add a payment method first.")
	} else {
		output.Print(user.PaymentAccounts, GetFormat())
		return fmt.Errorf("multiple payment accounts found. Use --payment flag to specify payment account ID")
	}

	debugLog("selected payment: %s", selectedPayment.PaymentAccountID)

	// Get zone details (need fee zone ID)
	debugLog("fetching zone details for zone %d", zoneID)
	zone, err := client.GetZone(zoneID)
	if err != nil {
		return fmt.Errorf("failed to get zone details: %w", err)
	}

	debugLog("zone %d has fee zone ID %d", zone.ID, zone.FeeZone.ID)

	// Start parking
	debugLog("starting parking for %d minutes", duration)
	parking, err := client.StartParking(zone.ID, zone.FeeZone.ID, selectedCar.ID, selectedPayment.PaymentAccountID, duration)
	if err != nil {
		return fmt.Errorf("failed to start parking: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Parking started successfully\n")

	return output.Print(parking, GetFormat())
}
