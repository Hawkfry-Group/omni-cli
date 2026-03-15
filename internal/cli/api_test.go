package cli

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/omni-co/omni-cli/internal/auth"
	"github.com/omni-co/omni-cli/internal/config"
)

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

func TestRunAPICallSuccess(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotAuth string
	var gotHeader string
	var gotBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotHeader = r.Header.Get("X-Test")
		bodyBytes := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(bodyBytes)
		gotBody = string(bodyBytes)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	rt := &runtime{
		JSON: true,
		Resolved: &auth.Resolved{
			Profile: config.Profile{
				BaseURL: server.URL,
				Token:   "pat-token",
			},
		},
	}

	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runAPICall(rt, []string{
			"--method", "POST",
			"--path", "documents",
			"--body", `{"name":"Quarterly"}`,
			"--header", "X-Test: hello",
		})
	})

	if exit != 0 {
		t.Fatalf("expected exit 0, got %d (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stdout, `"ok": true`) {
		t.Fatalf("expected success payload, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST method, got %q", gotMethod)
	}
	if gotPath != "/api/v1/documents" {
		t.Fatalf("expected /api/v1/documents path, got %q", gotPath)
	}
	if gotAuth != "Bearer pat-token" {
		t.Fatalf("expected bearer auth header, got %q", gotAuth)
	}
	if gotHeader != "hello" {
		t.Fatalf("expected X-Test header, got %q", gotHeader)
	}
	if gotBody != `{"name":"Quarterly"}` {
		t.Fatalf("expected request body to be forwarded, got %q", gotBody)
	}
}

func TestRunAPICallReadsBodyFile(t *testing.T) {
	tmp := t.TempDir()
	bodyPath := filepath.Join(tmp, "body.json")
	if err := os.WriteFile(bodyPath, []byte(`{"from":"file"}`), 0o600); err != nil {
		t.Fatalf("write body file: %v", err)
	}

	var gotContentType string
	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		bodyBytes, _ := io.ReadAll(r.Body)
		gotBody = string(bodyBytes)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"created":true}`))
	}))
	defer server.Close()

	rt := &runtime{
		JSON: true,
		Resolved: &auth.Resolved{
			Profile: config.Profile{BaseURL: server.URL, Token: "org-token"},
		},
	}

	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runAPICall(rt, []string{"--method", "POST", "--path", "/v1/documents", "--body-file", bodyPath})
	})
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stdout, `"created": true`) {
		t.Fatalf("expected created payload, got %q", stdout)
	}
	if gotContentType != "application/json" {
		t.Fatalf("expected default application/json content type, got %q", gotContentType)
	}
	if gotBody != `{"from":"file"}` {
		t.Fatalf("expected body file payload, got %q", gotBody)
	}
}

func TestRunAPICallUsageErrors(t *testing.T) {
	rt := &runtime{JSON: true}

	_, stderr, exit := captureRuntimeIO(t, func() int {
		return runAPICall(rt, []string{"--method", "POST", "--path", "/documents", "--body", "{}", "--body-file", "payload.json"})
	})
	if exit != 2 {
		t.Fatalf("expected usage exit 2, got %d", exit)
	}
	if !strings.Contains(stderr, "provide either --body-file or --body") {
		t.Fatalf("expected conflict message, got %q", stderr)
	}

	_, stderr, exit = captureRuntimeIO(t, func() int {
		return runAPICall(rt, []string{"--method", "GET"})
	})
	if exit != 2 {
		t.Fatalf("expected usage exit 2 for missing path, got %d", exit)
	}
	if !strings.Contains(stderr, "--path is required") {
		t.Fatalf("expected missing path message, got %q", stderr)
	}
}

func TestRunAPIHelp(t *testing.T) {
	rt := &runtime{}
	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runAPI(rt, nil)
	})
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d", exit)
	}
	if !strings.Contains(stdout, "omni api commands:") {
		t.Fatalf("expected API usage output, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}
