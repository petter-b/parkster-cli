package parkster

import (
	"encoding/json"
	"testing"
)

func TestUser_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"id": 12345,
		"email": "test@example.com",
		"accountType": "PRIVATE",
		"cars": [
			{"id": 67890, "licenseNbr": "ABC123", "countryCode": "SE"}
		],
		"paymentAccounts": [
			{"paymentAccountId": "pay_123"}
		]
	}`

	var user User
	err := json.Unmarshal([]byte(jsonData), &user)
	if err != nil {
		t.Fatalf("Failed to unmarshal user: %v", err)
	}

	if user.ID != 12345 {
		t.Errorf("Expected ID 12345, got %d", user.ID)
	}
	if user.Email != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", user.Email)
	}
	if user.AccountType != "PRIVATE" {
		t.Errorf("Expected accountType PRIVATE, got %s", user.AccountType)
	}
	if len(user.Cars) != 1 {
		t.Fatalf("Expected 1 car, got %d", len(user.Cars))
	}
	if user.Cars[0].LicenseNbr != "ABC123" {
		t.Errorf("Expected license ABC123, got %s", user.Cars[0].LicenseNbr)
	}
	if len(user.PaymentAccounts) != 1 {
		t.Fatalf("Expected 1 payment account, got %d", len(user.PaymentAccounts))
	}
	if user.PaymentAccounts[0].PaymentAccountID != "pay_123" {
		t.Errorf("Expected payment ID pay_123, got %s", user.PaymentAccounts[0].PaymentAccountID)
	}
}

func TestZone_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"id": 17429,
		"name": "Ericsson Kista",
		"feeZone": {
			"id": 27545,
			"currency": {
				"code": "SEK",
				"symbol": "kr"
			}
		}
	}`

	var zone Zone
	err := json.Unmarshal([]byte(jsonData), &zone)
	if err != nil {
		t.Fatalf("Failed to unmarshal zone: %v", err)
	}

	if zone.ID != 17429 {
		t.Errorf("Expected ID 17429, got %d", zone.ID)
	}
	if zone.Name != "Ericsson Kista" {
		t.Errorf("Expected name 'Ericsson Kista', got %s", zone.Name)
	}
	if zone.FeeZone.ID != 27545 {
		t.Errorf("Expected fee zone ID 27545, got %d", zone.FeeZone.ID)
	}
	if zone.FeeZone.Currency.Code != "SEK" {
		t.Errorf("Expected currency SEK, got %s", zone.FeeZone.Currency.Code)
	}
	if zone.FeeZone.Currency.Symbol != "kr" {
		t.Errorf("Expected symbol 'kr', got %s", zone.FeeZone.Currency.Symbol)
	}
}

func TestParking_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"id": 123456,
		"parkingZone": {
			"id": 17429,
			"name": "Test Zone",
			"feeZone": {
				"id": 27545,
				"currency": {"code": "SEK", "symbol": "kr"}
			}
		},
		"car": {
			"id": 67890,
			"licenseNbr": "ABC123",
			"countryCode": "SE"
		},
		"startTime": "2026-02-08T12:00:00Z",
		"timeout": 30,
		"cost": 10.50,
		"status": "ACTIVE"
	}`

	var parking Parking
	err := json.Unmarshal([]byte(jsonData), &parking)
	if err != nil {
		t.Fatalf("Failed to unmarshal parking: %v", err)
	}

	if parking.ID != 123456 {
		t.Errorf("Expected ID 123456, got %d", parking.ID)
	}
	if parking.ParkingZone.ID != 17429 {
		t.Errorf("Expected zone ID 17429, got %d", parking.ParkingZone.ID)
	}
	if parking.Car.LicenseNbr != "ABC123" {
		t.Errorf("Expected license ABC123, got %s", parking.Car.LicenseNbr)
	}
	if parking.StartTime != "2026-02-08T12:00:00Z" {
		t.Errorf("Expected startTime 2026-02-08T12:00:00Z, got %s", parking.StartTime)
	}
	if parking.Timeout != 30 {
		t.Errorf("Expected timeout 30, got %d", parking.Timeout)
	}
	if parking.Cost != 10.50 {
		t.Errorf("Expected cost 10.50, got %f", parking.Cost)
	}
	if parking.Status != "ACTIVE" {
		t.Errorf("Expected status ACTIVE, got %s", parking.Status)
	}
}
