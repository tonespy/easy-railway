package cmd

import (
	"strings"
	"testing"
)

func TestAuthReturnsFlagParseError(t *testing.T) {
	err := Auth([]string{"--bad-flag"})
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
	if !strings.Contains(err.Error(), "flag provided but not defined: -bad-flag") {
		t.Fatalf("expected parse error for unknown flag, got %q", err.Error())
	}
}
