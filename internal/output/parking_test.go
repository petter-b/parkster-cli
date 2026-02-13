package output

import (
	"strings"
	"testing"
	"time"

	"github.com/petter-b/parkster-cli/internal/parkster"
)

func TestFormatParking_Status(t *testing.T) {
	now := time.Now()
	parking := parkster.Parking{
		ID: 500,
		ParkingZone: parkster.Zone{
			ID:       17429,
			Name:     "Ericsson, Kista",
			ZoneCode: "80500",
		},
		Car: parkster.Car{
			ID:                 100,
			LicenseNbr:         "ABC123",
			CarPersonalization: parkster.CarPersonalization{Name: "Volkswagen"},
		},
		CheckInTime: now.Add(-30 * time.Minute).UnixMilli(),
		TimeoutTime: now.Add(2*time.Hour + 9*time.Minute).UnixMilli(),
		Cost:        0,
		Currency:    parkster.Currency{Code: "SEK", Symbol: "kr"},
	}

	out := FormatParking(parking)

	if !strings.Contains(out, "80500") {
		t.Errorf("expected zone code '80500' in output, got: %q", out)
	}
	if !strings.Contains(out, "Ericsson, Kista") {
		t.Errorf("expected zone name in output, got: %q", out)
	}
	if !strings.Contains(out, "Volkswagen") {
		t.Errorf("expected car name in output, got: %q", out)
	}
	if !strings.Contains(out, "ABC123") {
		t.Errorf("expected license plate in output, got: %q", out)
	}
	if !strings.Contains(out, "SEK") && !strings.Contains(out, "kr") {
		t.Errorf("expected currency in output, got: %q", out)
	}
	// Should NOT contain internal IDs
	if strings.Contains(out, "17429") {
		t.Errorf("should not contain zone ID 17429 in human output, got: %q", out)
	}
	if strings.Contains(out, " 500") {
		t.Errorf("should not contain parking ID 500 in human output, got: %q", out)
	}
}

func TestFormatParking_ContainsTimeInfo(t *testing.T) {
	now := time.Now()
	parking := parkster.Parking{
		ParkingZone: parkster.Zone{ZoneCode: "80500", Name: "Test"},
		Car:         parkster.Car{LicenseNbr: "ABC123"},
		CheckInTime: now.Add(-10 * time.Minute).UnixMilli(),
		TimeoutTime: now.Add(50 * time.Minute).UnixMilli(),
		Currency:    parkster.Currency{Code: "SEK"},
	}

	out := FormatParking(parking)

	if !strings.Contains(out, "Valid from:") {
		t.Errorf("expected 'Valid from:' in output, got: %q", out)
	}
	if !strings.Contains(out, "Ends at:") {
		t.Errorf("expected 'Ends at:' in output, got: %q", out)
	}
	if !strings.Contains(out, "remaining") {
		t.Errorf("expected 'remaining' in time output, got: %q", out)
	}
}

func TestFormatParkingStopped(t *testing.T) {
	parking := parkster.Parking{
		ParkingZone: parkster.Zone{ZoneCode: "80500", Name: "Ericsson, Kista"},
		Car:         parkster.Car{LicenseNbr: "ABC123", CarPersonalization: parkster.CarPersonalization{Name: "Volkswagen"}},
		Cost:        0,
		Currency:    parkster.Currency{Code: "SEK"},
	}

	out := FormatParkingStopped(parking)

	if !strings.Contains(out, "80500") {
		t.Errorf("expected zone code in output, got: %q", out)
	}
	if !strings.Contains(out, "ABC123") {
		t.Errorf("expected license plate in output, got: %q", out)
	}
	// Stopped format should NOT show "Valid from" or "Ends at"
	if strings.Contains(out, "Valid from:") {
		t.Errorf("stopped parking should not show 'Valid from:', got: %q", out)
	}
}

func TestFormatParkingList(t *testing.T) {
	now := time.Now()
	parkings := []parkster.Parking{
		{
			ParkingZone: parkster.Zone{ZoneCode: "80500", Name: "Ericsson, Kista"},
			Car:         parkster.Car{LicenseNbr: "ABC123"},
			CheckInTime: now.Add(-10 * time.Minute).UnixMilli(),
			TimeoutTime: now.Add(50 * time.Minute).UnixMilli(),
			Currency:    parkster.Currency{Code: "SEK"},
		},
		{
			ParkingZone: parkster.Zone{ZoneCode: "90100", Name: "Solna Centrum"},
			Car:         parkster.Car{LicenseNbr: "XYZ789"},
			CheckInTime: now.Add(-5 * time.Minute).UnixMilli(),
			TimeoutTime: now.Add(25 * time.Minute).UnixMilli(),
			Currency:    parkster.Currency{Code: "SEK"},
		},
	}

	out := FormatParkingList(parkings)

	if !strings.Contains(out, "80500") {
		t.Errorf("expected first zone code in output, got: %q", out)
	}
	if !strings.Contains(out, "90100") {
		t.Errorf("expected second zone code in output, got: %q", out)
	}
	// Multiple parkings should be separated by blank line
	if !strings.Contains(out, "\n\n") {
		t.Errorf("expected blank line separator between parkings, got: %q", out)
	}
}

