package turn

import (
	"strings"
	"testing"

	"decall_server/internal/config"
)

func TestBuildCredentials(t *testing.T) {
	cfg := config.Config{
		TurnSecret:        "test-secret",
		TurnHost:          "turn.example.com",
		TurnCredentialTTL: 3600,
		TurnTLS:           true,
	}

	resp, err := BuildCredentials(cfg, "deadbeef")
	if err != nil {
		t.Fatalf("BuildCredentials: %v", err)
	}
	if len(resp.IceServers) != 2 {
		t.Fatalf("expected 2 ice server entries, got %d", len(resp.IceServers))
	}

	turnEntry := resp.IceServers[1]
	urls, ok := turnEntry.URLs.([]string)
	if !ok {
		t.Fatalf("expected urls slice, got %T", turnEntry.URLs)
	}
	if len(urls) != 3 {
		t.Fatalf("expected 3 turn urls, got %d", len(urls))
	}
	if !strings.Contains(urls[2], "turns:turn.example.com:5349") {
		t.Fatalf("missing turns url: %v", urls)
	}
	if turnEntry.Username == "" || turnEntry.Credential == "" {
		t.Fatal("expected username and credential")
	}
	if !strings.Contains(turnEntry.Username, ":deadbeef") {
		t.Fatalf("unexpected username: %s", turnEntry.Username)
	}
}

func TestBuildCredentialsRequiresConfig(t *testing.T) {
	if _, err := BuildCredentials(config.Config{}, "id"); err == nil {
		t.Fatal("expected error without turn secret")
	}
}
