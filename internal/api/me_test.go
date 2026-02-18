package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMeSuccess(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer valid-token" {
			t.Error("expected Authorization header with Bearer token")
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected Content-Type application/json")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"me":{"id":"usr-1","name":"Test User","email":"test@example.com"}}}`))
	}))
	defer srv.Close()

	client := NewWithEndpoint(srv.URL)
	user, err := client.Me(context.Background(), "valid-token")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.ID != "usr-1" {
		t.Fatalf("expected ID %q, got %q", "usr-1", user.ID)
	}

	if user.Name != "Test User" {
		t.Fatalf("expected Name %q, got %q", "Test User", user.Name)
	}

	if user.Email != "test@example.com" {
		t.Fatalf("expected Email %q, got %q", "test@example.com", user.Email)
	}
}

func TestMeGraphQLError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":null,"errors":[{"message":"Not authenticated"}]}`))
	}))
	defer srv.Close()

	client := NewWithEndpoint(srv.URL)
	_, err := client.Me(context.Background(), "bad-token")

	if err == nil {
		t.Fatal("expected error for GraphQL error response")
	}

	var qe *QueryError
	if !errors.As(err, &qe) {
		t.Fatalf("expected QueryError, got %T: %v", err, err)
	}

	if qe.Message != "Not authenticated" {
		t.Fatalf("expected message %q, got %q", "Not authenticated", qe.Message)
	}
}

func TestMeHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer srv.Close()

	client := NewWithEndpoint(srv.URL)
	_, err := client.Me(context.Background(), "any-token")

	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}

	var ae *HTTPError
	if !errors.As(err, &ae) {
		t.Fatalf("expected HTTPError, got %T: %v", err, err)
	}

	if ae.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", ae.StatusCode)
	}
}

func TestMeNetworkError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Close immediately to simulate network failure.
	}))
	srv.Close()

	client := NewWithEndpoint(srv.URL)
	_, err := client.Me(context.Background(), "any-token")

	if err == nil {
		t.Fatal("expected error for closed server")
	}
}
