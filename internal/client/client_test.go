package client

import "testing"

func TestNormalizeBaseURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "trim whitespace", in: " https://acme.omniapp.co/ ", want: "https://acme.omniapp.co"},
		{name: "strip api", in: "https://acme.omniapp.co/api", want: "https://acme.omniapp.co"},
		{name: "strip api v1", in: "https://acme.omniapp.co/api/v1/", want: "https://acme.omniapp.co"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeBaseURL(tc.in); got != tc.want {
				t.Fatalf("normalizeBaseURL(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseBody(t *testing.T) {
	if got := ParseBody(nil).(map[string]any); len(got) != 0 {
		t.Fatalf("expected empty map for empty body, got %#v", got)
	}

	parsed := ParseBody([]byte(`{"ok":true}`)).(map[string]any)
	if parsed["ok"] != true {
		t.Fatalf("expected parsed JSON body, got %#v", parsed)
	}

	raw := ParseBody([]byte(`not-json`)).(map[string]any)
	if raw["raw"] != "not-json" {
		t.Fatalf("expected raw body wrapper, got %#v", raw)
	}
}
