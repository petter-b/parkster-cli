package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/petter-b/parkster-cli/internal/auth"
	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/petter-b/parkster-cli/internal/parkster"
)

// --- Error handling tests ---

func TestStart_MissingZone_Error(t *testing.T) {
	_, _, err := executeCommand("start")
	if err == nil {
		t.Error("start without --zone should return error")
	}
}

func TestStart_MissingZone_ErrorJSON(t *testing.T) {
	// Missing required --zone flag with --json should produce JSON error
	stdout, _, err := executeCommand("start", "--json")
	if err == nil {
		t.Fatal("start without --zone should return error")
	}

	// The error output should be valid JSON envelope
	if stdout != "" {
		var envelope output.Envelope
		if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
			t.Fatalf("Error output with --json should be valid JSON: %v\nOutput: %s", err, stdout)
		}
		if envelope.Success {
			t.Error("Error envelope should have success=false")
		}
	}
}

// --- Start command tests ---

func TestStart_SingleCarSinglePayment_Success(t *testing.T) {
	setAuth(t)

	now := time.Now()
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{ID: 17429, ZoneCode: "80500", Name: "Ericsson Kista", FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp: &parkster.Parking{
			ID:          999,
			ParkingZone: parkster.Zone{ID: 17429, ZoneCode: "80500", Name: "Ericsson Kista"},
			Car:         parkster.Car{ID: 100, LicenseNbr: "ABC123"},
			CheckInTime: now.UnixMilli(),
			TimeoutTime: now.Add(30 * time.Minute).UnixMilli(),
			Currency:    parkster.Currency{Code: "SEK"},
		},
	}
	withMockClient(t, mock)

	_, stderr, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stderr, "Parking started") {
		t.Errorf("expected 'Parking started' in stderr, got: %q", stderr)
	}
}

func TestStart_NoCars_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30")
	if err == nil {
		t.Fatal("expected error for no cars, got nil")
	}
	if !strings.Contains(err.Error(), "no cars") {
		t.Errorf("expected 'no cars' in error, got: %v", err)
	}
}

func TestStart_MultipleCarsWithoutFlag_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			Cars: []parkster.Car{
				{ID: 100, LicenseNbr: "ABC123"},
				{ID: 101, LicenseNbr: "DEF456"},
			},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30")
	if err == nil {
		t.Fatal("expected error for multiple cars without flag, got nil")
	}
	if !strings.Contains(err.Error(), "multiple cars") {
		t.Errorf("expected 'multiple cars' in error, got: %v", err)
	}
}

func TestStart_CarFlagSelectsCorrectCar(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			Cars: []parkster.Car{
				{ID: 100, LicenseNbr: "ABC123"},
				{ID: 101, LicenseNbr: "DEF456"},
			},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{ID: 17429, ZoneCode: "80500", FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp:  &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--car", "DEF456", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success with --car flag, got: %v", err)
	}
}

func TestStart_CarFlagUnknownPlate_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30", "--car", "UNKNOWN")
	if err == nil {
		t.Fatal("expected error for unknown car plate, got nil")
	}
	if !strings.Contains(err.Error(), "car not found") {
		t.Errorf("expected 'car not found' in error, got: %v", err)
	}
}

func TestStart_NoPaymentAccounts_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30")
	if err == nil {
		t.Fatal("expected error for no payment accounts, got nil")
	}
	if !strings.Contains(err.Error(), "no payment") {
		t.Errorf("expected 'no payment' in error, got: %v", err)
	}
}

func TestStart_MultiplePaymentsWithoutFlag_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:   1,
			Cars: []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{
				{PaymentAccountID: "pay1"},
				{PaymentAccountID: "pay2"},
			},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30")
	if err == nil {
		t.Fatal("expected error for multiple payment accounts without flag, got nil")
	}
	if !strings.Contains(err.Error(), "multiple payment") {
		t.Errorf("expected 'multiple payment' in error, got: %v", err)
	}
}

func TestStart_PaymentFlagSelectsCorrect(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:   1,
			Cars: []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{
				{PaymentAccountID: "pay1"},
				{PaymentAccountID: "pay2"},
			},
		},
		getZoneByCodeResp: &parkster.Zone{ID: 17429, ZoneCode: "80500", FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp:  &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--payment", "pay2", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success with --payment flag, got: %v", err)
	}
}

func TestStart_GetZoneFails_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeErr: errors.New("zone not found"),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "99999", "--duration", "30", "--lat", "59.373", "--lon", "17.893")
	if err == nil {
		t.Fatal("expected error when GetZone fails, got nil")
	}
	if !strings.Contains(err.Error(), "zone") {
		t.Errorf("expected 'zone' in error message, got: %v", err)
	}
}

func TestStart_StartParkingFails_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{ID: 17429, ZoneCode: "80500", FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingErr:   errors.New("server error"),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--lat", "59.373", "--lon", "17.893")
	if err == nil {
		t.Fatal("expected error when StartParking fails, got nil")
	}
	if !strings.Contains(err.Error(), "failed to start parking") {
		t.Errorf("expected 'failed to start parking' in error, got: %v", err)
	}
}

// --- Start command with zone code tests ---

func TestStart_WithZoneCode_Success(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{
			ID:       17429,
			ZoneCode: "80500",
			Name:     "Ericsson Kista",
			FeeZone:  parkster.FeeZone{ID: 27545},
		},
		startParkingResp: &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success with zone code, got: %v", err)
	}
}

