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

func TestRunSCIMCommands(t *testing.T) {
	const (
		userID  = "550e8400-e29b-41d4-a716-446655440000"
		groupID = "grp_coverage"
	)

	tmp := t.TempDir()
	userCreatePath := writeTempJSON(t, tmp, "scim-user-create.json", `{"userName":"jamie@example.com"}`)
	userUpdatePath := writeTempJSON(t, tmp, "scim-user-update.json", `{"Operations":[{"op":"replace","path":"displayName","value":"Jamie Fry"}]}`)
	userReplacePath := writeTempJSON(t, tmp, "scim-user-replace.json", `{"userName":"jamie+updated@example.com"}`)
	groupCreatePath := writeTempJSON(t, tmp, "scim-group-create.json", `{"displayName":"Coverage Analysts"}`)
	groupUpdatePath := writeTempJSON(t, tmp, "scim-group-update.json", `{"Operations":[{"op":"add","path":"members","value":[]}]}`)
	groupReplacePath := writeTempJSON(t, tmp, "scim-group-replace.json", `{"displayName":"Coverage Operators"}`)

	saw := map[string]bool{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/scim/v2/Users":
			saw["users-list"] = true
			if r.URL.Query().Get("count") != "6" || r.URL.Query().Get("startIndex") != "2" || r.URL.Query().Get("filter") != `userName eq "jamie@example.com"` {
				t.Fatalf("unexpected scim users list query %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"Resources":[{"id":"` + userID + `","userName":"jamie@example.com"}],"itemsPerPage":1,"schemas":["urn:ietf:params:scim:api:messages:2.0:ListResponse"],"startIndex":1,"totalResults":1}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/scim/v2/Users/"+userID:
			saw["users-get"] = true
			_, _ = w.Write([]byte(`{"id":"` + userID + `","userName":"jamie@example.com"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/scim/v2/Users":
			saw["users-create"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"userName":"jamie@example.com"`) {
				t.Fatalf("unexpected scim users create body %q", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"` + userID + `"}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/scim/v2/Users/"+userID:
			saw["users-update"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"displayName"`) {
				t.Fatalf("unexpected scim users update body %q", body)
			}
			_, _ = w.Write([]byte(`{"updated":true}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/scim/v2/Users/"+userID:
			saw["users-replace"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `jamie+updated@example.com`) {
				t.Fatalf("unexpected scim users replace body %q", body)
			}
			_, _ = w.Write([]byte(`{"replaced":true}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/scim/v2/Users/"+userID:
			saw["users-delete"] = true
			_, _ = w.Write([]byte(`{"deleted":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/scim/v2/Groups":
			saw["groups-list"] = true
			if r.URL.Query().Get("count") != "5" || r.URL.Query().Get("startIndex") != "4" {
				t.Fatalf("unexpected scim groups list query %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"Resources":[{"id":"` + groupID + `","displayName":"Coverage Analysts"}],"itemsPerPage":1,"schemas":["urn:ietf:params:scim:api:messages:2.0:ListResponse"],"startIndex":1,"totalResults":1}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/scim/v2/Groups/"+groupID:
			saw["groups-get"] = true
			_, _ = w.Write([]byte(`{"id":"` + groupID + `","displayName":"Coverage Analysts"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/scim/v2/Groups":
			saw["groups-create"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"displayName":"Coverage Analysts"`) {
				t.Fatalf("unexpected scim groups create body %q", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"` + groupID + `"}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/scim/v2/Groups/"+groupID:
			saw["groups-update"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"members"`) {
				t.Fatalf("unexpected scim groups update body %q", body)
			}
			_, _ = w.Write([]byte(`{"updated":true}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/scim/v2/Groups/"+groupID:
			saw["groups-replace"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"Coverage Operators"`) {
				t.Fatalf("unexpected scim groups replace body %q", body)
			}
			_, _ = w.Write([]byte(`{"replaced":true}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/scim/v2/Groups/"+groupID:
			saw["groups-delete"] = true
			_, _ = w.Write([]byte(`{"deleted":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/scim/v2/embed/Users":
			saw["embed-users-list"] = true
			if r.URL.Query().Get("count") != "3" || r.URL.Query().Get("startIndex") != "2" || r.URL.Query().Get("filter") != `userName co "jamie"` {
				t.Fatalf("unexpected scim embed-users list query %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"Resources":[{"id":"` + userID + `","userName":"jamie@example.com"}],"itemsPerPage":1,"schemas":["urn:ietf:params:scim:api:messages:2.0:ListResponse"],"startIndex":1,"totalResults":1}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/scim/v2/embed/Users/"+userID:
			saw["embed-users-get"] = true
			_, _ = w.Write([]byte(`{"id":"` + userID + `","userName":"jamie@example.com"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/scim/v2/embed/Users/"+userID:
			saw["embed-users-delete"] = true
			_, _ = w.Write([]byte(`{"deleted":true}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	orgRT := testRuntime(server.URL, "org-token", "org")
	patRT := testRuntime(server.URL, "pat-token", "pat")

	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runSCIM(orgRT, nil)
	})
	if exit != 0 || !strings.Contains(stdout, "omni scim commands:") || strings.TrimSpace(stderr) != "" {
		t.Fatalf("scim usage failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	_, stderr, exit = captureRuntimeIO(t, func() int {
		return runSCIM(patRT, []string{"users", "list"})
	})
	if exit != 1 || !strings.Contains(stderr, "scim commands require an org API key") {
		t.Fatalf("scim org guard failed: exit=%d stderr=%q", exit, stderr)
	}

	_, stderr, exit = captureRuntimeIO(t, func() int {
		return runSCIM(orgRT, []string{"teams"})
	})
	if exit != 2 || !strings.Contains(stderr, "unknown scim resource") {
		t.Fatalf("scim unknown resource failed: exit=%d stderr=%q", exit, stderr)
	}

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "users list", args: []string{"users", "list", "--count", "6", "--start-index", "2", "--filter", `userName eq "jamie@example.com"`}, want: `"userName": "jamie@example.com"`},
		{name: "users get", args: []string{"users", "get", userID}, want: `"id": "` + userID + `"`},
		{name: "users create", args: []string{"users", "create", "--file", userCreatePath}, want: `"id": "` + userID + `"`},
		{name: "users update", args: []string{"users", "update", "--file", userUpdatePath, userID}, want: `"updated": true`},
		{name: "users replace", args: []string{"users", "replace", "--file", userReplacePath, userID}, want: `"replaced": true`},
		{name: "users delete", args: []string{"users", "delete", userID}, want: `"deleted": true`},
		{name: "groups list", args: []string{"groups", "list", "--count", "5", "--start-index", "4"}, want: `"Coverage Analysts"`},
		{name: "groups get", args: []string{"groups", "get", groupID}, want: `"id": "` + groupID + `"`},
		{name: "groups create", args: []string{"groups", "create", "--file", groupCreatePath}, want: `"id": "` + groupID + `"`},
		{name: "groups update", args: []string{"groups", "update", "--file", groupUpdatePath, groupID}, want: `"updated": true`},
		{name: "groups replace", args: []string{"groups", "replace", "--file", groupReplacePath, groupID}, want: `"replaced": true`},
		{name: "groups delete", args: []string{"groups", "delete", groupID}, want: `"deleted": true`},
		{name: "embed-users list", args: []string{"embed-users", "list", "--count", "3", "--start-index", "2", "--filter", `userName co "jamie"`}, want: `"userName": "jamie@example.com"`},
		{name: "embed-users get", args: []string{"embed-users", "get", userID}, want: `"id": "` + userID + `"`},
		{name: "embed-users delete", args: []string{"embed-users", "delete", userID}, want: `"deleted": true`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, exit := captureRuntimeIO(t, func() int {
				return runSCIM(orgRT, tc.args)
			})
			assertCommandSuccess(t, stdout, stderr, exit, tc.want)
		})
	}

	for _, key := range []string{
		"users-list", "users-get", "users-create", "users-update", "users-replace", "users-delete",
		"groups-list", "groups-get", "groups-create", "groups-update", "groups-replace", "groups-delete",
		"embed-users-list", "embed-users-get", "embed-users-delete",
	} {
		if !saw[key] {
			t.Fatalf("expected scim endpoint %q to be exercised", key)
		}
	}
}

func TestRunConnectionsCommands(t *testing.T) {
	const (
		connectionID  = "550e8400-e29b-41d4-a716-446655440000"
		scheduleID    = "11111111-1111-1111-1111-111111111111"
		environmentID = "22222222-2222-2222-2222-222222222222"
	)

	tmp := t.TempDir()
	connectionCreatePath := writeTempJSON(t, tmp, "connection-create.json", `{"name":"Coverage Warehouse","dialect":"POSTGRES"}`)
	connectionUpdatePath := writeTempJSON(t, tmp, "connection-update.json", `{"name":"Coverage Warehouse Updated"}`)
	dbtUpdatePath := writeTempJSON(t, tmp, "connection-dbt-update.json", `{"projectPath":"/tmp/dbt"}`)
	scheduleCreatePath := writeTempJSON(t, tmp, "connection-schedule-create.json", `{"name":"Nightly Refresh"}`)
	scheduleUpdatePath := writeTempJSON(t, tmp, "connection-schedule-update.json", `{"name":"Nightly Refresh Updated"}`)
	environmentCreatePath := writeTempJSON(t, tmp, "connection-environment-create.json", `{"name":"Staging"}`)
	environmentUpdatePath := writeTempJSON(t, tmp, "connection-environment-update.json", `{"name":"Staging Updated"}`)

	saw := map[string]bool{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/connections":
			saw["list"] = true
			if r.URL.Query().Get("name") != "Coverage Warehouse" {
				t.Fatalf("unexpected connections list query %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"connections":[{"id":"` + connectionID + `","name":"Coverage Warehouse","dialect":"POSTGRES"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/connections":
			saw["create"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"Coverage Warehouse"`) {
				t.Fatalf("unexpected connections create body %q", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"` + connectionID + `"}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/connections/"+connectionID:
			saw["update"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"Coverage Warehouse Updated"`) {
				t.Fatalf("unexpected connections update body %q", body)
			}
			_, _ = w.Write([]byte(`{"updated":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/connections/"+connectionID+"/dbt":
			saw["dbt-get"] = true
			_, _ = w.Write([]byte(`{"projectPath":"/tmp/dbt"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/connections/"+connectionID+"/dbt":
			saw["dbt-update"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"/tmp/dbt"`) {
				t.Fatalf("unexpected connections dbt update body %q", body)
			}
			_, _ = w.Write([]byte(`{"updated":true}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/connections/"+connectionID+"/dbt":
			saw["dbt-delete"] = true
			_, _ = w.Write([]byte(`{"deleted":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/connections/"+connectionID+"/schedules":
			saw["schedules-list"] = true
			_, _ = w.Write([]byte(`{"records":[{"id":"` + scheduleID + `"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/connections/"+connectionID+"/schedules":
			saw["schedules-create"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"Nightly Refresh"`) {
				t.Fatalf("unexpected connections schedules create body %q", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"` + scheduleID + `"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/connections/"+connectionID+"/schedules/"+scheduleID:
			saw["schedules-get"] = true
			_, _ = w.Write([]byte(`{"id":"` + scheduleID + `"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/connections/"+connectionID+"/schedules/"+scheduleID:
			saw["schedules-update"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"Nightly Refresh Updated"`) {
				t.Fatalf("unexpected connections schedules update body %q", body)
			}
			_, _ = w.Write([]byte(`{"updated":true}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/connections/"+connectionID+"/schedules/"+scheduleID:
			saw["schedules-delete"] = true
			_, _ = w.Write([]byte(`{"deleted":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/connection-environments":
			saw["environments-list"] = true
			_, _ = w.Write([]byte(`{"records":[{"id":"` + environmentID + `","name":"Staging"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/connection-environments":
			saw["environments-create"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"Staging"`) {
				t.Fatalf("unexpected connections environments create body %q", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"` + environmentID + `"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/connection-environments/"+environmentID:
			saw["environments-update"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"Staging Updated"`) {
				t.Fatalf("unexpected connections environments update body %q", body)
			}
			_, _ = w.Write([]byte(`{"updated":true}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/connection-environments/"+environmentID:
			saw["environments-delete"] = true
			_, _ = w.Write([]byte(`{"deleted":true}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	rt := testRuntime(server.URL, "pat-token", "pat")

	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runConnections(rt, nil)
	})
	if exit != 0 || !strings.Contains(stdout, "omni connections commands:") || strings.TrimSpace(stderr) != "" {
		t.Fatalf("connections usage failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	_, stderr, exit = captureRuntimeIO(t, func() int {
		return runConnections(rt, []string{"nope"})
	})
	if exit != 2 || !strings.Contains(stderr, "unknown connections subcommand") {
		t.Fatalf("connections unknown subcommand failed: exit=%d stderr=%q", exit, stderr)
	}

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "list", args: []string{"list", "--name", "Coverage Warehouse"}, want: `"name": "Coverage Warehouse"`},
		{name: "create", args: []string{"create", "--file", connectionCreatePath}, want: `"id": "` + connectionID + `"`},
		{name: "update", args: []string{"update", "--file", connectionUpdatePath, connectionID}, want: `"updated": true`},
		{name: "dbt get", args: []string{"dbt", "get", connectionID}, want: `"/tmp/dbt"`},
		{name: "dbt update", args: []string{"dbt", "update", "--file", dbtUpdatePath, connectionID}, want: `"updated": true`},
		{name: "dbt delete", args: []string{"dbt", "delete", connectionID}, want: `"deleted": true`},
		{name: "schedules list", args: []string{"schedules", "list", connectionID}, want: `"id": "` + scheduleID + `"`},
		{name: "schedules create", args: []string{"schedules", "create", "--file", scheduleCreatePath, connectionID}, want: `"id": "` + scheduleID + `"`},
		{name: "schedules get", args: []string{"schedules", "get", connectionID, scheduleID}, want: `"id": "` + scheduleID + `"`},
		{name: "schedules update", args: []string{"schedules", "update", "--file", scheduleUpdatePath, connectionID, scheduleID}, want: `"updated": true`},
		{name: "schedules delete", args: []string{"schedules", "delete", connectionID, scheduleID}, want: `"deleted": true`},
		{name: "environments list", args: []string{"environments", "list"}, want: `"Staging"`},
		{name: "environments create", args: []string{"environments", "create", "--file", environmentCreatePath}, want: `"id": "` + environmentID + `"`},
		{name: "environments update", args: []string{"environments", "update", "--file", environmentUpdatePath, environmentID}, want: `"updated": true`},
		{name: "environments delete", args: []string{"environments", "delete", environmentID}, want: `"deleted": true`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, exit := captureRuntimeIO(t, func() int {
				return runConnections(rt, tc.args)
			})
			assertCommandSuccess(t, stdout, stderr, exit, tc.want)
		})
	}

	for _, key := range []string{
		"list", "create", "update",
		"dbt-get", "dbt-update", "dbt-delete",
		"schedules-list", "schedules-create", "schedules-get", "schedules-update", "schedules-delete",
		"environments-list", "environments-create", "environments-update", "environments-delete",
	} {
		if !saw[key] {
			t.Fatalf("expected connections endpoint %q to be exercised", key)
		}
	}
}

func TestRunUsersCommands(t *testing.T) {
	const (
		userID       = "550e8400-e29b-41d4-a716-446655440000"
		connectionID = "11111111-1111-1111-1111-111111111111"
		modelID      = "22222222-2222-2222-2222-222222222222"
		groupID      = "grp_coverage"
	)

	tmp := t.TempDir()
	createPath := writeTempJSON(t, tmp, "users-create-email-only.json", `{"email":"jamie@example.com"}`)
	bulkCreatePath := writeTempJSON(t, tmp, "users-create-email-only-bulk.json", `{"emails":["jamie@example.com","team@example.com"]}`)
	rolesAssignPath := writeTempJSON(t, tmp, "users-roles-assign.json", `{"role":"MODELER"}`)
	groupRolesAssignPath := writeTempJSON(t, tmp, "users-group-roles-assign.json", `{"role":"VIEWER"}`)

	saw := map[string]bool{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/users/email-only":
			saw["list-email-only"] = true
			q := r.URL.Query()
			if q.Get("cursor") != "users-next" || q.Get("pageSize") != "7" || q.Get("email") != "jamie@example.com" || q.Get("sortDirection") != "desc" {
				t.Fatalf("unexpected users list-email-only query %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"records":[{"id":"` + userID + `","email":"jamie@example.com"}],"pageInfo":{"hasNextPage":false,"nextCursor":null,"pageSize":7,"totalRecords":1}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/users/email-only":
			saw["create-email-only"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"email":"jamie@example.com"`) {
				t.Fatalf("unexpected users create-email-only body %q", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"` + userID + `"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/users/email-only/bulk":
			saw["create-email-only-bulk"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"team@example.com"`) {
				t.Fatalf("unexpected users create-email-only-bulk body %q", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"created":2}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/users/"+userID+"/model-roles":
			saw["roles-get"] = true
			q := r.URL.Query()
			if q.Get("connectionId") != connectionID || q.Get("modelId") != modelID {
				t.Fatalf("unexpected users roles get query %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"roles":["MODELER"]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/users/"+userID+"/model-roles":
			saw["roles-assign"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"MODELER"`) {
				t.Fatalf("unexpected users roles assign body %q", body)
			}
			_, _ = w.Write([]byte(`{"assigned":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/user-groups/"+groupID+"/model-roles":
			saw["group-roles-get"] = true
			q := r.URL.Query()
			if q.Get("connectionId") != connectionID || q.Get("modelId") != modelID {
				t.Fatalf("unexpected users group-roles get query %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"roles":["VIEWER"]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/user-groups/"+groupID+"/model-roles":
			saw["group-roles-assign"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"VIEWER"`) {
				t.Fatalf("unexpected users group-roles assign body %q", body)
			}
			_, _ = w.Write([]byte(`{"assigned":true}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	orgRT := testRuntime(server.URL, "org-token", "org")
	patRT := testRuntime(server.URL, "pat-token", "pat")

	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runUsers(orgRT, nil)
	})
	if exit != 0 || !strings.Contains(stdout, "omni users commands:") || strings.TrimSpace(stderr) != "" {
		t.Fatalf("users usage failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	_, stderr, exit = captureRuntimeIO(t, func() int {
		return runUsers(patRT, []string{"list-email-only"})
	})
	if exit != 1 || !strings.Contains(stderr, "users commands require an org API key") {
		t.Fatalf("users org guard failed: exit=%d stderr=%q", exit, stderr)
	}

	_, stderr, exit = captureRuntimeIO(t, func() int {
		return runUsers(orgRT, []string{"delete"})
	})
	if exit != 2 || !strings.Contains(stderr, "unknown users subcommand") {
		t.Fatalf("users unknown subcommand failed: exit=%d stderr=%q", exit, stderr)
	}

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "list-email-only", args: []string{"list-email-only", "--cursor", "users-next", "--page-size", "7", "--email", "jamie@example.com", "--sort-direction", "desc"}, want: `"email": "jamie@example.com"`},
		{name: "create-email-only", args: []string{"create-email-only", "--file", createPath}, want: `"id": "` + userID + `"`},
		{name: "create-email-only-bulk", args: []string{"create-email-only-bulk", "--file", bulkCreatePath}, want: `"created": 2`},
		{name: "roles get", args: []string{"roles", "get", "--connection-id", connectionID, "--model-id", modelID, userID}, want: `"roles": [`},
		{name: "roles assign", args: []string{"roles", "assign", "--file", rolesAssignPath, userID}, want: `"assigned": true`},
		{name: "group-roles get", args: []string{"group-roles", "get", "--connection-id", connectionID, "--model-id", modelID, groupID}, want: `"roles": [`},
		{name: "group-roles assign", args: []string{"group-roles", "assign", "--file", groupRolesAssignPath, groupID}, want: `"assigned": true`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, exit := captureRuntimeIO(t, func() int {
				return runUsers(orgRT, tc.args)
			})
			assertCommandSuccess(t, stdout, stderr, exit, tc.want)
		})
	}

	for _, key := range []string{
		"list-email-only", "create-email-only", "create-email-only-bulk",
		"roles-get", "roles-assign", "group-roles-get", "group-roles-assign",
	} {
		if !saw[key] {
			t.Fatalf("expected users endpoint %q to be exercised", key)
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
