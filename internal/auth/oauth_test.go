package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestGeneratePKCE(t *testing.T) {
	t.Parallel()

	pkce, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE: %v", err)
	}

	// 32 bytes → 43 base64url chars (no padding).
	if len(pkce.Verifier) != 43 {
		t.Fatalf("expected verifier length 43, got %d", len(pkce.Verifier))
	}

	if len(pkce.Challenge) != 43 {
		t.Fatalf("expected challenge length 43, got %d", len(pkce.Challenge))
	}

	if pkce.Verifier == pkce.Challenge {
		t.Fatal("verifier and challenge should differ")
	}
}

func TestGeneratePKCEUniqueness(t *testing.T) {
	t.Parallel()

	p1, _ := GeneratePKCE()
	p2, _ := GeneratePKCE()

	if p1.Verifier == p2.Verifier {
		t.Fatal("two PKCE verifiers should not be identical")
	}
}

func TestGenerateState(t *testing.T) {
	t.Parallel()

	s1, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState: %v", err)
	}

	s2, _ := GenerateState()

	if s1 == s2 {
		t.Fatal("two states should not be identical")
	}

	if len(s1) == 0 {
		t.Fatal("state should not be empty")
	}
}

func TestBuildAuthURL(t *testing.T) {
	t.Parallel()

	cfg := &OAuthConfig{AuthURL: "https://example.com/auth"}
	pkce := &PKCEParams{Verifier: "test-verifier", Challenge: "test-challenge"}

	u := BuildAuthURL(cfg, pkce, "test-state", "openid email profile")

	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}

	q := parsed.Query()

	if q.Get("client_id") != OAuthClientID {
		t.Fatalf("expected client_id %q, got %q", OAuthClientID, q.Get("client_id"))
	}

	if q.Get("redirect_uri") != OAuthCallbackURL {
		t.Fatalf("expected redirect_uri %q, got %q", OAuthCallbackURL, q.Get("redirect_uri"))
	}

	if q.Get("response_type") != "code" {
		t.Fatalf("expected response_type 'code', got %q", q.Get("response_type"))
	}

	if q.Get("code_challenge") != "test-challenge" {
		t.Fatalf("expected code_challenge 'test-challenge', got %q", q.Get("code_challenge"))
	}

	if q.Get("code_challenge_method") != "S256" {
		t.Fatalf("expected code_challenge_method 'S256', got %q", q.Get("code_challenge_method"))
	}

	if q.Get("state") != "test-state" {
		t.Fatalf("expected state 'test-state', got %q", q.Get("state"))
	}

	if q.Get("scope") != "openid email profile" {
		t.Fatalf("expected scope 'openid email profile', got %q", q.Get("scope"))
	}
}

func TestBuildAuthURLWithOfflineAccess(t *testing.T) {
	t.Parallel()

	cfg := &OAuthConfig{AuthURL: "https://example.com/auth"}
	pkce := &PKCEParams{Verifier: "v", Challenge: "c"}

	u := BuildAuthURL(cfg, pkce, "s", "openid email profile offline_access")

	parsed, _ := url.Parse(u)

	if parsed.Query().Get("scope") != "openid email profile offline_access" {
		t.Fatalf("expected offline_access in scope, got %q", parsed.Query().Get("scope"))
	}
}

func TestExchangeCode(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}

		if r.FormValue("grant_type") != "authorization_code" {
			t.Errorf("expected grant_type=authorization_code, got %q", r.FormValue("grant_type"))
		}

		if r.FormValue("code") != "test-code" {
			t.Errorf("expected code=test-code, got %q", r.FormValue("code"))
		}

		if r.FormValue("code_verifier") != "test-verifier" {
			t.Errorf("expected code_verifier=test-verifier, got %q", r.FormValue("code_verifier"))
		}

		if r.FormValue("client_id") != OAuthClientID {
			t.Errorf("expected client_id=%s, got %q", OAuthClientID, r.FormValue("client_id"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "test-access",
			"refresh_token": "test-refresh",
			"expires_in":    3600,
			"id_token":      "test-id-token",
			"token_type":    "Bearer",
		})
	}))
	defer srv.Close()

	cfg := &OAuthConfig{
		HTTPClient: srv.Client(),
		TokenURL:   srv.URL,
	}
	pkce := &PKCEParams{Verifier: "test-verifier", Challenge: "test-challenge"}

	result, err := ExchangeCode(cfg, "test-code", pkce)
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}

	if result.AccessToken != "test-access" {
		t.Fatalf("expected AccessToken %q, got %q", "test-access", result.AccessToken)
	}

	if result.RefreshToken != "test-refresh" {
		t.Fatalf("expected RefreshToken %q, got %q", "test-refresh", result.RefreshToken)
	}

	if result.ExpiresIn != 3600 {
		t.Fatalf("expected ExpiresIn 3600, got %d", result.ExpiresIn)
	}

	if result.IDToken != "test-id-token" {
		t.Fatalf("expected IDToken %q, got %q", "test-id-token", result.IDToken)
	}

	if result.TokenType != "Bearer" {
		t.Fatalf("expected TokenType %q, got %q", "Bearer", result.TokenType)
	}
}

