package auth

import (
	"errors"
	"strings"
	"testing"

	"github.com/omni-co/omni-cli/internal/config"
)

type mockStore struct {
	available bool
	tokens    map[string]string
	getCalls  int
}

func (m *mockStore) Name() string                                   { return "mock" }
func (m *mockStore) Available() bool                                { return m.available }
func (m *mockStore) Save(profileName, token string) (string, error) { return "", nil }
func (m *mockStore) Delete(ref string) error                        { return nil }
func (m *mockStore) Get(ref string) (string, error) {
	m.getCalls++
	v, ok := m.tokens[ref]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func TestResolveUsesConfigToken(t *testing.T) {
	t.Setenv("OMNI_PROFILE", "")
	t.Setenv("OMNI_URL", "")
	t.Setenv("OMNI_TOKEN", "")
	t.Setenv("OMNI_TOKEN_TYPE", "")

	cfg := &config.Config{
		CurrentProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {
				BaseURL:   "https://acme.omniapp.co",
				Token:     "cfg-token",
				TokenType: "pat",
			},
		},
	}

	resolved, err := Resolve(cfg, Options{})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if resolved.Profile.Token != "cfg-token" {
		t.Fatalf("expected cfg token, got %q", resolved.Profile.Token)
	}
}

func TestResolveLoadsTokenFromKeychainRef(t *testing.T) {
	t.Setenv("OMNI_PROFILE", "")
	t.Setenv("OMNI_URL", "")
	t.Setenv("OMNI_TOKEN", "")
	t.Setenv("OMNI_TOKEN_TYPE", "")

	cfg := &config.Config{
		CurrentProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {
				BaseURL:    "https://acme.omniapp.co",
				TokenType:  "pat",
				TokenStore: "keychain",
				TokenRef:   "default",
			},
		},
	}

	store := &mockStore{available: true, tokens: map[string]string{"default": "keychain-token"}}
	resolved, err := Resolve(cfg, Options{TokenStore: store})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if resolved.Profile.Token != "keychain-token" {
		t.Fatalf("expected keychain token, got %q", resolved.Profile.Token)
	}
	if store.getCalls != 1 {
		t.Fatalf("expected one keychain read, got %d", store.getCalls)
	}
}

func TestResolveKeychainUnavailableFails(t *testing.T) {
	cfg := &config.Config{
		CurrentProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {
				BaseURL:    "https://acme.omniapp.co",
				TokenType:  "pat",
				TokenStore: "keychain",
				TokenRef:   "default",
			},
		},
	}

	_, err := Resolve(cfg, Options{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "keychain") {
		t.Fatalf("expected keychain error, got %q", err.Error())
	}
}

func TestResolveTokenOverrideSkipsKeychainRead(t *testing.T) {
	cfg := &config.Config{
		CurrentProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {
				BaseURL:    "https://acme.omniapp.co",
				TokenType:  "pat",
				TokenStore: "keychain",
				TokenRef:   "default",
			},
		},
	}

	store := &mockStore{available: true, tokens: map[string]string{"default": "keychain-token"}}
	resolved, err := Resolve(cfg, Options{TokenStore: store, TokenFlag: "flag-token"})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if resolved.Profile.Token != "flag-token" {
		t.Fatalf("expected override token, got %q", resolved.Profile.Token)
	}
	if store.getCalls != 0 {
		t.Fatalf("expected no keychain reads when token override set, got %d", store.getCalls)
	}
}

func TestResolveUsesOrgEnvWhenRequired(t *testing.T) {
	t.Setenv("OMNI_ORG_KEY", "org-from-env")
	t.Setenv("OMNI_PAT", "")
	t.Setenv("OMNI_TOKEN", "")
	t.Setenv("OMNI_TOKEN_TYPE", "")
	t.Setenv("OMNI_URL", "")
	t.Setenv("OMNI_PROFILE", "")

	cfg := &config.Config{
		CurrentProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {BaseURL: "https://acme.omniapp.co"},
		},
	}

	resolved, err := Resolve(cfg, Options{RequireAuth: "org"})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if resolved.Profile.TokenType != "org" {
		t.Fatalf("expected org token type, got %q", resolved.Profile.TokenType)
	}
	if resolved.Profile.Token != "org-from-env" {
		t.Fatalf("expected org token from env, got %q", resolved.Profile.Token)
	}
}

func TestResolveMissingURLFails(t *testing.T) {
	t.Setenv("OMNI_PAT", "pat-from-env")
	t.Setenv("OMNI_TOKEN", "")
	t.Setenv("OMNI_URL", "")

	_, err := Resolve(&config.Config{Profiles: map[string]config.Profile{}}, Options{})
	if err == nil || !strings.Contains(err.Error(), "missing Omni URL") {
		t.Fatalf("expected missing URL error, got %v", err)
	}
}

func TestResolveMissingRequiredCredentialFails(t *testing.T) {
	cfg := &config.Config{
		CurrentProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {BaseURL: "https://acme.omniapp.co"},
		},
	}

	_, err := Resolve(cfg, Options{RequireAuth: "org"})
	if err == nil || !strings.Contains(err.Error(), "missing org API key") {
		t.Fatalf("expected missing org key error, got %v", err)
	}

	_, err = Resolve(cfg, Options{RequireAuth: "pat"})
	if err == nil || !strings.Contains(err.Error(), "missing PAT") {
		t.Fatalf("expected missing PAT error, got %v", err)
	}
}

func TestSaveProfileNormalizesAndSetsCurrent(t *testing.T) {
	cfg := &config.Config{}

	SaveProfile(cfg, "prod", config.Profile{
		BaseURL:     " https://acme.omniapp.co ",
		DefaultAuth: "pat",
		OrgKey:      " org-key ",
	}, true)

	profile := cfg.Profiles["prod"]
	if profile.BaseURL != "https://acme.omniapp.co" {
		t.Fatalf("expected trimmed base URL, got %q", profile.BaseURL)
	}
	if profile.DefaultAuth != "org" {
		t.Fatalf("expected default auth to switch to org, got %q", profile.DefaultAuth)
	}
	if profile.OrgKey != "org-key" {
		t.Fatalf("expected trimmed org key, got %q", profile.OrgKey)
	}
	if cfg.CurrentProfile != "prod" {
		t.Fatalf("expected current profile prod, got %q", cfg.CurrentProfile)
	}
}

func TestUseProfileAndRedactToken(t *testing.T) {
	cfg := &config.Config{
		Profiles: map[string]config.Profile{
			"prod": {BaseURL: "https://acme.omniapp.co"},
		},
	}

	if err := UseProfile(cfg, "prod"); err != nil {
		t.Fatalf("UseProfile returned error: %v", err)
	}
	if cfg.CurrentProfile != "prod" {
		t.Fatalf("expected current profile prod, got %q", cfg.CurrentProfile)
	}
	if err := UseProfile(cfg, "missing"); err == nil || !strings.Contains(err.Error(), `profile "missing" not found`) {
		t.Fatalf("expected missing profile error, got %v", err)
	}

	if got := RedactToken("short"); got != "****" {
		t.Fatalf("expected short token redaction, got %q", got)
	}
	if got := RedactToken("1234567890abcdef"); got != "1234...cdef" {
		t.Fatalf("expected long token redaction, got %q", got)
	}
}
