//go:build integration

package parkster

import (
	"os"
	"testing"
)

func integrationClient(t *testing.T) *Client {
	t.Helper()

	username := os.Getenv("PARKSTER_USERNAME")
	password := os.Getenv("PARKSTER_PASSWORD")
	if username == "" || password == "" {
		t.Skip("PARKSTER_USERNAME and PARKSTER_PASSWORD must be set")
	}

	return NewClient(username, password)
}

func TestIntegration_Login(t *testing.T) {
	client := integrationClient(t)

	user, err := client.Login()
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if user.ID == 0 {
		t.Error("Expected non-zero user ID")
	}
	if len(user.Cars) == 0 {
		t.Error("Expected at least one car")
	}
	if len(user.PaymentAccounts) == 0 {
		t.Error("Expected at least one payment account")
	}

	t.Logf("User ID: %d, Email: %s, Cars: %d, Payments: %d",
		user.ID, user.Email, len(user.Cars), len(user.PaymentAccounts))
}

func TestIntegration_GetZone_Sweden(t *testing.T) {
	client := integrationClient(t)

	zone, err := client.GetZone(17429)
	if err != nil {
		t.Fatalf("GetZone(17429) failed: %v", err)
	}

	if zone.ID != 17429 {
		t.Errorf("Expected zone ID 17429, got %d", zone.ID)
	}
	if zone.FeeZone.ID == 0 {
		t.Error("Expected non-zero feeZone ID")
	}
	if zone.FeeZone.Currency.Code != "SEK" {
		t.Errorf("Expected currency SEK, got %s", zone.FeeZone.Currency.Code)
	}

	t.Logf("Zone: %s (ID: %d), FeeZone: %d, Currency: %s",
		zone.Name, zone.ID, zone.FeeZone.ID, zone.FeeZone.Currency.Code)
}

func TestIntegration_GetZone_Germany(t *testing.T) {
	client := integrationClient(t)

	zone, err := client.GetZone(7713)
	if err != nil {
		t.Fatalf("GetZone(7713) failed: %v", err)
	}

	if zone.ID != 7713 {
		t.Errorf("Expected zone ID 7713, got %d", zone.ID)
	}
	if zone.FeeZone.ID == 0 {
		t.Error("Expected non-zero feeZone ID")
	}
	if zone.FeeZone.Currency.Code != "EUR" {
		t.Errorf("Expected currency EUR, got %s", zone.FeeZone.Currency.Code)
	}

	t.Logf("Zone: %s (ID: %d), FeeZone: %d, Currency: %s",
		zone.Name, zone.ID, zone.FeeZone.ID, zone.FeeZone.Currency.Code)
}

func TestIntegration_GetZone_Austria(t *testing.T) {
	client := integrationClient(t)

	zone, err := client.GetZone(25624)
	if err != nil {
		t.Fatalf("GetZone(25624) failed: %v", err)
	}

	if zone.ID != 25624 {
		t.Errorf("Expected zone ID 25624, got %d", zone.ID)
	}
	if zone.FeeZone.ID == 0 {
		t.Error("Expected non-zero feeZone ID")
	}
	if zone.FeeZone.Currency.Code != "EUR" {
		t.Errorf("Expected currency EUR, got %s", zone.FeeZone.Currency.Code)
	}

	t.Logf("Zone: %s (ID: %d), FeeZone: %d, Currency: %s",
		zone.Name, zone.ID, zone.FeeZone.ID, zone.FeeZone.Currency.Code)
}

func TestIntegration_LoginFailure(t *testing.T) {
	client := NewClient("invalid@example.com", "wrongpassword")

	_, err := client.Login()
	if err == nil {
		t.Fatal("Expected error for invalid credentials")
	}

	t.Logf("Expected error: %v", err)
}
