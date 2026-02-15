package secret

import "testing"

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
