package parkster

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test@example.com", "password123")

	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.email != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", client.email)
	}
	if client.password != "password123" {
		t.Errorf("Expected password password123, got %s", client.password)
	}
	if client.http == nil {
		t.Error("HTTP client not initialized")
	}
}

func TestDeviceParams(t *testing.T) {
	client := NewClient("test@example.com", "password")
	params := client.deviceParams()

	// Check all required parameters exist
	// Must use ios platform params — API rejects platform=cli with "Client too old"
	if params.Get("platform") != "ios" {
		t.Errorf("Expected platform 'ios', got %s", params.Get("platform"))
	}
	if params.Get("platformVersion") != "26.2" {
		t.Errorf("Expected platformVersion '26.2', got %s", params.Get("platformVersion"))
	}
	if params.Get("version") != "626" {
		t.Errorf("Expected version '626', got %s", params.Get("version"))
	}
	if params.Get("locale") != "en_US" {
		t.Errorf("Expected locale 'en_US', got %s", params.Get("locale"))
	}
	if params.Get("clientTime") == "" {
		t.Error("clientTime not set")
	}
}

func TestGet_BasicAuth(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Basic Auth header
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Basic ") {
			t.Error("Missing Basic auth prefix")
		}

		// Decode and verify credentials
		encoded := strings.TrimPrefix(auth, "Basic ")
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			t.Fatalf("Failed to decode auth: %v", err)
		}

		if string(decoded) != "test@example.com:password123" {
			t.Errorf("Expected credentials 'test@example.com:password123', got %s", string(decoded))
		}

		// Verify Accept header
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Expected Accept 'application/json', got %s", r.Header.Get("Accept"))
		}

		// Verify device params in query string
		query := r.URL.Query()
		if query.Get("platform") != "ios" {
			t.Errorf("Expected platform 'ios' in query, got %s", query.Get("platform"))
		}
		if query.Get("clientTime") == "" {
			t.Error("Missing clientTime in query")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client and override base URL
	client := NewClient("test@example.com", "password123")
	client.baseURL = server.URL

	// Make request to mock server
	resp, err := client.get("", nil)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestPost_FormEncoded(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Verify Content-Type
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Expected form-encoded content type, got %s", r.Header.Get("Content-Type"))
		}

		// Verify Basic Auth
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Basic ") {
			t.Error("Missing Basic auth")
		}

		// Parse form data
		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		// Verify custom data
		if r.FormValue("testKey") != "testValue" {
			t.Errorf("Expected testKey=testValue, got %s", r.FormValue("testKey"))
		}

		// Verify device params in body
		if r.FormValue("platform") != "ios" {
			t.Errorf("Expected platform 'ios' in body, got %s", r.FormValue("platform"))
		}
		if r.FormValue("clientTime") == "" {
			t.Error("Missing clientTime in body")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	// Create client and override base URL
	client := NewClient("test@example.com", "password123")
	client.baseURL = server.URL

	// Prepare form data
	data := url.Values{}
	data.Set("testKey", "testValue")

	// Make request
	resp, err := client.post("", data)
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestPut_FormEncoded(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method
		if r.Method != "PUT" {
			t.Errorf("Expected PUT, got %s", r.Method)
		}

		// Verify Content-Type
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Expected form-encoded content type, got %s", r.Header.Get("Content-Type"))
		}

		// Verify Basic Auth
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Basic ") {
			t.Error("Missing Basic auth")
		}

		// Parse form data
		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		// Verify device params in body
		if r.FormValue("platform") != "ios" {
			t.Errorf("Expected platform 'ios' in body, got %s", r.FormValue("platform"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client and override base URL
	client := NewClient("test@example.com", "password123")
	client.baseURL = server.URL

	// Make request
	resp, err := client.put("", nil)
	if err != nil {
		t.Fatalf("PUT request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// --- API method tests (Task 4) ---

func newTestClient(serverURL string) *Client {
	client := NewClient("test@example.com", "password123")
	client.baseURL = serverURL
	return client
}

func TestLogin_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/people/login" {
			t.Errorf("Expected path /people/login, got %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(User{
			ID:          12345,
			Email:       "test@example.com",
			AccountType: "PRIVATE",
			Cars:        []Car{{ID: 67890, LicenseNbr: "ABC123", CountryCode: "SE"}},
			PaymentAccounts: []PaymentAccount{{PaymentAccountID: "pay_123"}},
			ShortTermParkings: []Parking{
				{ID: 999, Status: "ACTIVE", Car: Car{LicenseNbr: "ABC123"}},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	user, err := client.Login()
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if user.ID != 12345 {
		t.Errorf("Expected ID 12345, got %d", user.ID)
	}
	if user.Email != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", user.Email)
	}
	if len(user.Cars) != 1 {
		t.Fatalf("Expected 1 car, got %d", len(user.Cars))
	}
	if user.Cars[0].LicenseNbr != "ABC123" {
		t.Errorf("Expected license ABC123, got %s", user.Cars[0].LicenseNbr)
	}
	if len(user.ShortTermParkings) != 1 {
		t.Fatalf("Expected 1 active parking, got %d", len(user.ShortTermParkings))
	}
	if user.ShortTermParkings[0].ID != 999 {
		t.Errorf("Expected parking ID 999, got %d", user.ShortTermParkings[0].ID)
	}
}

func TestLogin_AuthFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.Login()
	if err == nil {
		t.Fatal("Expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("Expected 'authentication failed' in error, got: %s", err.Error())
	}
}

func TestGetActiveParkings_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/parkings/short-term" {
			t.Errorf("Expected path /parkings/short-term, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Parking{
			{ID: 111, Status: "ACTIVE", Car: Car{LicenseNbr: "ABC123"}},
			{ID: 222, Status: "ACTIVE", Car: Car{LicenseNbr: "DEF456"}},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	parkings, err := client.GetActiveParkings()
	if err != nil {
		t.Fatalf("GetActiveParkings failed: %v", err)
	}
	if len(parkings) != 2 {
		t.Fatalf("Expected 2 parkings, got %d", len(parkings))
	}
	if parkings[0].ID != 111 {
		t.Errorf("Expected first parking ID 111, got %d", parkings[0].ID)
	}
}

func TestGetActiveParkings_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Parking{})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	parkings, err := client.GetActiveParkings()
	if err != nil {
		t.Fatalf("GetActiveParkings failed: %v", err)
	}
	if len(parkings) != 0 {
		t.Errorf("Expected 0 parkings, got %d", len(parkings))
	}
}

func TestGetZone_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/parking-zones/17429" {
			t.Errorf("Expected path /parking-zones/17429, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Zone{
			ID:   17429,
			Name: "Ericsson Kista",
			FeeZone: FeeZone{
				ID:       27545,
				Currency: Currency{Code: "SEK", Symbol: "kr"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	zone, err := client.GetZone(17429)
	if err != nil {
		t.Fatalf("GetZone failed: %v", err)
	}
	if zone.ID != 17429 {
		t.Errorf("Expected zone ID 17429, got %d", zone.ID)
	}
	if zone.FeeZone.ID != 27545 {
		t.Errorf("Expected fee zone ID 27545, got %d", zone.FeeZone.ID)
	}
}

func TestGetZone_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.GetZone(99999)
	if err == nil {
		t.Fatal("Expected error for 404 response")
	}
	if !strings.Contains(err.Error(), "zone not found") {
		t.Errorf("Expected 'zone not found' in error, got: %s", err.Error())
	}
}

func TestStartParking_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/parkings/short-term" {
			t.Errorf("Expected path /parkings/short-term, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}
		if r.FormValue("parkingZoneId") != "17429" {
			t.Errorf("Expected parkingZoneId=17429, got %s", r.FormValue("parkingZoneId"))
		}
		if r.FormValue("feeZoneId") != "27545" {
			t.Errorf("Expected feeZoneId=27545, got %s", r.FormValue("feeZoneId"))
		}
		if r.FormValue("carId") != "67890" {
			t.Errorf("Expected carId=67890, got %s", r.FormValue("carId"))
		}
		if r.FormValue("paymentAccountId") != "pay_123" {
			t.Errorf("Expected paymentAccountId=pay_123, got %s", r.FormValue("paymentAccountId"))
		}
		if r.FormValue("timeout") != "30" {
			t.Errorf("Expected timeout=30, got %s", r.FormValue("timeout"))
		}
		// Verify device params are in form body
		if r.FormValue("platform") != "ios" {
			t.Errorf("Expected platform=cli in body, got %s", r.FormValue("platform"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Parking{
			ID:     123456,
			Status: "ACTIVE",
			Car:    Car{ID: 67890, LicenseNbr: "ABC123"},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	parking, err := client.StartParking(17429, 27545, 67890, "pay_123", 30)
	if err != nil {
		t.Fatalf("StartParking failed: %v", err)
	}
	if parking.ID != 123456 {
		t.Errorf("Expected parking ID 123456, got %d", parking.ID)
	}
	if parking.Status != "ACTIVE" {
		t.Errorf("Expected status ACTIVE, got %s", parking.Status)
	}
}

func TestStartParking_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.StartParking(17429, 27545, 67890, "pay_123", 30)
	if err == nil {
		t.Fatal("Expected error for 400 response")
	}
	if !strings.Contains(err.Error(), "failed to start parking") {
		t.Errorf("Expected 'failed to start parking' in error, got: %s", err.Error())
	}
}

func TestStopParking_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/parkings/short-term/123456/park-out" {
			t.Errorf("Expected path /parkings/short-term/123456/park-out, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Parking{
			ID:     123456,
			Status: "COMPLETED",
			Cost:   15.50,
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	parking, err := client.StopParking(123456)
	if err != nil {
		t.Fatalf("StopParking failed: %v", err)
	}
	if parking.ID != 123456 {
		t.Errorf("Expected parking ID 123456, got %d", parking.ID)
	}
	if parking.Status != "COMPLETED" {
		t.Errorf("Expected status COMPLETED, got %s", parking.Status)
	}
	if parking.Cost != 15.50 {
		t.Errorf("Expected cost 15.50, got %f", parking.Cost)
	}
}

func TestStopParking_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.StopParking(99999)
	if err == nil {
		t.Fatal("Expected error for 404 response")
	}
}

func TestExtendParking_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/parkings/short-term/123456/timeout" {
			t.Errorf("Expected path /parkings/short-term/123456/timeout, got %s", r.URL.Path)
		}
		if r.Method != "PUT" {
			t.Errorf("Expected PUT, got %s", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}
		// Must use "offset" not "timeout"
		if r.FormValue("offset") != "30" {
			t.Errorf("Expected offset=30, got %s", r.FormValue("offset"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Parking{
			ID:      123456,
			Status:  "ACTIVE",
			Timeout: 60,
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	parking, err := client.ExtendParking(123456, 30)
	if err != nil {
		t.Fatalf("ExtendParking failed: %v", err)
	}
	if parking.ID != 123456 {
		t.Errorf("Expected parking ID 123456, got %d", parking.ID)
	}
	if parking.Timeout != 60 {
		t.Errorf("Expected timeout 60, got %d", parking.Timeout)
	}
}

func TestExtendParking_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.ExtendParking(123456, 30)
	if err == nil {
		t.Fatal("Expected error for 400 response")
	}
	if !strings.Contains(err.Error(), "failed to extend parking") {
		t.Errorf("Expected 'failed to extend parking' in error, got: %s", err.Error())
	}
}
