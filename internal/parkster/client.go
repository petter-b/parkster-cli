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

// Client is the Parkster API client
type Client struct {
	http    *http.Client
	baseURL string
	email   string
	password string
}

// NewClient creates a new Parkster API client
func NewClient(email, password string) *Client {
	return &Client{
		http:     &http.Client{Timeout: 30 * time.Second},
		baseURL:  DefaultBaseURL,
		email:    email,
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
	if extraParams != nil {
		for k, v := range extraParams {
			params[k] = v
		}
	}

	reqURL := fmt.Sprintf("%s%s?%s", c.baseURL, path, params.Encode())
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	// HTTP Basic Auth
	auth := base64.StdEncoding.EncodeToString([]byte(c.email + ":" + c.password))
	req.Header.Set("Authorization", "Basic "+auth)
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

	// HTTP Basic Auth
	auth := base64.StdEncoding.EncodeToString([]byte(c.email + ":" + c.password))
	req.Header.Set("Authorization", "Basic "+auth)
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

	// HTTP Basic Auth
	auth := base64.StdEncoding.EncodeToString([]byte(c.email + ":" + c.password))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return c.http.Do(req)
}

// Login authenticates and returns user profile
func (c *Client) Login() (*User, error) {
	resp, err := c.get("/people/login", nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

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
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("request failed (status %d)", resp.StatusCode)
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
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("zone not found (status %d)", resp.StatusCode)
	}

	var zone Zone
	if err := json.NewDecoder(resp.Body).Decode(&zone); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &zone, nil
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
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to start parking (status %d)", resp.StatusCode)
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
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to stop parking (status %d)", resp.StatusCode)
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
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to extend parking (status %d)", resp.StatusCode)
	}

	var parking Parking
	if err := json.NewDecoder(resp.Body).Decode(&parking); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &parking, nil
}