func TestStart_ZoneCodeNotFound_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeErr: errors.New("zone not found"),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "99999", "--duration", "30", "--lat", "59.373", "--lon", "17.893")
	if err == nil {
		t.Fatal("expected error when zone code not found, got nil")
	}
	if !strings.Contains(err.Error(), "zone") && !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected zone/not found in error, got: %v", err)
	}
}

// --- Start command --until tests ---

func TestStart_Until_Success(t *testing.T) {
	setAuth(t)

	// Use a time 2 hours from now
	untilTime := time.Now().Add(2 * time.Hour)
	untilStr := untilTime.Format("15:04")

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{ID: 17429, ZoneCode: "80500", FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp:  &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "80500", "--until", untilStr, "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success with --until, got: %v", err)
	}
}

func TestStart_BothDurationAndUntil_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30", "--until", "23:00")
	if err == nil {
		t.Fatal("expected error when both --duration and --until specified")
	}
}

func TestStart_NeitherDurationNorUntil_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429")
	if err == nil {
		t.Fatal("expected error when neither --duration nor --until specified")
	}
}

// --- Start command dry-run tests ---

func TestStart_DryRun_ShowsCost(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{
			ID:       17429,
			ZoneCode: "80500",
			Name:     "Ericsson Kista",
			FeeZone: parkster.FeeZone{
				ID: 27545,
				Currency: parkster.Currency{
					Code:   "SEK",
					Symbol: "kr",
				},
			},
		},
		estimateCostResp: &parkster.CostEstimate{
			Amount:   15.0,
			Currency: "SEK",
		},
		// startParkingResp intentionally nil - should not be called
	}
	withMockClient(t, mock)

	stdout, stderr, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--dry-run", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success with dry-run, got: %v", err)
	}
	if !strings.Contains(stderr, "DRY RUN") {
		t.Errorf("expected 'DRY RUN' in stderr, got: %q", stderr)
	}
	// Check that cost appears in combined output (either stdout or stderr)
	combined := stdout + stderr
	if !strings.Contains(combined, "15") {
		t.Errorf("expected cost '15' in output, got stdout: %q, stderr: %q", stdout, stderr)
	}
}

func TestStart_DryRun_JSON(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{
			ID:       17429,
			ZoneCode: "80500",
			Name:     "Ericsson Kista",
			FeeZone: parkster.FeeZone{
				ID:       27545,
				Currency: parkster.Currency{Code: "SEK", Symbol: "kr"},
			},
		},
		estimateCostResp: &parkster.CostEstimate{
			Amount:   15.0,
			Currency: "SEK",
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--dry-run", "--json", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected valid JSON envelope, got parse error: %v\nOutput: %s", err, stdout)
	}
	if !envelope.Success {
		t.Error("expected success=true in JSON envelope")
	}

	// Parse data to check for zone, car, duration, cost
	dataBytes, _ := json.Marshal(envelope.Data)
	var dryRunData map[string]interface{}
	if err := json.Unmarshal(dataBytes, &dryRunData); err != nil {
		t.Fatalf("failed to parse dry-run data: %v", err)
	}

	if dryRunData["zone"] == nil {
		t.Error("expected zone field in dry-run data")
	}
	if dryRunData["car"] == nil {
		t.Error("expected car field in dry-run data")
	}
	if dryRunData["duration"] == nil {
		t.Error("expected duration field in dry-run data")
	}
	if dryRunData["cost"] == nil {
		t.Error("expected cost field in dry-run data")
	}
	if dryRunData["dryRun"] != true {
		t.Error("expected dryRun=true in dry-run data")
	}
}

func TestStart_DryRun_CostEstimateFails_StillSucceeds(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{
			ID:       17429,
			ZoneCode: "80500",
			Name:     "Ericsson Kista",
			FeeZone:  parkster.FeeZone{ID: 27545},
		},
		estimateCostErr: errors.New("cost estimation unavailable"),
	}
	withMockClient(t, mock)

	_, stderr, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--dry-run", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success even when cost estimate fails, got: %v", err)
	}
	if !strings.Contains(stderr, "DRY RUN") {
		t.Errorf("expected 'DRY RUN' in stderr even when cost fails, got: %q", stderr)
	}
}

func TestStart_DryRun_OutputUsesZoneCode(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{
			ID: 17429, ZoneCode: "80500", Name: "Ericsson Kista",
			FeeZone: parkster.FeeZone{ID: 27545},
		},
		estimateCostResp: &parkster.CostEstimate{Amount: 15.0, Currency: "SEK"},
	}
	withMockClient(t, mock)

	stdout, stderr, err := executeCommand("start", "--zone", "80500", "--duration", "30",
		"--lat", "59.373", "--lon", "17.893", "--dry-run")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	combined := stdout + stderr
	// Should show zone code, not numeric ID
	if !strings.Contains(combined, "80500") {
		t.Errorf("expected zone code '80500' in dry-run output, got: %q", combined)
	}
	if strings.Contains(combined, "17429") {
		t.Errorf("should not show numeric zone ID '17429' in dry-run output, got: %q", combined)
	}
}

