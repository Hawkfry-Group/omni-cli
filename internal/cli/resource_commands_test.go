package cli

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/omni-co/omni-cli/internal/auth"
	"github.com/omni-co/omni-cli/internal/config"
)

func TestExecuteDoctorRunsWithoutSubcommandArgs(t *testing.T) {
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

	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.json")
	cfg := &config.Config{
		CurrentProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {
				BaseURL:     server.URL,
				DefaultAuth: "org",
				OrgKey:      "org-token",
				OrgKeyStore: "config",
			},
		},
	}
	if err := config.Save(configPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	t.Setenv("OMNI_CONFIG", configPath)

	stdout, stderr, exit := captureExecute(t, []string{"--json", "doctor"})
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stdout, `"status": "ok"`) {
		t.Fatalf("expected doctor JSON output, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}

func TestRunFoldersCommandsMatchLiveValidatedShapes(t *testing.T) {
	const folderID = "550e8400-e29b-41d4-a716-446655440000"
	var sawList bool
	var sawCreate bool
	var sawPerms bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/folders":
			sawList = true
			if r.URL.Query().Get("pageSize") != "20" {
				t.Fatalf("expected pageSize query, got %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"records":[],"pageInfo":{"hasNextPage":false,"nextCursor":null,"pageSize":20,"totalRecords":0}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/folders":
			sawCreate = true
			if got := r.Header.Get("Authorization"); got != "Bearer org-token" {
				t.Fatalf("expected org auth header, got %q", got)
			}
			body := mustReadAll(t, r)
			if !strings.Contains(body, `"name":"Coverage Seed Folder"`) || !strings.Contains(body, `"scope":"organization"`) {
				t.Fatalf("unexpected folder create body %q", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"` + folderID + `","name":"Coverage Seed Folder","path":"coverage-seed-folder","scope":"organization"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/folders/"+folderID+"/permissions":
			sawPerms = true
			_, _ = w.Write([]byte(`{"permits":[{"id":"user-1","type":"user","name":"Jamie Fry"}]}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	rt := &runtime{
		JSON: true,
		Resolved: &auth.Resolved{
			ProfileName: "default",
			Profile: config.Profile{
				BaseURL:   server.URL,
				Token:     "org-token",
				TokenType: "org",
			},
		},
	}

	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runFolders(rt, []string{"list", "--page-size", "20"})
	})
	if exit != 0 || !strings.Contains(stdout, `"records": []`) || strings.TrimSpace(stderr) != "" {
		t.Fatalf("folders list failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	stdout, stderr, exit = captureRuntimeIO(t, func() int {
		return runFolders(rt, []string{"create", "--scope", "organization", "Coverage Seed Folder"})
	})
	if exit != 0 || !strings.Contains(stdout, folderID) || strings.TrimSpace(stderr) != "" {
		t.Fatalf("folders create failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	stdout, stderr, exit = captureRuntimeIO(t, func() int {
		return runFolders(rt, []string{"permissions", "get", folderID})
	})
	if exit != 0 || !strings.Contains(stdout, `"permits"`) || strings.TrimSpace(stderr) != "" {
		t.Fatalf("folders permissions get failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	if !sawList || !sawCreate || !sawPerms {
		t.Fatalf("expected all folder endpoints to be exercised, got list=%v create=%v perms=%v", sawList, sawCreate, sawPerms)
	}
}

func TestRunFoldersDeleteAndPermissionsMutations(t *testing.T) {
	const folderID = "550e8400-e29b-41d4-a716-446655440000"

	tmp := t.TempDir()
	permissionsPath := writeTempJSON(t, tmp, "folder-permissions.json", `{"permits":[{"id":"user-1","type":"user","role":"viewer"}]}`)

	saw := map[string]bool{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/folders/"+folderID:
			saw["delete"] = true
			_, _ = w.Write([]byte(`{"deleted":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/folders/"+folderID+"/permissions":
			saw["permissions-add"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"role":"viewer"`) {
				t.Fatalf("unexpected folder permissions add body %q", body)
			}
			_, _ = w.Write([]byte(`{"added":true}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/folders/"+folderID+"/permissions":
			saw["permissions-update"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"role":"viewer"`) {
				t.Fatalf("unexpected folder permissions update body %q", body)
			}
			_, _ = w.Write([]byte(`{"updated":true}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/folders/"+folderID+"/permissions":
			saw["permissions-revoke"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"role":"viewer"`) {
				t.Fatalf("unexpected folder permissions revoke body %q", body)
			}
			_, _ = w.Write([]byte(`{"revoked":true}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	rt := testRuntime(server.URL, "org-token", "org")

	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runFolders(rt, nil)
	})
	if exit != 0 || !strings.Contains(stdout, "omni folders commands:") || strings.TrimSpace(stderr) != "" {
		t.Fatalf("folders usage failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	_, stderr, exit = captureRuntimeIO(t, func() int {
		return runFolders(rt, []string{"nope"})
	})
	if exit != 2 || !strings.Contains(stderr, "unknown folders subcommand") {
		t.Fatalf("folders unknown subcommand failed: exit=%d stderr=%q", exit, stderr)
	}

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "delete", args: []string{"delete", folderID}, want: `"deleted": true`},
		{name: "permissions add", args: []string{"permissions", "add", "--file", permissionsPath, folderID}, want: `"added": true`},
		{name: "permissions update", args: []string{"permissions", "update", "--file", permissionsPath, folderID}, want: `"updated": true`},
		{name: "permissions revoke", args: []string{"permissions", "revoke", "--file", permissionsPath, folderID}, want: `"revoked": true`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, exit := captureRuntimeIO(t, func() int {
				return runFolders(rt, tc.args)
			})
			assertCommandSuccess(t, stdout, stderr, exit, tc.want)
		})
	}

	for _, key := range []string{"delete", "permissions-add", "permissions-update", "permissions-revoke"} {
		if !saw[key] {
			t.Fatalf("expected folders endpoint %q to be exercised", key)
		}
	}
}

func TestRunFoldersCreateRequiresFlagsBeforeName(t *testing.T) {
	_, stderr, exit := captureRuntimeIO(t, func() int {
		return runFolders(&runtime{JSON: true}, []string{"create", "Coverage Seed Folder", "--scope", "organization"})
	})
	if exit != 2 {
		t.Fatalf("expected usage exit 2, got %d", exit)
	}
	if !strings.Contains(stderr, "usage: omni folders create") {
		t.Fatalf("expected usage error, got %q", stderr)
	}
}

func TestRunUserAttributesUsersAndSCIMListShapes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/user-attributes":
			_, _ = w.Write([]byte(`{"records":[{"name":"omni_user_email","system":true}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/users/email-only":
			if r.URL.Query().Get("pageSize") != "20" {
				t.Fatalf("expected users pageSize query, got %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"records":[],"pageInfo":{"hasNextPage":false,"nextCursor":null,"pageSize":20,"totalRecords":0}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/scim/v2/Users":
			if r.URL.Query().Get("count") != "20" {
				t.Fatalf("expected scim count query, got %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"Resources":[{"id":"user-1","userName":"jamie@hawkfry.com"}],"itemsPerPage":1,"schemas":["urn:ietf:params:scim:api:messages:2.0:ListResponse"],"startIndex":1,"totalResults":1}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	orgRT := &runtime{
		JSON: true,
		Resolved: &auth.Resolved{
			ProfileName: "default",
			Profile: config.Profile{
				BaseURL:   server.URL,
				Token:     "org-token",
				TokenType: "org",
			},
		},
	}

	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runUserAttributes(orgRT, []string{"list"})
	})
	if exit != 0 || !strings.Contains(stdout, `"omni_user_email"`) || strings.TrimSpace(stderr) != "" {
		t.Fatalf("user-attributes list failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	stdout, stderr, exit = captureRuntimeIO(t, func() int {
		return runUsers(orgRT, []string{"list-email-only", "--page-size", "20"})
	})
	if exit != 0 || !strings.Contains(stdout, `"records": []`) || strings.TrimSpace(stderr) != "" {
		t.Fatalf("users list-email-only failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	stdout, stderr, exit = captureRuntimeIO(t, func() int {
		return runSCIM(orgRT, []string{"users", "list", "--count", "20"})
	})
	if exit != 0 || !strings.Contains(stdout, `"userName": "jamie@hawkfry.com"`) || strings.TrimSpace(stderr) != "" {
		t.Fatalf("scim users list failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}
}

func TestRunUtilityAndUnstableCommands(t *testing.T) {
	const documentID = "wk_coverage_seed"

	tmp := t.TempDir()
	importPath := writeTempJSON(t, tmp, "unstable-import.json", `{"name":"Imported Coverage Workbook"}`)

	saw := map[string]bool{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/unstable/documents/"+documentID+"/export":
			saw["export"] = true
			_, _ = w.Write([]byte(`{"id":"` + documentID + `","name":"Coverage Export"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/unstable/documents/import":
			saw["import"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"name":"Imported Coverage Workbook"`) {
				t.Fatalf("unexpected unstable import body %q", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"wk_imported"}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	rt := testRuntime(server.URL, "pat-token", "pat")

	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runExitCodes(rt, nil)
	})
	if exit != 0 || !strings.Contains(stdout, `"exit_codes"`) || strings.TrimSpace(stderr) != "" {
		t.Fatalf("exit-codes failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	_, stderr, exit = captureRuntimeIO(t, func() int {
		return runExitCodes(rt, []string{"extra"})
	})
	if exit != 2 || !strings.Contains(stderr, "usage: omni exit-codes") {
		t.Fatalf("exit-codes usage failure missing: exit=%d stderr=%q", exit, stderr)
	}

	stdout, stderr, exit = captureRuntimeIO(t, func() int {
		return runSchema(rt, []string{"-h"})
	})
	if exit != 0 || !strings.Contains(stdout, "omni schema:") || strings.TrimSpace(stderr) != "" {
		t.Fatalf("schema help failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	stdout, stderr, exit = captureRuntimeIO(t, func() int {
		return runUserAttributes(rt, nil)
	})
	if exit != 0 || !strings.Contains(stdout, "omni user-attributes commands:") || strings.TrimSpace(stderr) != "" {
		t.Fatalf("user-attributes usage failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	stdout, stderr, exit = captureRuntimeIO(t, func() int {
		return runUnstable(rt, nil)
	})
	if exit != 0 || !strings.Contains(stdout, "omni unstable commands:") || strings.TrimSpace(stderr) != "" {
		t.Fatalf("unstable usage failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	stdout, stderr, exit = captureRuntimeIO(t, func() int {
		return runUnstable(rt, []string{"documents", "export", documentID})
	})
	if exit != 0 || !strings.Contains(stdout, `"Coverage Export"`) || strings.TrimSpace(stderr) != "" {
		t.Fatalf("unstable documents export failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	stdout, stderr, exit = captureRuntimeIO(t, func() int {
		return runUnstable(rt, []string{"documents", "import", "--file", importPath})
	})
	if exit != 0 || !strings.Contains(stdout, `"id": "wk_imported"`) || strings.TrimSpace(stderr) != "" {
		t.Fatalf("unstable documents import failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	for _, key := range []string{"export", "import"} {
		if !saw[key] {
			t.Fatalf("expected unstable endpoint %q to be exercised", key)
		}
	}
}

func mustReadAll(t *testing.T, r *http.Request) string {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	return string(body)
}
