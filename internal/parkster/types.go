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

// Car represents a registered vehicle
type Car struct {
	ID          int    `json:"id"`
	LicenseNbr  string `json:"licenseNbr"`
	CountryCode string `json:"countryCode"`
}

// PaymentAccount represents a payment method
type PaymentAccount struct {
	PaymentAccountID string `json:"paymentAccountId"`
}

// Zone represents a parking zone
type Zone struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	FeeZone FeeZone `json:"feeZone"`
}

// FeeZone contains pricing information
type FeeZone struct {
	ID       int      `json:"id"`
	Currency Currency `json:"currency"`
}

// Currency represents money denomination
type Currency struct {
	Code   string `json:"code"`
	Symbol string `json:"symbol"`
}

// Parking represents an active or completed parking session
type Parking struct {
	ID          int     `json:"id"`
	ParkingZone Zone    `json:"parkingZone"`
	Car         Car     `json:"car"`
	StartTime   string  `json:"startTime"`
	Timeout     int     `json:"timeout"`
	Cost        float64 `json:"cost"`
	Status      string  `json:"status"`
}
