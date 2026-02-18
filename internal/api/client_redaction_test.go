package api

import "testing"

func TestRedactHeadersMasksSensitiveValues(t *testing.T) {
	in := map[string]string{
		"Content-Type":         "application/json",
		"Authorization":        "Bearer abc123",
		"Project-Access-Token": "proj-secret",
		"X-API-Key":            "api-secret",
	}

	out := redactHeaders(in)

	if out["Content-Type"] != "application/json" {
		t.Fatalf("expected Content-Type unchanged, got %q", out["Content-Type"])
	}

	if out["Authorization"] != "Bearer [REDACTED]" {
		t.Fatalf("expected redacted Authorization header, got %q", out["Authorization"])
	}

	if out["Project-Access-Token"] != "[REDACTED]" {
		t.Fatalf("expected redacted Project-Access-Token header, got %q", out["Project-Access-Token"])
	}

	if out["X-API-Key"] != "[REDACTED]" {
		t.Fatalf("expected redacted X-API-Key header, got %q", out["X-API-Key"])
	}

	if in["Authorization"] != "Bearer abc123" {
		t.Fatalf("expected input headers map to remain unchanged, got %q", in["Authorization"])
	}
}

func TestRedactHeaderValueAuthorizationWithoutScheme(t *testing.T) {
	got := redactHeaderValue("Authorization", "raw-token")
	if got != "[REDACTED]" {
		t.Fatalf("expected fully redacted value, got %q", got)
	}
}