func TestStart_MultipleCars_HumanOutput_NoInternalIDs(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			Cars: []parkster.Car{
				{ID: 100, LicenseNbr: "ABC123", CarPersonalization: parkster.CarPersonalization{Name: "Volkswagen"}},
				{ID: 101, LicenseNbr: "UPC304", CarPersonalization: parkster.CarPersonalization{Name: "Saab"}},
			},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
	}
	withMockClient(t, mock)

	stdout, _, _ := executeCommand("start", "--zone", "17429", "--duration", "30")
	// Should show car names and plates
	if !strings.Contains(stdout, "Volkswagen") {
		t.Errorf("expected car name in output, got: %q", stdout)
	}
	// Should NOT show internal IDs or curly braces
	if strings.Contains(stdout, " 100") || strings.Contains(stdout, " 101") {
		t.Errorf("should not show internal car IDs, got: %q", stdout)
	}
	if strings.Contains(stdout, "{") {
		t.Errorf("should not show curly braces, got: %q", stdout)
	}
}

func TestStart_MultiplePayments_HumanOutput_Clean(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:   1,
			Cars: []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{
				{PaymentAccountID: "PRIVATE:9999999"},
				{PaymentAccountID: "AT_WORK:72624"},
			},
		},
	}
	withMockClient(t, mock)

	stdout, _, _ := executeCommand("start", "--zone", "17429", "--duration", "30")
	// Should show payment info
	if !strings.Contains(stdout, "PRIVATE") {
		t.Errorf("expected payment type in output, got: %q", stdout)
	}
	if !strings.Contains(stdout, "9999999") {
		t.Errorf("expected payment ID in output, got: %q", stdout)
	}
}

func TestStart_WithRadius_PassesToZoneLookup(t *testing.T) {
	setAuth(t)
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{
			ID: 17429, ZoneCode: "80500", Name: "Ericsson Kista",
			FeeZone: parkster.FeeZone{ID: 27545},
		},
		startParkingResp: &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30",
		"--lat", "59.373", "--lon", "17.893", "--radius", "1000")
	if err != nil {
		t.Fatalf("expected success with --radius, got: %v", err)
	}
}

// --- Car flag matching tests ---

func TestStart_CarFlag_MatchesByName(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			Cars: []parkster.Car{
				{ID: 100, LicenseNbr: "ABC123", CarPersonalization: parkster.CarPersonalization{Name: "Volkswagen"}},
				{ID: 101, LicenseNbr: "UPC304", CarPersonalization: parkster.CarPersonalization{Name: "Saab"}},
			},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{ID: 17429, ZoneCode: "80500", FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp:  &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--car", "Volkswagen", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected --car to match by name, got: %v", err)
	}
}

func TestStart_CarFlag_CaseInsensitive(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			Cars: []parkster.Car{
				{ID: 100, LicenseNbr: "ABC123", CarPersonalization: parkster.CarPersonalization{Name: "Volkswagen"}},
			},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{ID: 17429, ZoneCode: "80500", FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp:  &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--car", "volkswagen", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected case-insensitive match, got: %v", err)
	}
}

func TestStart_CarFlag_CaseInsensitivePlate(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			Cars: []parkster.Car{
				{ID: 100, LicenseNbr: "ABC123", CarPersonalization: parkster.CarPersonalization{Name: "Volkswagen"}},
			},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{ID: 17429, ZoneCode: "80500", FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp:  &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--car", "abc123", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected case-insensitive plate match, got: %v", err)
	}
}

// --- Payment flag flexible matching tests ---

func TestStart_PaymentFlag_MatchesByNumericPart(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:   1,
			Cars: []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{
				{PaymentAccountID: "PRIVATE:9999999"},
				{PaymentAccountID: "AT_WORK:72624"},
			},
		},
		getZoneByCodeResp: &parkster.Zone{ID: 17429, ZoneCode: "80500", FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp:  &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--payment", "9999999", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected --payment to match numeric suffix, got: %v", err)
	}
}

func TestStart_PaymentFlag_MatchesByTypePrefix(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:   1,
			Cars: []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{
				{PaymentAccountID: "PRIVATE:9999999"},
				{PaymentAccountID: "AT_WORK:72624"},
			},
		},
		getZoneByCodeResp: &parkster.Zone{ID: 17429, ZoneCode: "80500", FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp:  &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--payment", "PRIVATE", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected --payment to match type prefix, got: %v", err)
	}
}

func TestStart_PaymentFlag_FullID_StillWorks(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:   1,
			Cars: []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{
				{PaymentAccountID: "PRIVATE:9999999"},
			},
		},
		getZoneByCodeResp: &parkster.Zone{ID: 17429, ZoneCode: "80500", FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp:  &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--payment", "PRIVATE:9999999", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected full payment ID to still work, got: %v", err)
	}
}

// --- Duration validation tests ---

func TestStart_ZeroDuration_Error(t *testing.T) {
	setAuth(t)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "0")
	if err == nil {
		t.Fatal("expected error for --duration 0, got nil")
	}
	if !strings.Contains(err.Error(), "positive") {
		t.Errorf("expected 'positive' in error, got: %v", err)
	}
}

func TestStart_NegativeDuration_Error(t *testing.T) {
	setAuth(t)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "-5")
	if err == nil {
		t.Fatal("expected error for --duration -5, got nil")
	}
	if !strings.Contains(err.Error(), "positive") {
		t.Errorf("expected 'positive' in error, got: %v", err)
	}
}

