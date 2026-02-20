package commands

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/petter-b/parkster-cli/internal/parkster"
)

func TestExitError_Code(t *testing.T) {
	err := &ExitError{Code: ExitAuth, Err: errors.New("not authenticated")}
	if err.Code != 3 {
		t.Errorf("expected code 3, got %d", err.Code)
	}
}

func TestExitError_Error(t *testing.T) {
	err := &ExitError{Code: ExitAuth, Err: errors.New("not authenticated")}
	if err.Error() != "not authenticated" {
		t.Errorf("expected 'not authenticated', got %q", err.Error())
	}
}

func TestExitError_Unwrap(t *testing.T) {
	inner := errors.New("inner")
	err := &ExitError{Code: ExitAuth, Err: inner}
	if !errors.Is(err, inner) {
		t.Error("expected Unwrap to return inner error")
	}
}

func TestExitError_Silent(t *testing.T) {
	err := &ExitError{Code: ExitAuth, Silent: true}
	if !err.Silent {
		t.Error("expected Silent to be true")
	}
}

func TestExitCode_ExtractsCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil error", nil, 0},
		{"plain error", errors.New("oops"), ExitGeneral},
		{"exit error", &ExitError{Code: ExitAuth, Err: errors.New("auth")}, ExitAuth},
		{"wrapped exit error", fmt.Errorf("wrapped: %w", &ExitError{Code: ExitAPI, Err: errors.New("api")}), ExitAPI},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExitCode(tt.err); got != tt.want {
				t.Errorf("ExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestAuthRequiredError_ExitCode(t *testing.T) {
	err := authRequiredError()
	if ExitCode(err) != ExitAuth {
		t.Errorf("expected exit code %d (auth), got %d", ExitAuth, ExitCode(err))
	}
}

func TestExecute_ExitError_PreservesCode(t *testing.T) {
	// Trigger a usage error (unknown flag) — Cobra returns these directly
	_, _, err := executeCommandFull("--bogus-flag")
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if ExitCode(err) != ExitUsage {
		t.Errorf("expected exit code %d (usage), got %d", ExitUsage, ExitCode(err))
	}
}

func TestSelectParking_NotFound_ExitCode(t *testing.T) {
	setAuth(t)
	now := time.Now()
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 100, CheckInTime: now.UnixMilli(), TimeoutTime: now.Add(30 * time.Minute).UnixMilli()},
			},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("stop", "--parking-id", "999")
	if err == nil {
		t.Fatal("expected error for parking not found")
	}
	if ExitCode(err) != ExitNotFound {
		t.Errorf("expected exit code %d (not found), got %d", ExitNotFound, ExitCode(err))
	}
}

func TestSelectParking_Multiple_ExitCode(t *testing.T) {
	setAuth(t)
	now := time.Now()
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 100, ParkingZone: parkster.Zone{ZoneCode: "80500"}, Car: parkster.Car{LicenseNbr: "ABC123"}, CheckInTime: now.UnixMilli(), TimeoutTime: now.Add(30 * time.Minute).UnixMilli()},
				{ID: 200, ParkingZone: parkster.Zone{ZoneCode: "80501"}, Car: parkster.Car{LicenseNbr: "DEF456"}, CheckInTime: now.UnixMilli(), TimeoutTime: now.Add(60 * time.Minute).UnixMilli()},
			},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("stop")
	if err == nil {
		t.Fatal("expected error for multiple parkings")
	}
	if ExitCode(err) != ExitUsage {
		t.Errorf("expected exit code %d (usage), got %d", ExitUsage, ExitCode(err))
	}
}

func TestStart_ZoneNotFound_ExitCode(t *testing.T) {
	setAuth(t)
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeErr: fmt.Errorf("zone not found"),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "99999", "--duration", "30", "--lat", "59.37", "--lon", "17.89")
	if err == nil {
		t.Fatal("expected error for zone not found")
	}
	if ExitCode(err) != ExitNotFound {
		t.Errorf("expected exit code %d (not found), got %d", ExitNotFound, ExitCode(err))
	}
}

func TestStart_MultipleCars_ExitCode(t *testing.T) {
	setAuth(t)
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			Cars: []parkster.Car{
				{ID: 100, LicenseNbr: "ABC123"},
				{ID: 200, LicenseNbr: "DEF456"},
			},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--lat", "59.37", "--lon", "17.89")
	if err == nil {
		t.Fatal("expected error for multiple cars")
	}
	if ExitCode(err) != ExitUsage {
		t.Errorf("expected exit code %d (usage), got %d", ExitUsage, ExitCode(err))
	}
}

func TestStart_InvalidDuration_ExitCode(t *testing.T) {
	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "-5", "--lat", "59.37", "--lon", "17.89")
	if err == nil {
		t.Fatal("expected error for negative duration")
	}
	if ExitCode(err) != ExitUsage {
		t.Errorf("expected exit code %d (usage), got %d", ExitUsage, ExitCode(err))
	}
}

func TestStart_LoginFails_ExitCode(t *testing.T) {
	setAuth(t)
	mock := &mockAPI{
		loginErr: fmt.Errorf("network timeout"),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--lat", "59.37", "--lon", "17.89")
	if err == nil {
		t.Fatal("expected error for login failure")
	}
	if ExitCode(err) != ExitAPI {
		t.Errorf("expected exit code %d (API), got %d", ExitAPI, ExitCode(err))
	}
}

func TestStart_StartParkingFails_ExitCode(t *testing.T) {
	setAuth(t)
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID:              1,
			Cars:            []parkster.Car{{ID: 100, LicenseNbr: "ABC123"}},
			PaymentAccounts: []parkster.PaymentAccount{{PaymentAccountID: "pay1"}},
		},
		getZoneByCodeResp: &parkster.Zone{ID: 17429, ZoneCode: "80500", FeeZone: parkster.FeeZone{ID: 27545}},
		startParkingErr:   fmt.Errorf("server error"),
	}
	withMockClient(t, mock)

	_, _, err := executeCommand("start", "--zone", "80500", "--duration", "30", "--lat", "59.37", "--lon", "17.89")
	if err == nil {
		t.Fatal("expected error for start parking failure")
	}
	if ExitCode(err) != ExitAPI {
		t.Errorf("expected exit code %d (API), got %d", ExitAPI, ExitCode(err))
	}
}
