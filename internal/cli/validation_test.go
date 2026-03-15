package cli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/omni-co/omni-cli/internal/client"
)

func TestCollectValidationPATLimitedPermissions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/content":
			w.Header().Set("X-RateLimit-Limit", "100")
			w.Header().Set("X-RateLimit-Remaining", "75")
			w.Header().Set("X-RateLimit-Reset", "1700000000")
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		case "/api/v1/query/run":
			if r.Method != http.MethodPost {
				t.Fatalf("expected query probe POST, got %s", r.Method)
			}
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	api, err := client.New(server.URL, "pat-token")
	if err != nil {
		t.Fatalf("client.New returned error: %v", err)
	}

	summary, failure := collectValidation(context.Background(), api, "pat", false)
	if failure != nil {
		t.Fatalf("expected no fatal validation failure, got %v", failure)
	}
	if summary.Base.Status != "warn" || summary.Base.Message == "" {
		t.Fatalf("expected base warning, got %#v", summary.Base)
	}
	if summary.Query.Status != "fail" || summary.Query.Message != "permission denied" {
		t.Fatalf("expected query permission failure, got %#v", summary.Query)
	}
	if summary.Admin.Status != "skipped" {
		t.Fatalf("expected admin to be skipped for PAT, got %#v", summary.Admin)
	}
	if len(summary.Capabilities) != 1 || summary.Capabilities[0] != "base_api" {
		t.Fatalf("expected only base_api capability, got %#v", summary.Capabilities)
	}
	if summary.RateLimit["limit"] != 100 || summary.RateLimit["remaining"] != 75 {
		t.Fatalf("expected parsed rate limit headers, got %#v", summary.RateLimit)
	}
}

func TestCollectValidationOrgFullAccess(t *testing.T) {
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

	api, err := client.New(server.URL, "org-token")
	if err != nil {
		t.Fatalf("client.New returned error: %v", err)
	}

	summary, failure := collectValidation(context.Background(), api, "org", true)
	if failure != nil {
		t.Fatalf("expected no validation failure, got %v", failure)
	}
	if summary.Base.Status != "pass" || summary.Query.Status != "pass" || summary.Admin.Status != "pass" {
		t.Fatalf("expected full pass summary, got %#v", summary)
	}
	if got := strings.Join(summary.Capabilities, ","); got != "base_api,query_api,admin_scim" {
		t.Fatalf("unexpected capabilities %q", got)
	}
}

func TestCollectValidationUnauthorizedFailsFast(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/content" {
			t.Fatalf("expected only base probe before failure, got %s", r.URL.Path)
		}
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
	}))
	defer server.Close()

	api, err := client.New(server.URL, "bad-token")
	if err != nil {
		t.Fatalf("client.New returned error: %v", err)
	}

	summary, failure := collectValidation(context.Background(), api, "pat", false)
	if failure == nil {
		t.Fatal("expected validation failure for unauthorized token")
	}
	if failure.Code != codeAuthUnauthorized {
		t.Fatalf("expected auth unauthorized failure code, got %#v", failure)
	}
	if summary.Base.Status != "fail" || summary.Base.Message != "unauthorized" {
		t.Fatalf("expected unauthorized base summary, got %#v", summary.Base)
	}
}
