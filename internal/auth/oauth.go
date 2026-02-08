package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/99designs/keyring"
	"golang.org/x/oauth2"
)

// OAuthConfig holds OAuth2 configuration for a service
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	Endpoint     oauth2.Endpoint
	Scopes       []string
	RedirectPort int // Default: 8085
}

// OAuthToken represents stored OAuth tokens
type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
}

// BrowserFlow performs OAuth2 authorization via browser
func BrowserFlow(ctx context.Context, cfg OAuthConfig) (*oauth2.Token, error) {
	if cfg.RedirectPort == 0 {
		cfg.RedirectPort = 8085
	}

	redirectURL := fmt.Sprintf("http://localhost:%d/callback", cfg.RedirectPort)

	oauthCfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     cfg.Endpoint,
		Scopes:       cfg.Scopes,
		RedirectURL:  redirectURL,
	}

	// Generate state token for CSRF protection
	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	// Channel to receive the auth code
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	// Start local HTTP server
	srv := &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", cfg.RedirectPort),
		ReadHeaderTimeout: 10 * time.Second,
	}

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Verify state
		if r.URL.Query().Get("state") != state {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			errCh <- fmt.Errorf("state mismatch: possible CSRF attack")
			return
		}

		// Check for error
		if errParam := r.URL.Query().Get("error"); errParam != "" {
			errDesc := r.URL.Query().Get("error_description")
			http.Error(w, fmt.Sprintf("Authorization failed: %s", errDesc), http.StatusBadRequest)
			errCh <- fmt.Errorf("authorization denied: %s - %s", errParam, errDesc)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "No authorization code received", http.StatusBadRequest)
			errCh <- fmt.Errorf("no authorization code in callback")
			return
		}

		// Success response
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head><title>Authorization Successful</title></head>
<body>
<h1>✓ Authorization successful!</h1>
<p>You can close this window and return to the terminal.</p>
<script>window.close();</script>
</body>
</html>`)

		codeCh <- code
	})

	// Start server in background
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	// Build auth URL
	authURL := oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOffline)

	// Open browser
	fmt.Printf("Opening browser for authorization...\n")
	fmt.Printf("If browser doesn't open, visit:\n%s\n\n", authURL)

	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Could not open browser automatically: %v\n", err)
	}

	// Wait for callback or timeout
	var code string
	select {
	case code = <-codeCh:
		// Success
	case err := <-errCh:
		srv.Shutdown(ctx)
		return nil, err
	case <-time.After(5 * time.Minute):
		srv.Shutdown(ctx)
		return nil, fmt.Errorf("authorization timed out after 5 minutes")
	case <-ctx.Done():
		srv.Shutdown(ctx)
		return nil, ctx.Err()
	}

	// Shutdown server
	srv.Shutdown(ctx)

	// Exchange code for token
	token, err := oauthCfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	return token, nil
}

// StoreOAuthToken stores OAuth tokens in keyring
func StoreOAuthToken(service string, token *oauth2.Token) error {
	store, err := OpenKeyring()
	if err != nil {
		return err
	}

	data, err := json.Marshal(OAuthToken{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
	})
	if err != nil {
		return err
	}

	return store.ring.Set(keyring.Item{
		Key:  "oauth:" + service,
		Data: data,
	})
}

// GetOAuthToken retrieves and optionally refreshes OAuth tokens
func GetOAuthToken(ctx context.Context, service string, cfg *oauth2.Config) (*oauth2.Token, error) {
	store, err := OpenKeyring()
	if err != nil {
		return nil, err
	}

	item, err := store.ring.Get("oauth:" + service)
	if err != nil {
		return nil, fmt.Errorf("no OAuth token for %s: run 'mycli auth add %s --oauth'", service, service)
	}

	var stored OAuthToken
	if err := json.Unmarshal(item.Data, &stored); err != nil {
		return nil, fmt.Errorf("corrupted token data: %w", err)
	}

	token := &oauth2.Token{
		AccessToken:  stored.AccessToken,
		RefreshToken: stored.RefreshToken,
		TokenType:    stored.TokenType,
		Expiry:       stored.Expiry,
	}

	// Check if token needs refresh
	if token.Expiry.Before(time.Now()) && token.RefreshToken != "" && cfg != nil {
		tokenSource := cfg.TokenSource(ctx, token)
		newToken, err := tokenSource.Token()
		if err != nil {
			return nil, fmt.Errorf("token refresh failed: %w", err)
		}

		// Store refreshed token
		if err := StoreOAuthToken(service, newToken); err != nil {
			// Log but don't fail
			fmt.Printf("Warning: could not store refreshed token: %v\n", err)
		}

		return newToken, nil
	}

	return token, nil
}

// generateState creates a random state token
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// openBrowser opens URL in default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
