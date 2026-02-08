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

const BaseURL = "https://api.parkster.se/api/mobile/v2"

// Client is the Parkster API client
type Client struct {
	http     *http.Client
	email    string
	password string
}

// NewClient creates a new Parkster API client
func NewClient(email, password string) *Client {
	return &Client{
		http:     &http.Client{Timeout: 30 * time.Second},
		email:    email,
		password: password,
	}
}

// deviceParams returns required device parameters for all requests
func (c *Client) deviceParams() url.Values {
	params := url.Values{}
	params.Set("platform", "cli")
	params.Set("platformVersion", "1.0")
	params.Set("version", "1")
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

	reqURL := fmt.Sprintf("%s%s?%s", BaseURL, path, params.Encode())
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

	reqURL := fmt.Sprintf("%s%s", BaseURL, path)
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

	reqURL := fmt.Sprintf("%s%s", BaseURL, path)
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