func TestStart_LatWithoutLon_Error(t *testing.T) {
	setAuth(t)

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--lat", "59.37")
	if err == nil {
		t.Fatal("expected error for --lat without --lon, got nil")
	}
	if !strings.Contains(err.Error(), "together") {
		t.Errorf("expected 'together' in error, got: %v", err)
	}
}

func TestStart_LonWithoutLat_Error(t *testing.T) {
	setAuth(t)

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--lon", "17.89")
	if err == nil {
		t.Fatal("expected error for --lon without --lat, got nil")
	}
	if !strings.Contains(err.Error(), "together") {
		t.Errorf("expected 'together' in error, got: %v", err)
	}
}

// --- Start --until with past time ---

// --- Start --payment with unknown payment ---

func TestStart_PaymentFlagUnknown_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "PRIVATE:123"}},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30", "--payment", "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for unknown payment account")
	}
	if !strings.Contains(err.Error(), "payment account not found") {
		t.Errorf("expected 'payment account not found' in error, got: %v", err)
	}
}

// --- Start --dry-run with --until ---

func TestStart_DryRun_WithUntil(t *testing.T) {
	setAuth(t)

	now := time.Now()
	// Use a time 2 hours from now
	futureHour := now.Add(2 * time.Hour)
	untilStr := fmt.Sprintf("%02d:%02d", futureHour.Hour(), futureHour.Minute())

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{ID: 17429, ZoneCode: "80500", Name: "Ericsson Kista", FeeZone: parkster.FeeZone{ID: 27545}},
		estimateCostResp:  &parkster.CostEstimate{Amount: 20.0, Currency: "SEK"},
	}
	withMockClient(t, mock)

	stdout, stderr, err := executeCommand("start", "--zone", "80500", "--until", untilStr, "--dry-run", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stderr, "DRY RUN") {
		t.Errorf("expected 'DRY RUN' in stderr, got: %q", stderr)
	}
	if stdout == "" {
		t.Error("expected dry-run output on stdout")
	}
}

// --- Start auth required ---

func TestStart_NotAuthenticated_Error(t *testing.T) {
	origGet := getCredentials
	getCredentials = func() (string, string, auth.CredentialSource, error) {
		return "", "", "", errors.New("no credentials found")
	}
	t.Cleanup(func() { getCredentials = origGet })

	_, stderr, err := executeCommand("start", "--zone", "17429", "--duration", "30")
	if err == nil {
		t.Fatal("expected error for start without auth")
	}
	if !strings.Contains(stderr, "Not authenticated") {
		t.Errorf("expected 'Not authenticated' in stderr, got: %q", stderr)
	}
}

// --- Login failures ---

func TestStart_LoginFails_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginErr: errors.New("network timeout"),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30")
	if err == nil {
		t.Fatal("expected error when login fails")
	}
	if !strings.Contains(err.Error(), "failed to authenticate") {
		t.Errorf("expected 'failed to authenticate' in error, got: %v", err)
	}
}

// --- Additional coverage: start --json success output ---

func TestStart_SingleCarSinglePayment_JSON(t *testing.T) {
	setAuth(t)

	now := time.Now()
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{ID: 17429, ZoneCode: "80500", Name: "Ericsson Kista", FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp: &parkster.Parking{
			ID:          999,
			ParkingZone: parkster.Zone{ID: 17429, ZoneCode: "80500"},
			Car:         parkster.Car{ID: 100, LicenseNbr: "ABC123"},
			CheckInTime: now.UnixMilli(),
			TimeoutTime: now.Add(30 * time.Minute).UnixMilli(),
			Currency:    parkster.Currency{Code: "SEK"},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--json", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected valid JSON envelope, got: %v\nOutput: %s", err, stdout)
	}
	if !envelope.Success {
		t.Error("expected success=true")
	}
}

// --- Bug regression: disambiguation list contaminates JSON output ---

func TestStart_MultipleCars_JSON_OutputIsValidJSON(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			Cars: []parkster.Car{
				{ID: 100, LicenseNbr: "ABC123", CarPersonalization: parkster.CarPersonalization{Name: "Volkswagen"}},
				{ID: 101, LicenseNbr: "UPC304", CarPersonalization: parkster.CarPersonalization{Name: "Saab"}},
			},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
	}
	withMockClient(t, mock)

	stdout, _, _ := executeCommand("start", "--zone", "17429", "--duration", "30", "--json")

	// stdout should be ONLY valid JSON - no car list contamination
	var envelope map[string]any
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Errorf("stdout should be valid JSON, got parse error: %v\nstdout was:\n%s", err, stdout)
	}
}

func TestStart_MultiplePayments_JSON_OutputIsValidJSON(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:   1,
			Cars: []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{
				{PaymentAccountID: "PRIVATE:9999999"},
				{PaymentAccountID: "AT_WORK:72624"},
			},
		},
	}
	withMockClient(t, mock)

	stdout, _, _ := executeCommand("start", "--zone", "17429", "--duration", "30", "--json")

	var envelope map[string]any
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Errorf("stdout should be valid JSON, got parse error: %v\nstdout was:\n%s", err, stdout)
	}
}

// --- Edge case: start --radius help text vs actual default ---

