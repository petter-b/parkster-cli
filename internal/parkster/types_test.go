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
			{"id": 67890, "licenseNbr": "ABC123", "countryCode": "SE", "carPersonalization": {"name": "Volkswagen"}}
		],
		"paymentAccounts": [
			{"paymentAccountId": "pay_123"}
		],
		"shortTermParkings": [
			{
				"id": 999,
				"parkingZone": {"id": 17429, "name": "Ericsson"},
				"car": {"id": 67890, "licenseNbr": "ABC123", "carPersonalization": {"name": "Volkswagen"}},
				"checkInTime": 1707400000000,
				"timeoutTime": 1707401800000
			}
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
	if user.Cars[0].CarPersonalization.Name != "Volkswagen" {
		t.Errorf("Expected car name 'Volkswagen', got %s", user.Cars[0].CarPersonalization.Name)
	}
	if len(user.PaymentAccounts) != 1 {
		t.Fatalf("Expected 1 payment account, got %d", len(user.PaymentAccounts))
	}
	if user.PaymentAccounts[0].PaymentAccountID != "pay_123" {
		t.Errorf("Expected payment ID pay_123, got %s", user.PaymentAccounts[0].PaymentAccountID)
	}
	if len(user.ShortTermParkings) != 1 {
		t.Fatalf("Expected 1 short term parking, got %d", len(user.ShortTermParkings))
	}
	if user.ShortTermParkings[0].ID != 999 {
		t.Errorf("Expected parking ID 999, got %d", user.ShortTermParkings[0].ID)
	}
	if user.ShortTermParkings[0].CheckInTime != 1707400000000 {
		t.Errorf("Expected checkInTime 1707400000000, got %d", user.ShortTermParkings[0].CheckInTime)
	}
	if user.ShortTermParkings[0].TimeoutTime != 1707401800000 {
		t.Errorf("Expected timeoutTime 1707401800000, got %d", user.ShortTermParkings[0].TimeoutTime)
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
			"name": "Ericsson, Kista",
			"zoneCode": "80500",
			"feeZone": {
				"id": 27545,
				"currency": {"code": "SEK", "symbol": "kr"}
			}
		},
		"car": {
			"id": 67890,
			"licenseNbr": "ABC123",
			"countryCode": "SE",
			"carPersonalization": {"name": "Volkswagen"}
		},
		"checkInTime": 1707400000000,
		"timeoutTime": 1707401800000,
		"cost": 10.50,
		"totalCost": 10.50,
		"currency": {"code": "SEK", "symbol": "kr"}
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
	if parking.Car.CarPersonalization.Name != "Volkswagen" {
		t.Errorf("Expected car name 'Volkswagen', got %s", parking.Car.CarPersonalization.Name)
	}
	if parking.CheckInTime != 1707400000000 {
		t.Errorf("Expected checkInTime 1707400000000, got %d", parking.CheckInTime)
	}
	if parking.TimeoutTime != 1707401800000 {
		t.Errorf("Expected timeoutTime 1707401800000, got %d", parking.TimeoutTime)
	}
	if parking.Cost != 10.50 {
		t.Errorf("Expected cost 10.50, got %f", parking.Cost)
	}
	if parking.TotalCost != 10.50 {
		t.Errorf("Expected totalCost 10.50, got %f", parking.TotalCost)
	}
	if parking.Currency.Code != "SEK" {
		t.Errorf("Expected currency SEK, got %s", parking.Currency.Code)
	}
}

// --- Zone search type tests (Task 1) ---

