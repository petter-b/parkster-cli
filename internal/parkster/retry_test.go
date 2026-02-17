package parkster

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestDoWithRetry_SuccessFirstAttempt(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewClient("u", "p")
	client.baseURL = server.URL

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := client.doWithRetry(req)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestDoWithRetry_RetriesOn500ThenSucceeds(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewClient("u", "p")
	client.baseURL = server.URL
	client.retryBaseBackoff = 10 * time.Millisecond // fast for tests

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := client.doWithRetry(req)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200 after retries, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestDoWithRetry_ExhaustsRetries_ReturnsLastResponse(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(503)
		_, _ = w.Write([]byte(`{"error":"service unavailable"}`))
	}))
	defer server.Close()

	client := NewClient("u", "p")
	client.baseURL = server.URL
	client.retryBaseBackoff = 10 * time.Millisecond

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := client.doWithRetry(req)
	if err != nil {
		t.Fatalf("expected last response returned (not error), got: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Should return the last 503 response so callers can handle it normally
	if resp.StatusCode != 503 {
		t.Errorf("expected 503, got %d", resp.StatusCode)
	}
	// maxRetries=3, so 1 initial + 3 retries = 4 total
	if atomic.LoadInt32(&attempts) != 4 {
		t.Errorf("expected 4 total attempts, got %d", attempts)
	}
}

func TestDoWithRetry_NoRetryOn4xx(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := NewClient("u", "p")
	client.baseURL = server.URL

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := client.doWithRetry(req)
	if err != nil {
		t.Fatalf("4xx should return response without error, got: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("should not retry on 4xx, got %d attempts", attempts)
	}
}

func TestDoWithRetry_NoRetryOn401(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(401)
	}))
	defer server.Close()

	client := NewClient("u", "p")
	client.baseURL = server.URL

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := client.doWithRetry(req)
	if err != nil {
		t.Fatalf("401 should return response without error, got: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 401 {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("should not retry on 401, got %d attempts", attempts)
	}
}

func TestDoWithRetry_CallsOnRetry(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 2 {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	var retryAttempts []int
	client := NewClient("u", "p")
	client.baseURL = server.URL
	client.retryBaseBackoff = 10 * time.Millisecond
	client.onRetry = func(attempt int, backoff time.Duration) {
		retryAttempts = append(retryAttempts, attempt)
	}

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := client.doWithRetry(req)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if len(retryAttempts) != 1 {
		t.Errorf("expected 1 retry callback, got %d", len(retryAttempts))
	}
	if len(retryAttempts) > 0 && retryAttempts[0] != 1 {
		t.Errorf("expected retry attempt 1, got %d", retryAttempts[0])
	}
}

func TestDoWithRetry_ExponentialBackoff(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n <= 3 {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	var backoffs []time.Duration
	client := NewClient("u", "p")
	client.baseURL = server.URL
	client.retryBaseBackoff = 10 * time.Millisecond
	client.onRetry = func(attempt int, backoff time.Duration) {
		backoffs = append(backoffs, backoff)
	}

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := client.doWithRetry(req)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if len(backoffs) != 3 {
		t.Fatalf("expected 3 retry callbacks, got %d", len(backoffs))
	}
	// Exponential: base*1, base*2, base*4
	if backoffs[0] != 10*time.Millisecond {
		t.Errorf("retry 1: expected 10ms backoff, got %v", backoffs[0])
	}
	if backoffs[1] != 20*time.Millisecond {
		t.Errorf("retry 2: expected 20ms backoff, got %v", backoffs[1])
	}
	if backoffs[2] != 40*time.Millisecond {
		t.Errorf("retry 3: expected 40ms backoff, got %v", backoffs[2])
	}
}

func TestDoWithRetry_NetworkError_Retries(t *testing.T) {
	// Start a server and close it immediately to simulate connection refused
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close() // close immediately so requests fail

	client := NewClient("u", "p")
	client.baseURL = serverURL
	client.retryBaseBackoff = 10 * time.Millisecond

	var retryCount int
	client.onRetry = func(attempt int, backoff time.Duration) {
		retryCount++
	}

	req, _ := http.NewRequest("GET", serverURL+"/test", nil)
	_, err := client.doWithRetry(req)
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
	// Should have retried 3 times (maxRetries=3)
	if retryCount != 3 {
		t.Errorf("expected 3 retries for network error, got %d", retryCount)
	}
}

func TestOnRetry_Setter(t *testing.T) {
	client := NewClient("u", "p")
	if client.onRetry != nil {
		t.Error("onRetry should be nil by default")
	}

	called := false
	client.OnRetry(func(attempt int, backoff time.Duration) {
		called = true
	})

	if client.onRetry == nil {
		t.Fatal("OnRetry setter did not set callback")
	}

	// Invoke to verify it's the right function
	client.onRetry(1, time.Second)
	if !called {
		t.Error("callback was not invoked")
	}
}
