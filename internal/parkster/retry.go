package parkster

import (
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	defaultMaxRetries  = 3
	defaultBaseBackoff = 500 * time.Millisecond
)

// doWithRetry executes an HTTP request with retry on transient errors.
// Retries on 5xx status codes, connection errors, and timeouts.
// Does NOT retry on 4xx or successful responses.
// When retries are exhausted on 5xx, returns the last response so callers
// can handle errors normally (e.g. parse error bodies).
// When retries are exhausted on network errors, returns the last error.
func (c *Client) doWithRetry(req *http.Request) (*http.Response, error) {
	maxRetries := defaultMaxRetries
	baseBackoff := c.retryBaseBackoff
	if baseBackoff == 0 {
		baseBackoff = defaultBaseBackoff
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := baseBackoff * time.Duration(1<<(attempt-1))
			if c.onRetry != nil {
				c.onRetry(attempt, backoff)
			}
			time.Sleep(backoff)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			if isTransient(err) {
				lastErr = err
				continue
			}
			return nil, err
		}

		if resp.StatusCode >= 500 {
			// On last attempt, return the response so caller can handle the error
			if attempt == maxRetries {
				return resp, nil
			}
			_ = resp.Body.Close()
			continue
		}

		return resp, nil
	}

	return nil, lastErr
}

// isTransient checks whether an error is likely transient and worth retrying.
func isTransient(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	if errors.Is(err, io.EOF) {
		return true
	}
	if strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "connection reset") {
		return true
	}
	return false
}
