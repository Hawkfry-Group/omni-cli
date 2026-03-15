package cli

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/omni-co/omni-cli/internal/config"
)

func TestRunSetupPATUsesBrowserLoginFlow(t *testing.T) {
	tmp := t.TempDir()
	rt := &runtime{
		ConfigPath: filepath.Join(tmp, "config.json"),
		Config:     &config.Config{Profiles: map[string]config.Profile{}},
		Keychain:   &mockSecretStore{available: false},
		PATLogin: func(baseURL string) (string, error) {
			return "pat-from-browser", nil
		},
	}

	exit := captureRuntimeOutput(t, func() int {
		return runSetup(rt, []string{
			"--profile", "prod",
			"--url", "https://acme.omniapp.co",
			"--auth-mode", "pat",
			"--token-store", "config",
			"--non-interactive",
			"--no-validate",
		}, setupDefaults{})
	})
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d", exit)
	}

	profile := rt.Config.Profiles["prod"]
	if profile.PATToken != "pat-from-browser" {
		t.Fatalf("expected PAT from browser login, got %#v", profile)
	}
}

func TestRunAuthAddBothUsesBrowserLoginFlowForPAT(t *testing.T) {
	tmp := t.TempDir()
	rt := &runtime{
		ConfigPath: filepath.Join(tmp, "config.json"),
		Config:     &config.Config{Profiles: map[string]config.Profile{}},
		Keychain:   &mockSecretStore{available: false},
		PATLogin: func(baseURL string) (string, error) {
			return "pat-from-browser", nil
		},
	}

	exit := captureRuntimeOutput(t, func() int {
		return runAuthAdd(rt, []string{
			"--name", "prod",
			"--url", "https://acme.omniapp.co",
			"--auth-mode", "both",
			"--org-key", "org-key-123",
			"--default-auth", "pat",
			"--token-store", "config",
		})
	})
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d", exit)
	}

	profile := rt.Config.Profiles["prod"]
	if profile.PATToken != "pat-from-browser" || profile.OrgKey != "org-key-123" {
		t.Fatalf("expected browser PAT and org key stored, got %#v", profile)
	}
}

func captureRuntimeOutput(t *testing.T, fn func() int) int {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}

	os.Stdout = outW
	os.Stderr = errW

	exit := fn()

	_ = outW.Close()
	_ = errW.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	_, _ = io.ReadAll(outR)
	_, _ = io.ReadAll(errR)

	return exit
}