func TestSearchResult_JSONUnmarshal(t *testing.T) {
	// Realistic JSON from API.md location-search response
	jsonData := `{
		"parkingZonesAtPosition": [
			{
				"id": 17429,
				"name": "Ericsson Kista",
				"zoneCode": "80500",
				"city": {"name": "Stockholm"},
				"latitude": 59.404833,
				"longitude": 17.953333
			}
		],
		"parkingZonesNearbyPosition": [
			{
				"id": 7713,
				"name": "Berlin Zone",
				"zoneCode": "100028",
				"city": {"name": "Berlin"},
				"latitude": 52.520008,
				"longitude": 13.404954,
				"distance": 150
			}
		]
	}`

	var result SearchResult
	err := json.Unmarshal([]byte(jsonData), &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal search result: %v", err)
	}

	// Verify parkingZonesAtPosition
	if len(result.ParkingZonesAtPosition) != 1 {
		t.Fatalf("Expected 1 zone at position, got %d", len(result.ParkingZonesAtPosition))
	}
	atPos := result.ParkingZonesAtPosition[0]
	if atPos.ID != 17429 {
		t.Errorf("Expected ID 17429, got %d", atPos.ID)
	}
	if atPos.Name != "Ericsson Kista" {
		t.Errorf("Expected name 'Ericsson Kista', got %s", atPos.Name)
	}
	if atPos.ZoneCode != "80500" {
		t.Errorf("Expected zoneCode '80500', got %s", atPos.ZoneCode)
	}
	if atPos.City.Name != "Stockholm" {
		t.Errorf("Expected city 'Stockholm', got %s", atPos.City.Name)
	}
	if atPos.Latitude != 59.404833 {
		t.Errorf("Expected latitude 59.404833, got %f", atPos.Latitude)
	}
	if atPos.Longitude != 17.953333 {
		t.Errorf("Expected longitude 17.953333, got %f", atPos.Longitude)
	}

	// Verify parkingZonesNearbyPosition
	if len(result.ParkingZonesNearbyPosition) != 1 {
		t.Fatalf("Expected 1 zone nearby, got %d", len(result.ParkingZonesNearbyPosition))
	}
	nearby := result.ParkingZonesNearbyPosition[0]
	if nearby.ID != 7713 {
		t.Errorf("Expected ID 7713, got %d", nearby.ID)
	}
	if nearby.ZoneCode != "100028" {
		t.Errorf("Expected zoneCode '100028', got %s", nearby.ZoneCode)
	}
	if nearby.City.Name != "Berlin" {
		t.Errorf("Expected city 'Berlin', got %s", nearby.City.Name)
	}
	if nearby.Distance != 150 {
		t.Errorf("Expected distance 150, got %d", nearby.Distance)
	}
}

func TestZoneDetail_JSONUnmarshal(t *testing.T) {
	// Realistic JSON from API.md zone detail response
	jsonData := `{
		"id": 17429,
		"name": "Ericsson Kista",
		"zoneCode": "80500",
		"city": {"name": "Stockholm"},
		"latitude": 59.404833,
		"longitude": 17.953333,
		"feeZone": {
			"id": 27545,
			"currency": {"code": "SEK", "symbol": "kr"},
			"parkingFees": [
				{
					"amountPerHour": 10.0,
					"description": "Mon-Fri 08:00-18:00",
					"startTime": 480,
					"endTime": 1080
				},
				{
					"amountPerHour": 0.0,
					"description": "Evenings and weekends",
					"startTime": 1080,
					"endTime": 480
				}
			]
		}
	}`

	var zone Zone
	err := json.Unmarshal([]byte(jsonData), &zone)
	if err != nil {
		t.Fatalf("Failed to unmarshal zone detail: %v", err)
	}

	if zone.ID != 17429 {
		t.Errorf("Expected ID 17429, got %d", zone.ID)
	}
	if zone.ZoneCode != "80500" {
		t.Errorf("Expected zoneCode '80500', got %s", zone.ZoneCode)
	}
	if zone.City.Name != "Stockholm" {
		t.Errorf("Expected city 'Stockholm', got %s", zone.City.Name)
	}
	if zone.Latitude != 59.404833 {
		t.Errorf("Expected latitude 59.404833, got %f", zone.Latitude)
	}
	if zone.Longitude != 17.953333 {
		t.Errorf("Expected longitude 17.953333, got %f", zone.Longitude)
	}
	if zone.FeeZone.ID != 27545 {
		t.Errorf("Expected fee zone ID 27545, got %d", zone.FeeZone.ID)
	}
	if len(zone.FeeZone.ParkingFees) != 2 {
		t.Fatalf("Expected 2 parking fees, got %d", len(zone.FeeZone.ParkingFees))
	}
	if zone.FeeZone.ParkingFees[0].AmountPerHour != 10.0 {
		t.Errorf("Expected amountPerHour 10.0, got %f", zone.FeeZone.ParkingFees[0].AmountPerHour)
	}
	if zone.FeeZone.ParkingFees[0].Description != "Mon-Fri 08:00-18:00" {
		t.Errorf("Expected description 'Mon-Fri 08:00-18:00', got %s", zone.FeeZone.ParkingFees[0].Description)
	}
	if zone.FeeZone.ParkingFees[0].StartTime != 480 {
		t.Errorf("Expected startTime 480, got %d", zone.FeeZone.ParkingFees[0].StartTime)
	}
	if zone.FeeZone.ParkingFees[0].EndTime != 1080 {
		t.Errorf("Expected endTime 1080, got %d", zone.FeeZone.ParkingFees[0].EndTime)
	}
}

