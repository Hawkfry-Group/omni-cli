package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/omni-co/omni-cli/internal/client/gen"
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

func TestProbeAndDocumentWrappers(t *testing.T) {
	const (
		testUUID   = "550e8400-e29b-41d4-a716-446655440000"
		documentID = "wk_coverage_seed"
	)

	id := uuid.MustParse(testUUID)
	var sawRunQuery int
	var sawQueryWait bool
	var sawJobStatus bool
	var sawBaseProbe bool
	var sawAdminProbe bool
	var sawListDocuments bool
	var sawDocumentMutations int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer token-123" {
			t.Fatalf("missing auth header: %q", got)
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/query/run":
			sawRunQuery++
			body := mustReadBody(t, r)
			if !strings.Contains(body, `"query":{"sql":"select 1"}`) && !strings.Contains(body, `"planOnly":true`) {
				t.Fatalf("unexpected query run body %q", body)
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/query/wait":
			sawQueryWait = true
			assertQueryValue(t, r.URL, "jobIds", "job-1,job-2")
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/job-1/status":
			sawJobStatus = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/content":
			sawBaseProbe = true
			assertQueryValue(t, r.URL, "pageSize", "1")
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/scim/v2/Users":
			if r.URL.Query().Get("count") == "1" {
				sawAdminProbe = true
				w.WriteHeader(http.StatusNoContent)
				return
			}
			t.Fatalf("unexpected users query %q", r.URL.RawQuery)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/documents":
			sawListDocuments = true
			assertQueryValue(t, r.URL, "cursor", "cursor-1")
			assertQueryValue(t, r.URL, "pageSize", "5")
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/documents/"+documentID:
			sawDocumentMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/documents":
			sawDocumentMutations++
			if body := mustReadBody(t, r); !strings.Contains(body, `"name":"Coverage Seed Workbook"`) {
				t.Fatalf("unexpected documents create body %q", body)
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/documents/"+documentID:
			sawDocumentMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/documents/"+documentID:
			sawDocumentMutations++
			if body := mustReadBody(t, r); !strings.Contains(body, `"name":"Coverage Seed Workbook Renamed"`) {
				t.Fatalf("unexpected documents rename body %q", body)
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/documents/"+documentID+"/draft":
			sawDocumentMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/documents/"+documentID+"/draft":
			sawDocumentMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/documents/"+documentID+"/duplicate":
			sawDocumentMutations++
			assertQueryValue(t, r.URL, "userId", testUUID)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/documents/"+documentID+"/favorite":
			sawDocumentMutations++
			assertQueryValue(t, r.URL, "userId", testUUID)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/documents/"+documentID+"/favorite":
			sawDocumentMutations++
			assertQueryValue(t, r.URL, "userId", testUUID)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/documents/"+documentID+"/labels":
			sawDocumentMutations++
			assertQueryValue(t, r.URL, "userId", testUUID)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/documents/"+documentID+"/permissions":
			sawDocumentMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/documents/"+documentID+"/queries":
			sawDocumentMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/documents/"+documentID+"/transfer-ownership":
			sawDocumentMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/documents/"+documentID+"/move":
			sawDocumentMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/documents/"+documentID+"/access-list":
			sawDocumentMutations++
			assertQueryValue(t, r.URL, "cursor", "access-cursor")
			assertQueryValue(t, r.URL, "pageSize", "3")
			assertQueryValue(t, r.URL, "type", "user")
			assertQueryValue(t, r.URL, "accessSource", "folder")
			assertQueryValue(t, r.URL, "sortField", "updatedAt")
			assertQueryValue(t, r.URL, "sortDirection", "desc")
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/documents/"+documentID+"/permissions":
			sawDocumentMutations++
			assertQueryValue(t, r.URL, "userId", testUUID)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/documents/"+documentID+"/permissions":
			sawDocumentMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/documents/"+documentID+"/permissions":
			sawDocumentMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/documents/"+documentID+"/permissions":
			sawDocumentMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/documents/"+documentID+"/labels/finance":
			sawDocumentMutations++
			assertQueryValue(t, r.URL, "userId", testUUID)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/documents/"+documentID+"/labels/finance":
			sawDocumentMutations++
			assertQueryValue(t, r.URL, "userId", testUUID)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	cli := mustNewClient(t, server.URL)
	ctx := context.Background()

	queryBody := gen.QueryRunBody{Query: map[string]any{"sql": "select 1"}}
	if _, err := cli.RunQuery(ctx, queryBody); err != nil {
		t.Fatalf("RunQuery: %v", err)
	}
	if _, err := cli.WaitForQueryResults(ctx, []string{"job-1", "job-2"}); err != nil {
		t.Fatalf("WaitForQueryResults: %v", err)
	}
	if _, err := cli.GetJobStatus(ctx, "job-1"); err != nil {
		t.Fatalf("GetJobStatus: %v", err)
	}
	if _, err := cli.BaseProbe(ctx); err != nil {
		t.Fatalf("BaseProbe: %v", err)
	}
	if status, payload, err := cli.ValidateAuth(ctx); err != nil {
		t.Fatalf("ValidateAuth: %v", err)
	} else if status != http.StatusNoContent || !strings.Contains(strings.TrimSpace(fmt.Sprintf("%v", payload)), "map[]") {
		t.Fatalf("ValidateAuth returned unexpected result: status=%d payload=%#v", status, payload)
	}
	if _, err := cli.QueryProbe(ctx); err != nil {
		t.Fatalf("QueryProbe: %v", err)
	}
	if _, err := cli.AdminProbe(ctx); err != nil {
		t.Fatalf("AdminProbe: %v", err)
	}

	if _, err := cli.ListDocuments(ctx, "cursor-1", 5); err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if _, err := cli.GetDocument(ctx, documentID); err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if _, err := cli.CreateDocument(ctx, []byte(`{"name":"Coverage Seed Workbook"}`)); err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}
	if _, err := cli.DeleteDocument(ctx, documentID); err != nil {
		t.Fatalf("DeleteDocument: %v", err)
	}
	if _, err := cli.RenameDocument(ctx, documentID, []byte(`{"name":"Coverage Seed Workbook Renamed"}`)); err != nil {
		t.Fatalf("RenameDocument: %v", err)
	}
	if _, err := cli.CreateDocumentDraft(ctx, documentID, []byte(`{"draft":true}`)); err != nil {
		t.Fatalf("CreateDocumentDraft: %v", err)
	}
	if _, err := cli.DiscardDocumentDraft(ctx, documentID, []byte(`{}`)); err != nil {
		t.Fatalf("DiscardDocumentDraft: %v", err)
	}
	if _, err := cli.DuplicateDocument(ctx, documentID, []byte(`{"name":"Coverage Seed Workbook Copy"}`), &id); err != nil {
		t.Fatalf("DuplicateDocument: %v", err)
	}
	if _, err := cli.AddDocumentFavorite(ctx, documentID, &id); err != nil {
		t.Fatalf("AddDocumentFavorite: %v", err)
	}
	if _, err := cli.RemoveDocumentFavorite(ctx, documentID, &id); err != nil {
		t.Fatalf("RemoveDocumentFavorite: %v", err)
	}
	if _, err := cli.BulkUpdateDocumentLabels(ctx, documentID, []byte(`{"labels":["finance"]}`), &id); err != nil {
		t.Fatalf("BulkUpdateDocumentLabels: %v", err)
	}
	if _, err := cli.UpdateDocumentPermissionSettings(ctx, documentID, []byte(`{"scope":"organization"}`)); err != nil {
		t.Fatalf("UpdateDocumentPermissionSettings: %v", err)
	}
	if _, err := cli.ListDocumentQueries(ctx, documentID); err != nil {
		t.Fatalf("ListDocumentQueries: %v", err)
	}
	if _, err := cli.TransferDocumentOwnership(ctx, documentID, []byte(`{"userId":"`+testUUID+`"}`)); err != nil {
		t.Fatalf("TransferDocumentOwnership: %v", err)
	}
	if _, err := cli.MoveDocument(ctx, documentID, []byte(`{"folderPath":"finance"}`)); err != nil {
		t.Fatalf("MoveDocument: %v", err)
	}
	if _, err := cli.ListDocumentAccess(ctx, documentID, "access-cursor", 3, "user", "folder", "updatedAt", "desc"); err != nil {
		t.Fatalf("ListDocumentAccess: %v", err)
	}
	if _, err := cli.GetDocumentPermissions(ctx, documentID, id); err != nil {
		t.Fatalf("GetDocumentPermissions: %v", err)
	}
	if _, err := cli.AddDocumentPermits(ctx, documentID, []byte(`{"permits":[]}`)); err != nil {
		t.Fatalf("AddDocumentPermits: %v", err)
	}
	if _, err := cli.UpdateDocumentPermits(ctx, documentID, []byte(`{"permits":[]}`)); err != nil {
		t.Fatalf("UpdateDocumentPermits: %v", err)
	}
	if _, err := cli.RevokeDocumentPermits(ctx, documentID, []byte(`{"permits":[]}`)); err != nil {
		t.Fatalf("RevokeDocumentPermits: %v", err)
	}
	if _, err := cli.AddDocumentLabel(ctx, documentID, "finance", &id); err != nil {
		t.Fatalf("AddDocumentLabel: %v", err)
	}
	if _, err := cli.RemoveDocumentLabel(ctx, documentID, "finance", &id); err != nil {
		t.Fatalf("RemoveDocumentLabel: %v", err)
	}

	if sawRunQuery != 2 || !sawQueryWait || !sawJobStatus || !sawBaseProbe || !sawAdminProbe || !sawListDocuments || sawDocumentMutations != 21 {
		t.Fatalf("unexpected wrapper counts: runQuery=%d wait=%v job=%v base=%v admin=%v listDocs=%v docMutations=%d", sawRunQuery, sawQueryWait, sawJobStatus, sawBaseProbe, sawAdminProbe, sawListDocuments, sawDocumentMutations)
	}
}

func TestModelWrappers(t *testing.T) {
	const (
		testUUID = "550e8400-e29b-41d4-a716-446655440000"
		branchID = "11111111-1111-1111-1111-111111111111"
	)

	modelID := uuid.MustParse(testUUID)
	branchUUID := uuid.MustParse(branchID)
	includePersonal := true
	var sawModelsList bool
	var sawModelMutations int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/models" && r.URL.Query().Get("modelId") == "":
			sawModelsList = true
			assertQueryValue(t, r.URL, "cursor", "next-model")
			assertQueryValue(t, r.URL, "pageSize", "4")
			assertQueryValue(t, r.URL, "name", "orders")
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/models" && r.URL.Query().Get("modelId") == testUUID:
			sawModelMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/models":
			sawModelMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/models/"+testUUID+"/branch/coverage-branch":
			sawModelMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/models/"+testUUID+"/branch/coverage-branch/merge":
			sawModelMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/models/"+testUUID+"/cache_reset/default":
			sawModelMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/models/"+testUUID+"/field":
			sawModelMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/models/"+testUUID+"/git":
			sawModelMutations++
			assertQueryValue(t, r.URL, "include", "webhookSecret")
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/models/"+testUUID+"/git":
			sawModelMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/models/"+testUUID+"/git":
			sawModelMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/models/"+testUUID+"/git":
			sawModelMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/models/"+testUUID+"/git/sync":
			sawModelMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/models/"+testUUID+"/migrate":
			sawModelMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/models/"+testUUID+"/refresh":
			sawModelMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/models/"+testUUID+"/topic":
			sawModelMutations++
			assertQueryValue(t, r.URL, "branch_id", branchID)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/models/"+testUUID+"/topic/orders":
			sawModelMutations++
			assertQueryValue(t, r.URL, "branch_id", branchID)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/models/"+testUUID+"/topic/orders":
			sawModelMutations++
			assertQueryValue(t, r.URL, "branch_id", branchID)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/models/"+testUUID+"/topic/orders":
			sawModelMutations++
			assertQueryValue(t, r.URL, "branch_id", branchID)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/models/"+testUUID+"/validate":
			sawModelMutations++
			assertQueryValue(t, r.URL, "branchId", branchID)
			assertQueryValue(t, r.URL, "limit", "7")
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/models/"+testUUID+"/view":
			sawModelMutations++
			assertQueryValue(t, r.URL, "branch_id", branchID)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/models/"+testUUID+"/view/default":
			sawModelMutations++
			assertQueryValue(t, r.URL, "branch_id", branchID)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/models/"+testUUID+"/view/default":
			sawModelMutations++
			assertQueryValue(t, r.URL, "branch_id", branchID)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/models/"+testUUID+"/view/default/field/orders_total":
			sawModelMutations++
			assertQueryValue(t, r.URL, "branch_id", branchID)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/models/"+testUUID+"/view/default/field/orders_total":
			sawModelMutations++
			assertQueryValue(t, r.URL, "branch_id", branchID)
			assertQueryValue(t, r.URL, "topic_context", "orders")
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/models/"+testUUID+"/content-validator":
			sawModelMutations++
			assertQueryValue(t, r.URL, "branch_id", "branch-alpha")
			assertQueryValue(t, r.URL, "userId", "member-1")
			assertQueryValue(t, r.URL, "include_personal_folders", "true")
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/models/"+testUUID+"/content-validator":
			sawModelMutations++
			assertQueryValue(t, r.URL, "userId", "member-1")
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/models/"+testUUID+"/yaml":
			sawModelMutations++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/models/"+testUUID+"/yaml":
			sawModelMutations++
			assertQueryValue(t, r.URL, "branchId", branchID)
			assertQueryValue(t, r.URL, "fileName", "orders.yaml")
			assertQueryValue(t, r.URL, "mode", "merged")
			assertQueryValue(t, r.URL, "includeChecksums", "true")
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/models/"+testUUID+"/yaml":
			sawModelMutations++
			assertQueryValue(t, r.URL, "branchId", branchID)
			assertQueryValue(t, r.URL, "fileName", "orders.yaml")
			assertQueryValue(t, r.URL, "mode", "merged")
			assertQueryValue(t, r.URL, "commitMessage", "remove yaml")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	cli := mustNewClient(t, server.URL)
	ctx := context.Background()

	if _, err := cli.ListModels(ctx, "next-model", 4, "orders"); err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if _, err := cli.GetModelByID(ctx, modelID); err != nil {
		t.Fatalf("GetModelByID: %v", err)
	}
	if _, err := cli.CreateModel(ctx, []byte(`{"name":"orders"}`)); err != nil {
		t.Fatalf("CreateModel: %v", err)
	}
	if _, err := cli.DeleteModelBranch(ctx, modelID, "coverage-branch"); err != nil {
		t.Fatalf("DeleteModelBranch: %v", err)
	}
	if _, err := cli.MergeModelBranch(ctx, modelID, "coverage-branch", []byte(`{"merge":true}`)); err != nil {
		t.Fatalf("MergeModelBranch: %v", err)
	}
	if _, err := cli.ResetModelCache(ctx, modelID, "default", []byte(`{"scope":"all"}`)); err != nil {
		t.Fatalf("ResetModelCache: %v", err)
	}
	if _, err := cli.CreateModelField(ctx, modelID, []byte(`{"field":"orders_total"}`)); err != nil {
		t.Fatalf("CreateModelField: %v", err)
	}
	if _, err := cli.GetModelGit(ctx, modelID, "webhookSecret"); err != nil {
		t.Fatalf("GetModelGit: %v", err)
	}
	if _, err := cli.CreateModelGit(ctx, modelID, []byte(`{"repo":"omni/coverage"}`)); err != nil {
		t.Fatalf("CreateModelGit: %v", err)
	}
	if _, err := cli.UpdateModelGit(ctx, modelID, []byte(`{"branch":"main"}`)); err != nil {
		t.Fatalf("UpdateModelGit: %v", err)
	}
	if _, err := cli.DeleteModelGit(ctx, modelID); err != nil {
		t.Fatalf("DeleteModelGit: %v", err)
	}
	if _, err := cli.SyncModelGit(ctx, modelID, []byte(`{"force":true}`)); err != nil {
		t.Fatalf("SyncModelGit: %v", err)
	}
	if _, err := cli.MigrateModel(ctx, modelID, []byte(`{"target":"warehouse"}`)); err != nil {
		t.Fatalf("MigrateModel: %v", err)
	}
	if _, err := cli.RefreshModel(ctx, modelID); err != nil {
		t.Fatalf("RefreshModel: %v", err)
	}
	if _, err := cli.ListModelTopics(ctx, modelID, &branchUUID); err != nil {
		t.Fatalf("ListModelTopics: %v", err)
	}
	if _, err := cli.GetModelTopic(ctx, modelID, "orders", &branchUUID); err != nil {
		t.Fatalf("GetModelTopic: %v", err)
	}
	if _, err := cli.UpdateModelTopic(ctx, modelID, "orders", &branchUUID, []byte(`{"label":"Orders"}`)); err != nil {
		t.Fatalf("UpdateModelTopic: %v", err)
	}
	if _, err := cli.DeleteModelTopic(ctx, modelID, "orders", &branchUUID); err != nil {
		t.Fatalf("DeleteModelTopic: %v", err)
	}
	if _, err := cli.ValidateModel(ctx, modelID, &branchUUID, 7); err != nil {
		t.Fatalf("ValidateModel: %v", err)
	}
	if _, err := cli.ListModelViews(ctx, modelID, &branchUUID); err != nil {
		t.Fatalf("ListModelViews: %v", err)
	}
	if _, err := cli.UpdateModelView(ctx, modelID, "default", &branchUUID, []byte(`{"hidden":false}`)); err != nil {
		t.Fatalf("UpdateModelView: %v", err)
	}
	if _, err := cli.DeleteModelView(ctx, modelID, "default", &branchUUID); err != nil {
		t.Fatalf("DeleteModelView: %v", err)
	}
	if _, err := cli.UpdateModelField(ctx, modelID, "default", "orders_total", &branchUUID, []byte(`{"label":"Orders Total"}`)); err != nil {
		t.Fatalf("UpdateModelField: %v", err)
	}
	if _, err := cli.DeleteModelField(ctx, modelID, "default", "orders_total", &branchUUID, "orders"); err != nil {
		t.Fatalf("DeleteModelField: %v", err)
	}
	if _, err := cli.GetModelContentValidator(ctx, modelID, "branch-alpha", "member-1", &includePersonal); err != nil {
		t.Fatalf("GetModelContentValidator: %v", err)
	}
	if _, err := cli.ReplaceModelContentValidator(ctx, modelID, "member-1", []byte(`{"replace":"x"}`)); err != nil {
		t.Fatalf("ReplaceModelContentValidator: %v", err)
	}
	if _, err := cli.CreateModelYAML(ctx, modelID, []byte(`{"fileName":"orders.yaml"}`)); err != nil {
		t.Fatalf("CreateModelYAML: %v", err)
	}
	if _, err := cli.GetModelYAML(ctx, modelID, &branchUUID, "orders.yaml", "merged", &includePersonal); err != nil {
		t.Fatalf("GetModelYAML: %v", err)
	}
	if _, err := cli.DeleteModelYAML(ctx, modelID, "orders.yaml", &branchUUID, "merged", "remove yaml"); err != nil {
		t.Fatalf("DeleteModelYAML: %v", err)
	}

	if !sawModelsList || sawModelMutations != 28 {
		t.Fatalf("unexpected model wrapper counts: list=%v mutations=%d", sawModelsList, sawModelMutations)
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
