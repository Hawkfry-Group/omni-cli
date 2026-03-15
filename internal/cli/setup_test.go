package cli

import (
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
