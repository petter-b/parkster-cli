package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/petter-b/parkster-cli/internal/parkster"
)

// --- Zones no-auth tests ---

func TestZonesSearch_NoAuth_Success(t *testing.T) {
	// Do NOT call setAuth — no PARKSTER_USERNAME/PASSWORD set
	// Zone commands should work without credentials
	mock := &mockAPI{
		searchZonesResp: &parkster.SearchResult{
			ParkingZonesAtPosition: []parkster.ZoneSearchItem{
				{ID: 17429, Name: "Ericsson Kista", ZoneCode: "80500"},
			},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("zones", "search", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("zones search should work without auth, got: %v", err)
	}
}

func TestZonesInfo_NoAuth_Success(t *testing.T) {
	mock := &mockAPI{
		getZoneByCodeResp: &parkster.Zone{
			ID: 17429, Name: "Ericsson Kista", ZoneCode: "80500",
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("zones", "info", "80500", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("zones info should work without auth, got: %v", err)
	}
}

// --- Zones search command tests ---

func TestZonesSearch_Success(t *testing.T) {
	mock := &mockAPI{
		searchZonesResp: &parkster.SearchResult{
			ParkingZonesAtPosition: []parkster.ZoneSearchItem{
				{ID: 17429, Name: "Ericsson Kista", ZoneCode: "80500", City: parkster.City{Name: "Stockholm"}},
			},
			ParkingZonesNearbyPosition: []parkster.ZoneSearchItem{
				{ID: 17430, Name: "Kistagången", ZoneCode: "80501", City: parkster.City{Name: "Stockholm"}},
			},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("zones", "search", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stdout, "80500") {
		t.Errorf("expected zone code 80500 in output, got: %q", stdout)
	}
	if !strings.Contains(stdout, "Ericsson Kista") {
		t.Errorf("expected zone name in output, got: %q", stdout)
	}
}

func TestZonesSearch_Success_JSON(t *testing.T) {
	mock := &mockAPI{
		searchZonesResp: &parkster.SearchResult{
			ParkingZonesAtPosition: []parkster.ZoneSearchItem{
				{ID: 17429, Name: "Ericsson Kista", ZoneCode: "80500"},
			},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("zones", "search", "--lat", "59.373", "--lon", "17.893", "--json")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected valid JSON, got parse error: %v\nOutput: %s", err, stdout)
	}
	if !envelope.Success {
		t.Error("expected success=true in JSON envelope")
	}
	dataBytes, _ := json.Marshal(envelope.Data)
	if !strings.Contains(string(dataBytes), "80500") {
		t.Error("expected zone data in JSON output")
	}
}

func TestZonesSearch_NoResults(t *testing.T) {
	mock := &mockAPI{
		searchZonesResp: &parkster.SearchResult{
			ParkingZonesAtPosition:     []parkster.ZoneSearchItem{},
			ParkingZonesNearbyPosition: []parkster.ZoneSearchItem{},
		},
	}
	withMockClient(t, mock)

	_, stderr, err := executeCommand("zones", "search", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success with empty results, got: %v", err)
	}
	if !strings.Contains(stderr, "No zones found") {
		t.Errorf("expected 'No zones found' in stderr, got: %q", stderr)
	}
}

func TestZonesSearch_NoResults_JSON(t *testing.T) {
	mock := &mockAPI{
		searchZonesResp: &parkster.SearchResult{
			ParkingZonesAtPosition:     []parkster.ZoneSearchItem{},
			ParkingZonesNearbyPosition: []parkster.ZoneSearchItem{},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("zones", "search", "--lat", "59.373", "--lon", "17.893", "--json")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected valid JSON, got parse error: %v\nOutput: %s", err, stdout)
	}
	if !envelope.Success {
		t.Error("expected success=true in JSON envelope")
	}
	// Data should be an empty array
	dataBytes, _ := json.Marshal(envelope.Data)
	if string(dataBytes) != "[]" {
		t.Errorf("expected empty array in data, got: %s", string(dataBytes))
	}
}

func TestZonesSearch_MissingLatLon_Error(t *testing.T) {
	_, _, err := executeCommand("zones", "search")
	if err == nil {
		t.Fatal("expected error for missing --lat and --lon flags, got nil")
	}
}

func TestZonesSearch_InvalidCoordinates_Error(t *testing.T) {
	_, _, err := executeCommand("zones", "search", "--lat", "999", "--lon", "17.893")
	if err == nil {
		t.Fatal("expected error for invalid latitude, got nil")
	}
	if !strings.Contains(err.Error(), "latitude") {
		t.Errorf("expected 'latitude' in error, got: %v", err)
	}
}

func TestZonesSearch_SearchFails_Error(t *testing.T) {
	mock := &mockAPI{
		searchZonesErr: errors.New("search failed"),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("zones", "search", "--lat", "59.373", "--lon", "17.893")
	if err == nil {
		t.Fatal("expected error when SearchZones fails, got nil")
	}
	if !strings.Contains(err.Error(), "search") {
		t.Errorf("expected 'search' in error, got: %v", err)
	}
}

func TestZonesSearch_CustomRadius(t *testing.T) {
	mock := &mockAPI{
		searchZonesResp: &parkster.SearchResult{
			ParkingZonesAtPosition: []parkster.ZoneSearchItem{
				{ID: 17429, Name: "Ericsson Kista", ZoneCode: "80500"},
			},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("zones", "search", "--lat", "59.373", "--lon", "17.893", "--radius", "500")
	if err != nil {
		t.Fatalf("expected success with custom radius, got: %v", err)
	}
}

func TestZonesSearch_RadiusDefault_IsZero(t *testing.T) {
	f := zonesSearchCmd.Flags().Lookup("radius")
	if f == nil {
		t.Fatal("--radius flag not found")
	}
	if f.DefValue != "0" {
		t.Errorf("expected --radius default '0', got %q", f.DefValue)
	}
}

func TestZonesSearch_HumanOutput_NoInternalIDs(t *testing.T) {
	mock := &mockAPI{
		searchZonesResp: &parkster.SearchResult{
			ParkingZonesAtPosition: []parkster.ZoneSearchItem{
				{ID: 17429, Name: "Ericsson", ZoneCode: "80500", City: parkster.City{Name: "Kista"}},
			},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("zones", "search", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	// Should show zone code and name
	if !strings.Contains(stdout, "80500") {
		t.Errorf("expected zone code in output, got: %q", stdout)
	}
	// Should NOT show internal ID or curly braces
	if strings.Contains(stdout, "17429") {
		t.Errorf("should not show internal zone ID, got: %q", stdout)
	}
	if strings.Contains(stdout, "{") {
		t.Errorf("should not show curly braces, got: %q", stdout)
	}
}

func TestZonesSearch_NegativeRadius_Error(t *testing.T) {
	_, _, err := executeCommand("zones", "search", "--lat", "59.373", "--lon", "17.893", "--radius", "-100")
	if err == nil {
		t.Fatal("expected error for negative radius, got nil")
	}
}

func TestZonesSearch_Debug_WritesToStderr(t *testing.T) {
	mock := &mockAPI{
		searchZonesResp: &parkster.SearchResult{
			ParkingZonesAtPosition: []parkster.ZoneSearchItem{
				{ID: 17429, Name: "Ericsson", ZoneCode: "80500"},
			},
		},
	}
	withMockClient(t, mock)

	_, stderr, err := executeCommand("zones", "search", "--lat", "59.373", "--lon", "17.893", "-d")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stderr, "DEBUG:") {
		t.Errorf("expected DEBUG output with -d flag, got stderr: %q", stderr)
	}
}

func TestZonesSearch_InvalidCoordinates_JSON_Error(t *testing.T) {
	stdout, _, err := executeCommand("zones", "search", "--lat", "999", "--lon", "17.893", "--json")
	if err == nil {
		t.Fatal("expected error for invalid latitude, got nil")
	}
	// When --json is set, error should still be formatted as JSON
	if stdout != "" {
		var envelope output.Envelope
		if jsonErr := json.Unmarshal([]byte(stdout), &envelope); jsonErr != nil {
			t.Fatalf("error output with --json should be valid JSON: %v\nOutput: %s", jsonErr, stdout)
		}
		if envelope.Success {
			t.Error("error envelope should have success=false")
		}
	}
}

func TestZonesSearch_Quiet_HasResults(t *testing.T) {
	mock := &mockAPI{
		searchZonesResp: &parkster.SearchResult{
			ParkingZonesAtPosition: []parkster.ZoneSearchItem{
				{ID: 1, ZoneCode: "80500", Name: "Ericsson", City: parkster.City{Name: "Kista"}},
			},
		},
	}
	withMockClient(t, mock)

	stdout, stderr, err := executeCommand("zones", "search", "--lat", "59.3", "--lon", "17.9", "--quiet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Data should still appear on stdout
	if !strings.Contains(stdout, "80500") {
		t.Errorf("expected zone data on stdout even with --quiet, got: %q", stdout)
	}
	// Status messages should be suppressed on stderr
	if strings.Contains(stderr, "Found") || strings.Contains(stderr, "Searching") {
		t.Errorf("--quiet should suppress status messages on stderr, got: %q", stderr)
	}
}

// Error JSON envelope: zones search missing flags with --json
func TestZonesSearch_MissingFlags_JSON_Error(t *testing.T) {
	stdout, _, _ := executeCommandFull("zones", "search", "--json")
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected valid JSON error envelope, got: %s", stdout)
	}
	if envelope["success"] != false {
		t.Errorf("expected success=false, got %v", envelope["success"])
	}
}

// --- Zones info command tests ---

func TestZonesInfo_Success(t *testing.T) {
	mock := &mockAPI{
		getZoneByCodeResp: &parkster.Zone{
			ID:       80500,
			Name:     "Ericsson Kista",
			ZoneCode: "80500",
			City:     parkster.City{Name: "Stockholm"},
			FeeZone: parkster.FeeZone{
				ID:       27545,
				Currency: parkster.Currency{Code: "SEK", Symbol: "kr"},
				ParkingFees: []parkster.ParkingFee{
					{AmountPerHour: 10.0, Description: "Weekdays 8-18", StartTime: 480, EndTime: 1080},
				},
			},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("zones", "info", "80500", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stdout, "80500") {
		t.Errorf("expected zone code in output, got: %q", stdout)
	}
	if !strings.Contains(stdout, "10") || !strings.Contains(stdout, "kr") {
		t.Errorf("expected pricing info in output, got: %q", stdout)
	}
}

func TestZonesInfo_Success_JSON(t *testing.T) {
	mock := &mockAPI{
		getZoneByCodeResp: &parkster.Zone{
			ID:       80500,
			Name:     "Ericsson Kista",
			ZoneCode: "80500",
			City:     parkster.City{Name: "Stockholm"},
			FeeZone: parkster.FeeZone{
				ID:       27545,
				Currency: parkster.Currency{Code: "SEK", Symbol: "kr"},
			},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("zones", "info", "80500", "--lat", "59.373", "--lon", "17.893", "--json")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected valid JSON, got parse error: %v\nOutput: %s", err, stdout)
	}
	if !envelope.Success {
		t.Error("expected success=true in JSON envelope")
	}
	dataBytes, _ := json.Marshal(envelope.Data)
	if !strings.Contains(string(dataBytes), "80500") {
		t.Error("expected zone data in JSON output")
	}
}

func TestZonesInfo_WithLatLon_Success(t *testing.T) {
	mock := &mockAPI{
		getZoneByCodeResp: &parkster.Zone{
			ID:       17429,
			Name:     "Ericsson Kista",
			ZoneCode: "80500",
			FeeZone:  parkster.FeeZone{ID: 27545},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("zones", "info", "80500", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success with zone code + lat/lon, got: %v", err)
	}
	if !strings.Contains(stdout, "Ericsson Kista") {
		t.Errorf("expected zone name in output, got: %q", stdout)
	}
}

func TestZonesInfo_NonNumericCode_MissingLatLon_Error(t *testing.T) {
	_, _, err := executeCommand("zones", "info", "ABC123")
	if err == nil {
		t.Fatal("expected error for non-numeric code without --lat/--lon, got nil")
	}
	if !strings.Contains(err.Error(), "lat") || !strings.Contains(err.Error(), "lon") {
		t.Errorf("expected error about lat/lon required, got: %v", err)
	}
}

func TestZonesInfo_MissingArg_Error(t *testing.T) {
	_, _, err := executeCommand("zones", "info")
	if err == nil {
		t.Fatal("expected error for missing zone code argument, got nil")
	}
}

func TestZonesInfo_NotFound_Error(t *testing.T) {
	mock := &mockAPI{
		getZoneByCodeErr: errors.New("zone not found"),
		getZoneErr:       errors.New("zone not found"),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("zones", "info", "99999", "--lat", "59.373", "--lon", "17.893")
	if err == nil {
		t.Fatal("expected error when zone not found, got nil")
	}
	if !strings.Contains(err.Error(), "zone") {
		t.Errorf("expected 'zone' in error message, got: %v", err)
	}
}

func TestZonesInfo_HelpText_SaysZoneCode(t *testing.T) {
	stdout, _, err := executeCommand("zones", "info", "--help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(stdout, "zone-code-or-id") {
		t.Error("help text should not say 'zone-code-or-id', should say 'zone-code'")
	}
	if !strings.Contains(stdout, "zone-code") {
		t.Error("help text should mention 'zone-code'")
	}
}

func TestZonesInfo_HumanOutput_NoInternalIDs(t *testing.T) {
	mock := &mockAPI{
		getZoneByCodeResp: &parkster.Zone{
			ID: 17429, Name: "Ericsson", ZoneCode: "80500",
			City: parkster.City{Name: "Kista"},
			FeeZone: parkster.FeeZone{
				ID:       27545,
				Currency: parkster.Currency{Code: "SEK", Symbol: "kr"},
			},
		},
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommand("zones", "info", "80500", "--lat", "59.373", "--lon", "17.893")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !strings.Contains(stdout, "80500") {
		t.Errorf("expected zone code, got: %q", stdout)
	}
	if strings.Contains(stdout, "17429") {
		t.Errorf("should not show internal zone ID, got: %q", stdout)
	}
	if strings.Contains(stdout, "27545") {
		t.Errorf("should not show fee zone ID, got: %q", stdout)
	}
	if strings.Contains(stdout, "{") {
		t.Errorf("should not show curly braces, got: %q", stdout)
	}
}

// --- zones info missing lat/lon pairing validation ---

func TestZonesInfo_LatWithoutLon_Error(t *testing.T) {
	_, _, err := executeCommand("zones", "info", "80500", "--lat", "59.37")
	if err == nil {
		t.Fatal("expected error for --lat without --lon, got nil")
	}
	if !strings.Contains(err.Error(), "together") {
		t.Errorf("expected 'together' in error, got: %v", err)
	}
}

func TestZonesInfo_LonWithoutLat_Error(t *testing.T) {
	_, _, err := executeCommand("zones", "info", "80500", "--lon", "17.89")
	if err == nil {
		t.Fatal("expected error for --lon without --lat, got nil")
	}
	if !strings.Contains(err.Error(), "together") {
		t.Errorf("expected 'together' in error, got: %v", err)
	}
}

// --- Edge case: zones info with lat only or lon only ---

func TestZonesInfo_LatOnly_Error(t *testing.T) {
	_, _, err := executeCommand("zones", "info", "80500", "--lat", "59.3")
	if err == nil {
		t.Fatal("expected error for --lat without --lon")
	}
	if !strings.Contains(err.Error(), "--lat and --lon must be used together") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestZonesInfo_LonOnly_Error(t *testing.T) {
	_, _, err := executeCommand("zones", "info", "80500", "--lon", "17.9")
	if err == nil {
		t.Fatal("expected error for --lon without --lat")
	}
	if !strings.Contains(err.Error(), "--lat and --lon must be used together") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestZonesInfo_NumericInput_NoLatLon_HintsAboutCoordinates(t *testing.T) {
	mock := &mockAPI{
		getZoneErr: fmt.Errorf("Parking zone not found."),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("zones", "info", "80500")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--lat") || !strings.Contains(err.Error(), "--lon") {
		t.Errorf("expected hint about --lat/--lon in error, got: %v", err)
	}
}

func TestZonesInfo_NotFound_JSON_ErrorEnvelope(t *testing.T) {
	mock := &mockAPI{
		getZoneErr: fmt.Errorf("Parking zone not found."),
	}
	withMockClient(t, mock)

	stdout, _, err := executeCommandFull("zones", "info", "99999", "--json")
	if err == nil {
		t.Fatal("expected error for zone not found")
	}

	var envelope output.Envelope
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected valid JSON error envelope, got: %v\nstdout: %q", err, stdout)
	}
	if envelope.Success {
		t.Error("expected success=false")
	}
}

// --- zones info: too many positional args ---

func TestZonesInfo_TooManyArgs_Error(t *testing.T) {
	_, _, err := executeCommand("zones", "info", "80500", "extra")
	if err == nil {
		t.Fatal("expected error for too many positional args, got nil")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg(s), received 2") {
		t.Errorf("expected ExactArgs error, got: %v", err)
	}
}

// --- zones search: individual required flag tests ---

func TestZonesSearch_LatOnly_Error(t *testing.T) {
	_, _, err := executeCommand("zones", "search", "--lat", "59.373")
	if err == nil {
		t.Fatal("expected error for --lat without --lon, got nil")
	}
	if !strings.Contains(err.Error(), `"lon"`) {
		t.Errorf("expected error about missing --lon, got: %v", err)
	}
}

func TestZonesSearch_ExtraArgs_Error(t *testing.T) {
	mock := &mockAPI{
		searchZonesResp: &parkster.SearchResult{},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("zones", "search", "--lat", "59.373", "--lon", "17.893", "extra")
	if err == nil {
		t.Fatal("expected error for extra positional args on zones search")
	}
}

func TestZonesSearch_LonOnly_Error(t *testing.T) {
	_, _, err := executeCommand("zones", "search", "--lon", "17.893")
	if err == nil {
		t.Fatal("expected error for --lon without --lat, got nil")
	}
	if !strings.Contains(err.Error(), `"lat"`) {
		t.Errorf("expected error about missing --lat, got: %v", err)
	}
}
