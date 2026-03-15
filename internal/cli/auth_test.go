package cli

import (
	"path/filepath"
	"testing"

	"github.com/omni-co/omni-cli/internal/config"
)

type testKeychain struct {
	deleted []string
}

func (t *testKeychain) Name() string                                   { return "keychain" }
func (t *testKeychain) Available() bool                                { return true }
func (t *testKeychain) Save(profileName, token string) (string, error) { return profileName, nil }
func (t *testKeychain) Get(ref string) (string, error)                 { return "", nil }
func (t *testKeychain) Delete(ref string) error {
	t.deleted = append(t.deleted, ref)
	return nil
}

func TestRunAuthRemoveDeletesProfileAndKeychainRef(t *testing.T) {
	tmp := t.TempDir()
	k := &testKeychain{}
	rt := &runtime{
		ConfigPath: filepath.Join(tmp, "config.json"),
		Config: &config.Config{
			CurrentProfile: "p1",
			Profiles: map[string]config.Profile{
				"p1": {
					BaseURL:  "https://a.omniapp.co",
					PATStore: "keychain",
					PATRef:   "ref-p1",
				},
				"p2": {
					BaseURL:  "https://b.omniapp.co",
					PATStore: "config",
					PATToken: "abc",
				},
			},
		},
		Keychain: k,
	}

	if exit := runAuthRemove(rt, []string{"p1"}); exit != 0 {
		t.Fatalf("expected exit 0, got %d", exit)
	}
	if _, ok := rt.Config.Profiles["p1"]; ok {
		t.Fatalf("profile p1 should be removed")
	}
	if len(k.deleted) != 1 || k.deleted[0] != "ref-p1" {
		t.Fatalf("expected keychain ref deletion, got %#v", k.deleted)
	}
	if rt.Config.CurrentProfile == "p1" {
		t.Fatalf("current profile should move away from removed profile")
	}
}
