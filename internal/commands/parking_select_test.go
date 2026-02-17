package commands

import (
	"testing"
	"time"

	"github.com/petter-b/parkster-cli/internal/parkster"
)

func TestSelectParking_SingleParking_AutoSelects(t *testing.T) {
	setAuth(t)
	now := time.Now()
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{
					ID:          100,
					CheckInTime: now.Add(-30 * time.Minute).UnixMilli(),
					TimeoutTime: now.Add(30 * time.Minute).UnixMilli(),
					ParkingZone: parkster.Zone{ZoneCode: "80500", Name: "Test"},
					Car:         parkster.Car{LicenseNbr: "ABC123"},
					Currency:    parkster.Currency{Code: "SEK"},
				},
			},
		},
	}
	withMockClient(t, mock)

	parking, _, err := selectParking(0)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if parking == nil {
		t.Fatal("expected non-nil parking")
	}
	if parking.ID != 100 {
		t.Errorf("expected parking ID 100, got %d", parking.ID)
	}
}

func TestSelectParking_NoParkings_ReturnsNil(t *testing.T) {
	setAuth(t)
	mock := &mockAPI{
		loginResp: &parkster.User{ID: 1},
	}
	withMockClient(t, mock)

	parking, client, err := selectParking(0)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if parking != nil {
		t.Errorf("expected nil parking for empty list, got %+v", parking)
	}
	if client == nil {
		t.Error("expected non-nil client even with no parkings")
	}
}

func TestSelectParking_ByID_FindsCorrect(t *testing.T) {
	setAuth(t)
	now := time.Now()
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 100, CheckInTime: now.UnixMilli(), TimeoutTime: now.Add(30 * time.Minute).UnixMilli()},
				{ID: 200, CheckInTime: now.UnixMilli(), TimeoutTime: now.Add(60 * time.Minute).UnixMilli()},
			},
		},
	}
	withMockClient(t, mock)

	parking, _, err := selectParking(200)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if parking.ID != 200 {
		t.Errorf("expected parking ID 200, got %d", parking.ID)
	}
}

func TestSelectParking_ByID_NotFound(t *testing.T) {
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

	_, _, err := selectParking(999)
	if err == nil {
		t.Fatal("expected error for non-existent parking ID")
	}
}

func TestSelectParking_Multiple_NoID_Error(t *testing.T) {
	setAuth(t)
	now := time.Now()
	mock := &mockAPI{
		loginResp: &parkster.User{
			ID: 1,
			ShortTermParkings: []parkster.Parking{
				{ID: 100, CheckInTime: now.UnixMilli(), TimeoutTime: now.Add(30 * time.Minute).UnixMilli(), ParkingZone: parkster.Zone{ZoneCode: "A"}, Car: parkster.Car{LicenseNbr: "X"}, Currency: parkster.Currency{Code: "SEK"}},
				{ID: 200, CheckInTime: now.UnixMilli(), TimeoutTime: now.Add(60 * time.Minute).UnixMilli(), ParkingZone: parkster.Zone{ZoneCode: "B"}, Car: parkster.Car{LicenseNbr: "Y"}, Currency: parkster.Currency{Code: "SEK"}},
			},
		},
	}
	withMockClient(t, mock)

	_, _, err := selectParking(0)
	if err == nil {
		t.Fatal("expected error for multiple parkings without --parking-id")
	}
}
