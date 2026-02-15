package cli

import (
	"path/filepath"
	"testing"

	"github.com/omni-co/omni-cli/internal/config"
)

type mockSecretStore struct {
	available  bool
	saveRef    string
	savedToken string
	deleteRefs []string
}

func (m *mockSecretStore) Name() string                   { return "keychain" }
func (m *mockSecretStore) Available() bool                { return m.available }
func (m *mockSecretStore) Get(ref string) (string, error) { return "", nil }
func (m *mockSecretStore) Save(profileName, token string) (string, error) {
	m.savedToken = token
	if m.saveRef == "" {
		m.saveRef = profileName
	}
	return m.saveRef, nil
}
func (m *mockSecretStore) Delete(ref string) error {
	m.deleteRefs = append(m.deleteRefs, ref)
	return nil
}

func TestSaveProfileWithTokenAutoUsesKeychain(t *testing.T) {
	tmp := t.TempDir()
	rt := &runtime{
		ConfigPath: filepath.Join(tmp, "config.json"),
		Config:     &config.Config{Profiles: map[string]config.Profile{}},
		Keychain:   &mockSecretStore{available: true, saveRef: "profile-ref"},
	}

	profile, warning, err := saveProfileWithToken(rt, "prod", "https://acme.omniapp.co", "pat", "super-token", "auto", true)
	if err != nil {
		t.Fatalf("saveProfileWithToken returned error: %v", err)
	}
	if warning != "" {
		t.Fatalf("expected no warning, got %q", warning)
	}
	if profile.TokenStore != "keychain" {
		t.Fatalf("expected keychain token store, got %q", profile.TokenStore)
	}
	if profile.TokenRef != "profile-ref" {
		t.Fatalf("expected token ref profile-ref, got %q", profile.TokenRef)
	}
	if profile.Token != "" {
		t.Fatalf("expected no plaintext token in config profile, got %q", profile.Token)
	}
}

func TestSaveProfileWithTokenAutoFallsBackToConfig(t *testing.T) {
	tmp := t.TempDir()
	rt := &runtime{
		ConfigPath: filepath.Join(tmp, "config.json"),
		Config:     &config.Config{Profiles: map[string]config.Profile{}},
		Keychain:   &mockSecretStore{available: false},
	}

	profile, warning, err := saveProfileWithToken(rt, "prod", "https://acme.omniapp.co", "pat", "super-token", "auto", true)
	if err != nil {
		t.Fatalf("saveProfileWithToken returned error: %v", err)
	}
	if warning == "" {
		t.Fatal("expected fallback warning, got empty string")
	}
	if profile.TokenStore != "config" {
		t.Fatalf("expected config token store, got %q", profile.TokenStore)
	}
	if profile.Token != "super-token" {
		t.Fatalf("expected plaintext token in config fallback, got %q", profile.Token)
	}
}

func TestSaveProfileWithTokenDeletesOldKeychainRefWhenSwitchingToConfig(t *testing.T) {
	store := &mockSecretStore{available: true}
	tmp := t.TempDir()
	rt := &runtime{
		ConfigPath: filepath.Join(tmp, "config.json"),
		Config: &config.Config{Profiles: map[string]config.Profile{
			"prod": {
				BaseURL:    "https://acme.omniapp.co",
				TokenType:  "pat",
				TokenStore: "keychain",
				TokenRef:   "old-ref",
			},
		}},
		Keychain: store,
	}

	_, _, err := saveProfileWithToken(rt, "prod", "https://acme.omniapp.co", "pat", "new-token", "config", true)
	if err != nil {
		t.Fatalf("saveProfileWithToken returned error: %v", err)
	}
	if len(store.deleteRefs) != 1 || store.deleteRefs[0] != "old-ref" {
		t.Fatalf("expected old keychain ref deletion, got %#v", store.deleteRefs)
	}
}
