package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yourorg/mycli/internal/auth"
)

// Client is a generic API client template
// Copy and modify this for your specific service integration
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient(service string, opts ...Option) (*Client, error) {
	// Get credentials with env var fallback
	apiKey, err := auth.GetCredential(service)
	if err != nil {
		return nil, err
	}

	c := &Client{
		baseURL: "https://api.example.com/v1", // Override per service
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// Option configures the client
type Option func(*Client)

// WithBaseURL sets the API base URL
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithTimeout sets the HTTP client timeout
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// Request performs an HTTP request with authentication
func (c *Client) Request(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "mycli/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// Get performs a GET request and decodes JSON response
func (c *Client) Get(ctx context.Context, path string, result any) error {
	resp, err := c.Request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// Example: List items from the API
// Modify this for your specific endpoint
type Item struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func (c *Client) ListItems(ctx context.Context, limit int) ([]Item, error) {
	path := fmt.Sprintf("/items?limit=%d", limit)

	var response struct {
		Items []Item `json:"items"`
	}

	if err := c.Get(ctx, path, &response); err != nil {
		return nil, err
	}

	return response.Items, nil
}

func (c *Client) GetItem(ctx context.Context, id string) (*Item, error) {
	path := "/items/" + id

	var item Item
	if err := c.Get(ctx, path, &item); err != nil {
		return nil, err
	}

	return &item, nil
}
