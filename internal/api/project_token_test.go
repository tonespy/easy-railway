package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateProjectTokenSuccess(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Project-Access-Token") != "proj-token" {
			t.Error("expected Project-Access-Token header")
		}

		if r.Header.Get("Authorization") != "" {
			t.Error("expected no Authorization header for project tokens")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"projectToken":{"id":"pt-1","name":"My Token","projectId":"proj-1","environmentId":"env-1"}}}`))
	}))
	defer srv.Close()

	client := NewWithEndpoint(srv.URL)
	info, err := client.ValidateProjectToken(context.Background(), "proj-token")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.ID != "pt-1" {
		t.Fatalf("expected ID %q, got %q", "pt-1", info.ID)
	}

	if info.Name != "My Token" {
		t.Fatalf("expected Name %q, got %q", "My Token", info.Name)
	}

	if info.ProjectID != "proj-1" {
		t.Fatalf("expected ProjectID %q, got %q", "proj-1", info.ProjectID)
	}

	if info.EnvironmentID != "env-1" {
		t.Fatalf("expected EnvironmentID %q, got %q", "env-1", info.EnvironmentID)
	}
}

func TestValidateProjectTokenGraphQLError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":null,"errors":[{"message":"Invalid project token"}]}`))
	}))
	defer srv.Close()

	client := NewWithEndpoint(srv.URL)
	_, err := client.ValidateProjectToken(context.Background(), "bad-token")

	if err == nil {
		t.Fatal("expected error for invalid project token")
	}

	var qe *QueryError
	if !errors.As(err, &qe) {
		t.Fatalf("expected QueryError, got %T: %v", err, err)
	}

	if qe.Message != "Invalid project token" {
		t.Fatalf("expected message %q, got %q", "Invalid project token", qe.Message)
	}
}

func TestValidateProjectTokenHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	}))
	defer srv.Close()

	client := NewWithEndpoint(srv.URL)
	_, err := client.ValidateProjectToken(context.Background(), "bad-token")

	if err == nil {
		t.Fatal("expected error for HTTP 401")
	}

	var he *HTTPError
	if !errors.As(err, &he) {
		t.Fatalf("expected HTTPError, got %T: %v", err, err)
	}

	if he.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", he.StatusCode)
	}
}
