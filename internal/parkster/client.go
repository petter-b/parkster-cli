package parkster

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.parkster.se/api/mobile/v2"

// API defines the methods commands use to interact with the Parkster API.
type API interface {
	Login() (*User, error)
	GetZone(zoneID int) (*Zone, error)
	SearchZones(lat, lon float64, radius int) (*SearchResult, error)
	GetZoneByCode(code string, lat, lon float64, radiusMeters int) (*Zone, error)
	StartParking(zoneID, feeZoneID, carID int, paymentID string, timeout int) (*Parking, error)
	StopParking(parkingID int) (*Parking, error)
	ExtendParking(parkingID, minutes int) (*Parking, error)
	EstimateCost(zoneID, feeZoneID, carID int, paymentID string, timeout int) (*CostEstimate, error)
}

// Client is the Parkster API client
type Client struct {
	http     *http.Client
	baseURL  string
	username string
	password string
}

// NewClient creates a new Parkster API client
func NewClient(username, password string) *Client {
	return &Client{
		http:     &http.Client{Timeout: 30 * time.Second},
		baseURL:  DefaultBaseURL,
		username: username,
		password: password,
	}
}

// deviceParams returns required device parameters for all requests
func (c *Client) deviceParams() url.Values {
	params := url.Values{}
	params.Set("platform", "ios")
	params.Set("platformVersion", "26.2")
	params.Set("version", "626")
	params.Set("locale", "en_US")
	params.Set("clientTime", fmt.Sprintf("%d", time.Now().UnixMilli()))
	return params
}

// get makes a GET request with device params in query string
func (c *Client) get(path string, extraParams url.Values) (*http.Response, error) {
	params := c.deviceParams()
	for k, v := range extraParams {
		params[k] = v
	}

	reqURL := fmt.Sprintf("%s%s?%s", c.baseURL, path, params.Encode())
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	if c.username != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(c.username + ":" + c.password))
		req.Header.Set("Authorization", "Basic "+auth)
	}
	req.Header.Set("Accept", "application/json")

	return c.http.Do(req)
}

// post makes a POST request with device params in form body
func (c *Client) post(path string, data url.Values) (*http.Response, error) {
	if data == nil {
		data = url.Values{}
	}

	// Merge device params into body
	for k, v := range c.deviceParams() {
		data[k] = v
	}

	reqURL := fmt.Sprintf("%s%s", c.baseURL, path)
	req, err := http.NewRequest("POST", reqURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	if c.username != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(c.username + ":" + c.password))
		req.Header.Set("Authorization", "Basic "+auth)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return c.http.Do(req)
}

// put makes a PUT request with device params in form body
func (c *Client) put(path string, data url.Values) (*http.Response, error) {
	if data == nil {
		data = url.Values{}
	}

	// Merge device params into body
	for k, v := range c.deviceParams() {
		data[k] = v
	}

	reqURL := fmt.Sprintf("%s%s", c.baseURL, path)
	req, err := http.NewRequest("PUT", reqURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	if c.username != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(c.username + ":" + c.password))
		req.Header.Set("Authorization", "Basic "+auth)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return c.http.Do(req)
}

// parseErrorResponse reads a non-200 response body and extracts displayMessage if present.
// Falls back to a generic "description (status N)" message.
func parseErrorResponse(resp *http.Response, description string) error {
	var apiErr APIError
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Data.DisplayMessage != "" {
		return fmt.Errorf("%s", apiErr.Data.DisplayMessage)
	}
	return fmt.Errorf("%s (status %d)", description, resp.StatusCode)
}

