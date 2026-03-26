package cli

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunDocumentsCommands(t *testing.T) {
	const (
		documentID   = "wk_coverage_seed"
		favoriteUser = "11111111-1111-1111-1111-111111111111"
	)

	tmp := t.TempDir()
	createPath := writeTempJSON(t, tmp, "documents-create.json", `{"name":"Coverage Seed Workbook"}`)
	movePath := writeTempJSON(t, tmp, "documents-move.json", `{"folderPath":"finance/coverage","scope":"organization"}`)
	draftPath := writeTempJSON(t, tmp, "documents-draft.json", `{"replace":"coverage draft"}`)
	duplicatePath := writeTempJSON(t, tmp, "documents-duplicate.json", `{"name":"Coverage Seed Workbook Copy","scope":"organization"}`)
	permissionsPath := writeTempJSON(t, tmp, "documents-permissions.json", `{"permits":[{"id":"user-2","type":"user","role":"viewer"}]}`)
	permissionSettingsPath := writeTempJSON(t, tmp, "documents-permission-settings.json", `{"scope":"organization"}`)
	labelsBulkPath := writeTempJSON(t, tmp, "documents-labels-bulk.json", `{"labels":["finance","coverage"]}`)
	transferPath := writeTempJSON(t, tmp, "documents-transfer.json", `{"userId":"22222222-2222-2222-2222-222222222222"}`)

	saw := map[string]bool{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/documents":
			saw["list"] = true
			if r.URL.Query().Get("cursor") != "next-page" || r.URL.Query().Get("pageSize") != "7" {
				t.Fatalf("unexpected documents list query: %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"records":[],"pageInfo":{"hasNextPage":false,"nextCursor":null,"pageSize":7,"totalRecords":0}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/documents/"+documentID:
			saw["get"] = true
			_, _ = w.Write([]byte(`{"id":"` + documentID + `","name":"Coverage Seed Workbook"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/documents":
			saw["create"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"name":"Coverage Seed Workbook"`) {
				t.Fatalf("unexpected documents create body %q", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"` + documentID + `"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/documents/"+documentID:
			saw["delete"] = true
			_, _ = w.Write([]byte(`{"deleted":true}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/documents/"+documentID:
			saw["rename"] = true
			body := mustReadAll(t, r)
			if !strings.Contains(body, `"name":"Coverage Seed Workbook Renamed"`) || !strings.Contains(body, `"clearExistingDraft":true`) {
				t.Fatalf("unexpected documents rename body %q", body)
			}
			_, _ = w.Write([]byte(`{"renamed":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/documents/"+documentID+"/move":
			saw["move"] = true
			body := mustReadAll(t, r)
			if !strings.Contains(body, `"folderPath":"finance/coverage"`) || !strings.Contains(body, `"scope":"organization"`) {
				t.Fatalf("unexpected documents move body %q", body)
			}
			_, _ = w.Write([]byte(`{"moved":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/documents/"+documentID+"/draft":
			saw["draft-create"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"replace":"coverage draft"`) {
				t.Fatalf("unexpected documents draft create body %q", body)
			}
			_, _ = w.Write([]byte(`{"draft":true}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/documents/"+documentID+"/draft":
			saw["draft-discard"] = true
			if body := strings.TrimSpace(mustReadAll(t, r)); body != "{}" {
				t.Fatalf("expected default discard draft body, got %q", body)
			}
			_, _ = w.Write([]byte(`{"discarded":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/documents/"+documentID+"/duplicate":
			saw["duplicate"] = true
			if r.URL.Query().Get("userId") != favoriteUser {
				t.Fatalf("unexpected documents duplicate query: %q", r.URL.RawQuery)
			}
			if body := mustReadAll(t, r); !strings.Contains(body, `"name":"Coverage Seed Workbook Copy"`) {
				t.Fatalf("unexpected documents duplicate body %q", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"wk_duplicate"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/documents/"+documentID+"/favorite":
			saw["favorite-add"] = true
			if r.URL.Query().Get("userId") != favoriteUser {
				t.Fatalf("unexpected documents favorite add query: %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"favorited":true}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/documents/"+documentID+"/favorite":
			saw["favorite-remove"] = true
			if r.URL.Query().Get("userId") != favoriteUser {
				t.Fatalf("unexpected documents favorite remove query: %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"favorited":false}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/documents/"+documentID+"/access-list":
			saw["access-list"] = true
			q := r.URL.Query()
			if q.Get("cursor") != "access-next" || q.Get("pageSize") != "3" || q.Get("type") != "user" || q.Get("accessSource") != "folder" || q.Get("sortField") != "updatedAt" || q.Get("sortDirection") != "desc" {
				t.Fatalf("unexpected documents access query: %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"records":[],"pageInfo":{"hasNextPage":false,"nextCursor":null,"pageSize":3,"totalRecords":0}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/documents/"+documentID+"/permissions":
			saw["permissions-get"] = true
			if r.URL.Query().Get("userId") != favoriteUser {
				t.Fatalf("unexpected documents permissions get query: %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"permits":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/documents/"+documentID+"/permissions":
			saw["permissions-add"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"role":"viewer"`) {
				t.Fatalf("unexpected documents permissions add body %q", body)
			}
			_, _ = w.Write([]byte(`{"added":true}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/documents/"+documentID+"/permissions":
			saw["permissions-update"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"role":"viewer"`) {
				t.Fatalf("unexpected documents permissions update body %q", body)
			}
			_, _ = w.Write([]byte(`{"updated":true}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/documents/"+documentID+"/permissions":
			saw["permissions-revoke"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"role":"viewer"`) {
				t.Fatalf("unexpected documents permissions revoke body %q", body)
			}
			_, _ = w.Write([]byte(`{"revoked":true}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/documents/"+documentID+"/permissions":
			saw["permissions-settings"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"scope":"organization"`) {
				t.Fatalf("unexpected documents permissions settings body %q", body)
			}
			_, _ = w.Write([]byte(`{"settingsUpdated":true}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/documents/"+documentID+"/labels/finance":
			saw["label-add"] = true
			if r.URL.Query().Get("userId") != favoriteUser {
				t.Fatalf("unexpected documents label add query: %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"label":"finance"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/documents/"+documentID+"/labels/finance":
			saw["label-remove"] = true
			if r.URL.Query().Get("userId") != favoriteUser {
				t.Fatalf("unexpected documents label remove query: %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"removed":true}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/documents/"+documentID+"/labels":
			saw["labels-bulk-update"] = true
			if r.URL.Query().Get("userId") != favoriteUser {
				t.Fatalf("unexpected documents labels bulk update query: %q", r.URL.RawQuery)
			}
			if body := mustReadAll(t, r); !strings.Contains(body, `"labels":["finance","coverage"]`) {
				t.Fatalf("unexpected documents labels bulk update body %q", body)
			}
			_, _ = w.Write([]byte(`{"labelsUpdated":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/documents/"+documentID+"/queries":
			saw["queries"] = true
			_, _ = w.Write([]byte(`{"queries":[]}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/documents/"+documentID+"/transfer-ownership":
			saw["transfer-ownership"] = true
			body := mustReadAll(t, r)
			if !strings.Contains(body, `"userId":"22222222-2222-2222-2222-222222222222"`) && !strings.Contains(body, `"userId":"11111111-1111-1111-1111-111111111111"`) {
				t.Fatalf("unexpected documents transfer ownership body %q", body)
			}
			_, _ = w.Write([]byte(`{"transferred":true}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	rt := testRuntime(server.URL, "pat-token", "pat")

	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runDocuments(rt, nil)
	})
	if exit != 0 || !strings.Contains(stdout, "omni documents commands:") || strings.TrimSpace(stderr) != "" {
		t.Fatalf("documents usage failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	_, stderr, exit = captureRuntimeIO(t, func() int {
		return runDocuments(rt, []string{"nope"})
	})
	if exit != 2 || !strings.Contains(stderr, "unknown documents subcommand") {
		t.Fatalf("documents unknown subcommand failed: exit=%d stderr=%q", exit, stderr)
	}

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "list", args: []string{"list", "--cursor", "next-page", "--page-size", "7"}, want: `"records": []`},
		{name: "get", args: []string{"get", documentID}, want: documentID},
		{name: "create", args: []string{"create", "--file", createPath}, want: documentID},
		{name: "delete", args: []string{"delete", documentID}, want: `"deleted": true`},
		{name: "rename", args: []string{"rename", "--name", "Coverage Seed Workbook Renamed", "--clear-existing-draft", documentID}, want: `"renamed": true`},
		{name: "move", args: []string{"move", "--file", movePath, documentID}, want: `"moved": true`},
		{name: "draft create", args: []string{"draft", "create", "--file", draftPath, documentID}, want: `"draft": true`},
		{name: "draft discard", args: []string{"draft", "discard", documentID}, want: `"discarded": true`},
		{name: "duplicate", args: []string{"duplicate", "--file", duplicatePath, "--user-id", favoriteUser, documentID}, want: `"id": "wk_duplicate"`},
		{name: "favorite add", args: []string{"favorite", "add", "--user-id", favoriteUser, documentID}, want: `"favorited": true`},
		{name: "favorite remove", args: []string{"favorite", "remove", "--user-id", favoriteUser, documentID}, want: `"favorited": false`},
		{name: "access list", args: []string{"access", "list", "--cursor", "access-next", "--page-size", "3", "--type", "user", "--access-source", "folder", "--sort-field", "updatedAt", "--sort-direction", "desc", documentID}, want: `"records": []`},
		{name: "permissions get", args: []string{"permissions", "get", "--user-id", favoriteUser, documentID}, want: `"permits": []`},
		{name: "permissions add", args: []string{"permissions", "add", "--file", permissionsPath, documentID}, want: `"added": true`},
		{name: "permissions update", args: []string{"permissions", "update", "--file", permissionsPath, documentID}, want: `"updated": true`},
		{name: "permissions revoke", args: []string{"permissions", "revoke", "--file", permissionsPath, documentID}, want: `"revoked": true`},
		{name: "permissions settings", args: []string{"permissions", "settings", "--file", permissionSettingsPath, documentID}, want: `"settingsUpdated": true`},
		{name: "label add", args: []string{"label", "add", "--user-id", favoriteUser, documentID, "finance"}, want: `"label": "finance"`},
		{name: "label remove", args: []string{"label", "remove", "--user-id", favoriteUser, documentID, "finance"}, want: `"removed": true`},
		{name: "labels bulk-update", args: []string{"labels", "bulk-update", "--file", labelsBulkPath, "--user-id", favoriteUser, documentID}, want: `"labelsUpdated": true`},
		{name: "queries", args: []string{"queries", documentID}, want: `"queries": []`},
		{name: "transfer ownership with user", args: []string{"transfer-ownership", "--user-id", favoriteUser, documentID}, want: `"transferred": true`},
		{name: "transfer ownership with file", args: []string{"transfer-ownership", "--file", transferPath, documentID}, want: `"transferred": true`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, exit := captureRuntimeIO(t, func() int {
				return runDocuments(rt, tc.args)
			})
			assertCommandSuccess(t, stdout, stderr, exit, tc.want)
		})
	}

	for _, key := range []string{"list", "get", "create", "delete", "rename", "move", "draft-create", "draft-discard", "duplicate", "favorite-add", "favorite-remove", "access-list", "permissions-get", "permissions-add", "permissions-update", "permissions-revoke", "permissions-settings", "label-add", "label-remove", "labels-bulk-update", "queries", "transfer-ownership"} {
		if !saw[key] {
			t.Fatalf("expected documents endpoint %q to be exercised", key)
		}
	}
}

func TestRunJobsStatusAndErrors(t *testing.T) {
	const jobID = "job-coverage-123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/jobs/" + jobID + "/status":
			_, _ = w.Write([]byte(`{"job_id":"` + jobID + `","job_type":"REFRESH_SCHEMA","status":"COMPLETED"}`))
		case "/api/v1/jobs/denied/status":
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"detail":"token expired"}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	rt := testRuntime(server.URL, "pat-token", "pat")

	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runJobs(rt, nil)
	})
	if exit != 0 || !strings.Contains(stdout, "omni jobs commands:") || strings.TrimSpace(stderr) != "" {
		t.Fatalf("jobs usage failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	_, stderr, exit = captureRuntimeIO(t, func() int {
		return runJobs(rt, []string{"cancel"})
	})
	if exit != 2 || !strings.Contains(stderr, "unknown jobs subcommand") {
		t.Fatalf("jobs unknown subcommand failed: exit=%d stderr=%q", exit, stderr)
	}

	stdout, stderr, exit = captureRuntimeIO(t, func() int {
		return runJobs(rt, []string{"status", jobID})
	})
	assertCommandSuccess(t, stdout, stderr, exit, `"job_id": "job-coverage-123"`)

	_, stderr, exit = captureRuntimeIO(t, func() int {
		return runJobs(rt, []string{"status", "denied"})
	})
	if exit != 1 || !strings.Contains(stderr, codeAuthUnauthorized) || !strings.Contains(stderr, "token expired") {
		t.Fatalf("jobs unauthorized response failed: exit=%d stderr=%q", exit, stderr)
	}
}

func TestRunModelsCommands(t *testing.T) {
	const (
		modelID  = testCommandUUID
		branchID = "22222222-2222-2222-2222-222222222222"
	)

	tmp := t.TempDir()
	createPath := writeTempJSON(t, tmp, "models-create.json", `{"name":"Orders Coverage Model"}`)
	mergePath := writeTempJSON(t, tmp, "models-merge.json", `{"commitMessage":"merge coverage branch"}`)
	cacheResetPath := writeTempJSON(t, tmp, "models-cache-reset.json", `{"scope":"all"}`)
	topicUpdatePath := writeTempJSON(t, tmp, "models-topic-update.json", `{"label":"Revenue"}`)
	viewUpdatePath := writeTempJSON(t, tmp, "models-view-update.json", `{"hidden":false}`)
	fieldCreatePath := writeTempJSON(t, tmp, "models-field-create.json", `{"viewName":"orders","fieldName":"total_revenue"}`)
	fieldUpdatePath := writeTempJSON(t, tmp, "models-field-update.json", `{"label":"Total Revenue"}`)
	gitCreatePath := writeTempJSON(t, tmp, "models-git-create.json", `{"repository":"omni/coverage-model"}`)
	gitUpdatePath := writeTempJSON(t, tmp, "models-git-update.json", `{"branch":"main"}`)
	gitSyncPath := writeTempJSON(t, tmp, "models-git-sync.json", `{"force":true}`)
	migratePath := writeTempJSON(t, tmp, "models-migrate.json", `{"targetConnectionId":"`+branchID+`"}`)
	contentValidatorPath := writeTempJSON(t, tmp, "models-content-validator.json", `{"find_or_replace_type":"FIELD","find":"old_name","replace":"new_name"}`)
	yamlCreatePath := writeTempJSON(t, tmp, "models-yaml-create.json", `{"fileName":"orders.yaml","content":"name: orders"}`)

	saw := map[string]bool{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/models" && r.URL.Query().Get("modelId") == "":
			saw["list"] = true
			if r.URL.Query().Get("cursor") != "models-next" || r.URL.Query().Get("pageSize") != "5" || r.URL.Query().Get("name") != "Orders" {
				t.Fatalf("unexpected models list query: %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"records":[{"id":"` + modelID + `","name":"Orders Coverage Model","createdAt":"2025-01-01T00:00:00Z","updatedAt":"2025-01-02T00:00:00Z","deletedAt":null,"baseModelId":null,"connectionId":null,"modelKind":"WORKBOOK"}],"pageInfo":{"hasNextPage":false,"nextCursor":null,"pageSize":5,"totalRecords":1}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/models" && r.URL.Query().Get("modelId") == modelID:
			saw["get"] = true
			_, _ = w.Write([]byte(`{"records":[{"id":"` + modelID + `","name":"Orders Coverage Model"}],"pageInfo":{"hasNextPage":false,"nextCursor":null,"pageSize":1,"totalRecords":1}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/models":
			saw["create"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"name":"Orders Coverage Model"`) {
				t.Fatalf("unexpected models create body %q", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"` + modelID + `"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/models/"+modelID+"/refresh":
			saw["refresh"] = true
			_, _ = w.Write([]byte(`{"status":"running"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/models/"+modelID+"/validate":
			saw["validate"] = true
			if r.URL.Query().Get("branchId") != branchID || r.URL.Query().Get("limit") != "9" {
				t.Fatalf("unexpected models validate query: %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"issues":[]}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/models/"+modelID+"/branch/coverage-branch":
			saw["branch-delete"] = true
			_, _ = w.Write([]byte(`{"deleted":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/models/"+modelID+"/branch/coverage-branch/merge":
			saw["branch-merge"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"commitMessage":"merge coverage branch"`) {
				t.Fatalf("unexpected models branch merge body %q", body)
			}
			_, _ = w.Write([]byte(`{"merged":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/models/"+modelID+"/cache_reset/default":
			saw["cache-reset"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"scope":"all"`) {
				t.Fatalf("unexpected models cache-reset body %q", body)
			}
			_, _ = w.Write([]byte(`{"reset":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/models/"+modelID+"/topic":
			saw["topics-list"] = true
			if r.URL.Query().Get("branch_id") != branchID {
				t.Fatalf("unexpected models topics list query: %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"success":true,"records":[]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/models/"+modelID+"/topic/orders":
			saw["topics-get"] = true
			if r.URL.Query().Get("branch_id") != branchID {
				t.Fatalf("unexpected models topics get query: %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"success":true,"topic":{"base_view_name":"default","name":"orders","relationships":[],"views":[]}}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/models/"+modelID+"/topic/orders":
			saw["topics-update"] = true
			if r.URL.Query().Get("branch_id") != branchID {
				t.Fatalf("unexpected models topics update query: %q", r.URL.RawQuery)
			}
			if body := mustReadAll(t, r); !strings.Contains(body, `"label":"Revenue"`) {
				t.Fatalf("unexpected models topics update body %q", body)
			}
			_, _ = w.Write([]byte(`{"updated":true}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/models/"+modelID+"/topic/orders":
			saw["topics-delete"] = true
			if r.URL.Query().Get("branch_id") != branchID {
				t.Fatalf("unexpected models topics delete query: %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"deleted":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/models/"+modelID+"/view":
			saw["views-list"] = true
			if r.URL.Query().Get("branch_id") != branchID {
				t.Fatalf("unexpected models views list query: %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"records":[]}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/models/"+modelID+"/view/default":
			saw["views-update"] = true
			if r.URL.Query().Get("branch_id") != branchID {
				t.Fatalf("unexpected models views update query: %q", r.URL.RawQuery)
			}
			if body := mustReadAll(t, r); !strings.Contains(body, `"hidden":false`) {
				t.Fatalf("unexpected models views update body %q", body)
			}
			_, _ = w.Write([]byte(`{"updated":true}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/models/"+modelID+"/view/default":
			saw["views-delete"] = true
			if r.URL.Query().Get("branch_id") != branchID {
				t.Fatalf("unexpected models views delete query: %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"deleted":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/models/"+modelID+"/field":
			saw["fields-create"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"fieldName":"total_revenue"`) {
				t.Fatalf("unexpected models fields create body %q", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"created":true}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/models/"+modelID+"/view/default/field/total_revenue":
			saw["fields-update"] = true
			if r.URL.Query().Get("branch_id") != branchID {
				t.Fatalf("unexpected models fields update query: %q", r.URL.RawQuery)
			}
			if body := mustReadAll(t, r); !strings.Contains(body, `"label":"Total Revenue"`) {
				t.Fatalf("unexpected models fields update body %q", body)
			}
			_, _ = w.Write([]byte(`{"updated":true}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/models/"+modelID+"/view/default/field/total_revenue":
			saw["fields-delete"] = true
			if r.URL.Query().Get("branch_id") != branchID || r.URL.Query().Get("topic_context") != "orders" {
				t.Fatalf("unexpected models fields delete query: %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"deleted":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/models/"+modelID+"/git":
			saw["git-get"] = true
			if r.URL.Query().Get("include") != "webhookSecret" {
				t.Fatalf("unexpected models git get query: %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"repository":"omni/coverage-model"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/models/"+modelID+"/git":
			saw["git-create"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"repository":"omni/coverage-model"`) {
				t.Fatalf("unexpected models git create body %q", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"created":true}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/models/"+modelID+"/git":
			saw["git-update"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"branch":"main"`) {
				t.Fatalf("unexpected models git update body %q", body)
			}
			_, _ = w.Write([]byte(`{"updated":true}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/models/"+modelID+"/git":
			saw["git-delete"] = true
			_, _ = w.Write([]byte(`{"deleted":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/models/"+modelID+"/git/sync":
			saw["git-sync"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"force":true`) {
				t.Fatalf("unexpected models git sync body %q", body)
			}
			_, _ = w.Write([]byte(`{"synced":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/models/"+modelID+"/migrate":
			saw["migrate"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"targetConnectionId":"`+branchID+`"`) {
				t.Fatalf("unexpected models migrate body %q", body)
			}
			_, _ = w.Write([]byte(`{"migrated":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/models/"+modelID+"/content-validator":
			saw["content-validator-get"] = true
			q := r.URL.Query()
			if q.Get("branch_id") != "branch-alpha" || q.Get("userId") != "member-1" || q.Get("include_personal_folders") != "true" {
				t.Fatalf("unexpected models content-validator get query: %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"rules":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/models/"+modelID+"/content-validator":
			saw["content-validator-replace"] = true
			if r.URL.Query().Get("userId") != "member-1" {
				t.Fatalf("unexpected models content-validator replace query: %q", r.URL.RawQuery)
			}
			if body := mustReadAll(t, r); !strings.Contains(body, `"replace":"new_name"`) {
				t.Fatalf("unexpected models content-validator replace body %q", body)
			}
			_, _ = w.Write([]byte(`{"replaced":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/models/"+modelID+"/yaml":
			saw["yaml-get"] = true
			q := r.URL.Query()
			if q.Get("branchId") != branchID || q.Get("fileName") != "orders.yaml" || q.Get("mode") != "merged" || q.Get("includeChecksums") != "true" {
				t.Fatalf("unexpected models yaml get query: %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"fileName":"orders.yaml"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/models/"+modelID+"/yaml":
			saw["yaml-create"] = true
			if body := mustReadAll(t, r); !strings.Contains(body, `"fileName":"orders.yaml"`) {
				t.Fatalf("unexpected models yaml create body %q", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"created":true}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/models/"+modelID+"/yaml":
			saw["yaml-delete"] = true
			q := r.URL.Query()
			if q.Get("branchId") != branchID || q.Get("fileName") != "orders.yaml" || q.Get("mode") != "merged" || q.Get("commitMessage") != "remove yaml" {
				t.Fatalf("unexpected models yaml delete query: %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"deleted":true}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	rt := testRuntime(server.URL, "pat-token", "pat")

	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runModels(rt, nil)
	})
	if exit != 0 || !strings.Contains(stdout, "omni models commands:") || strings.TrimSpace(stderr) != "" {
		t.Fatalf("models usage failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	_, stderr, exit = captureRuntimeIO(t, func() int {
		return runModels(rt, []string{"nope"})
	})
	if exit != 2 || !strings.Contains(stderr, "unknown models subcommand") {
		t.Fatalf("models unknown subcommand failed: exit=%d stderr=%q", exit, stderr)
	}

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "list", args: []string{"list", "--cursor", "models-next", "--page-size", "5", "--name", "Orders"}, want: `"Orders Coverage Model"`},
		{name: "get", args: []string{"get", modelID}, want: modelID},
		{name: "create", args: []string{"create", "--file", createPath}, want: modelID},
		{name: "refresh", args: []string{"refresh", modelID}, want: `"status": "running"`},
		{name: "validate", args: []string{"validate", "--branch-id", branchID, "--limit", "9", modelID}, want: `"issues": []`},
		{name: "branch delete", args: []string{"branch", "delete", modelID, "coverage-branch"}, want: `"deleted": true`},
		{name: "branch merge", args: []string{"branch", "merge", "--file", mergePath, modelID, "coverage-branch"}, want: `"merged": true`},
		{name: "cache-reset", args: []string{"cache-reset", "--file", cacheResetPath, modelID, "default"}, want: `"reset": true`},
		{name: "topics list", args: []string{"topics", "list", "--branch-id", branchID, modelID}, want: `"success": true`},
		{name: "topics get", args: []string{"topics", "get", "--branch-id", branchID, modelID, "orders"}, want: `"name": "orders"`},
		{name: "topics update", args: []string{"topics", "update", "--branch-id", branchID, "--file", topicUpdatePath, modelID, "orders"}, want: `"updated": true`},
		{name: "topics delete", args: []string{"topics", "delete", "--branch-id", branchID, modelID, "orders"}, want: `"deleted": true`},
		{name: "views list", args: []string{"views", "list", "--branch-id", branchID, modelID}, want: `"records": []`},
		{name: "views update", args: []string{"views", "update", "--branch-id", branchID, "--file", viewUpdatePath, modelID, "default"}, want: `"updated": true`},
		{name: "views delete", args: []string{"views", "delete", "--branch-id", branchID, modelID, "default"}, want: `"deleted": true`},
		{name: "fields create", args: []string{"fields", "create", "--file", fieldCreatePath, modelID}, want: `"created": true`},
		{name: "fields update", args: []string{"fields", "update", "--branch-id", branchID, "--file", fieldUpdatePath, modelID, "default", "total_revenue"}, want: `"updated": true`},
		{name: "fields delete", args: []string{"fields", "delete", "--branch-id", branchID, "--topic-context", "orders", modelID, "default", "total_revenue"}, want: `"deleted": true`},
		{name: "git get", args: []string{"git", "get", "--include", "webhookSecret", modelID}, want: `"repository": "omni/coverage-model"`},
		{name: "git create", args: []string{"git", "create", "--file", gitCreatePath, modelID}, want: `"created": true`},
		{name: "git update", args: []string{"git", "update", "--file", gitUpdatePath, modelID}, want: `"updated": true`},
		{name: "git delete", args: []string{"git", "delete", modelID}, want: `"deleted": true`},
		{name: "git sync", args: []string{"git", "sync", "--file", gitSyncPath, modelID}, want: `"synced": true`},
		{name: "migrate", args: []string{"migrate", "--file", migratePath, modelID}, want: `"migrated": true`},
		{name: "content-validator get", args: []string{"content-validator", "get", "--branch-id", "branch-alpha", "--user-id", "member-1", "--include-personal-folders", "true", modelID}, want: `"rules": []`},
		{name: "content-validator replace", args: []string{"content-validator", "replace", "--user-id", "member-1", "--file", contentValidatorPath, modelID}, want: `"replaced": true`},
		{name: "yaml get", args: []string{"yaml", "get", "--branch-id", branchID, "--file-name", "orders.yaml", "--mode", "merged", "--include-checksums", "true", modelID}, want: `"fileName": "orders.yaml"`},
		{name: "yaml create", args: []string{"yaml", "create", "--file", yamlCreatePath, modelID}, want: `"created": true`},
		{name: "yaml delete", args: []string{"yaml", "delete", "--file-name", "orders.yaml", "--branch-id", branchID, "--mode", "merged", "--commit-message", "remove yaml", modelID}, want: `"deleted": true`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, exit := captureRuntimeIO(t, func() int {
				return runModels(rt, tc.args)
			})
			assertCommandSuccess(t, stdout, stderr, exit, tc.want)
		})
	}

	for _, key := range []string{
		"list", "get", "create", "refresh", "validate",
		"branch-delete", "branch-merge", "cache-reset",
		"topics-list", "topics-get", "topics-update", "topics-delete",
		"views-list", "views-update", "views-delete",
		"fields-create", "fields-update", "fields-delete",
		"git-get", "git-create", "git-update", "git-delete", "git-sync",
		"migrate", "content-validator-get", "content-validator-replace",
		"yaml-get", "yaml-create", "yaml-delete",
	} {
		if !saw[key] {
			t.Fatalf("expected models endpoint %q to be exercised", key)
		}
	}
}

func assertCommandSuccess(t *testing.T, stdout, stderr string, exit int, want string) {
	t.Helper()

	if exit != 0 {
		t.Fatalf("expected exit 0, got %d (stdout=%q stderr=%q)", exit, stdout, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, want) {
		t.Fatalf("expected stdout to contain %q, got %q", want, stdout)
	}
}
