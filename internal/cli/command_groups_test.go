package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/omni-co/omni-cli/internal/auth"
	"github.com/omni-co/omni-cli/internal/config"
)

const testCommandUUID = "550e8400-e29b-41d4-a716-446655440000"

func TestRunAdminUsersAndGroupsList(t *testing.T) {
	var sawUsers bool
	var sawGroups bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/scim/v2/Users":
			sawUsers = true
			if r.URL.Query().Get("count") != "5" || r.URL.Query().Get("startIndex") != "2" {
				t.Fatalf("unexpected users query: %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"Resources":[{"id":"` + testCommandUUID + `","displayName":"Jamie Fry","userName":"jamie@example.com"}],"itemsPerPage":1,"schemas":["urn:ietf:params:scim:api:messages:2.0:ListResponse"],"startIndex":1,"totalResults":1}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/scim/v2/Groups":
			sawGroups = true
			if r.URL.Query().Get("count") != "3" || r.URL.Query().Get("startIndex") != "4" {
				t.Fatalf("unexpected groups query: %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"Resources":[{"id":"` + testCommandUUID + `","displayName":"Analysts"}],"itemsPerPage":1,"schemas":["urn:ietf:params:scim:api:messages:2.0:ListResponse"],"startIndex":1,"totalResults":1}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	orgRT := testRuntime(server.URL, "org-token", "org")
	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runAdmin(orgRT, []string{"users", "list", "--count", "5", "--start-index", "2"})
	})
	if exit != 0 || !strings.Contains(stdout, `"Resources"`) || strings.TrimSpace(stderr) != "" {
		t.Fatalf("admin users list failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	stdout, stderr, exit = captureRuntimeIO(t, func() int {
		return runAdmin(orgRT, []string{"groups", "list", "--count", "3", "--start-index", "4"})
	})
	if exit != 0 || !strings.Contains(stdout, `"Analysts"`) || strings.TrimSpace(stderr) != "" {
		t.Fatalf("admin groups list failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	if !sawUsers || !sawGroups {
		t.Fatalf("expected both admin endpoints to be exercised, got users=%v groups=%v", sawUsers, sawGroups)
	}

	plainRT := testRuntime(server.URL, "pat-token", "pat")
	_, stderr, exit = captureRuntimeIO(t, func() int {
		return runAdmin(plainRT, []string{"users", "list"})
	})
	if exit != 1 || !strings.Contains(stderr, "admin commands require an org API key") {
		t.Fatalf("expected org-key guard, got exit=%d stderr=%q", exit, stderr)
	}
}

func TestRunAgenticCommands(t *testing.T) {
	tmp := t.TempDir()
	payloadPath := writeTempJSON(t, tmp, "agentic.json", `{"prompt":"show revenue","modelId":"`+testCommandUUID+`"}`)

	var sawSubmit bool
	var sawStatus bool
	var sawCancel bool
	var sawResult bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/agentic/jobs":
			sawSubmit = true
			body := mustReadAll(t, r)
			if !strings.Contains(body, `"prompt":"show revenue"`) {
				t.Fatalf("unexpected submit body %q", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"jobId":"` + testCommandUUID + `","conversationId":"` + testCommandUUID + `"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/agentic/jobs/"+testCommandUUID:
			sawStatus = true
			_, _ = w.Write([]byte(`{"id":"` + testCommandUUID + `","state":"COMPLETE","conversationId":"` + testCommandUUID + `","createdAt":"2025-01-01T00:00:00Z","organizationId":"` + testCommandUUID + `","prompt":"show revenue","updatedAt":"2025-01-01T00:00:00Z","userId":"` + testCommandUUID + `","branchId":null,"modelId":null,"topicName":null}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/agentic/jobs/"+testCommandUUID+"/cancel":
			sawCancel = true
			_, _ = w.Write([]byte(`{"jobId":"` + testCommandUUID + `","state":"CANCELLED"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/agentic/jobs/"+testCommandUUID+"/result":
			sawResult = true
			_, _ = w.Write([]byte(`{"id":"` + testCommandUUID + `","state":"COMPLETE","conversationId":"` + testCommandUUID + `","createdAt":"2025-01-01T00:00:00Z","organizationId":"` + testCommandUUID + `","prompt":"show revenue","resultSummary":"done","updatedAt":"2025-01-01T00:00:00Z","userId":"` + testCommandUUID + `","branchId":null,"modelId":null,"topicName":null}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	rt := testRuntime(server.URL, "pat-token", "pat")

	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runAgentic(rt, []string{"submit", "--file", payloadPath})
	})
	if exit != 0 || !strings.Contains(stdout, `"jobId": "`) || strings.TrimSpace(stderr) != "" {
		t.Fatalf("agentic submit failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	for _, sub := range []string{"status", "cancel", "result"} {
		stdout, stderr, exit = captureRuntimeIO(t, func() int {
			return runAgentic(rt, []string{sub, testCommandUUID})
		})
		if exit != 0 || !strings.Contains(stdout, testCommandUUID) || strings.TrimSpace(stderr) != "" {
			t.Fatalf("agentic %s failed: exit=%d stdout=%q stderr=%q", sub, exit, stdout, stderr)
		}
	}

	if !sawSubmit || !sawStatus || !sawCancel || !sawResult {
		t.Fatalf("expected all agentic endpoints, got submit=%v status=%v cancel=%v result=%v", sawSubmit, sawStatus, sawCancel, sawResult)
	}
}

func TestRunAICommands(t *testing.T) {
	tmp := t.TempDir()
	contextPath := writeTempJSON(t, tmp, "context.json", `{"fields":["orders.id"]}`)

	var sawGenerate bool
	var sawWorkbook bool
	var sawPickTopic bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/ai/generate-query":
			body := mustReadAll(t, r)
			isWorkbook := strings.Contains(body, `"workbookUrl":true`)
			if isWorkbook {
				sawWorkbook = true
			} else {
				sawGenerate = true
			}
			if !strings.Contains(body, `"modelId":"`+testCommandUUID+`"`) || !strings.Contains(body, `"prompt":"show revenue"`) {
				t.Fatalf("unexpected ai generate body %q", body)
			}
			if !isWorkbook && !strings.Contains(body, `"contextQuery":{"fields":["orders.id"]}`) {
				t.Fatalf("expected context query body, got %q", body)
			}
			_, _ = w.Write([]byte(`{"topic":"orders","query":{"ok":true},"error":null}`))
		case "/api/v1/ai/pick-topic":
			sawPickTopic = true
			body := mustReadAll(t, r)
			if !strings.Contains(body, `"potentialTopicNames":["orders","customers"]`) {
				t.Fatalf("unexpected ai pick-topic body %q", body)
			}
			_, _ = w.Write([]byte(`{"topicId":"orders"}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	rt := testRuntime(server.URL, "pat-token", "pat")

	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runAI(rt, []string{"generate-query", "--model-id", testCommandUUID, "--prompt", "show revenue", "--current-topic-name", "orders", "--branch-id", testCommandUUID, "--structured", "--context-query-file", contextPath})
	})
	if exit != 0 || !strings.Contains(stdout, `"topic": "orders"`) || strings.TrimSpace(stderr) != "" {
		t.Fatalf("ai generate-query failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	stdout, stderr, exit = captureRuntimeIO(t, func() int {
		return runAI(rt, []string{"workbook", "--model-id", testCommandUUID, "--prompt", "show revenue"})
	})
	if exit != 0 || !strings.Contains(stdout, `"topic": "orders"`) || strings.TrimSpace(stderr) != "" {
		t.Fatalf("ai workbook failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	stdout, stderr, exit = captureRuntimeIO(t, func() int {
		return runAI(rt, []string{"pick-topic", "--model-id", testCommandUUID, "--prompt", "show revenue", "--potential-topic-names", "orders, customers"})
	})
	if exit != 0 || !strings.Contains(stdout, `"topicId": "orders"`) || strings.TrimSpace(stderr) != "" {
		t.Fatalf("ai pick-topic failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}

	if !sawGenerate || !sawWorkbook || !sawPickTopic {
		t.Fatalf("expected all ai endpoints, got generate=%v workbook=%v pickTopic=%v", sawGenerate, sawWorkbook, sawPickTopic)
	}
}

func TestRunDashboardsAndEmbedCommands(t *testing.T) {
	tmp := t.TempDir()
	downloadPath := writeTempJSON(t, tmp, "download.json", `{"format":"pdf"}`)
	filtersPath := writeTempJSON(t, tmp, "filters.json", `{"filters":[]}`)
	embedPath := writeTempJSON(t, tmp, "embed.json", `{"userId":"`+testCommandUUID+`"}`)

	const dashboardID = "wk_abc123"
	const jobID = "job-123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/dashboards/"+dashboardID+"/download":
			if r.URL.Query().Get("userId") != "member-1" {
				t.Fatalf("unexpected download query %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"jobId":"` + jobID + `"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/dashboards/"+dashboardID+"/download/"+jobID+"/status":
			if r.URL.Query().Get("userId") != "member-1" {
				t.Fatalf("unexpected dashboard query %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"status":"ready"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/dashboards/"+dashboardID+"/download/"+jobID:
			if r.URL.Query().Get("userId") != "member-1" {
				t.Fatalf("unexpected dashboard file query %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"url":"https://example.com/download.pdf"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/dashboards/"+dashboardID+"/filters":
			if r.URL.Query().Get("userId") != "member-1" {
				t.Fatalf("unexpected filters get query %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"filters":[]}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/dashboards/"+dashboardID+"/filters":
			if r.URL.Query().Get("userId") != "member-1" {
				t.Fatalf("unexpected filters update query %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"updated":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/embed/sso/generate-session":
			body := mustReadAll(t, r)
			if !strings.Contains(body, `"userId":"`+testCommandUUID+`"`) {
				t.Fatalf("unexpected embed body %q", body)
			}
			_, _ = w.Write([]byte(`{"url":"https://embed.example.com/session"}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	rt := testRuntime(server.URL, "org-token", "org")

	for _, args := range [][]string{
		{"download", "--file", downloadPath, "--user-id", "member-1", dashboardID},
		{"download-status", "--user-id", "member-1", dashboardID, jobID},
		{"download-file", "--user-id", "member-1", dashboardID, jobID},
		{"filters", "get", "--user-id", "member-1", dashboardID},
		{"filters", "update", "--file", filtersPath, "--user-id", "member-1", dashboardID},
	} {
		stdout, stderr, exit := captureRuntimeIO(t, func() int {
			return runDashboards(rt, args)
		})
		if exit != 0 || strings.TrimSpace(stderr) != "" {
			t.Fatalf("dashboards command %v failed: exit=%d stdout=%q stderr=%q", args, exit, stdout, stderr)
		}
	}

	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runEmbed(rt, []string{"sso", "generate-session", "--file", embedPath})
	})
	if exit != 0 || !strings.Contains(stdout, `embed.example.com/session`) || strings.TrimSpace(stderr) != "" {
		t.Fatalf("embed sso generate-session failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}
}

func TestRunLabelsCommands(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/labels":
			_, _ = w.Write([]byte(`{"records":[{"name":"finance"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/labels/finance":
			_, _ = w.Write([]byte(`{"name":"finance"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/labels":
			body := mustReadAll(t, r)
			if r.URL.Query().Get("userId") != "member-1" || !strings.Contains(body, `"homepage":true`) {
				t.Fatalf("unexpected label create request query=%q body=%q", r.URL.RawQuery, body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"name":"finance"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/labels/finance":
			body := mustReadAll(t, r)
			if !strings.Contains(body, `"name":"finance-updated"`) {
				t.Fatalf("unexpected label update body %q", body)
			}
			_, _ = w.Write([]byte(`{"name":"finance-updated"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/labels/finance":
			if r.URL.Query().Get("userId") != "member-1" {
				t.Fatalf("unexpected label delete query %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"deleted":true}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	rt := testRuntime(server.URL, "org-token", "org")

	for _, args := range [][]string{
		{"list"},
		{"get", "finance"},
		{"create", "--homepage", "true", "--user-id", "member-1", "finance"},
		{"update", "--new-name", "finance-updated", "finance"},
		{"delete", "--user-id", "member-1", "finance"},
	} {
		stdout, stderr, exit := captureRuntimeIO(t, func() int {
			return runLabels(rt, args)
		})
		if exit != 0 || strings.TrimSpace(stderr) != "" {
			t.Fatalf("labels command %v failed: exit=%d stdout=%q stderr=%q", args, exit, stdout, stderr)
		}
	}

	_, stderr, exit := captureRuntimeIO(t, func() int {
		return runLabels(rt, []string{"update", "finance"})
	})
	if exit != 2 || !strings.Contains(stderr, "no changes provided") {
		t.Fatalf("expected no-changes usage error, got exit=%d stderr=%q", exit, stderr)
	}
}

func TestRunSchedulesCommands(t *testing.T) {
	tmp := t.TempDir()
	createPath := writeTempJSON(t, tmp, "schedule-create.json", `{"name":"Daily"}`)
	recipientsPath := writeTempJSON(t, tmp, "recipients.json", `{"userIds":["`+testCommandUUID+`"]}`)
	ownershipPath := writeTempJSON(t, tmp, "ownership.json", `{"userId":"`+testCommandUUID+`"}`)

	var transferCalls int
	var sawTransferFile bool
	var sawTransferFlag bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/schedules":
			if r.URL.Query().Get("cursor") != "abc" || r.URL.Query().Get("pageSize") != "10" || r.URL.Query().Get("q") != "daily" {
				t.Fatalf("unexpected schedules list query %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"records":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/schedules":
			if r.URL.Query().Get("userId") != testCommandUUID {
				t.Fatalf("unexpected schedules create query %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"id":"` + testCommandUUID + `"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/schedules/"+testCommandUUID:
			_, _ = w.Write([]byte(`{"id":"` + testCommandUUID + `"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/schedules/"+testCommandUUID:
			_, _ = w.Write([]byte(`{"deleted":true}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/schedules/"+testCommandUUID:
			_, _ = w.Write([]byte(`{"updated":true}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/schedules/"+testCommandUUID+"/pause":
			_, _ = w.Write([]byte(`{"status":"paused"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/schedules/"+testCommandUUID+"/resume":
			_, _ = w.Write([]byte(`{"status":"resumed"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/schedules/"+testCommandUUID+"/trigger":
			_, _ = w.Write([]byte(`{"status":"triggered"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/schedules/"+testCommandUUID+"/recipients":
			_, _ = w.Write([]byte(`{"recipients":[]}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/schedules/"+testCommandUUID+"/add-recipients":
			_, _ = w.Write([]byte(`{"added":true}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/schedules/"+testCommandUUID+"/remove-recipients":
			_, _ = w.Write([]byte(`{"removed":true}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/schedules/"+testCommandUUID+"/transfer-ownership":
			body := mustReadAll(t, r)
			if body != `{"userId":"`+testCommandUUID+`"}` {
				t.Fatalf("unexpected transfer ownership body %q", body)
			}
			transferCalls++
			if transferCalls == 1 {
				sawTransferFlag = true
			} else {
				sawTransferFile = true
			}
			_, _ = w.Write([]byte(`{"transferred":true}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	rt := testRuntime(server.URL, "org-token", "org")

	cases := []struct {
		args []string
	}{
		{[]string{"list", "--cursor", "abc", "--page-size", "10", "--q", "daily"}},
		{[]string{"create", "--file", createPath, "--user-id", testCommandUUID}},
		{[]string{"get", testCommandUUID}},
		{[]string{"update", testCommandUUID}},
		{[]string{"delete", testCommandUUID}},
		{[]string{"pause", testCommandUUID}},
		{[]string{"resume", testCommandUUID}},
		{[]string{"trigger", testCommandUUID}},
		{[]string{"recipients", "get", testCommandUUID}},
		{[]string{"recipients", "add", "--file", recipientsPath, testCommandUUID}},
		{[]string{"recipients", "remove", "--file", recipientsPath, testCommandUUID}},
		{[]string{"transfer-ownership", "--user-id", testCommandUUID, testCommandUUID}},
		{[]string{"transfer-ownership", "--file", ownershipPath, testCommandUUID}},
	}

	for i, tc := range cases {
		stdout, stderr, exit := captureRuntimeIO(t, func() int {
			return runSchedules(rt, tc.args)
		})
		if exit != 0 || strings.TrimSpace(stderr) != "" {
			t.Fatalf("schedules command #%d %v failed: exit=%d stdout=%q stderr=%q", i, tc.args, exit, stdout, stderr)
		}
	}

	if !sawTransferFlag || !sawTransferFile {
		t.Fatalf("expected transfer ownership to exercise both flag and file paths, got flag=%v file=%v", sawTransferFlag, sawTransferFile)
	}
}

func TestRunQueryRunWaitsForResults(t *testing.T) {
	tmp := t.TempDir()
	queryPath := writeTempJSON(t, tmp, "query.json", `{"query":{"fields":["orders.id"]}}`)
	var sawRun bool
	var sawWait bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/query/run":
			sawRun = true
			body := mustReadAll(t, r)
			if !strings.Contains(body, `"resultType":"json"`) {
				t.Fatalf("expected query result type in body, got %q", body)
			}
			_, _ = w.Write([]byte(`{"jobIds":["` + testCommandUUID + `"]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/query/wait":
			sawWait = true
			if r.URL.Query().Get("jobIds") != testCommandUUID {
				t.Fatalf("unexpected wait query %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"results":[{"ok":true}]}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	rt := testRuntime(server.URL, "pat-token", "pat")
	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return runQuery(rt, []string{"run", "--file", queryPath, "--result-type", "json", "--wait"})
	})
	if exit != 0 || !strings.Contains(stdout, `"results"`) || strings.TrimSpace(stderr) != "" {
		t.Fatalf("query run failed: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}
	if !sawRun || !sawWait {
		t.Fatalf("expected run and wait endpoints, got run=%v wait=%v", sawRun, sawWait)
	}
}

func testRuntime(baseURL, token, tokenType string) *runtime {
	return &runtime{
		JSON: true,
		Resolved: &auth.Resolved{
			ProfileName: "default",
			Profile: config.Profile{
				BaseURL:   baseURL,
				Token:     token,
				TokenType: tokenType,
			},
		},
	}
}

func writeTempJSON(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write temp json %s: %v", name, err)
	}
	return path
}
