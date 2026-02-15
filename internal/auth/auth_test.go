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