func TestExchangeCodeError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":             "invalid_grant",
			"error_description": "code expired",
		})
	}))
	defer srv.Close()

	cfg := &OAuthConfig{HTTPClient: srv.Client(), TokenURL: srv.URL}

	_, err := ExchangeCode(cfg, "bad-code", &PKCEParams{Verifier: "v", Challenge: "c"})
	if err == nil {
		t.Fatal("expected error for invalid grant")
	}
}

func TestExchangeCodeHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	}))
	defer srv.Close()

	cfg := &OAuthConfig{HTTPClient: srv.Client(), TokenURL: srv.URL}

	_, err := ExchangeCode(cfg, "code", &PKCEParams{Verifier: "v", Challenge: "c"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestRefreshAccessToken(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}

		if r.FormValue("grant_type") != "refresh_token" {
			t.Errorf("expected grant_type=refresh_token, got %q", r.FormValue("grant_type"))
		}

		if r.FormValue("refresh_token") != "old-refresh" {
			t.Errorf("expected refresh_token=old-refresh, got %q", r.FormValue("refresh_token"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "new-access",
			"refresh_token": "new-refresh",
			"expires_in":    3600,
			"token_type":    "Bearer",
		})
	}))
	defer srv.Close()

	cfg := &OAuthConfig{HTTPClient: srv.Client(), TokenURL: srv.URL}

	result, err := RefreshAccessToken(cfg, "old-refresh")
	if err != nil {
		t.Fatalf("RefreshAccessToken: %v", err)
	}

	if result.AccessToken != "new-access" {
		t.Fatalf("expected AccessToken %q, got %q", "new-access", result.AccessToken)
	}

	if result.RefreshToken != "new-refresh" {
		t.Fatalf("expected RefreshToken %q, got %q", "new-refresh", result.RefreshToken)
	}
}

func TestRefreshAccessTokenError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":             "invalid_grant",
			"error_description": "refresh token expired",
		})
	}))
	defer srv.Close()

	cfg := &OAuthConfig{HTTPClient: srv.Client(), TokenURL: srv.URL}

	_, err := RefreshAccessToken(cfg, "expired-refresh")
	if err == nil {
		t.Fatal("expected error for expired refresh token")
	}
}

func TestListenForCallback(t *testing.T) {
	t.Parallel()

	cfg := &OAuthConfig{CallbackAddr: "127.0.0.1:0"} // Random port.

	// We need the actual port, so start separately.
	resultCh := make(chan struct {
		code string
		err  error
	}, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Run ListenForCallback in a goroutine — it blocks until callback arrives.
	// We need to find the port it's listening on. Use a known port instead.
	cfg.CallbackAddr = "127.0.0.1:0"

	go func() {
		code, err := ListenForCallback(ctx, cfg, "test-state")
		resultCh <- struct {
			code string
			err  error
		}{code, err}
	}()

	// Give server a moment to start.
	time.Sleep(50 * time.Millisecond)

	// Since we used port 0, we don't know the port. Use a fixed port for this test.
	// Let's re-approach: use a known free port.
	cancel() // Cancel the random-port attempt.
	<-resultCh

	// Use a specific port for a deterministic test.
	cfg2 := &OAuthConfig{CallbackAddr: "127.0.0.1:13350"}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	resultCh2 := make(chan struct {
		code string
		err  error
	}, 1)

	go func() {
		code, err := ListenForCallback(ctx2, cfg2, "test-state")
		resultCh2 <- struct {
			code string
			err  error
		}{code, err}
	}()

	time.Sleep(100 * time.Millisecond)

	// Simulate the browser callback.
	resp, err := http.Get("http://127.0.0.1:13350/callback?code=auth-code&state=test-state")
	if err != nil {
		t.Fatalf("GET callback: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	result := <-resultCh2
	if result.err != nil {
		t.Fatalf("ListenForCallback: %v", result.err)
	}

	if result.code != "auth-code" {
		t.Fatalf("expected code %q, got %q", "auth-code", result.code)
	}
}

func TestListenForCallbackStateMismatch(t *testing.T) {
	t.Parallel()

	cfg := &OAuthConfig{CallbackAddr: "127.0.0.1:13351"}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resultCh := make(chan error, 1)

	go func() {
		_, err := ListenForCallback(ctx, cfg, "expected-state")
		resultCh <- err
	}()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://127.0.0.1:13351/callback?code=code&state=wrong-state")
	if err != nil {
		t.Fatalf("GET callback: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := <-resultCh; err == nil {
		t.Fatal("expected state mismatch error")
	}
}

func TestListenForCallbackOAuthError(t *testing.T) {
	t.Parallel()

	cfg := &OAuthConfig{CallbackAddr: "127.0.0.1:13352"}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resultCh := make(chan error, 1)

	go func() {
		_, err := ListenForCallback(ctx, cfg, "state")
		resultCh <- err
	}()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://127.0.0.1:13352/callback?error=access_denied&error_description=user+denied")
	if err != nil {
		t.Fatalf("GET callback: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := <-resultCh; err == nil {
		t.Fatal("expected access_denied error")
	}
}

func TestListenForCallbackTimeout(t *testing.T) {
	t.Parallel()

	cfg := &OAuthConfig{CallbackAddr: "127.0.0.1:13353"}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := ListenForCallback(ctx, cfg, "state")
	if err == nil {
		t.Fatal("expected timeout error")
	}

	_ = fmt.Sprintf("%v", err) // Ensure error is printable.
}