func TestStart_RadiusDefault_IsZero(t *testing.T) {
	// Verify that the radius flag defaults to 0 (API decides default),
	// matching the behavior of zones search
	resetFlags()
	f := startCmd.Flags().Lookup("radius")
	if f == nil {
		t.Fatal("expected radius flag on start command")
		return
	}
	if f.DefValue != "0" {
		t.Errorf("radius default should be 0, got %q", f.DefValue)
	}
}

// --- Disambiguation lists go to stderr, not stdout ---

func TestStart_MultipleCars_DisambiguationOnStderr(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			Cars: []parkster.Car{
				{ID: 100, LicenseNbr: "ABC123", CarPersonalization: parkster.CarPersonalization{Name: "Volvo"}},
				{ID: 101, LicenseNbr: "DEF456", CarPersonalization: parkster.CarPersonalization{Name: "Saab"}},
			},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
	}
	withMockClient(t, mock)

	stdout, stderr, _ := executeCommand("start", "--zone", "17429", "--duration", "30", "--json")

	// Car list should appear on stderr, not stdout
	if strings.Contains(stdout, "ABC123") || strings.Contains(stdout, "DEF456") {
		t.Errorf("disambiguation list should not appear on stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "ABC123") && !strings.Contains(stderr, "DEF456") {
		t.Errorf("disambiguation list should appear on stderr, got: %q", stderr)
	}
}

func TestStart_MultiplePayments_DisambiguationOnStderr(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:   1,
			Cars: []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{
				{PaymentAccountID: "PRIVATE:111"},
				{PaymentAccountID: "AT_WORK:222"},
			},
		},
	}
	withMockClient(t, mock)

	stdout, stderr, _ := executeCommand("start", "--zone", "17429", "--duration", "30", "--json")

	if strings.Contains(stdout, "PRIVATE") || strings.Contains(stdout, "AT_WORK") {
		t.Errorf("payment disambiguation should not appear on stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "PRIVATE") && !strings.Contains(stderr, "AT_WORK") {
		t.Errorf("payment disambiguation should appear on stderr, got: %q", stderr)
	}
}

// --- JSON error envelopes for validation errors ---

func TestStart_ZeroDuration_JSON_Error(t *testing.T) {
	setAuth(t)

	stdout, _, err := executeCommandFull("start", "--zone", "17429", "--duration", "0", "--json")
	if err == nil {
		t.Fatal("expected error for zero duration")
	}
	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Errorf("error output should be valid JSON: %v\nstdout: %s", err, stdout)
	}
	if envelope.Success {
		t.Error("expected success=false")
	}
}

func TestStart_NegativeDuration_JSON_Error(t *testing.T) {
	setAuth(t)

	stdout, _, err := executeCommandFull("start", "--zone", "17429", "--duration", "-5", "--json")
	if err == nil {
		t.Fatal("expected error for negative duration")
	}
	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Errorf("error output should be valid JSON: %v\nstdout: %s", err, stdout)
	}
}

// parseUntil wraps past times to tomorrow, so --until 00:01 at any time
// after midnight proceeds normally (not rejected as past time).
func TestStart_UntilInPast_WrapsToTomorrow(t *testing.T) {
	if time.Now().Hour() == 0 && time.Now().Minute() <= 1 {
		t.Skip("too close to midnight to test past time wrapping")
	}
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{ID: 17429, ZoneCode: "80500", FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp:  &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	// 00:01 is in the past, but parseUntil wraps to tomorrow.
	// The command should succeed (no "past time" error).
	stdout, _, err := executeCommand("start", "--zone", "80500", "--until", "00:01", "--lat", "59.37", "--lon", "17.89", "--json")
	if err != nil {
		t.Fatalf("--until 00:01 should wrap to tomorrow, got error: %v", err)
	}

	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Errorf("output should be valid JSON: %v\nstdout: %s", err, stdout)
	}
	if !envelope.Success {
		t.Error("expected success: true in JSON envelope")
	}
}

// --- Human-mode disambiguation shows list on stdout ---

func TestStart_MultipleCars_HumanMode_ShowsList(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			Cars: []parkster.Car{
				{ID: 100, LicenseNbr: "ABC123", CarPersonalization: parkster.CarPersonalization{Name: "Volvo"}},
				{ID: 101, LicenseNbr: "DEF456", CarPersonalization: parkster.CarPersonalization{Name: "Saab"}},
			},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("start", "--zone", "17429", "--duration", "30")
	if err == nil {
		t.Fatal("expected error for multiple cars")
	}
	// In human mode, car list should appear on stdout
	if !strings.Contains(stdout, "ABC123") && !strings.Contains(stdout, "Volvo") {
		t.Errorf("expected car list on stdout in human mode, got: %q", stdout)
	}
}

// Flag validation: start --zone accepts non-numeric zone codes
func TestStart_ZoneCode_NonNumeric_Accepted(t *testing.T) {
	setAuth(t)
	mock := &mockAPI{
		loginResp: &parkster.User{
			Cars:            []parkster.Car{{ID: 1, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{ID: 100, FeeZone: parkster.FeeZone{ID: 200}},
		startParkingResp:  &parkster.Parking{ID: 999, ParkingZone: parkster.Zone{Name: "Test"}},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "ABC123", "--duration", "30", "--lat", "59.0", "--lon", "17.0")
	if err != nil {
		t.Fatalf("expected non-numeric zone code to be accepted, got error: %v", err)
	}
}

func TestStart_NumericZone_NoLatLon_HintsAboutCoordinates(t *testing.T) {
	setAuth(t)
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 1, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "PAY:1"}},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--lat") || !strings.Contains(err.Error(), "--lon") {
		t.Errorf("expected hint about --lat/--lon in error, got: %v", err)
	}
}

