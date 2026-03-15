package secret

import (
	"strings"
	"testing"
)

type fakeStore struct{ available bool }

func (f fakeStore) Name() string                                   { return "keychain" }
func (f fakeStore) Available() bool                                { return f.available }
func (f fakeStore) Save(profileName, token string) (string, error) { return "", nil }
func (f fakeStore) Get(ref string) (string, error)                 { return "", nil }
func (f fakeStore) Delete(ref string) error                        { return nil }

func TestPickAutoPrefersKeychainWhenAvailable(t *testing.T) {
	picked, err := Pick("auto", fakeStore{available: true})
	if err != nil {
		t.Fatalf("Pick returned error: %v", err)
	}
	if picked.Name() != "keychain" {
		t.Fatalf("expected keychain store, got %q", picked.Name())
	}
}

func TestPickAutoFallsBackToConfigWhenUnavailable(t *testing.T) {
	picked, err := Pick("auto", fakeStore{available: false})
	if err != nil {
		t.Fatalf("Pick returned error: %v", err)
	}
	if picked.Name() != "config" {
		t.Fatalf("expected config store, got %q", picked.Name())
	}
}

func TestPickExplicitKeychainUnavailableErrors(t *testing.T) {
	_, err := Pick("keychain", fakeStore{available: false})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestConfigStoreOperations(t *testing.T) {
	store := NewConfigStore()
	if !store.Available() {
		t.Fatal("expected config store to be available")
	}
	if store.Name() != "config" {
		t.Fatalf("expected config store name, got %q", store.Name())
	}
	if _, err := store.Save("default", ""); err == nil {
		t.Fatal("expected empty token save to fail")
	}
	ref, err := store.Save("default", "token")
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if ref != "" {
		t.Fatalf("expected config store save to return empty ref, got %q", ref)
	}
	if err := store.Delete("ignored"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := store.Get("ignored"); err == nil || !strings.Contains(err.Error(), "does not support token references") {
		t.Fatalf("expected unsupported get error, got %v", err)
	}
}

func TestPickConfigAndInvalidPreference(t *testing.T) {
	picked, err := Pick("config", fakeStore{available: true})
	if err != nil {
		t.Fatalf("Pick returned error: %v", err)
	}
	if picked.Name() != "config" {
		t.Fatalf("expected config store, got %q", picked.Name())
	}

	_, err = Pick("bogus", fakeStore{available: true})
	if err == nil || !strings.Contains(err.Error(), "invalid token store") {
		t.Fatalf("expected invalid token store error, got %v", err)
	}
}