// Login authenticates and returns user profile
func (c *Client) Login() (*User, error) {
	resp, err := c.get("/people/login", nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("authentication failed (status %d)", resp.StatusCode)
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &user, nil
}

// GetActiveParkings returns all active parking sessions
func (c *Client) GetActiveParkings() ([]Parking, error) {
	resp, err := c.get("/parkings/short-term", nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, parseErrorResponse(resp, "request failed")
	}

	var parkings []Parking
	if err := json.NewDecoder(resp.Body).Decode(&parkings); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return parkings, nil
}

// GetZone fetches zone details including fee zone ID
func (c *Client) GetZone(zoneID int) (*Zone, error) {
	path := fmt.Sprintf("/parking-zones/%d", zoneID)
	resp, err := c.get(path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, parseErrorResponse(resp, "zone not found")
	}

	var zone Zone
	if err := json.NewDecoder(resp.Body).Decode(&zone); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &zone, nil
}

// SearchZones searches for parking zones by location
func (c *Client) SearchZones(lat, lon float64, radius int) (*SearchResult, error) {
	params := url.Values{}
	params.Set("searchLat", fmt.Sprintf("%.6f", lat))
	params.Set("searchLong", fmt.Sprintf("%.6f", lon))
	params.Set("userLat", fmt.Sprintf("%.6f", lat))
	params.Set("userLong", fmt.Sprintf("%.6f", lon))
	params.Set("radius", fmt.Sprintf("%d", radius))

	resp, err := c.get("/parking-zones/location-search", params)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, parseErrorResponse(resp, "search failed")
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetZoneByCode searches for a zone by sign code near a location
func (c *Client) GetZoneByCode(code string, lat, lon float64, radiusMeters int) (*Zone, error) {
	// Use 500m default for code lookup if radius not specified
	if radiusMeters <= 0 {
		radiusMeters = 500
	}

	result, err := c.SearchZones(lat, lon, radiusMeters)
	if err != nil {
		return nil, fmt.Errorf("zone search failed: %w", err)
	}

	// Search both arrays for matching code
	allZones := append(result.ParkingZonesAtPosition, result.ParkingZonesNearbyPosition...)
	for _, z := range allZones {
		if z.ZoneCode == code {
			// Fetch full details (includes feeZone with pricing)
			return c.GetZone(z.ID)
		}
	}

	return nil, fmt.Errorf("zone code %q not found near %.4f,%.4f", code, lat, lon)
}

// StartParking starts a new parking session
func (c *Client) StartParking(zoneID, feeZoneID, carID int, paymentID string, timeout int) (*Parking, error) {
	data := url.Values{}
	data.Set("parkingZoneId", fmt.Sprintf("%d", zoneID))
	data.Set("feeZoneId", fmt.Sprintf("%d", feeZoneID))
	data.Set("carId", fmt.Sprintf("%d", carID))
	data.Set("paymentAccountId", paymentID)
	data.Set("timeout", fmt.Sprintf("%d", timeout))

	resp, err := c.post("/parkings/short-term", data)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, parseErrorResponse(resp, "failed to start parking")
	}

	var parking Parking
	if err := json.NewDecoder(resp.Body).Decode(&parking); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &parking, nil
}

// StopParking ends a parking session
func (c *Client) StopParking(parkingID int) (*Parking, error) {
	path := fmt.Sprintf("/parkings/short-term/%d/park-out", parkingID)
	resp, err := c.post(path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, parseErrorResponse(resp, "failed to stop parking")
	}

	var parking Parking
	if err := json.NewDecoder(resp.Body).Decode(&parking); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &parking, nil
}

// ExtendParking adds more time to a parking session
// Note: uses "offset" parameter to ADD minutes, not "timeout" to SET absolute timeout
func (c *Client) ExtendParking(parkingID, minutes int) (*Parking, error) {
	path := fmt.Sprintf("/parkings/short-term/%d/timeout", parkingID)
	data := url.Values{}
	data.Set("offset", fmt.Sprintf("%d", minutes))

	resp, err := c.put(path, data)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, parseErrorResponse(resp, "failed to extend parking")
	}

	var parking Parking
	if err := json.NewDecoder(resp.Body).Decode(&parking); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &parking, nil
}

// EstimateCost returns the probable cost of a parking session
func (c *Client) EstimateCost(zoneID, feeZoneID, carID int, paymentID string, timeout int) (*CostEstimate, error) {
	params := url.Values{}
	params.Set("parkingZoneId", fmt.Sprintf("%d", zoneID))
	params.Set("feeZoneId", fmt.Sprintf("%d", feeZoneID))
	params.Set("carId", fmt.Sprintf("%d", carID))
	params.Set("paymentAccountId", paymentID)
	params.Set("timeout", fmt.Sprintf("%d", timeout))

	resp, err := c.get("/parkings/short-term/probable-cost", params)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, parseErrorResponse(resp, "failed to estimate cost")
	}

	var estimate CostEstimate
	if err := json.NewDecoder(resp.Body).Decode(&estimate); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &estimate, nil
}