func TestZoneSearchItem_JSONUnmarshal(t *testing.T) {
	// Test with distance field (nearby result)
	jsonWithDistance := `{
		"id": 7713,
		"name": "Berlin Zone",
		"zoneCode": "100028",
		"city": {"name": "Berlin"},
		"latitude": 52.520008,
		"longitude": 13.404954,
		"distance": 150
	}`

	var itemWithDistance ZoneSearchItem
	err := json.Unmarshal([]byte(jsonWithDistance), &itemWithDistance)
	if err != nil {
		t.Fatalf("Failed to unmarshal zone with distance: %v", err)
	}
	if itemWithDistance.Distance != 150 {
		t.Errorf("Expected distance 150, got %d", itemWithDistance.Distance)
	}

	// Test without distance field (at-position result)
	jsonWithoutDistance := `{
		"id": 17429,
		"name": "Ericsson Kista",
		"zoneCode": "80500",
		"city": {"name": "Stockholm"},
		"latitude": 59.404833,
		"longitude": 17.953333
	}`

	var itemWithoutDistance ZoneSearchItem
	err = json.Unmarshal([]byte(jsonWithoutDistance), &itemWithoutDistance)
	if err != nil {
		t.Fatalf("Failed to unmarshal zone without distance: %v", err)
	}
	if itemWithoutDistance.ID != 17429 {
		t.Errorf("Expected ID 17429, got %d", itemWithoutDistance.ID)
	}
	if itemWithoutDistance.Distance != 0 {
		t.Errorf("Expected distance 0 (omitted), got %d", itemWithoutDistance.Distance)
	}
}

// --- Favorite zone type test ---

func TestUser_UnmarshalFavoriteZones(t *testing.T) {
	raw := `{
		"id": 1,
		"email": "test@example.com",
		"accountType": "NEUTRAL",
		"cars": [],
		"paymentAccounts": [],
		"shortTermParkings": [],
		"favoriteZones": [
			{"id": 17429, "name": "Ericsson", "zoneCode": "80500", "city": {"name": "Kista"}}
		]
	}`
	var user User
	if err := json.Unmarshal([]byte(raw), &user); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(user.FavoriteZones) != 1 {
		t.Fatalf("expected 1 favorite zone, got %d", len(user.FavoriteZones))
	}
	fz := user.FavoriteZones[0]
	if fz.ID != 17429 {
		t.Errorf("expected zone ID 17429, got %d", fz.ID)
	}
	if fz.Name != "Ericsson" {
		t.Errorf("expected zone name Ericsson, got %s", fz.Name)
	}
	if fz.ZoneCode != "80500" {
		t.Errorf("expected zone code 80500, got %s", fz.ZoneCode)
	}
	if fz.City.Name != "Kista" {
		t.Errorf("expected city Kista, got %s", fz.City.Name)
	}
}

// --- Cost estimate type test ---

func TestCostEstimate_JSONUnmarshal(t *testing.T) {
	// Test the CostEstimate type can be unmarshaled from JSON
	jsonData := `{
		"amount": 15.0,
		"currency": "SEK"
	}`

	var estimate CostEstimate
	err := json.Unmarshal([]byte(jsonData), &estimate)
	if err != nil {
		t.Fatalf("Failed to unmarshal cost estimate: %v", err)
	}

	if estimate.Amount != 15.0 {
		t.Errorf("Expected amount 15.0, got %f", estimate.Amount)
	}
	if estimate.Currency != "SEK" {
		t.Errorf("Expected currency 'SEK', got %s", estimate.Currency)
	}
}
