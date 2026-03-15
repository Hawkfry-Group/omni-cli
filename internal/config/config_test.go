package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadMissingFileReturnsEmptyConfig(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.CurrentProfile != "" {
		t.Fatalf("expected empty current profile, got %q", cfg.CurrentProfile)
	}
	if cfg.Profiles == nil || len(cfg.Profiles) != 0 {
		t.Fatalf("expected empty profiles map, got %#v", cfg.Profiles)
	}
}

func TestSaveAndLoadNormalizesProfiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	input := &Config{
		CurrentProfile: "prod",
		Profiles: map[string]Profile{
			"prod": {
				BaseURL:     " https://acme.omniapp.co/ ",
				DefaultAuth: "PAT",
				PATToken:    " pat-token ",
				OrgKeyRef:   " org-ref ",
			},
		},
	}

	if err := Save(path, input); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	profile := cfg.Profiles["prod"]
	if profile.BaseURL != "https://acme.omniapp.co/" {
		t.Fatalf("expected trimmed base URL, got %q", profile.BaseURL)
	}
	if profile.DefaultAuth != "pat" {
		t.Fatalf("expected normalized default auth, got %q", profile.DefaultAuth)
	}
	if profile.PATStore != "config" {
		t.Fatalf("expected config PAT store, got %q", profile.PATStore)
	}
	if profile.PATToken != "pat-token" {
		t.Fatalf("expected trimmed PAT token, got %q", profile.PATToken)
	}
	if profile.OrgKeyStore != "keychain" {
		t.Fatalf("expected keychain org store, got %q", profile.OrgKeyStore)
	}
	if profile.OrgKeyRef != "org-ref" {
		t.Fatalf("expected trimmed org ref, got %q", profile.OrgKeyRef)
	}
}

func TestNormalizeProfileMigratesLegacyOrgToken(t *testing.T) {
	profile := normalizeProfile(Profile{
		BaseURL:   " https://acme.omniapp.co ",
		Token:     " legacy-org ",
		TokenType: " org ",
	})

	if profile.BaseURL != "https://acme.omniapp.co" {
		t.Fatalf("expected trimmed base URL, got %q", profile.BaseURL)
	}
	if profile.OrgKey != "legacy-org" {
		t.Fatalf("expected legacy org token to migrate, got %q", profile.OrgKey)
	}
	if profile.OrgKeyStore != "config" {
		t.Fatalf("expected legacy org token to default to config store, got %q", profile.OrgKeyStore)
	}
	if profile.DefaultAuth != "org" {
		t.Fatalf("expected default auth org, got %q", profile.DefaultAuth)
	}
	if profile.Token != "" || profile.TokenType != "" || profile.TokenStore != "" || profile.TokenRef != "" {
		t.Fatalf("expected legacy fields cleared, got %#v", profile)
	}
}

func TestNormalizeProfileAdjustsDefaultAuthToAvailableCredential(t *testing.T) {
	profile := normalizeProfile(Profile{
		DefaultAuth: "pat",
		OrgKeyRef:   "org-ref",
		OrgKeyStore: "keychain",
	})

	if profile.DefaultAuth != "org" {
		t.Fatalf("expected default auth to switch to org, got %q", profile.DefaultAuth)
	}
}

func TestProfileHelpers(t *testing.T) {
	profile := Profile{
		PATToken:  "pat-token",
		OrgKeyRef: "org-ref",
	}

	if !profile.HasPAT() {
		t.Fatal("expected PAT to be configured")
	}
	if !profile.HasOrg() {
		t.Fatal("expected org key to be configured")
	}
	if profile.AuthMode() != "both" {
		t.Fatalf("expected auth mode both, got %q", profile.AuthMode())
	}
	if got, want := profile.ConfiguredAuths(), []string{"pat", "org"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("expected configured auths %v, got %v", want, got)
	}
}

func TestDefaultStoreForCredential(t *testing.T) {
	if got := defaultStoreForCredential("", "ref"); got != "keychain" {
		t.Fatalf("expected keychain store for ref, got %q", got)
	}
	if got := defaultStoreForCredential("token", ""); got != "config" {
		t.Fatalf("expected config store for token, got %q", got)
	}
	if got := defaultStoreForCredential("", ""); got != "" {
		t.Fatalf("expected empty store when no credential is present, got %q", got)
	}
}

func TestDefaultPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath returned error: %v", err)
	}
	if path != filepath.Join(home, ".omni", "config.json") {
		t.Fatalf("unexpected default path %q", path)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) && err != nil {
		t.Fatalf("unexpected stat error: %v", err)
	}
}
