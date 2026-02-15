package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/omni-co/omni-cli/internal/config"
)

func TestAuthListJSONContractStable(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.json")
	t.Setenv("OMNI_CONFIG", configPath)

	cfg := &config.Config{
		CurrentProfile: "alpha",
		Profiles: map[string]config.Profile{
			"zeta": {
				BaseURL:   "https://zeta.omniapp.co",
				TokenType: "pat",
			},
			"alpha": {
				BaseURL:    "https://alpha.omniapp.co",
				TokenType:  "org",
				TokenStore: "keychain",
			},
		},
	}
	if err := config.Save(configPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	stdout, stderr, exit := captureExecute(t, []string{"--json", "auth", "list"})
	if exit != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%q)", exit, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var payload struct {
		Profiles []struct {
			Name       string `json:"name"`
			BaseURL    string `json:"base_url"`
			TokenType  string `json:"token_type"`
			TokenStore string `json:"token_store"`
			Current    bool   `json:"current"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("unmarshal auth list json: %v\nbody=%q", err, stdout)
	}
	if len(payload.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(payload.Profiles))
	}
	if payload.Profiles[0].Name != "alpha" || payload.Profiles[1].Name != "zeta" {
		t.Fatalf("expected sorted profile names [alpha zeta], got [%s %s]", payload.Profiles[0].Name, payload.Profiles[1].Name)
	}
	if !payload.Profiles[0].Current {
		t.Fatalf("expected alpha to be current profile, got %#v", payload.Profiles[0])
	}
	if payload.Profiles[0].TokenStore != "keychain" {
		t.Fatalf("expected alpha token_store keychain, got %q", payload.Profiles[0].TokenStore)
	}
	if payload.Profiles[1].TokenStore != "config" {
		t.Fatalf("expected default token_store config, got %q", payload.Profiles[1].TokenStore)
	}
}

func TestAuthListPlainContractStable(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.json")
	t.Setenv("OMNI_CONFIG", configPath)

	cfg := &config.Config{
		CurrentProfile: "alpha",
		Profiles: map[string]config.Profile{
			"zeta": {
				BaseURL:   "https://zeta.omniapp.co",
				TokenType: "pat",
			},
			"alpha": {
				BaseURL:    "https://alpha.omniapp.co",
				TokenType:  "org",
				TokenStore: "keychain",
			},
		},
	}
	if err := config.Save(configPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	stdout, stderr, exit := captureExecute(t, []string{"--plain", "auth", "list"})
	if exit != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%q)", exit, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 plain lines, got %d: %q", len(lines), stdout)
	}
	if lines[0] != "base_url\tcurrent\tname\ttoken_store\ttoken_type" {
		t.Fatalf("unexpected plain header: %q", lines[0])
	}
	if lines[1] != "https://alpha.omniapp.co\ttrue\talpha\tkeychain\torg" {
		t.Fatalf("unexpected first plain row: %q", lines[1])
	}
	if lines[2] != "https://zeta.omniapp.co\tfalse\tzeta\tconfig\tpat" {
		t.Fatalf("unexpected second plain row: %q", lines[2])
	}
}
