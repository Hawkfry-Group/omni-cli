package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseOptionalBool(t *testing.T) {
	cases := []struct {
		in      string
		nilWant bool
		valWant bool
		errWant bool
	}{
		{"", true, false, false},
		{"true", false, true, false},
		{"false", false, false, false},
		{"yes", false, true, false},
		{"no", false, false, false},
		{"maybe", true, false, true},
	}

	for _, tc := range cases {
		got, err := parseOptionalBool(tc.in)
		if tc.errWant {
			if err == nil {
				t.Fatalf("parseOptionalBool(%q): expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("parseOptionalBool(%q): unexpected error: %v", tc.in, err)
		}
		if tc.nilWant {
			if got != nil {
				t.Fatalf("parseOptionalBool(%q): expected nil, got %#v", tc.in, *got)
			}
			continue
		}
		if got == nil || *got != tc.valWant {
			t.Fatalf("parseOptionalBool(%q): expected %v, got %#v", tc.in, tc.valWant, got)
		}
	}
}

func TestParseUUIDArg(t *testing.T) {
	valid := "550e8400-e29b-41d4-a716-446655440000"
	if _, err := parseUUIDArg(valid, "schedule-id"); err != nil {
		t.Fatalf("expected valid UUID parse, got error: %v", err)
	}
	if _, err := parseUUIDArg("not-a-uuid", "schedule-id"); err == nil {
		t.Fatal("expected invalid UUID error")
	}
}

func TestParseOptionalUUIDArg(t *testing.T) {
	if got, err := parseOptionalUUIDArg("", "user-id"); err != nil || got != nil {
		t.Fatalf("expected nil UUID for empty input, got %v (err=%v)", got, err)
	}
	if got, err := parseOptionalUUIDArg("550e8400-e29b-41d4-a716-446655440000", "user-id"); err != nil || got == nil {
		t.Fatalf("expected parsed UUID, got %v (err=%v)", got, err)
	}
	if _, err := parseOptionalUUIDArg("bad", "user-id"); err == nil {
		t.Fatal("expected parse error for invalid UUID")
	}
}

func TestReadJSONFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "payload.json")
	if err := os.WriteFile(path, []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got, err := readJSONFile(path)
	if err != nil {
		t.Fatalf("readJSONFile failed: %v", err)
	}
	if string(got) != `{"ok":true}` {
		t.Fatalf("unexpected file contents: %s", string(got))
	}
}
