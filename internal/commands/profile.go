package commands

import (
	"fmt"

	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/petter-b/parkster-cli/internal/parkster"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Show account info, cars, payments, and favorite zones",
	Long: `Display account details for the authenticated user.

Shows registered cars, payment methods, and favorite zones.
Useful for agents to discover available options before starting parking.

Examples:
  parkster profile
  parkster profile --json`,
	Args: cobra.NoArgs,
	RunE: runProfile,
}

func init() {
	rootCmd.AddCommand(profileCmd)
}

// profileData is the JSON output shape for the profile command.
// Excludes shortTermParkings (use 'status' for that).
type profileData struct {
	Username      string                `json:"username"`
	AccountType   string                `json:"accountType"`
	Cars          []profileCar          `json:"cars"`
	Payments      []profilePayment      `json:"payments"`
	FavoriteZones []profileFavoriteZone `json:"favoriteZones"`
}

type profileCar struct {
	LicenseNbr  string `json:"licenseNbr"`
	Name        string `json:"name,omitempty"`
	CountryCode string `json:"countryCode,omitempty"`
}

type profilePayment struct {
	ID string `json:"id"`
}

type profileFavoriteZone struct {
	ZoneCode string `json:"zoneCode"`
	Name     string `json:"name"`
	City     string `json:"city,omitempty"`
}

func runProfile(cmd *cobra.Command, args []string) error {
	username, password, _, err := getCredentials()
	if err != nil {
		return authRequiredError()
	}

	client := newAPIClient(username, password)

	debugLog("fetching profile")

	user, err := client.Login()
	if err != nil {
		return &ExitError{Code: ExitAPI, Err: fmt.Errorf("failed to fetch profile: %w", err)}
	}

	mode := OutputMode()
	if mode != output.ModeHuman {
		return output.PrintSuccess(buildProfileData(user), mode)
	}

	fmt.Println(output.FormatProfile(user.Email, user.AccountType, user.Cars, user.PaymentAccounts, user.FavoriteZones))
	return nil
}

func buildProfileData(user *parkster.User) profileData {
	cars := make([]profileCar, len(user.Cars))
	for i, c := range user.Cars {
		cars[i] = profileCar{
			LicenseNbr:  c.LicenseNbr,
			Name:        c.CarPersonalization.Name,
			CountryCode: c.CountryCode,
		}
	}

	payments := make([]profilePayment, len(user.PaymentAccounts))
	for i, p := range user.PaymentAccounts {
		payments[i] = profilePayment{ID: p.PaymentAccountID}
	}

	favorites := make([]profileFavoriteZone, len(user.FavoriteZones))
	for i, z := range user.FavoriteZones {
		favorites[i] = profileFavoriteZone{
			ZoneCode: z.ZoneCode,
			Name:     z.Name,
			City:     z.City.Name,
		}
	}

	return profileData{
		Username:      user.Email,
		AccountType:   user.AccountType,
		Cars:          cars,
		Payments:      payments,
		FavoriteZones: favorites,
	}
}
