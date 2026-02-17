package output

import (
	"fmt"
	"strings"
	"time"

	"github.com/petter-b/parkster-cli/internal/parkster"
)

// formatTime formats a unix millisecond timestamp as "Today at HH:MM" or "YYYY-MM-DD HH:MM"
func formatTime(ms int64) string {
	t := time.UnixMilli(ms)
	now := time.Now()
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return "Today at " + t.Format("15:04")
	}
	return t.Format("2006-01-02 15:04")
}

// formatRemaining returns a human-friendly remaining duration like "2h 09m remaining"
func formatRemaining(ms int64) string {
	remaining := time.Until(time.UnixMilli(ms))
	if remaining <= 0 {
		return "expired"
	}
	hours := int(remaining.Hours())
	minutes := int(remaining.Minutes()) % 60
	if hours > 0 {
		return fmt.Sprintf("%dh %02dm remaining", hours, minutes)
	}
	return fmt.Sprintf("%dm remaining", minutes)
}

// formatCar returns "Name - Plate" or just "Plate" if no personalization
func formatCar(car parkster.Car) string {
	if car.CarPersonalization.Name != "" {
		return car.CarPersonalization.Name + " - " + car.LicenseNbr
	}
	return car.LicenseNbr
}

// formatZone returns "ZoneCode ZoneName"
func formatZone(zone parkster.Zone) string {
	return zone.ZoneCode + " " + zone.Name
}

// formatCost returns "Amount CurrencyCode" (e.g. "0 SEK")
func formatCost(cost float64, currency parkster.Currency) string {
	code := currency.Code
	if code == "" {
		code = "?"
	}
	if cost == 0 {
		return "0 " + code
	}
	return fmt.Sprintf("%.2f %s", cost, code)
}

// FormatParking formats a parking for status display (full details)
func FormatParking(p parkster.Parking) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Zone:       %s\n", formatZone(p.ParkingZone))
	fmt.Fprintf(&b, "Car:        %s\n", formatCar(p.Car))
	fmt.Fprintf(&b, "Valid from: %s\n", formatTime(p.CheckInTime))
	fmt.Fprintf(&b, "Ends at:    %s (%s)\n", formatTime(p.TimeoutTime), formatRemaining(p.TimeoutTime))
	fmt.Fprintf(&b, "Cost:       %s", formatCost(p.Cost, p.Currency))
	return b.String()
}

// FormatParkingStopped formats a parking after it was stopped
func FormatParkingStopped(p parkster.Parking) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Zone:       %s\n", formatZone(p.ParkingZone))
	fmt.Fprintf(&b, "Car:        %s\n", formatCar(p.Car))
	fmt.Fprintf(&b, "Cost:       %s", formatCost(p.Cost, p.Currency))
	return b.String()
}

// FormatParkingChanged formats a parking after its end time was changed
func FormatParkingChanged(p parkster.Parking) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Zone:       %s\n", formatZone(p.ParkingZone))
	fmt.Fprintf(&b, "Car:        %s\n", formatCar(p.Car))
	fmt.Fprintf(&b, "Ends at:    %s (%s)\n", formatTime(p.TimeoutTime), formatRemaining(p.TimeoutTime))
	fmt.Fprintf(&b, "Cost:       %s", formatCost(p.Cost, p.Currency))
	return b.String()
}

// FormatZoneSearchList formats zone search results as a compact table
func FormatZoneSearchList(zones []parkster.ZoneSearchItem) string {
	var b strings.Builder
	for i, z := range zones {
		if i > 0 {
			b.WriteString("\n")
		}
		if z.City.Name != "" {
			fmt.Fprintf(&b, "%-6s %s, %s", z.ZoneCode, z.Name, z.City.Name)
		} else {
			fmt.Fprintf(&b, "%-6s %s", z.ZoneCode, z.Name)
		}
	}
	return b.String()
}

// FormatZoneInfo formats a zone's full details for display
func FormatZoneInfo(z parkster.Zone) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Zone:     %s %s\n", z.ZoneCode, z.Name)
	if z.City.Name != "" {
		fmt.Fprintf(&b, "City:     %s\n", z.City.Name)
	}
	if z.FeeZone.Currency.Code != "" {
		fmt.Fprintf(&b, "Currency: %s", z.FeeZone.Currency.Code)
		if z.FeeZone.Currency.Symbol != "" {
			fmt.Fprintf(&b, " (%s)", z.FeeZone.Currency.Symbol)
		}
		b.WriteString("\n")
	}
	for _, fee := range z.FeeZone.ParkingFees {
		desc := fee.Description
		if desc == "" {
			desc = fmt.Sprintf("%s-%s", formatMinutesSinceMidnight(fee.StartTime), formatMinutesSinceMidnight(fee.EndTime))
		}
		switch {
		case fee.AmountPerHour > 0:
			fmt.Fprintf(&b, "Rate:     %.2f %s/h (%s)\n", fee.AmountPerHour, z.FeeZone.Currency.Symbol, desc)
		case fee.Description != "":
			fmt.Fprintf(&b, "Rate:     %s\n", fee.Description)
		default:
			fmt.Fprintf(&b, "Rate:     (see parking sign for rates)\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// formatMinutesSinceMidnight converts minutes since midnight to "HH:MM"
func formatMinutesSinceMidnight(m int) string {
	return fmt.Sprintf("%02d:%02d", m/60, m%60)
}

// FormatCarList formats multiple cars for disambiguation display
func FormatCarList(cars []parkster.Car) string {
	var b strings.Builder
	for i, c := range cars {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "  %s", formatCar(c))
	}
	return b.String()
}

// FormatPaymentList formats multiple payment accounts for disambiguation display
func FormatPaymentList(accounts []parkster.PaymentAccount) string {
	var b strings.Builder
	for i, a := range accounts {
		if i > 0 {
			b.WriteString("\n")
		}
		id := a.PaymentAccountID
		if idx := strings.Index(id, ":"); idx >= 0 {
			fmt.Fprintf(&b, "  %-10s %s", id[:idx], id[idx+1:])
		} else {
			fmt.Fprintf(&b, "  %s", id)
		}
	}
	return b.String()
}

// FormatParkingList formats multiple parkings for status display
func FormatParkingList(parkings []parkster.Parking) string {
	var b strings.Builder
	for i, p := range parkings {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(FormatParking(p))
	}
	return b.String()
}
