package cli

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/omni-co/omni-cli/internal/auth"
	"github.com/omni-co/omni-cli/internal/client/gen"
	"github.com/omni-co/omni-cli/internal/config"
)

func TestRunConnectionsCreateWithFile(t *testing.T) {
	tmp := t.TempDir()
	bodyPath := filepath.Join(tmp, "connection.json")
	if err := os.WriteFile(bodyPath, []byte(`{"dialect":"postgres","name":"Coverage Postgres","passwordUnencrypted":"secret"}`), 0o600); err != nil {
		t.Fatalf("write connection body: %v", err)
	}

	var gotAuth string
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/connections" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"conn-1"}`))
	}))
	defer server.Close()

	rt := &runtime{
		JSON: true,
		Resolved: &auth.Resolved{
			Profile: config.Profile{
				BaseURL: server.URL,
				Token:   "org-token",
			},
		},
	}

	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runConnectionsCreateWithPrompts(rt, []string{"--file", bodyPath}, false, nil, nil)
	})
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stdout, `"id": "conn-1"`) {
		t.Fatalf("expected create response, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if gotAuth != "Bearer org-token" {
		t.Fatalf("expected bearer auth header, got %q", gotAuth)
	}
	if gotBody["dialect"] != "postgres" || gotBody["name"] != "Coverage Postgres" {
		t.Fatalf("unexpected request body: %#v", gotBody)
	}
}

func TestRunConnectionsCreateInteractivePostgresWizard(t *testing.T) {
	var gotBody gen.ConnectionsCreateJSONBody

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/connections" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"conn-wizard"}`))
	}))
	defer server.Close()

	rt := &runtime{
		JSON: true,
		Resolved: &auth.Resolved{
			Profile: config.Profile{
				BaseURL: server.URL,
				Token:   "org-token",
			},
		},
	}

	reader := bufio.NewReader(strings.NewReader(strings.Join([]string{
		"",
		"Coverage Postgres",
		"db.example.internal",
		"",
		"analytics",
		"omni",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
	}, "\n") + "\n"))

	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runConnectionsCreateWithPrompts(rt, nil, true, reader, func(_ *bufio.Reader, _ string) (string, error) {
			return "top-secret", nil
		})
	})
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stdout, `"id": "conn-wizard"`) {
		t.Fatalf("expected create response, got %q", stdout)
	}
	if !strings.Contains(stderr, "Interactive connection setup") {
		t.Fatalf("expected wizard prompt text, got %q", stderr)
	}
	if gotBody.Dialect != gen.Postgres {
		t.Fatalf("expected postgres dialect, got %q", gotBody.Dialect)
	}
	if gotBody.Name != "Coverage Postgres" {
		t.Fatalf("expected name to be preserved, got %q", gotBody.Name)
	}
	if gotBody.Host == nil || *gotBody.Host != "db.example.internal" {
		t.Fatalf("expected host to be set, got %#v", gotBody.Host)
	}
	if gotBody.Port == nil || *gotBody.Port != 5432 {
		t.Fatalf("expected default port 5432, got %#v", gotBody.Port)
	}
	if gotBody.Database == nil || *gotBody.Database != "analytics" {
		t.Fatalf("expected database to be set, got %#v", gotBody.Database)
	}
	if gotBody.Username == nil || *gotBody.Username != "omni" {
		t.Fatalf("expected username to be set, got %#v", gotBody.Username)
	}
	if gotBody.PasswordUnencrypted != "top-secret" {
		t.Fatalf("expected password to be forwarded, got %q", gotBody.PasswordUnencrypted)
	}
	if gotBody.BaseRole == nil || *gotBody.BaseRole != gen.QUERIER {
		t.Fatalf("expected default base role QUERIER, got %#v", gotBody.BaseRole)
	}
	if gotBody.SystemTimezone == nil || *gotBody.SystemTimezone != "UTC" {
		t.Fatalf("expected default system timezone UTC, got %#v", gotBody.SystemTimezone)
	}
	if gotBody.QueryTimezone != nil {
		t.Fatalf("expected blank query timezone to be omitted, got %#v", gotBody.QueryTimezone)
	}
	if gotBody.QueryTimeoutSeconds == nil || *gotBody.QueryTimeoutSeconds != 900 {
		t.Fatalf("expected default query timeout 900, got %#v", gotBody.QueryTimeoutSeconds)
	}
}

func TestRunConnectionsCreateRequiresFileOrInteractiveTTY(t *testing.T) {
	rt := &runtime{JSON: true}

	_, stderr, exit := captureRuntimeIO(t, func() int {
		return runConnectionsCreateWithPrompts(rt, nil, false, nil, nil)
	})
	if exit != 2 {
		t.Fatalf("expected usage exit 2, got %d", exit)
	}
	if !strings.Contains(stderr, "run from a terminal for interactive setup") {
		t.Fatalf("expected interactive guidance, got %q", stderr)
	}
}

func TestPromptConnectionCreatePayloadValidation(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		secret string
		want   string
	}{
		{
			name:  "invalid dialect",
			input: "duckdb\n",
			want:  `invalid dialect "duckdb"`,
		},
		{
			name: "invalid port",
			input: strings.Join([]string{
				"postgres",
				"Coverage Postgres",
				"db.example.internal",
				"zero",
			}, "\n") + "\n",
			want: "port must be a positive integer",
		},
		{
			name: "invalid base role",
			input: strings.Join([]string{
				"postgres",
				"Coverage Postgres",
				"db.example.internal",
				"5432",
				"analytics",
				"omni",
				"not-a-role",
			}, "\n") + "\n",
			secret: "top-secret",
			want:   `invalid base role "not-a-role"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := promptConnectionCreatePayload(bufio.NewReader(strings.NewReader(tc.input)), func(_ *bufio.Reader, _ string) (string, error) {
				return tc.secret, nil
			})
			if err == nil {
				t.Fatal("expected prompt error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestPromptConnectionCreatePayloadBigQueryUsesServiceAccountDefaults(t *testing.T) {
	tmp := t.TempDir()
	serviceAccountPath := filepath.Join(tmp, "service-account.json")
	serviceAccount := `{"project_id":"coverage-project","client_email":"svc@coverage-project.iam.gserviceaccount.com"}`
	if err := os.WriteFile(serviceAccountPath, []byte(serviceAccount), 0o600); err != nil {
		t.Fatalf("write service account file: %v", err)
	}

	reader := bufio.NewReader(strings.NewReader(strings.Join([]string{
		"bigquery",
		"Coverage BigQuery",
		serviceAccountPath,
		"",
		"",
		"",
		"us",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
	}, "\n") + "\n"))

	payload, err := promptConnectionCreatePayload(reader, func(_ *bufio.Reader, _ string) (string, error) {
		return "", nil
	})
	if err != nil {
		t.Fatalf("expected bigquery wizard to succeed, got %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if got["dialect"] != "bigquery" {
		t.Fatalf("expected bigquery dialect, got %#v", got["dialect"])
	}
	if got["database"] != "coverage-project" {
		t.Fatalf("expected project ID default, got %#v", got["database"])
	}
	if got["username"] != "svc@coverage-project.iam.gserviceaccount.com" {
		t.Fatalf("expected client email default, got %#v", got["username"])
	}
	if got["region"] != "us" {
		t.Fatalf("expected region to be set, got %#v", got["region"])
	}
	if got["passwordUnencrypted"] != serviceAccount {
		t.Fatalf("expected raw service account JSON to be stored, got %#v", got["passwordUnencrypted"])
	}
}
