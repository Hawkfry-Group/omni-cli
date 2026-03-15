package cli

import (
	"path/filepath"
	"strings"
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

func TestRunAuthUseSetsCurrentProfile(t *testing.T) {
	tmp := t.TempDir()
	rt := &runtime{
		JSON:       true,
		ConfigPath: filepath.Join(tmp, "config.json"),
		Config: &config.Config{
			CurrentProfile: "alpha",
			Profiles: map[string]config.Profile{
				"alpha": {BaseURL: "https://alpha.omniapp.co"},
				"beta":  {BaseURL: "https://beta.omniapp.co"},
			},
		},
	}

	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runAuthUse(rt, []string{"beta"})
	})
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d (stderr=%q)", exit, stderr)
	}
	if rt.Config.CurrentProfile != "beta" {
		t.Fatalf("expected current profile beta, got %q", rt.Config.CurrentProfile)
	}
	if !strings.Contains(stdout, `"current_profile": "beta"`) {
		t.Fatalf("expected current profile in output, got %q", stdout)
	}
}

func TestRunAuthShowAndRedactConfiguredToken(t *testing.T) {
	rt := &runtime{
		JSON: true,
		Config: &config.Config{
			CurrentProfile: "alpha",
			Profiles: map[string]config.Profile{
				"alpha": {
					BaseURL:     "https://alpha.omniapp.co",
					DefaultAuth: "org",
					PATToken:    "1234567890abcdef",
					PATStore:    "config",
					OrgKeyStore: "keychain",
					OrgKeyRef:   "alpha:org",
				},
			},
		},
	}

	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runAuthShow(rt, nil)
	})
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stdout, `"pat_token": "1234...cdef"`) {
		t.Fatalf("expected PAT redaction in output, got %q", stdout)
	}
	if !strings.Contains(stdout, `"org_key": "stored-in-keychain"`) {
		t.Fatalf("expected keychain marker in output, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	if got := redactConfiguredToken("", "alpha:org"); got != "stored-in-keychain" {
		t.Fatalf("expected keychain marker, got %q", got)
	}
	if got := redactConfiguredToken("1234567890abcdef", ""); got != "1234...cdef" {
		t.Fatalf("expected redacted inline token, got %q", got)
	}
	if got := redactConfiguredToken("", ""); got != "" {
		t.Fatalf("expected empty redaction when no credential exists, got %q", got)
	}
}

func TestRunAuthUsageAndMissingProfile(t *testing.T) {
	rt := &runtime{
		JSON: true,
		Config: &config.Config{
			Profiles: map[string]config.Profile{},
		},
	}

	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runAuth(rt, nil)
	})
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d", exit)
	}
	if !strings.Contains(stdout, "omni auth commands:") {
		t.Fatalf("expected auth usage output, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	_, stderr, exit = captureRuntimeIO(t, func() int {
		return runAuthShow(rt, []string{"--profile", "missing"})
	})
	if exit != 1 {
		t.Fatalf("expected exit 1 for missing profile, got %d", exit)
	}
	if !strings.Contains(stderr, codeConfigMissing) {
		t.Fatalf("expected config missing error, got %q", stderr)
	}
}
