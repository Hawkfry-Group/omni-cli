package cli

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupValidationNoteForBothAuthMode(t *testing.T) {
	note := setupValidationNote("both", map[string]validationSummary{
		"pat": {
			Query: capabilityCheck{Status: "fail"},
		},
		"org": {
			Query: capabilityCheck{Status: "pass"},
			Admin: capabilityCheck{Status: "pass"},
		},
	})

	if note == "" {
		t.Fatal("expected validation note for limited PAT in both mode")
	}
}

func TestSetupValidationNoteSkipsOtherCases(t *testing.T) {
	if note := setupValidationNote("pat", map[string]validationSummary{}); note != "" {
		t.Fatalf("expected no note outside both mode, got %q", note)
	}
	if note := setupValidationNote("both", map[string]validationSummary{
		"pat": {Query: capabilityCheck{Status: "pass"}},
		"org": {Query: capabilityCheck{Status: "pass"}, Admin: capabilityCheck{Status: "pass"}},
	}); note != "" {
		t.Fatalf("expected no note when PAT validation passed, got %q", note)
	}
}

func TestApplyLegacySetupFlags(t *testing.T) {
	authMode, defaultAuth, patToken, orgKey := applyLegacySetupFlags("", "", "", "", " legacy-org ", "org")
	if authMode != "org" || defaultAuth != "org" || orgKey != "legacy-org" || patToken != "" {
		t.Fatalf("unexpected org legacy mapping: authMode=%q defaultAuth=%q pat=%q org=%q", authMode, defaultAuth, patToken, orgKey)
	}

	authMode, defaultAuth, patToken, orgKey = applyLegacySetupFlags("", "", "", "", " legacy-pat ", "pat")
	if authMode != "pat" || defaultAuth != "pat" || patToken != "legacy-pat" || orgKey != "" {
		t.Fatalf("unexpected PAT legacy mapping: authMode=%q defaultAuth=%q pat=%q org=%q", authMode, defaultAuth, patToken, orgKey)
	}
}

func TestEmitSetupValidationMessagesForBothMode(t *testing.T) {
	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		emitSetupValidationMessages("both", map[string]validationSummary{
			"pat": {Query: capabilityCheck{Status: "fail", Message: "permission denied"}},
			"org": {
				Query: capabilityCheck{Status: "pass", Message: "ok"},
				Admin: capabilityCheck{Status: "pass", Message: "ok"},
			},
		})
		return 0
	})
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d", exit)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "setup succeeded") {
		t.Fatalf("expected user-facing success note, got %q", stderr)
	}
}

func TestValidateSetupCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/content":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"records":[]}`))
		case "/api/v1/query/run":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"bad request"}`))
		case "/api/scim/v2/Users":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"Resources":[]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	summary, failure := validateSetupCredential(server.URL, "org-token", "org", 5)
	if failure != nil {
		t.Fatalf("expected no validation failure, got %v", failure)
	}
	if summary.Base.Status != "pass" || summary.Query.Status != "pass" || summary.Admin.Status != "pass" {
		t.Fatalf("expected full validation pass, got %#v", summary)
	}
}

func TestRunSetupHelp(t *testing.T) {
	tmp := t.TempDir()
	rt := &runtime{
		ConfigPath: filepath.Join(tmp, "config.json"),
	}

	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runSetup(rt, []string{"-h"}, setupDefaults{})
	})
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d", exit)
	}
	if !strings.Contains(stdout, "omni setup:") {
		t.Fatalf("expected setup usage, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}

func TestPromptSecretInputUsesReaderWhenNotTerminal(t *testing.T) {
	oldStdin := os.Stdin
	oldStderr := os.Stderr

	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	defer func() {
		os.Stdin = oldStdin
		os.Stderr = oldStderr
		_ = stdinR.Close()
		_ = stdinW.Close()
		_ = stderrR.Close()
		_ = stderrW.Close()
	}()

	os.Stdin = stdinR
	os.Stderr = stderrW

	secret, err := promptSecretInput(bufio.NewReader(strings.NewReader("  top-secret  \n")), "Org API key")
	if err != nil {
		t.Fatalf("promptSecretInput returned error: %v", err)
	}
	if secret != "top-secret" {
		t.Fatalf("expected trimmed secret, got %q", secret)
	}
}
