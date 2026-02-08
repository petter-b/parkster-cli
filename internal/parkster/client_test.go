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
	if params.Get("platform") != "cli" {
		t.Errorf("Expected platform 'cli', got %s", params.Get("platform"))
	}
	if params.Get("platformVersion") != "1.0" {
		t.Errorf("Expected platformVersion '1.0', got %s", params.Get("platformVersion"))
	}
	if params.Get("version") != "1" {
		t.Errorf("Expected version '1', got %s", params.Get("version"))
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
		if query.Get("platform") != "cli" {
			t.Errorf("Expected platform 'cli' in query, got %s", query.Get("platform"))
		}
		if query.Get("clientTime") == "" {
			t.Error("Missing clientTime in query")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client and override base URL
	client := NewClient("test@example.com", "password123")

	// Make request to mock server
	resp, err := client.get(strings.TrimPrefix(server.URL, "http://"+server.Listener.Addr().String()), nil)
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
		if r.FormValue("platform") != "cli" {
			t.Errorf("Expected platform 'cli' in body, got %s", r.FormValue("platform"))
		}
		if r.FormValue("clientTime") == "" {
			t.Error("Missing clientTime in body")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	// Create client
	client := NewClient("test@example.com", "password123")

	// Prepare form data
	data := url.Values{}
	data.Set("testKey", "testValue")

	// Make request
	resp, err := client.post(strings.TrimPrefix(server.URL, "http://"+server.Listener.Addr().String()), data)
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
		if r.FormValue("platform") != "cli" {
			t.Errorf("Expected platform 'cli' in body, got %s", r.FormValue("platform"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client
	client := NewClient("test@example.com", "password123")

	// Make request
	resp, err := client.put(strings.TrimPrefix(server.URL, "http://"+server.Listener.Addr().String()), nil)
	if err != nil {
		t.Fatalf("PUT request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}
