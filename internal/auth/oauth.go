// Package auth provides authentication helpers for API key and OAuth flows.
package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// OAuth constants for Railway's OAuth2 endpoints.
const (
	OAuthClientID    = "rlwy_oaci_gk9b2mxdGYqx7QzwFlvyrzPx"
	OAuthCallbackURL = "http://127.0.0.1:3350/callback"
	OAuthAuthURL     = "https://backboard.railway.com/oauth/auth"
	OAuthTokenURL    = "https://backboard.railway.com/oauth/token" //nolint:gosec // Not a credential; endpoint URL.

	defaultCallbackAddr = "127.0.0.1:3350"
	pkceVerifierLength  = 32 // 32 random bytes → 43 base64url chars.
)

// OAuthConfig holds injectable dependencies for the OAuth flow.
type OAuthConfig struct {
	// OpenBrowser opens a URL in the user's default browser.
	OpenBrowser func(url string) error

	// HTTPClient is used for the token exchange POST.
	HTTPClient *http.Client

	// TokenURL overrides the token exchange endpoint (for testing).
	TokenURL string

	// AuthURL overrides the authorization endpoint (for testing).
	AuthURL string

	// CallbackAddr overrides the local callback server address (for testing).
	CallbackAddr string

	// Now returns the current time (injectable for testing expiry).
	Now func() time.Time
}

// DefaultOAuthConfig returns a config suitable for production use.
func DefaultOAuthConfig() *OAuthConfig {
	return &OAuthConfig{
		OpenBrowser:  openBrowser,
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
		TokenURL:     OAuthTokenURL,
		AuthURL:      OAuthAuthURL,
		CallbackAddr: defaultCallbackAddr,
		Now:          time.Now,
	}
}

// OAuthResult holds the token exchange response.
type OAuthResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	IDToken      string
	TokenType    string
}

// PKCEParams holds the PKCE verifier and challenge pair.
type PKCEParams struct {
	Verifier  string
	Challenge string
}

// GeneratePKCE creates a cryptographically random code_verifier and
// its SHA-256 code_challenge in base64url encoding.
func GeneratePKCE() (*PKCEParams, error) {
	raw, err := randBytes(pkceVerifierLength)
	if err != nil {
		return nil, fmt.Errorf("generate PKCE verifier: %w", err)
	}

	verifier := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return &PKCEParams{Verifier: verifier, Challenge: challenge}, nil
}

// GenerateState creates a random state parameter for CSRF protection.
func GenerateState() (string, error) {
	raw, err := randBytes(16)
	if err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// BuildAuthURL constructs the full authorization URL with PKCE and state.
func BuildAuthURL(cfg *OAuthConfig, pkce *PKCEParams, state, scopes string) string {
	params := url.Values{
		"client_id":             {OAuthClientID},
		"redirect_uri":          {OAuthCallbackURL},
		"response_type":         {"code"},
		"scope":                 {scopes},
		"state":                 {state},
		"code_challenge":        {pkce.Challenge},
		"code_challenge_method": {"S256"},
	}

	return cfg.AuthURL + "?" + params.Encode()
}

// callbackResult is sent from the HTTP handler to the main goroutine.
type callbackResult struct {
	Code  string
	State string
	Err   error
}

// ListenForCallback starts a local HTTP server, waits for the OAuth callback,
// and returns the authorization code. It shuts down after receiving one request
// or after the context is cancelled.
func ListenForCallback(ctx context.Context, cfg *OAuthConfig, expectedState string) (string, error) {
	resultCh := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		if errParam := q.Get("error"); errParam != "" {
			desc := q.Get("error_description")
			if desc == "" {
				desc = errParam
			}

			escDesc := html.EscapeString(desc)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, callbackErrorHTML, escDesc)
			resultCh <- callbackResult{Err: fmt.Errorf("oauth: %s", desc)}

			return
		}

		code := q.Get("code")
		state := q.Get("state")

		if code == "" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintf(w, callbackErrorHTML, html.EscapeString("missing authorization code in callback"))
			resultCh <- callbackResult{Err: fmt.Errorf("oauth: missing authorization code")}

			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, callbackSuccessHTML)
		resultCh <- callbackResult{Code: code, State: state}
	})

	addr := cfg.CallbackAddr
	if addr == "" {
		addr = defaultCallbackAddr
	}

	lc := net.ListenConfig{}

	listener, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return "", fmt.Errorf("start callback server: %w", err)
	}

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() { _ = srv.Serve(listener) }()

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_ = srv.Shutdown(shutdownCtx)
	}()

	select {
	case result := <-resultCh:
		if result.Err != nil {
			return "", result.Err
		}

		if result.State != expectedState {
			return "", fmt.Errorf("oauth: state mismatch (possible CSRF)")
		}

		return result.Code, nil
	case <-ctx.Done():
		return "", fmt.Errorf("oauth: timed out waiting for browser callback")
	}
}

// ExchangeCode exchanges the authorization code for tokens.
func ExchangeCode(cfg *OAuthConfig, code string, pkce *PKCEParams) (*OAuthResult, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {OAuthClientID},
		"code":          {code},
		"redirect_uri":  {OAuthCallbackURL},
		"code_verifier": {pkce.Verifier},
	}

	return postTokenRequest(cfg, data)
}

// RefreshAccessToken uses a refresh token to obtain a new access token.
func RefreshAccessToken(cfg *OAuthConfig, refreshToken string) (*OAuthResult, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {OAuthClientID},
		"refresh_token": {refreshToken},
	}

	return postTokenRequest(cfg, data)
}

// postTokenRequest sends a form POST to the token endpoint and parses the response.
func postTokenRequest(cfg *OAuthConfig, data url.Values) (*OAuthResult, error) {
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		cfg.TokenURL,
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("token request: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		IDToken      string `json:"id_token"`
		TokenType    string `json:"token_type"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("token request: decode: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("token request: %s: %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	return &OAuthResult{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		IDToken:      tokenResp.IDToken,
		TokenType:    tokenResp.TokenType,
	}, nil
}

// openBrowser opens the given URL in the default browser.
func openBrowser(rawURL string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(context.Background(), "open", rawURL)
	case "linux":
		cmd = exec.CommandContext(context.Background(), "xdg-open", rawURL)
	case "windows":
		cmd = exec.CommandContext(
			context.Background(),
			"rundll32",
			"url.dll,FileProtocolHandler",
			rawURL,
		)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}

	return nil
}

const callbackSuccessHTML = `<!DOCTYPE html>
<html><body style="font-family:sans-serif;text-align:center;padding:4em">
<h1>Login successful</h1>
<p>You can close this window and return to the terminal.</p>
</body></html>`

const callbackErrorHTML = `<!DOCTYPE html>
<html><body style="font-family:sans-serif;text-align:center;padding:4em">
<h1>Login failed</h1>
<p>%s</p>
<p>Please return to the terminal.</p>
</body></html>`
