package cli

import "testing"

func TestBuildAPIURL(t *testing.T) {
	cases := []struct {
		base string
		path string
		want string
	}{
		{"https://acme.omniapp.co", "/documents", "https://acme.omniapp.co/api/v1/documents"},
		{"https://acme.omniapp.co/api", "documents", "https://acme.omniapp.co/api/v1/documents"},
		{"https://acme.omniapp.co/api/v1", "/api/v1/documents", "https://acme.omniapp.co/api/v1/documents"},
		{"https://acme.omniapp.co", "/v1/documents", "https://acme.omniapp.co/api/v1/documents"},
	}

	for _, tc := range cases {
		got := buildAPIURL(tc.base, tc.path)
		if got != tc.want {
			t.Fatalf("buildAPIURL(%q, %q) = %q, want %q", tc.base, tc.path, got, tc.want)
		}
	}
}

func TestHeaderArgsSet(t *testing.T) {
	var h headerArgs
	if err := h.Set("X-Test: hello"); err != nil {
		t.Fatalf("unexpected header parse error: %v", err)
	}
	values := h.Values()
	if values["X-Test"] != "hello" {
		t.Fatalf("expected X-Test header to equal hello, got %q", values["X-Test"])
	}
	if err := h.Set("broken"); err == nil {
		t.Fatal("expected invalid header error")
	}
}