func TestFormatZoneSearchList(t *testing.T) {
	zones := []parkster.ZoneSearchItem{
		{ID: 17429, Name: "Ericsson", ZoneCode: "80500", City: parkster.City{Name: "Kista"}},
		{ID: 6388, Name: "MC inom Taxa 3", ZoneCode: "13", City: parkster.City{Name: "Stockholm"}},
	}

	out := FormatZoneSearchList(zones)

	if !strings.Contains(out, "80500") {
		t.Errorf("expected zone code 80500, got: %q", out)
	}
	if !strings.Contains(out, "Ericsson") {
		t.Errorf("expected zone name, got: %q", out)
	}
	if !strings.Contains(out, "Kista") {
		t.Errorf("expected city name, got: %q", out)
	}
	// Must NOT contain internal IDs or curly braces
	if strings.Contains(out, "17429") {
		t.Errorf("should not contain internal zone ID, got: %q", out)
	}
	if strings.Contains(out, "{") {
		t.Errorf("should not contain curly braces, got: %q", out)
	}
}

func TestFormatZoneInfo(t *testing.T) {
	zone := parkster.Zone{
		ID:       17429,
		Name:     "Ericsson",
		ZoneCode: "80500",
		City:     parkster.City{Name: "Kista"},
		FeeZone: parkster.FeeZone{
			ID:       27545,
			Currency: parkster.Currency{Code: "SEK", Symbol: "kr"},
			ParkingFees: []parkster.ParkingFee{
				{AmountPerHour: 10.0, Description: "Weekdays 8-18", StartTime: 480, EndTime: 1080},
			},
		},
	}

	out := FormatZoneInfo(zone)

	if !strings.Contains(out, "80500") {
		t.Errorf("expected zone code, got: %q", out)
	}
	if !strings.Contains(out, "Ericsson") {
		t.Errorf("expected zone name, got: %q", out)
	}
	if !strings.Contains(out, "Kista") {
		t.Errorf("expected city, got: %q", out)
	}
	if !strings.Contains(out, "10.00") {
		t.Errorf("expected rate, got: %q", out)
	}
	// Must NOT contain internal IDs or curly braces
	if strings.Contains(out, "17429") {
		t.Errorf("should not contain internal zone ID, got: %q", out)
	}
	if strings.Contains(out, "27545") {
		t.Errorf("should not contain fee zone ID, got: %q", out)
	}
	if strings.Contains(out, "{") {
		t.Errorf("should not contain curly braces, got: %q", out)
	}
}

func TestFormatZoneInfo_NoFees(t *testing.T) {
	zone := parkster.Zone{
		ID:       17429,
		Name:     "Test Zone",
		ZoneCode: "12345",
		City:     parkster.City{Name: "Stockholm"},
		FeeZone: parkster.FeeZone{
			Currency: parkster.Currency{Code: "SEK", Symbol: "kr"},
		},
	}

	out := FormatZoneInfo(zone)

	if !strings.Contains(out, "12345") {
		t.Errorf("expected zone code, got: %q", out)
	}
	// Should still work without fees
	if strings.Contains(out, "{") {
		t.Errorf("should not contain curly braces, got: %q", out)
	}
}

func TestFormatParkingChanged(t *testing.T) {
	now := time.Now()
	parking := parkster.Parking{
		ParkingZone: parkster.Zone{ZoneCode: "80500", Name: "Ericsson, Kista"},
		Car:         parkster.Car{LicenseNbr: "ABC123", CarPersonalization: parkster.CarPersonalization{Name: "Volkswagen"}},
		TimeoutTime: now.Add(3 * time.Hour).UnixMilli(),
		Cost:        0,
		Currency:    parkster.Currency{Code: "SEK"},
	}

	out := FormatParkingChanged(parking)

	if !strings.Contains(out, "Ends at:") {
		t.Errorf("expected 'Ends at:' in output, got: %q", out)
	}
	if !strings.Contains(out, "80500") {
		t.Errorf("expected zone code in output, got: %q", out)
	}
}