func TestStart_ExtraArgs_Error(t *testing.T) {
	_, _, err := executeCommand("start", "extra")
	if err == nil {
		t.Fatal("expected error for extra positional args on start")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected 'unknown command' error, got: %v", err)
	}
}

// --- JSON error envelope tests ---

func TestStartCommand_AuthFailure_JSON_ErrorEnvelope(t *testing.T) {
	orig := getCredentials
	getCredentials = func() (string, string, auth.CredentialSource, error) {
		return "", "", "", fmt.Errorf("no credentials")
	}
	t.Cleanup(func() { getCredentials = orig })

	stdout, _, err := executeCommandFull("start", "--zone", "17429", "--duration", "30", "--json")
	if err == nil {
		t.Fatal("expected error for missing credentials")
	}

	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected valid JSON error envelope, got: %v\nstdout: %q", err, stdout)
	}
	if envelope.Success {
		t.Error("expected success=false")
	}
	if envelope.Error == nil {
		t.Error("expected non-nil error field")
	}
}

// --- No cars / no payments tests ---

func TestStart_NoCars_ErrorMessage(t *testing.T) {
	setAuth(t)
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "PAY:1"}},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30")
	if err == nil {
		t.Fatal("expected error for no cars")
	}
	if !strings.Contains(err.Error(), "no cars registered") {
		t.Errorf("expected 'no cars registered' error, got: %v", err)
	}
}

func TestStart_NoPaymentAccounts_ErrorMessage(t *testing.T) {
	setAuth(t)
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 1, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "17429", "--duration", "30")
	if err == nil {
		t.Fatal("expected error for no payment accounts")
	}
	if !strings.Contains(err.Error(), "no payment methods") {
		t.Errorf("expected 'no payment methods' error, got: %v", err)
	}
}

func TestStart_NoCars_JSON_ErrorEnvelope(t *testing.T) {
	setAuth(t)
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "PAY:1"}},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommandFull("start", "--zone", "17429", "--duration", "30", "--json")
	if err == nil {
		t.Fatal("expected error for no cars")
	}

	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected valid JSON, got: %v\nstdout: %q", err, stdout)
	}
	if envelope.Success {
		t.Error("expected success=false")
	}
}

// --- parseUntil tests ---

func TestParseUntil_DotSeparator(t *testing.T) {
	setAuth(t)

	now := time.Now()
	untilTime := now.Add(2 * time.Hour)
	// Use dot format: "HH.MM"
	untilStr := fmt.Sprintf("%02d.%02d", untilTime.Hour(), untilTime.Minute())

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 500, TimeoutTime: now.Add(30 * time.Minute).UnixMilli()},
			},
		},
		extendResp: &parkster.Parking{ID: 500, TimeoutTime: untilTime.UnixMilli()},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("change", "--until", untilStr)
	if err != nil {
		t.Fatalf("expected dot-separated time to work, got: %v", err)
	}
}

func TestParseUntil_BareHour(t *testing.T) {
	setAuth(t)

	now := time.Now()
	// Use a bare hour 2 hours from now
	futureHour := (now.Hour() + 2) % 24
	untilStr := fmt.Sprintf("%d", futureHour)

	// Skip if the target would be in the past (near midnight)
	target := time.Date(now.Year(), now.Month(), now.Day(), futureHour, 0, 0, 0, now.Location())
	if target.Before(now) {
		t.Skip("skipping: bare hour would be in the past near midnight")
	}

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 500, TimeoutTime: now.Add(30 * time.Minute).UnixMilli()},
			},
		},
		extendResp: &parkster.Parking{ID: 500, TimeoutTime: target.UnixMilli()},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("change", "--until", untilStr)
	if err != nil {
		t.Fatalf("expected bare hour to work, got: %v", err)
	}
}

func TestParseUntil_MidnightFormats(t *testing.T) {
	// "0" should parse as 00:00
	result, err := parseUntil("0")
	if err != nil {
		t.Fatalf("parseUntil(\"0\") failed: %v", err)
	}
	if result.Hour() != 0 || result.Minute() != 0 {
		t.Errorf("parseUntil(\"0\") = %v, want 00:00", result.Format("15:04"))
	}

	// "00:00" should parse as midnight
	result, err = parseUntil("00:00")
	if err != nil {
		t.Fatalf("parseUntil(\"00:00\") failed: %v", err)
	}
	if result.Hour() != 0 || result.Minute() != 0 {
		t.Errorf("parseUntil(\"00:00\") = %v, want 00:00", result.Format("15:04"))
	}

	// "23:59" should parse as 23:59
	result, err = parseUntil("23:59")
	if err != nil {
		t.Fatalf("parseUntil(\"23:59\") failed: %v", err)
	}
	if result.Hour() != 23 || result.Minute() != 59 {
		t.Errorf("parseUntil(\"23:59\") = %v, want 23:59", result.Format("15:04"))
	}
}

