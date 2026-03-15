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
				BaseURL:     "https://zeta.omniapp.co",
				DefaultAuth: "pat",
				PATToken:    "pat-token",
				PATStore:    "config",
			},
			"alpha": {
				BaseURL:     "https://alpha.omniapp.co",
				DefaultAuth: "org",
				OrgKeyStore: "keychain",
				OrgKeyRef:   "alpha:org",
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
			Name            string   `json:"name"`
			BaseURL         string   `json:"base_url"`
			AuthMode        string   `json:"auth_mode"`
			DefaultAuth     string   `json:"default_auth"`
			ConfiguredAuths []string `json:"configured_auths"`
			PATStore        string   `json:"pat_store"`
			OrgStore        string   `json:"org_store"`
			Current         bool     `json:"current"`
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
	if payload.Profiles[0].AuthMode != "org" || payload.Profiles[0].DefaultAuth != "org" {
		t.Fatalf("expected alpha org auth mode/default, got %#v", payload.Profiles[0])
	}
	if payload.Profiles[0].OrgStore != "keychain" {
		t.Fatalf("expected alpha org_store keychain, got %q", payload.Profiles[0].OrgStore)
	}
	if payload.Profiles[1].PATStore != "config" {
		t.Fatalf("expected zeta pat_store config, got %q", payload.Profiles[1].PATStore)
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
				BaseURL:     "https://zeta.omniapp.co",
				DefaultAuth: "pat",
				PATToken:    "pat-token",
				PATStore:    "config",
			},
			"alpha": {
				BaseURL:     "https://alpha.omniapp.co",
				DefaultAuth: "org",
				OrgKeyStore: "keychain",
				OrgKeyRef:   "alpha:org",
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
	if lines[0] != "auth_mode\tbase_url\tconfigured_auths\tcurrent\tdefault_auth\tname\torg_store\tpat_store" {
		t.Fatalf("unexpected plain header: %q", lines[0])
	}
	if lines[1] != "org\thttps://alpha.omniapp.co\t[\"org\"]\ttrue\torg\talpha\tkeychain\t" {
		t.Fatalf("unexpected first plain row: %q", lines[1])
	}
	if lines[2] != "pat\thttps://zeta.omniapp.co\t[\"pat\"]\tfalse\tpat\tzeta\t\tconfig" {
		t.Fatalf("unexpected second plain row: %q", lines[2])
	}
}
