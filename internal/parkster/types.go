package parkster

// User represents a Parkster account
type User struct {
	ID                int              `json:"id"`
	Email             string           `json:"email"`
	AccountType       string           `json:"accountType"`
	Cars              []Car            `json:"cars"`
	PaymentAccounts   []PaymentAccount `json:"paymentAccounts"`
	ShortTermParkings []Parking        `json:"shortTermParkings"`
}

// CarPersonalization contains user-assigned vehicle details
type CarPersonalization struct {
	Name string `json:"name"`
}

// Car represents a registered vehicle
type Car struct {
	ID                 int                `json:"id"`
	LicenseNbr         string             `json:"licenseNbr"`
	CountryCode        string             `json:"countryCode"`
	CarPersonalization CarPersonalization `json:"carPersonalization"`
}

// PaymentAccount represents a payment method
type PaymentAccount struct {
	PaymentAccountID string `json:"paymentAccountId"`
}

// City represents a city/municipality
type City struct {
	Name string `json:"name"`
}

// ParkingFee represents a time-based fee rule
type ParkingFee struct {
	AmountPerHour float64 `json:"amountPerHour"`
	Description   string  `json:"description"`
	StartTime     int     `json:"startTime"` // Minutes since midnight
	EndTime       int     `json:"endTime"`   // Minutes since midnight
}

// Zone represents a parking zone
type Zone struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	ZoneCode  string  `json:"zoneCode"`
	City      City    `json:"city"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	FeeZone   FeeZone `json:"feeZone"`
}

// FeeZone contains pricing information
type FeeZone struct {
	ID          int          `json:"id"`
	Currency    Currency     `json:"currency"`
	ParkingFees []ParkingFee `json:"parkingFees"`
}

// Currency represents money denomination
type Currency struct {
	Code   string `json:"code"`
	Symbol string `json:"symbol"`
}

// ZoneSearchItem represents a zone from location search results
type ZoneSearchItem struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	ZoneCode  string  `json:"zoneCode"`
	City      City    `json:"city"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Distance  int     `json:"distance,omitempty"`
}

// SearchResult holds the two arrays from location search API
type SearchResult struct {
	ParkingZonesAtPosition     []ZoneSearchItem `json:"parkingZonesAtPosition"`
	ParkingZonesNearbyPosition []ZoneSearchItem `json:"parkingZonesNearbyPosition"`
}

// Parking represents an active or completed parking session
type Parking struct {
	ID          int      `json:"id"`
	ParkingZone Zone     `json:"parkingZone"`
	Car         Car      `json:"car"`
	CheckInTime int64    `json:"checkInTime"`
	TimeoutTime int64    `json:"timeoutTime"`
	Cost        float64  `json:"cost"`
	TotalCost   float64  `json:"totalCost"`
	Currency    Currency `json:"currency"`
}

// CostEstimate represents the probable cost of a parking session
type CostEstimate struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}