func TestParseUntil_HH_MM_Format(t *testing.T) {
	result, err := parseUntil("14:30")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if result.Hour() != 14 || result.Minute() != 30 {
		t.Errorf("expected 14:30, got %02d:%02d", result.Hour(), result.Minute())
	}
}

func TestParseUntil_Dot_Format(t *testing.T) {
	result, err := parseUntil("14.30")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if result.Hour() != 14 || result.Minute() != 30 {
		t.Errorf("expected 14:30, got %02d:%02d", result.Hour(), result.Minute())
	}
}

func TestParseUntil_SingleDigitHour(t *testing.T) {
	result, err := parseUntil("9")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if result.Hour() != 9 {
		t.Errorf("expected 09:00, got %02d:%02d", result.Hour(), result.Minute())
	}
}

func TestParseUntil_InvalidHour_25(t *testing.T) {
	_, err := parseUntil("25:00")
	if err == nil {
		t.Error("expected error for hour 25")
	}
}

func TestParseUntil_InvalidFormat_Letters(t *testing.T) {
	_, err := parseUntil("abc")
	if err == nil {
		t.Error("expected error for non-numeric input")
	}
	if !strings.Contains(err.Error(), "invalid time format") {
		t.Errorf("expected 'invalid time format' in error, got: %v", err)
	}
}

func TestParseUntil_EmptyString(t *testing.T) {
	_, err := parseUntil("")
	if err == nil {
		t.Error("expected error for empty string")
	}
}

func TestParseUntil_LeadingZero(t *testing.T) {
	result, err := parseUntil("08:05")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if result.Hour() != 8 || result.Minute() != 5 {
		t.Errorf("expected 08:05, got %02d:%02d", result.Hour(), result.Minute())
	}
}

func TestParseUntil_ResultIsToday(t *testing.T) {
	// Use a controlled time so the test is deterministic
	fakeNow := time.Date(2026, 2, 17, 10, 0, 0, 0, time.Local)
	result, err := parseUntilFrom("15:00", fakeNow)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if result.Day() != 17 {
		t.Errorf("expected day 17, got %d", result.Day())
	}
}

