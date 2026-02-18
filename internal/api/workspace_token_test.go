package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateWorkspaceTokenSuccess(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should use Authorization: Bearer header.
		if r.Header.Get("Authorization") != "Bearer ws-token" {
			t.Errorf("expected Authorization header, got %q", r.Header.Get("Authorization"))
		}

		// Parse request to verify variables.
		var req GraphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}

		if req.Variables["id"] != "ws-123" {
			t.Errorf("expected workspace ID %q, got %q", "ws-123", req.Variables["id"])
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"workspace":{"id":"ws-123","name":"My Workspace"}}}`))
	}))
	defer srv.Close()

	client := NewWithEndpoint(srv.URL)
	info, err := client.ValidateWorkspaceToken(context.Background(), "ws-token", "ws-123")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.ID != "ws-123" {
		t.Fatalf("expected ID %q, got %q", "ws-123", info.ID)
	}

	if info.Name != "My Workspace" {
		t.Fatalf("expected Name %q, got %q", "My Workspace", info.Name)
	}
}

func TestValidateWorkspaceTokenGraphQLError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":null,"errors":[{"message":"Workspace not found"}]}`))
	}))
	defer srv.Close()

	client := NewWithEndpoint(srv.URL)
	_, err := client.ValidateWorkspaceToken(context.Background(), "ws-token", "bad-id")

	if err == nil {
		t.Fatal("expected error for invalid workspace")
	}

	var qe *QueryError
	if !errors.As(err, &qe) {
		t.Fatalf("expected QueryError, got %T: %v", err, err)
	}

	if qe.Message != "Workspace not found" {
		t.Fatalf("expected message %q, got %q", "Workspace not found", qe.Message)
	}
}

func TestValidateWorkspaceTokenHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	}))
	defer srv.Close()

	client := NewWithEndpoint(srv.URL)
	_, err := client.ValidateWorkspaceToken(context.Background(), "bad-token", "ws-123")

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
