package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestAdminAndLabelWrappers(t *testing.T) {
	const testUUID = "550e8400-e29b-41d4-a716-446655440000"
	var sawUsers bool
	var sawGroups bool
	var sawListLabels bool
	var sawGetLabel bool
	var sawCreateLabel bool
	var sawUpdateLabel bool
	var sawDeleteLabel bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Authorization") != "Bearer token-123" {
			t.Fatalf("missing auth header: %q", r.Header.Get("Authorization"))
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/scim/v2/Users":
			sawUsers = true
			assertQueryValue(t, r.URL, "count", "5")
			assertQueryValue(t, r.URL, "startIndex", "2")
			assertQueryValue(t, r.URL, "filter", `userName eq "jamie@example.com"`)
			_, _ = w.Write([]byte(`{"Resources":[],"itemsPerPage":0,"schemas":["urn:ietf:params:scim:api:messages:2.0:ListResponse"],"startIndex":1,"totalResults":0}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/scim/v2/Groups":
			sawGroups = true
			assertQueryValue(t, r.URL, "count", "7")
			assertQueryValue(t, r.URL, "startIndex", "3")
			_, _ = w.Write([]byte(`{"Resources":[],"itemsPerPage":0,"schemas":["urn:ietf:params:scim:api:messages:2.0:ListResponse"],"startIndex":1,"totalResults":0}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/labels":
			sawListLabels = true
			_, _ = w.Write([]byte(`{"records":[]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/labels/finance":
			sawGetLabel = true
			_, _ = w.Write([]byte(`{"name":"finance"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/labels":
			sawCreateLabel = true
			assertQueryValue(t, r.URL, "userId", "member-1")
			body := mustReadBody(t, r)
			if !strings.Contains(body, `"name":"finance"`) || !strings.Contains(body, `"homepage":true`) {
				t.Fatalf("unexpected label create body %q", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"name":"finance"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/labels/finance":
			sawUpdateLabel = true
			assertQueryValue(t, r.URL, "userId", "member-1")
			body := mustReadBody(t, r)
			if !strings.Contains(body, `"name":"finance-2"`) || !strings.Contains(body, `"verified":true`) {
				t.Fatalf("unexpected label update body %q", body)
			}
			_, _ = w.Write([]byte(`{"name":"finance-2"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/labels/finance":
			sawDeleteLabel = true
			assertQueryValue(t, r.URL, "userId", "member-1")
			_, _ = w.Write([]byte(`{"deleted":true}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	cli := mustNewClient(t, server.URL)
	ctx := context.Background()

	if _, err := cli.ListSCIMUsers(ctx, 5, 2, `userName eq "jamie@example.com"`); err != nil {
		t.Fatalf("ListSCIMUsers: %v", err)
	}
	if _, err := cli.ListSCIMGroups(ctx, 7, 3); err != nil {
		t.Fatalf("ListSCIMGroups: %v", err)
	}
	if _, err := cli.ListLabels(ctx); err != nil {
		t.Fatalf("ListLabels: %v", err)
	}
	if _, err := cli.GetLabel(ctx, "finance"); err != nil {
		t.Fatalf("GetLabel: %v", err)
	}
	home := true
	if _, err := cli.CreateLabel(ctx, "finance", &home, nil, "member-1"); err != nil {
		t.Fatalf("CreateLabel: %v", err)
	}
	newName := "finance-2"
	verified := true
	if _, err := cli.UpdateLabel(ctx, "finance", &newName, nil, &verified, "member-1"); err != nil {
		t.Fatalf("UpdateLabel: %v", err)
	}
	if _, err := cli.DeleteLabel(ctx, "finance", "member-1"); err != nil {
		t.Fatalf("DeleteLabel: %v", err)
	}

	if !sawUsers || !sawGroups || !sawListLabels || !sawGetLabel || !sawCreateLabel || !sawUpdateLabel || !sawDeleteLabel {
		t.Fatalf("missing wrapper calls: users=%v groups=%v list=%v get=%v create=%v update=%v delete=%v", sawUsers, sawGroups, sawListLabels, sawGetLabel, sawCreateLabel, sawUpdateLabel, sawDeleteLabel)
	}
	_ = testUUID
}

func TestScheduleAIAndDashboardWrappers(t *testing.T) {
	const testUUID = "550e8400-e29b-41d4-a716-446655440000"
	id := uuid.MustParse(testUUID)
	var sawListSchedules bool
	var sawCreateSchedule bool
	var sawScheduleLifecycle int
	var sawRecipients int
	var sawAI int
	var sawAgentic int
	var sawDashboards int
	var sawEmbed bool
	var sawUserAttributes bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/schedules":
			sawListSchedules = true
			assertQueryValue(t, r.URL, "cursor", "abc")
			assertQueryValue(t, r.URL, "pageSize", "10")
			assertQueryValue(t, r.URL, "q", "daily")
			_, _ = w.Write([]byte(`{"records":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/schedules":
			sawCreateSchedule = true
			assertQueryValue(t, r.URL, "userId", testUUID)
			_, _ = w.Write([]byte(`{"id":"` + testUUID + `"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/schedules/"+testUUID:
			sawScheduleLifecycle++
			_, _ = w.Write([]byte(`{"id":"` + testUUID + `"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/schedules/"+testUUID:
			sawScheduleLifecycle++
			_, _ = w.Write([]byte(`{"deleted":true}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/schedules/"+testUUID:
			sawScheduleLifecycle++
			_, _ = w.Write([]byte(`{"updated":true}`))
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/pause"):
			sawScheduleLifecycle++
			_, _ = w.Write([]byte(`{"status":"paused"}`))
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/resume"):
			sawScheduleLifecycle++
			_, _ = w.Write([]byte(`{"status":"resumed"}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/trigger"):
			sawScheduleLifecycle++
			_, _ = w.Write([]byte(`{"status":"triggered"}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/recipients"):
			sawRecipients++
			_, _ = w.Write([]byte(`{"recipients":[]}`))
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/add-recipients"):
			sawRecipients++
			_, _ = w.Write([]byte(`{"added":true}`))
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/remove-recipients"):
			sawRecipients++
			_, _ = w.Write([]byte(`{"removed":true}`))
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/transfer-ownership"):
			sawRecipients++
			_, _ = w.Write([]byte(`{"transferred":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/ai/generate-query":
			sawAI++
			_, _ = w.Write([]byte(`{"topic":"orders","query":{"ok":true},"error":null}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/ai/pick-topic":
			sawAI++
			_, _ = w.Write([]byte(`{"topicId":"orders"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/agentic/jobs":
			sawAgentic++
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"jobId":"` + testUUID + `","conversationId":"` + testUUID + `"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/agentic/jobs/"+testUUID:
			sawAgentic++
			_, _ = w.Write([]byte(`{"id":"` + testUUID + `","state":"COMPLETE","conversationId":"` + testUUID + `","createdAt":"2025-01-01T00:00:00Z","organizationId":"` + testUUID + `","prompt":"show revenue","updatedAt":"2025-01-01T00:00:00Z","userId":"` + testUUID + `","branchId":null,"modelId":null,"topicName":null}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/agentic/jobs/"+testUUID+"/cancel":
			sawAgentic++
			_, _ = w.Write([]byte(`{"jobId":"` + testUUID + `","state":"CANCELLED"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/agentic/jobs/"+testUUID+"/result":
			sawAgentic++
			_, _ = w.Write([]byte(`{"id":"` + testUUID + `","state":"COMPLETE","conversationId":"` + testUUID + `","createdAt":"2025-01-01T00:00:00Z","organizationId":"` + testUUID + `","prompt":"show revenue","resultSummary":"done","updatedAt":"2025-01-01T00:00:00Z","userId":"` + testUUID + `","branchId":null,"modelId":null,"topicName":null}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/dashboards/wk_abc/download":
			sawDashboards++
			assertQueryValue(t, r.URL, "userId", "member-1")
			_, _ = w.Write([]byte(`{"jobId":"job-123"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/dashboards/wk_abc/download/job-123/status":
			sawDashboards++
			_, _ = w.Write([]byte(`{"status":"ready"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/dashboards/wk_abc/download/job-123":
			sawDashboards++
			_, _ = w.Write([]byte(`{"url":"https://example.com/out.pdf"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/dashboards/wk_abc/filters":
			sawDashboards++
			_, _ = w.Write([]byte(`{"filters":[]}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/dashboards/wk_abc/filters":
			sawDashboards++
			_, _ = w.Write([]byte(`{"updated":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/embed/sso/generate-session":
			sawEmbed = true
			_, _ = w.Write([]byte(`{"url":"https://embed.example.com/session"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/user-attributes":
			sawUserAttributes = true
			_, _ = w.Write([]byte(`{"records":[]}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	cli := mustNewClient(t, server.URL)
	ctx := context.Background()

	if _, err := cli.ListSchedules(ctx, "abc", 10, "daily"); err != nil {
		t.Fatalf("ListSchedules: %v", err)
	}
	if _, err := cli.CreateSchedule(ctx, &id, []byte(`{"name":"Daily"}`)); err != nil {
		t.Fatalf("CreateSchedule: %v", err)
	}
	if _, err := cli.GetSchedule(ctx, id); err != nil {
		t.Fatalf("GetSchedule: %v", err)
	}
	if _, err := cli.DeleteSchedule(ctx, id); err != nil {
		t.Fatalf("DeleteSchedule: %v", err)
	}
	if _, err := cli.UpdateSchedule(ctx, id); err != nil {
		t.Fatalf("UpdateSchedule: %v", err)
	}
	if _, err := cli.PauseSchedule(ctx, id); err != nil {
		t.Fatalf("PauseSchedule: %v", err)
	}
	if _, err := cli.ResumeSchedule(ctx, id); err != nil {
		t.Fatalf("ResumeSchedule: %v", err)
	}
	if _, err := cli.TriggerSchedule(ctx, id); err != nil {
		t.Fatalf("TriggerSchedule: %v", err)
	}
	if _, err := cli.GetScheduleRecipients(ctx, id); err != nil {
		t.Fatalf("GetScheduleRecipients: %v", err)
	}
	if _, err := cli.AddScheduleRecipients(ctx, id, []byte(`{"userIds":["`+testUUID+`"]}`)); err != nil {
		t.Fatalf("AddScheduleRecipients: %v", err)
	}
	if _, err := cli.RemoveScheduleRecipients(ctx, id, []byte(`{"userIds":["`+testUUID+`"]}`)); err != nil {
		t.Fatalf("RemoveScheduleRecipients: %v", err)
	}
	if _, err := cli.TransferScheduleOwnership(ctx, id, []byte(`{"userId":"`+testUUID+`"}`)); err != nil {
		t.Fatalf("TransferScheduleOwnership: %v", err)
	}
	if _, err := cli.AIGenerateQuery(ctx, []byte(`{"modelId":"`+testUUID+`","prompt":"show revenue"}`)); err != nil {
		t.Fatalf("AIGenerateQuery: %v", err)
	}
	if _, err := cli.AIPickTopic(ctx, []byte(`{"modelId":"`+testUUID+`","prompt":"show revenue"}`)); err != nil {
		t.Fatalf("AIPickTopic: %v", err)
	}
	if _, err := cli.SubmitAgenticJob(ctx, []byte(`{"modelId":"`+testUUID+`","prompt":"show revenue"}`)); err != nil {
		t.Fatalf("SubmitAgenticJob: %v", err)
	}
	if _, err := cli.GetAgenticJobStatus(ctx, id); err != nil {
		t.Fatalf("GetAgenticJobStatus: %v", err)
	}
	if _, err := cli.CancelAgenticJob(ctx, id); err != nil {
		t.Fatalf("CancelAgenticJob: %v", err)
	}
	if _, err := cli.GetAgenticJobResult(ctx, id); err != nil {
		t.Fatalf("GetAgenticJobResult: %v", err)
	}
	if _, err := cli.DashboardDownload(ctx, "wk_abc", "member-1", []byte(`{"format":"pdf"}`)); err != nil {
		t.Fatalf("DashboardDownload: %v", err)
	}
	if _, err := cli.DashboardDownloadStatus(ctx, "wk_abc", "job-123", "member-1"); err != nil {
		t.Fatalf("DashboardDownloadStatus: %v", err)
	}
	if _, err := cli.DashboardDownloadFile(ctx, "wk_abc", "job-123", "member-1"); err != nil {
		t.Fatalf("DashboardDownloadFile: %v", err)
	}
	if _, err := cli.GetDashboardFilters(ctx, "wk_abc", "member-1"); err != nil {
		t.Fatalf("GetDashboardFilters: %v", err)
	}
	if _, err := cli.UpdateDashboardFilters(ctx, "wk_abc", "member-1", []byte(`{"filters":[]}`)); err != nil {
		t.Fatalf("UpdateDashboardFilters: %v", err)
	}
	if _, err := cli.GenerateEmbedSSOSession(ctx, []byte(`{"userId":"`+testUUID+`"}`)); err != nil {
		t.Fatalf("GenerateEmbedSSOSession: %v", err)
	}
	if _, err := cli.ListUserAttributes(ctx); err != nil {
		t.Fatalf("ListUserAttributes: %v", err)
	}

	if !sawListSchedules || !sawCreateSchedule || sawScheduleLifecycle != 6 || sawRecipients != 4 || sawAI != 2 || sawAgentic != 4 || sawDashboards != 5 || !sawEmbed || !sawUserAttributes {
		t.Fatalf("unexpected wrapper counts: schedules=%v create=%v lifecycle=%d recipients=%d ai=%d agentic=%d dashboards=%d embed=%v attrs=%v", sawListSchedules, sawCreateSchedule, sawScheduleLifecycle, sawRecipients, sawAI, sawAgentic, sawDashboards, sawEmbed, sawUserAttributes)
	}
}

func mustNewClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	cli, err := New(baseURL, "token-123")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return cli
}

func mustReadBody(t *testing.T, r *http.Request) string {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(body)
}

func assertQueryValue(t *testing.T, u *url.URL, key, want string) {
	t.Helper()
	if got := u.Query().Get(key); got != want {
		t.Fatalf("expected query %s=%q, got %q", key, want, got)
	}
}