func TestParseUntil_WrapsToTomorrowWhenPast(t *testing.T) {
	// At 23:00, --until 01:00 should mean tomorrow 01:00, not today 01:00
	fakeNow := time.Date(2026, 2, 17, 23, 0, 0, 0, time.Local)

	result, err := parseUntilFrom("01:00", fakeNow)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	expected := time.Date(2026, 2, 18, 1, 0, 0, 0, time.Local)
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestParseUntil_NoWrapWhenFuture(t *testing.T) {
	// At 10:00, --until 14:00 should stay on same day
	fakeNow := time.Date(2026, 2, 17, 10, 0, 0, 0, time.Local)

	result, err := parseUntilFrom("14:00", fakeNow)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	expected := time.Date(2026, 2, 17, 14, 0, 0, 0, time.Local)
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

// --- resolveZone unit tests ---

func TestResolveZone_ByCode_WithLatLon_Success(t *testing.T) {
	mock := &mockAPI{
		getZoneByCodeResp: &parkster.Zone{ID: 17429, Name: "Ericsson", ZoneCode: "80500"},
	}

	zone, err := resolveZone(mock, "80500", 59.373, 17.893, 500, nil)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if zone.ZoneCode != "80500" {
		t.Errorf("expected zone code 80500, got %s", zone.ZoneCode)
	}
}

func TestResolveZone_ByCode_WithoutLatLon_ErrorHints(t *testing.T) {
	mock := &mockAPI{}

	// Non-numeric input without lat/lon should hint about --lat/--lon
	_, err := resolveZone(mock, "ABC80500", 0, 0, 0, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--lat") || !strings.Contains(err.Error(), "--lon") {
		t.Errorf("expected hint about --lat/--lon, got: %v", err)
	}
}

func TestResolveZone_CodeNotFound_Error(t *testing.T) {
	mock := &mockAPI{
		getZoneByCodeErr: fmt.Errorf("zone code \"XXXXX\" not found near 59.3730,17.8930"),
	}

	_, err := resolveZone(mock, "XXXXX", 59.373, 17.893, 500, nil)
	if err == nil {
		t.Fatal("expected error when zone code not found")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestResolveZone_FavoriteZone_NoLatLon_Success(t *testing.T) {
	mock := &mockAPI{
		getZoneResp: &parkster.Zone{ID: 17429, Name: "Ericsson Kista", ZoneCode: "80500", FeeZone: parkster.FeeZone{ID: 27545}},
	}

	favorites := []parkster.FavoriteZone{
		{ID: 17429, ZoneCode: "80500", Name: "Ericsson Kista"},
	}

	zone, err := resolveZone(mock, "80500", 0, 0, 0, favorites)
	if err != nil {
		t.Fatalf("expected favorite zone to resolve without lat/lon, got: %v", err)
	}
	if zone.ID != 17429 {
		t.Errorf("expected zone ID 17429, got %d", zone.ID)
	}
}

func TestResolveZone_NotFavorite_NoLatLon_Error(t *testing.T) {
	mock := &mockAPI{}

	favorites := []parkster.FavoriteZone{
		{ID: 17429, ZoneCode: "80500", Name: "Ericsson Kista"},
	}

	_, err := resolveZone(mock, "99999", 0, 0, 0, favorites)
	if err == nil {
		t.Fatal("expected error for non-favorite zone without lat/lon")
	}
	if !strings.Contains(err.Error(), "not in your favorites") {
		t.Errorf("expected 'not in your favorites' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "--lat") {
		t.Errorf("expected '--lat' hint in error, got: %v", err)
	}
}

func TestResolveZone_NoFavorites_NoLatLon_Error(t *testing.T) {
	mock := &mockAPI{}

	_, err := resolveZone(mock, "80500", 0, 0, 0, nil)
	if err == nil {
		t.Fatal("expected error without favorites or lat/lon")
	}
	if !strings.Contains(err.Error(), "--lat") {
		t.Errorf("expected '--lat' hint in error, got: %v", err)
	}
}

func TestResolveZone_FavoriteZone_WithLatLon_UsesLocationSearch(t *testing.T) {
	mock := &mockAPI{
		getZoneByCodeResp: &parkster.Zone{ID: 17429, Name: "Ericsson Kista", ZoneCode: "80500", FeeZone: parkster.FeeZone{ID: 27545}},
	}

	favorites := []parkster.FavoriteZone{
		{ID: 17429, ZoneCode: "80500", Name: "Ericsson Kista"},
	}

	// Even though zone is a favorite, explicit lat/lon should use location search
	zone, err := resolveZone(mock, "80500", 59.373, 17.893, 500, favorites)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if zone.ID != 17429 {
		t.Errorf("expected zone ID 17429, got %d", zone.ID)
	}
}

// --- Dry-run does NOT call StartParking ---

func TestStart_DryRun_DoesNotStartParking(t *testing.T) {
	setAuth(t)

	startCalled := false
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{ID: 17429, ZoneCode: "80500", FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp:  &parkster.Parking{ID: 999},
		estimateCostResp:  &parkster.CostEstimate{Amount: 10.0, Currency: "SEK"},
	}
	// Wrap StartParking to detect if it's called
	origNew := newAPIClient
	newAPIClient = func(u, p string) parkster.API {
		return &startTracker{mockAPI: mock, called: &startCalled}
	}
	t.Cleanup(func() { newAPIClient = origNew })

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--lat", "59.37", "--lon", "17.89", "--dry-run")
	if err != nil {
		t.Fatalf("dry-run should succeed, got: %v", err)
	}
	if startCalled {
		t.Error("--dry-run should NOT call StartParking")
	}
}

// --- Debug mode outputs to stderr ---

func TestStart_Debug_LogsSteps(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{ID: 17429, ZoneCode: "80500", FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp:  &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, stderr, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--lat", "59.37", "--lon", "17.89", "--debug")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	for _, want := range []string{"authenticating", "selected car", "resolving zone"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("expected %q in debug output, got: %q", want, stderr)
		}
	}
}

// --- Favorite zone start (no lat/lon) integration tests ---

func TestStart_FavoriteZone_NoLatLon_Success(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
			FavoriteZones:   []parkster.FavoriteZone{{ID: 17429, ZoneCode: "80500", Name: "Ericsson Kista"}},
		},
		getZoneResp:      &parkster.Zone{ID: 17429, ZoneCode: "80500", Name: "Ericsson Kista", FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingResp: &parkster.Parking{ID: 999},
	}
	withMockClient(t, mock)

	_, stderr, err := executeCommand("start", "--zone", "80500", "--duration", "30")
	if err != nil {
		t.Fatalf("expected success with favorite zone, got: %v", err)
	}
	if !strings.Contains(stderr, "Parking started") {
		t.Errorf("expected 'Parking started' in stderr, got: %q", stderr)
	}
}

func TestStart_NonFavoriteZone_NoLatLon_Error(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
			FavoriteZones:   []parkster.FavoriteZone{{ID: 17429, ZoneCode: "80500", Name: "Ericsson Kista"}},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "99999", "--duration", "30")
	if err == nil {
		t.Fatal("expected error for non-favorite zone without lat/lon")
	}
	if !strings.Contains(err.Error(), "not in your favorites") {
		t.Errorf("expected 'not in your favorites' in error, got: %v", err)
	}
}

func TestStart_DryRun_FavoriteZone_NoLatLon_Success(t *testing.T) {
	setAuth(t)

	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
			FavoriteZones:   []parkster.FavoriteZone{{ID: 17429, ZoneCode: "80500", Name: "Ericsson Kista"}},
		},
		getZoneResp:      &parkster.Zone{ID: 17429, ZoneCode: "80500", Name: "Ericsson Kista", FeeZone: parkster.FeeZone{ID: 27545}},
		estimateCostResp: &parkster.CostEstimate{Amount: 15.0, Currency: "SEK"},
	}
	withMockClient(t, mock)

	_, stderr, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--dry-run")
	if err != nil {
		t.Fatalf("expected dry-run success with favorite zone, got: %v", err)
	}
	if !strings.Contains(stderr, "DRY RUN") {
		t.Errorf("expected 'DRY RUN' in stderr, got: %q", stderr)
	}
}

// startTracker wraps mockAPI to detect StartParking calls.
type startTracker struct {
	*mockAPI
	called *bool
}

func (s *startTracker) StartParking(a, b, c int, d string, e int) (*parkster.Parking, error) {
	*s.called = true
	return s.mockAPI.StartParking(a, b, c, d, e)
}
