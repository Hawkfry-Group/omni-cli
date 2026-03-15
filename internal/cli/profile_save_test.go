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

func TestSaveProfileWithCredentialsAutoUsesKeychain(t *testing.T) {
	tmp := t.TempDir()
	rt := &runtime{
		ConfigPath: filepath.Join(tmp, "config.json"),
		Config:     &config.Config{Profiles: map[string]config.Profile{}},
		Keychain:   &mockSecretStore{available: true},
	}

	profile, warning, err := saveProfileWithCredentials(rt, "prod", "https://acme.omniapp.co", "pat", "super-token", "", "auto", true)
	if err != nil {
		t.Fatalf("saveProfileWithCredentials returned error: %v", err)
	}
	if warning != "" {
		t.Fatalf("expected no warning, got %q", warning)
	}
	if profile.PATStore != "keychain" {
		t.Fatalf("expected keychain PAT store, got %q", profile.PATStore)
	}
	if profile.PATRef != "prod:pat" {
		t.Fatalf("expected PAT ref prod:pat, got %q", profile.PATRef)
	}
	if profile.PATToken != "" {
		t.Fatalf("expected no plaintext PAT in config profile, got %q", profile.PATToken)
	}
}

func TestSaveProfileWithCredentialsAutoFallsBackToConfig(t *testing.T) {
	tmp := t.TempDir()
	rt := &runtime{
		ConfigPath: filepath.Join(tmp, "config.json"),
		Config:     &config.Config{Profiles: map[string]config.Profile{}},
		Keychain:   &mockSecretStore{available: false},
	}

	profile, warning, err := saveProfileWithCredentials(rt, "prod", "https://acme.omniapp.co", "pat", "super-token", "", "auto", true)
	if err != nil {
		t.Fatalf("saveProfileWithCredentials returned error: %v", err)
	}
	if warning == "" {
		t.Fatal("expected fallback warning, got empty string")
	}
	if profile.PATStore != "config" {
		t.Fatalf("expected config PAT store, got %q", profile.PATStore)
	}
	if profile.PATToken != "super-token" {
		t.Fatalf("expected plaintext PAT in config fallback, got %q", profile.PATToken)
	}
}

func TestSaveProfileWithCredentialsDeletesOldKeychainRefWhenSwitchingToConfig(t *testing.T) {
	store := &mockSecretStore{available: true}
	tmp := t.TempDir()
	rt := &runtime{
		ConfigPath: filepath.Join(tmp, "config.json"),
		Config: &config.Config{Profiles: map[string]config.Profile{
			"prod": {
				BaseURL:  "https://acme.omniapp.co",
				PATStore: "keychain",
				PATRef:   "old-ref",
			},
		}},
		Keychain: store,
	}

	_, _, err := saveProfileWithCredentials(rt, "prod", "https://acme.omniapp.co", "pat", "new-token", "", "config", true)
	if err != nil {
		t.Fatalf("saveProfileWithCredentials returned error: %v", err)
	}
	if len(store.deleteRefs) != 1 || store.deleteRefs[0] != "old-ref" {
		t.Fatalf("expected old keychain ref deletion, got %#v", store.deleteRefs)
	}
}

func TestSaveProfileWithCredentialsStoresBothAuths(t *testing.T) {
	tmp := t.TempDir()
	rt := &runtime{
		ConfigPath: filepath.Join(tmp, "config.json"),
		Config:     &config.Config{Profiles: map[string]config.Profile{}},
		Keychain:   &mockSecretStore{available: false},
	}

	profile, _, err := saveProfileWithCredentials(rt, "prod", "https://acme.omniapp.co", "pat", "pat-token", "org-token", "config", true)
	if err != nil {
		t.Fatalf("saveProfileWithCredentials returned error: %v", err)
	}
	if profile.DefaultAuth != "pat" {
		t.Fatalf("expected default auth pat, got %q", profile.DefaultAuth)
	}
	if profile.PATToken != "pat-token" || profile.OrgKey != "org-token" {
		t.Fatalf("expected both credentials stored, got %#v", profile)
	}
}
