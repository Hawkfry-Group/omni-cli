package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omni-co/omni-cli/internal/client/gen"
)

type Client struct {
	api *gen.ClientWithResponses
}

func New(baseURL, token string) (*Client, error) {
	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: newRetryTransport(http.DefaultTransport),
	}
	api, err := gen.NewClientWithResponses(
		normalizeBaseURL(baseURL),
		gen.WithHTTPClient(httpClient),
		gen.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Accept", "application/json")
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}
	return &Client{api: api}, nil
}

func normalizeBaseURL(base string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	base = strings.TrimSuffix(base, "/api/v1")
	base = strings.TrimSuffix(base, "/api")
	return base
}

func (c *Client) RunQuery(ctx context.Context, body gen.QueryRunBody) (*gen.QueryRunResp, error) {
	return c.api.QueryRunWithResponse(ctx, body)
}

func (c *Client) WaitForQueryResults(ctx context.Context, jobIDs []string) (*gen.QueryWaitResp, error) {
	params := &gen.QueryWaitParams{JobIds: strings.Join(jobIDs, ",")}
	return c.api.QueryWaitWithResponse(ctx, params)
}

func (c *Client) GetJobStatus(ctx context.Context, jobID string) (*gen.JobsGetStatusResp, error) {
	return c.api.JobsGetStatusWithResponse(ctx, jobID)
}

func (c *Client) ValidateAuth(ctx context.Context) (int, any, error) {
	resp, err := c.BaseProbe(ctx)
	if err != nil {
		return 0, nil, err
	}
	if resp.JSON200 != nil {
		return resp.StatusCode(), resp.JSON200, nil
	}
	return resp.StatusCode(), ParseBody(resp.Body), nil
}

func (c *Client) BaseProbe(ctx context.Context) (*gen.ContentListResp, error) {
	pageSize := float32(1)
	return c.api.ContentListWithResponse(ctx, &gen.ContentListParams{PageSize: &pageSize})
}

func (c *Client) QueryProbe(ctx context.Context) (*gen.QueryRunResp, error) {
	planOnly := true
	resultType := gen.QueryRunBodyResultTypeJson
	return c.api.QueryRunWithResponse(ctx, gen.QueryRunBody{
		PlanOnly:   &planOnly,
		Query:      map[string]any{},
		ResultType: &resultType,
	})
}

func (c *Client) AdminProbe(ctx context.Context) (*gen.ScimUsersListResp, error) {
	count := "1"
	return c.api.ScimUsersListWithResponse(ctx, &gen.ScimUsersListParams{
		Count: &count,
	})
}

func (c *Client) ListConnections(ctx context.Context, name string) (*gen.ConnectionsListResp, error) {
	params := &gen.ConnectionsListParams{}
	if strings.TrimSpace(name) != "" {
		n := strings.TrimSpace(name)
		params.Name = &n
	}
	return c.api.ConnectionsListWithResponse(ctx, params)
}

func (c *Client) CreateConnection(ctx context.Context, body []byte) (*gen.ConnectionsCreateResp, error) {
	return c.api.ConnectionsCreateWithBodyWithResponse(ctx, "application/json", bytes.NewReader(body))
}

func (c *Client) UpdateConnection(ctx context.Context, id uuid.UUID, body []byte) (*gen.ConnectionsUpdateResp, error) {
	return c.api.ConnectionsUpdateWithBodyWithResponse(ctx, id, "application/json", bytes.NewReader(body))
}

func (c *Client) GetConnectionDBT(ctx context.Context, connectionID uuid.UUID) (*gen.ConnectionsDbtGetResp, error) {
	return c.api.ConnectionsDbtGetWithResponse(ctx, connectionID)
}

func (c *Client) UpdateConnectionDBT(ctx context.Context, connectionID uuid.UUID, body []byte) (*gen.ConnectionsDbtUpdateResp, error) {
	return c.api.ConnectionsDbtUpdateWithBodyWithResponse(ctx, connectionID, "application/json", bytes.NewReader(body))
}

func (c *Client) DeleteConnectionDBT(ctx context.Context, connectionID uuid.UUID) (*gen.ConnectionsDbtDeleteResp, error) {
	return c.api.ConnectionsDbtDeleteWithResponse(ctx, connectionID)
}

func (c *Client) ListConnectionSchedules(ctx context.Context, connectionID uuid.UUID) (*gen.ConnectionsSchedulesListResp, error) {
	return c.api.ConnectionsSchedulesListWithResponse(ctx, connectionID)
}

func (c *Client) CreateConnectionSchedule(ctx context.Context, connectionID uuid.UUID, body []byte) (*gen.ConnectionsSchedulesCreateResp, error) {
	return c.api.ConnectionsSchedulesCreateWithBodyWithResponse(ctx, connectionID, "application/json", bytes.NewReader(body))
}

func (c *Client) GetConnectionSchedule(ctx context.Context, connectionID uuid.UUID, scheduleID uuid.UUID) (*gen.ConnectionsSchedulesGetResp, error) {
	return c.api.ConnectionsSchedulesGetWithResponse(ctx, connectionID, scheduleID)
}

func (c *Client) UpdateConnectionSchedule(ctx context.Context, connectionID uuid.UUID, scheduleID uuid.UUID, body []byte) (*gen.ConnectionsSchedulesUpdateResp, error) {
	return c.api.ConnectionsSchedulesUpdateWithBodyWithResponse(ctx, connectionID, scheduleID, "application/json", bytes.NewReader(body))
}

func (c *Client) DeleteConnectionSchedule(ctx context.Context, connectionID uuid.UUID, scheduleID uuid.UUID) (*gen.ConnectionsSchedulesDeleteResp, error) {
	return c.api.ConnectionsSchedulesDeleteWithResponse(ctx, connectionID, scheduleID)
}

func (c *Client) ListConnectionEnvironments(ctx context.Context) (*gen.ConnectionEnvironmentsListResp, error) {
	return c.api.ConnectionEnvironmentsListWithResponse(ctx)
}

func (c *Client) CreateConnectionEnvironment(ctx context.Context, body []byte) (*gen.ConnectionEnvironmentsCreateResp, error) {
	return c.api.ConnectionEnvironmentsCreateWithBodyWithResponse(ctx, "application/json", bytes.NewReader(body))
}

func (c *Client) UpdateConnectionEnvironment(ctx context.Context, id uuid.UUID, body []byte) (*gen.ConnectionEnvironmentsUpdateResp, error) {
	return c.api.ConnectionEnvironmentsUpdateWithBodyWithResponse(ctx, id, "application/json", bytes.NewReader(body))
}

func (c *Client) DeleteConnectionEnvironment(ctx context.Context, id uuid.UUID) (*gen.ConnectionEnvironmentsDeleteResp, error) {
	return c.api.ConnectionEnvironmentsDeleteWithResponse(ctx, id)
}

func (c *Client) ListDocuments(ctx context.Context, cursor string, pageSize int) (*gen.DocumentsListResp, error) {
	params := &gen.DocumentsListParams{}
	if strings.TrimSpace(cursor) != "" {
		cur := strings.TrimSpace(cursor)
		params.Cursor = &cur
	}
	if pageSize > 0 {
		size := pageSize
		params.PageSize = &size
	}
	return c.api.DocumentsListWithResponse(ctx, params)
}

func (c *Client) GetDocument(ctx context.Context, identifier string) (*gen.DocumentsGetResp, error) {
	return c.api.DocumentsGetWithResponse(ctx, identifier)
}

func (c *Client) CreateDocument(ctx context.Context, body []byte) (*gen.DocumentsCreateResp, error) {
	return c.api.DocumentsCreateWithBodyWithResponse(ctx, "application/json", bytes.NewReader(body))
}

func (c *Client) DeleteDocument(ctx context.Context, identifier string) (*gen.DocumentsDeleteResp, error) {
	return c.api.DocumentsDeleteWithResponse(ctx, identifier)
}

func (c *Client) RenameDocument(ctx context.Context, identifier string, body []byte) (*gen.DocumentsRenameResp, error) {
	return c.api.DocumentsRenameWithBodyWithResponse(ctx, identifier, "application/json", bytes.NewReader(body))
}

func (c *Client) CreateDocumentDraft(ctx context.Context, identifier string, body []byte) (*gen.DocumentsCreateDraftResp, error) {
	return c.api.DocumentsCreateDraftWithBodyWithResponse(ctx, identifier, "application/json", bytes.NewReader(body))
}

func (c *Client) DiscardDocumentDraft(ctx context.Context, identifier string, body []byte) (*gen.DocumentsDiscardDraftResp, error) {
	return c.api.DocumentsDiscardDraftWithBodyWithResponse(ctx, identifier, "application/json", bytes.NewReader(body))
}

func (c *Client) DuplicateDocument(ctx context.Context, identifier string, body []byte, userID *uuid.UUID) (*gen.DocumentsDuplicateResp, error) {
	var params *gen.DocumentsDuplicateParams
	if userID != nil {
		params = &gen.DocumentsDuplicateParams{UserId: userID}
	}
	return c.api.DocumentsDuplicateWithBodyWithResponse(ctx, identifier, params, "application/json", bytes.NewReader(body))
}

func (c *Client) AddDocumentFavorite(ctx context.Context, identifier string, userID *uuid.UUID) (*gen.DocumentsAddFavoriteResp, error) {
	var params *gen.DocumentsAddFavoriteParams
	if userID != nil {
		params = &gen.DocumentsAddFavoriteParams{UserId: userID}
	}
	return c.api.DocumentsAddFavoriteWithResponse(ctx, identifier, params)
}

func (c *Client) RemoveDocumentFavorite(ctx context.Context, identifier string, userID *uuid.UUID) (*gen.DocumentsRemoveFavoriteResp, error) {
	var params *gen.DocumentsRemoveFavoriteParams
	if userID != nil {
		params = &gen.DocumentsRemoveFavoriteParams{UserId: userID}
	}
	return c.api.DocumentsRemoveFavoriteWithResponse(ctx, identifier, params)
}

func (c *Client) BulkUpdateDocumentLabels(ctx context.Context, identifier string, body []byte, userID *uuid.UUID) (*gen.DocumentsBulkUpdateLabelsResp, error) {
	var params *gen.DocumentsBulkUpdateLabelsParams
	if userID != nil {
		params = &gen.DocumentsBulkUpdateLabelsParams{UserId: userID}
	}
	return c.api.DocumentsBulkUpdateLabelsWithBodyWithResponse(ctx, identifier, params, "application/json", bytes.NewReader(body))
}

func (c *Client) UpdateDocumentPermissionSettings(ctx context.Context, identifier string, body []byte) (*gen.DocumentsUpdatePermissionSettingsResp, error) {
	return c.api.DocumentsUpdatePermissionSettingsWithBodyWithResponse(ctx, identifier, "application/json", bytes.NewReader(body))
}

func (c *Client) ListDocumentQueries(ctx context.Context, identifier string) (*gen.DocumentsGetQueriesResp, error) {
	return c.api.DocumentsGetQueriesWithResponse(ctx, identifier)
}

func (c *Client) TransferDocumentOwnership(ctx context.Context, identifier string, body []byte) (*gen.DocumentsTransferOwnershipResp, error) {
	return c.api.DocumentsTransferOwnershipWithBodyWithResponse(ctx, identifier, "application/json", bytes.NewReader(body))
}

func (c *Client) MoveDocument(ctx context.Context, identifier string, body []byte) (*gen.DocumentsMoveResp, error) {
	return c.api.DocumentsMoveWithBodyWithResponse(ctx, identifier, "application/json", bytes.NewReader(body))
}

func (c *Client) ListDocumentAccess(ctx context.Context, identifier string, cursor string, pageSize int, principalType string, accessSource string, sortField string, sortDirection string) (*gen.DocumentsAccessListResp, error) {
	params := &gen.DocumentsAccessListParams{}
	if strings.TrimSpace(cursor) != "" {
		cur := strings.TrimSpace(cursor)
		params.Cursor = &cur
	}
	if pageSize > 0 {
		size := float32(pageSize)
		params.PageSize = &size
	}
	if strings.TrimSpace(principalType) != "" {
		t := gen.DocumentsAccessListParamsType(strings.TrimSpace(principalType))
		params.Type = &t
	}
	if strings.TrimSpace(accessSource) != "" {
		src := gen.DocumentsAccessListParamsAccessSource(strings.TrimSpace(accessSource))
		params.AccessSource = &src
	}
	if strings.TrimSpace(sortField) != "" {
		f := strings.TrimSpace(sortField)
		params.SortField = &f
	}
	if strings.TrimSpace(sortDirection) != "" {
		dir := gen.DocumentsAccessListParamsSortDirection(strings.TrimSpace(sortDirection))
		params.SortDirection = &dir
	}
	return c.api.DocumentsAccessListWithResponse(ctx, identifier, params)
}

func (c *Client) GetDocumentPermissions(ctx context.Context, identifier string, userID uuid.UUID) (*gen.DocumentsGetPermissionsResp, error) {
	return c.api.DocumentsGetPermissionsWithResponse(ctx, identifier, &gen.DocumentsGetPermissionsParams{
		UserId: userID,
	})
}

func (c *Client) AddDocumentPermits(ctx context.Context, identifier string, body []byte) (*gen.DocumentsAddPermitsResp, error) {
	return c.api.DocumentsAddPermitsWithBodyWithResponse(ctx, identifier, "application/json", bytes.NewReader(body))
}

func (c *Client) UpdateDocumentPermits(ctx context.Context, identifier string, body []byte) (*gen.DocumentsUpdatePermitsResp, error) {
	return c.api.DocumentsUpdatePermitsWithBodyWithResponse(ctx, identifier, "application/json", bytes.NewReader(body))
}

func (c *Client) RevokeDocumentPermits(ctx context.Context, identifier string, body []byte) (*gen.DocumentsRevokePermitsResp, error) {
	return c.api.DocumentsRevokePermitsWithBodyWithResponse(ctx, identifier, "application/json", bytes.NewReader(body))
}

func (c *Client) AddDocumentLabel(ctx context.Context, identifier string, labelName string, userID *uuid.UUID) (*gen.DocumentsAddLabelResp, error) {
	var params *gen.DocumentsAddLabelParams
	if userID != nil {
		params = &gen.DocumentsAddLabelParams{UserId: userID}
	}
	return c.api.DocumentsAddLabelWithResponse(ctx, identifier, labelName, params)
}

func (c *Client) RemoveDocumentLabel(ctx context.Context, identifier string, labelName string, userID *uuid.UUID) (*gen.DocumentsRemoveLabelResp, error) {
	var params *gen.DocumentsRemoveLabelParams
	if userID != nil {
		params = &gen.DocumentsRemoveLabelParams{UserId: userID}
	}
	return c.api.DocumentsRemoveLabelWithResponse(ctx, identifier, labelName, params)
}

func (c *Client) ListModels(ctx context.Context, cursor string, pageSize int, name string) (*gen.ModelsListResp, error) {
	params := &gen.ModelsListParams{}
	if strings.TrimSpace(cursor) != "" {
		cur := strings.TrimSpace(cursor)
		params.Cursor = &cur
	}
	if pageSize > 0 {
		size := pageSize
		params.PageSize = &size
	}
	if strings.TrimSpace(name) != "" {
		n := strings.TrimSpace(name)
		params.Name = &n
	}
	return c.api.ModelsListWithResponse(ctx, params)
}

func (c *Client) GetModelByID(ctx context.Context, modelID uuid.UUID) (*gen.ModelsListResp, error) {
	return c.api.ModelsListWithResponse(ctx, &gen.ModelsListParams{
		ModelId: &modelID,
	})
}

func (c *Client) CreateModel(ctx context.Context, body []byte) (*gen.ModelsCreateResp, error) {
	return c.api.ModelsCreateWithBodyWithResponse(ctx, "application/json", bytes.NewReader(body))
}

func (c *Client) DeleteModelBranch(ctx context.Context, modelID uuid.UUID, branchName string) (*gen.ModelsDeleteBranchResp, error) {
	return c.api.ModelsDeleteBranchWithResponse(ctx, modelID, branchName)
}

func (c *Client) MergeModelBranch(ctx context.Context, modelID uuid.UUID, branchName string, body []byte) (*gen.ModelsMergeBranchResp, error) {
	return c.api.ModelsMergeBranchWithBodyWithResponse(ctx, modelID, branchName, "application/json", bytes.NewReader(body))
}

func (c *Client) ResetModelCache(ctx context.Context, modelID uuid.UUID, policyName string, body []byte) (*gen.ModelsCacheResetResp, error) {
	return c.api.ModelsCacheResetWithBodyWithResponse(ctx, modelID, policyName, "application/json", bytes.NewReader(body))
}

func (c *Client) CreateModelField(ctx context.Context, modelID uuid.UUID, body []byte) (*gen.ModelsCreateFieldResp, error) {
	return c.api.ModelsCreateFieldWithBodyWithResponse(ctx, modelID, "application/json", bytes.NewReader(body))
}

func (c *Client) GetModelGit(ctx context.Context, modelID uuid.UUID, include string) (*gen.ModelsGitGetResp, error) {
	var params *gen.ModelsGitGetParams
	if strings.TrimSpace(include) != "" {
		v := strings.TrimSpace(include)
		params = &gen.ModelsGitGetParams{Include: &v}
	}
	return c.api.ModelsGitGetWithResponse(ctx, modelID, params)
}

func (c *Client) CreateModelGit(ctx context.Context, modelID uuid.UUID, body []byte) (*gen.ModelsGitCreateResp, error) {
	return c.api.ModelsGitCreateWithBodyWithResponse(ctx, modelID, "application/json", bytes.NewReader(body))
}

func (c *Client) UpdateModelGit(ctx context.Context, modelID uuid.UUID, body []byte) (*gen.ModelsGitUpdateResp, error) {
	return c.api.ModelsGitUpdateWithBodyWithResponse(ctx, modelID, "application/json", bytes.NewReader(body))
}

func (c *Client) DeleteModelGit(ctx context.Context, modelID uuid.UUID) (*gen.ModelsGitDeleteResp, error) {
	return c.api.ModelsGitDeleteWithResponse(ctx, modelID)
}

func (c *Client) SyncModelGit(ctx context.Context, modelID uuid.UUID, body []byte) (*gen.ModelsGitSyncResp, error) {
	return c.api.ModelsGitSyncWithBodyWithResponse(ctx, modelID, "application/json", bytes.NewReader(body))
}

func (c *Client) MigrateModel(ctx context.Context, modelID uuid.UUID, body []byte) (*gen.ModelsMigrateResp, error) {
	return c.api.ModelsMigrateWithBodyWithResponse(ctx, modelID, "application/json", bytes.NewReader(body))
}

func (c *Client) RefreshModel(ctx context.Context, modelID uuid.UUID) (*gen.ModelsRefreshResp, error) {
	return c.api.ModelsRefreshWithResponse(ctx, modelID)
}

func (c *Client) ListModelTopics(ctx context.Context, modelID uuid.UUID, branchID *uuid.UUID) (*gen.ModelsListTopicsResp, error) {
	var params *gen.ModelsListTopicsParams
	if branchID != nil {
		params = &gen.ModelsListTopicsParams{BranchId: branchID}
	}
	return c.api.ModelsListTopicsWithResponse(ctx, modelID, params)
}

func (c *Client) GetModelTopic(ctx context.Context, modelID uuid.UUID, topicName string, branchID *uuid.UUID) (*gen.ModelsGetTopicResp, error) {
	var params *gen.ModelsGetTopicParams
	if branchID != nil {
		params = &gen.ModelsGetTopicParams{BranchId: branchID}
	}
	return c.api.ModelsGetTopicWithResponse(ctx, modelID, topicName, params)
}

func (c *Client) UpdateModelTopic(ctx context.Context, modelID uuid.UUID, topicName string, branchID *uuid.UUID, body []byte) (*gen.ModelsUpdateTopicResp, error) {
	var params *gen.ModelsUpdateTopicParams
	if branchID != nil {
		params = &gen.ModelsUpdateTopicParams{BranchId: branchID}
	}
	return c.api.ModelsUpdateTopicWithBodyWithResponse(ctx, modelID, topicName, params, "application/json", bytes.NewReader(body))
}

func (c *Client) DeleteModelTopic(ctx context.Context, modelID uuid.UUID, topicName string, branchID *uuid.UUID) (*gen.ModelsDeleteTopicResp, error) {
	var params *gen.ModelsDeleteTopicParams
	if branchID != nil {
		params = &gen.ModelsDeleteTopicParams{BranchId: branchID}
	}
	return c.api.ModelsDeleteTopicWithResponse(ctx, modelID, topicName, params)
}

func (c *Client) ValidateModel(ctx context.Context, modelID uuid.UUID, branchID *uuid.UUID, limit int) (*gen.ModelsValidateResp, error) {
	var params *gen.ModelsValidateParams
	if branchID != nil || limit > 0 {
		params = &gen.ModelsValidateParams{}
		if branchID != nil {
			params.BranchId = branchID
		}
		if limit > 0 {
			params.Limit = &limit
		}
	}
	return c.api.ModelsValidateWithResponse(ctx, modelID, params)
}

func (c *Client) ListModelViews(ctx context.Context, modelID uuid.UUID, branchID *uuid.UUID) (*gen.ModelsGetViewsResp, error) {
	var params *gen.ModelsGetViewsParams
	if branchID != nil {
		params = &gen.ModelsGetViewsParams{BranchId: branchID}
	}
	return c.api.ModelsGetViewsWithResponse(ctx, modelID, params)
}

func (c *Client) UpdateModelView(ctx context.Context, modelID uuid.UUID, viewName string, branchID *uuid.UUID, body []byte) (*gen.ModelsUpdateViewResp, error) {
	var params *gen.ModelsUpdateViewParams
	if branchID != nil {
		params = &gen.ModelsUpdateViewParams{BranchId: branchID}
	}
	return c.api.ModelsUpdateViewWithBodyWithResponse(ctx, modelID, viewName, params, "application/json", bytes.NewReader(body))
}

func (c *Client) DeleteModelView(ctx context.Context, modelID uuid.UUID, viewName string, branchID *uuid.UUID) (*gen.ModelsDeleteViewResp, error) {
	var params *gen.ModelsDeleteViewParams
	if branchID != nil {
		params = &gen.ModelsDeleteViewParams{BranchId: branchID}
	}
	return c.api.ModelsDeleteViewWithResponse(ctx, modelID, viewName, params)
}

func (c *Client) UpdateModelField(ctx context.Context, modelID uuid.UUID, viewName string, fieldName string, branchID *uuid.UUID, body []byte) (*gen.ModelsUpdateFieldResp, error) {
	var params *gen.ModelsUpdateFieldParams
	if branchID != nil {
		params = &gen.ModelsUpdateFieldParams{BranchId: branchID}
	}
	return c.api.ModelsUpdateFieldWithBodyWithResponse(ctx, modelID, viewName, fieldName, params, "application/json", bytes.NewReader(body))
}

func (c *Client) DeleteModelField(ctx context.Context, modelID uuid.UUID, viewName string, fieldName string, branchID *uuid.UUID, topicContext string) (*gen.ModelsDeleteFieldResp, error) {
	var params *gen.ModelsDeleteFieldParams
	if branchID != nil || strings.TrimSpace(topicContext) != "" {
		params = &gen.ModelsDeleteFieldParams{}
		if branchID != nil {
			params.BranchId = branchID
		}
		if strings.TrimSpace(topicContext) != "" {
			tc := strings.TrimSpace(topicContext)
			params.TopicContext = &tc
		}
	}
	return c.api.ModelsDeleteFieldWithResponse(ctx, modelID, viewName, fieldName, params)
}

func (c *Client) GetModelContentValidator(ctx context.Context, modelID uuid.UUID, branchID string, userID string, includePersonalFolders *bool) (*gen.ModelsContentValidatorGetResp, error) {
	var params *gen.ModelsContentValidatorGetParams
	if strings.TrimSpace(branchID) != "" || strings.TrimSpace(userID) != "" || includePersonalFolders != nil {
		params = &gen.ModelsContentValidatorGetParams{}
		if strings.TrimSpace(branchID) != "" {
			b := strings.TrimSpace(branchID)
			params.BranchId = &b
		}
		if strings.TrimSpace(userID) != "" {
			u := strings.TrimSpace(userID)
			params.UserId = &u
		}
		if includePersonalFolders != nil {
			params.IncludePersonalFolders = includePersonalFolders
		}
	}
	return c.api.ModelsContentValidatorGetWithResponse(ctx, modelID, params)
}

func (c *Client) ReplaceModelContentValidator(ctx context.Context, modelID uuid.UUID, userID string, body []byte) (*gen.ModelsContentValidatorReplaceResp, error) {
	var params *gen.ModelsContentValidatorReplaceParams
	if strings.TrimSpace(userID) != "" {
		u := strings.TrimSpace(userID)
		params = &gen.ModelsContentValidatorReplaceParams{UserId: &u}
	}
	return c.api.ModelsContentValidatorReplaceWithBodyWithResponse(ctx, modelID, params, "application/json", bytes.NewReader(body))
}

func (c *Client) CreateModelYAML(ctx context.Context, modelID uuid.UUID, body []byte) (*gen.ModelsYamlCreateResp, error) {
	return c.api.ModelsYamlCreateWithBodyWithResponse(ctx, modelID, "application/json", bytes.NewReader(body))
}

func (c *Client) GetModelYAML(ctx context.Context, modelID uuid.UUID, branchID *uuid.UUID, fileName string, mode string, includeChecksums *bool) (*gen.ModelsYamlGetResp, error) {
	var params *gen.ModelsYamlGetParams
	if branchID != nil || strings.TrimSpace(fileName) != "" || strings.TrimSpace(mode) != "" || includeChecksums != nil {
		params = &gen.ModelsYamlGetParams{}
		if branchID != nil {
			params.BranchId = branchID
		}
		if strings.TrimSpace(fileName) != "" {
			f := strings.TrimSpace(fileName)
			params.FileName = &f
		}
		if strings.TrimSpace(mode) != "" {
			m := gen.ModelsYamlGetParamsMode(strings.TrimSpace(mode))
			params.Mode = &m
		}
		if includeChecksums != nil {
			params.IncludeChecksums = includeChecksums
		}
	}
	return c.api.ModelsYamlGetWithResponse(ctx, modelID, params)
}

func (c *Client) DeleteModelYAML(ctx context.Context, modelID uuid.UUID, fileName string, branchID *uuid.UUID, mode string, commitMessage string) (*gen.ModelsYamlDeleteResp, error) {
	payload := map[string]any{
		"fileName": fileName,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var params gen.ModelsYamlDeleteParams
	if err := json.Unmarshal(encoded, &params); err != nil {
		return nil, err
	}
	if branchID != nil {
		params.BranchId = branchID
	}
	if strings.TrimSpace(mode) != "" {
		m := gen.ModelsYamlDeleteParamsMode(strings.TrimSpace(mode))
		params.Mode = &m
	}
	if strings.TrimSpace(commitMessage) != "" {
		msg := strings.TrimSpace(commitMessage)
		params.CommitMessage = &msg
	}
	return c.api.ModelsYamlDeleteWithResponse(ctx, modelID, &params)
}

func (c *Client) ListSCIMUsers(ctx context.Context, count int, startIndex int, filter string) (*gen.ScimUsersListResp, error) {
	params := &gen.ScimUsersListParams{}
	if count > 0 {
		v := fmt.Sprintf("%d", count)
		params.Count = &v
	}
	if startIndex > 0 {
		v := fmt.Sprintf("%d", startIndex)
		params.StartIndex = &v
	}
	if strings.TrimSpace(filter) != "" {
		v := strings.TrimSpace(filter)
		params.Filter = &v
	}
	return c.api.ScimUsersListWithResponse(ctx, params)
}

func (c *Client) CreateSCIMUser(ctx context.Context, body []byte) (*gen.ScimUsersCreateResp, error) {
	return c.api.ScimUsersCreateWithBodyWithResponse(ctx, "application/json", bytes.NewReader(body))
}

func (c *Client) GetSCIMUser(ctx context.Context, id uuid.UUID) (*gen.ScimUsersGetResp, error) {
	return c.api.ScimUsersGetWithResponse(ctx, id)
}

func (c *Client) UpdateSCIMUser(ctx context.Context, id uuid.UUID, body []byte) (*gen.ScimUsersUpdateResp, error) {
	return c.api.ScimUsersUpdateWithBodyWithResponse(ctx, id, "application/json", bytes.NewReader(body))
}

func (c *Client) ReplaceSCIMUser(ctx context.Context, id uuid.UUID, body []byte) (*gen.ScimUsersReplaceResp, error) {
	return c.api.ScimUsersReplaceWithBodyWithResponse(ctx, id, "application/json", bytes.NewReader(body))
}

func (c *Client) DeleteSCIMUser(ctx context.Context, id uuid.UUID) (*gen.ScimUsersDeleteResp, error) {
	return c.api.ScimUsersDeleteWithResponse(ctx, id)
}

func (c *Client) ListSCIMGroups(ctx context.Context, count int, startIndex int) (*gen.ScimGroupsListResp, error) {
	params := &gen.ScimGroupsListParams{}
	if count > 0 {
		v := fmt.Sprintf("%d", count)
		params.Count = &v
	}
	if startIndex > 0 {
		v := fmt.Sprintf("%d", startIndex)
		params.StartIndex = &v
	}
	return c.api.ScimGroupsListWithResponse(ctx, params)
}

func (c *Client) CreateSCIMGroup(ctx context.Context, body []byte) (*gen.ScimGroupsCreateResp, error) {
	return c.api.ScimGroupsCreateWithBodyWithResponse(ctx, "application/json", bytes.NewReader(body))
}

func (c *Client) GetSCIMGroup(ctx context.Context, miniUUID string) (*gen.ScimGroupsGetResp, error) {
	return c.api.ScimGroupsGetWithResponse(ctx, miniUUID, nil)
}

func (c *Client) UpdateSCIMGroup(ctx context.Context, miniUUID string, body []byte) (*gen.ScimGroupsUpdateResp, error) {
	return c.api.ScimGroupsUpdateWithBodyWithResponse(ctx, miniUUID, "application/json", bytes.NewReader(body))
}

func (c *Client) ReplaceSCIMGroup(ctx context.Context, miniUUID string, body []byte) (*gen.ScimGroupsReplaceResp, error) {
	return c.api.ScimGroupsReplaceWithBodyWithResponse(ctx, miniUUID, "application/json", bytes.NewReader(body))
}

func (c *Client) DeleteSCIMGroup(ctx context.Context, miniUUID string) (*gen.ScimGroupsDeleteResp, error) {
	return c.api.ScimGroupsDeleteWithResponse(ctx, miniUUID)
}

func (c *Client) ListSCIMEmbedUsers(ctx context.Context, count int, startIndex int, filter string) (*gen.ScimEmbedUsersListResp, error) {
	params := &gen.ScimEmbedUsersListParams{}
	if count > 0 {
		v := fmt.Sprintf("%d", count)
		params.Count = &v
	}
	if startIndex > 0 {
		v := fmt.Sprintf("%d", startIndex)
		params.StartIndex = &v
	}
	if strings.TrimSpace(filter) != "" {
		f := strings.TrimSpace(filter)
		params.Filter = &f
	}
	return c.api.ScimEmbedUsersListWithResponse(ctx, params)
}

func (c *Client) GetSCIMEmbedUser(ctx context.Context, id uuid.UUID) (*gen.ScimEmbedUsersGetResp, error) {
	return c.api.ScimEmbedUsersGetWithResponse(ctx, id)
}

func (c *Client) DeleteSCIMEmbedUser(ctx context.Context, id uuid.UUID) (*gen.ScimEmbedUsersDeleteResp, error) {
	return c.api.ScimEmbedUsersDeleteWithResponse(ctx, id)
}

func (c *Client) ListEmailOnlyUsers(ctx context.Context, cursor string, pageSize int, email string, sortDirection string) (*gen.UsersListEmailOnlyResp, error) {
	params := &gen.UsersListEmailOnlyParams{}
	if strings.TrimSpace(cursor) != "" {
		v := strings.TrimSpace(cursor)
		params.Cursor = &v
	}
	if pageSize > 0 {
		size := float32(pageSize)
		params.PageSize = &size
	}
	if strings.TrimSpace(email) != "" {
		v := strings.TrimSpace(email)
		params.Email = &v
	}
	if strings.TrimSpace(sortDirection) != "" {
		v := gen.UsersListEmailOnlyParamsSortDirection(strings.TrimSpace(sortDirection))
		params.SortDirection = &v
	}
	return c.api.UsersListEmailOnlyWithResponse(ctx, params)
}

func (c *Client) CreateEmailOnlyUser(ctx context.Context, body []byte) (*gen.UsersCreateEmailOnlyResp, error) {
	return c.api.UsersCreateEmailOnlyWithBodyWithResponse(ctx, "application/json", bytes.NewReader(body))
}

func (c *Client) CreateEmailOnlyUsersBulk(ctx context.Context, body []byte) (*gen.UsersCreateEmailOnlyBulkResp, error) {
	return c.api.UsersCreateEmailOnlyBulkWithBodyWithResponse(ctx, "application/json", bytes.NewReader(body))
}

func (c *Client) GetUserModelRoles(ctx context.Context, id uuid.UUID, connectionID *uuid.UUID, modelID *uuid.UUID) (*gen.UsersGetModelRolesResp, error) {
	var params *gen.UsersGetModelRolesParams
	if connectionID != nil || modelID != nil {
		params = &gen.UsersGetModelRolesParams{
			ConnectionId: connectionID,
			ModelId:      modelID,
		}
	}
	return c.api.UsersGetModelRolesWithResponse(ctx, id, params)
}

func (c *Client) AssignUserModelRole(ctx context.Context, id uuid.UUID, body []byte) (*gen.UsersAssignModelRoleResp, error) {
	return c.api.UsersAssignModelRoleWithBodyWithResponse(ctx, id, "application/json", bytes.NewReader(body))
}

func (c *Client) GetUserGroupModelRoles(ctx context.Context, id string, connectionID *uuid.UUID, modelID *uuid.UUID) (*gen.UserGroupsGetModelRolesResp, error) {
	var params *gen.UserGroupsGetModelRolesParams
	if connectionID != nil || modelID != nil {
		params = &gen.UserGroupsGetModelRolesParams{
			ConnectionId: connectionID,
			ModelId:      modelID,
		}
	}
	return c.api.UserGroupsGetModelRolesWithResponse(ctx, id, params)
}

func (c *Client) AssignUserGroupModelRole(ctx context.Context, id string, body []byte) (*gen.UserGroupsAssignModelRoleResp, error) {
	return c.api.UserGroupsAssignModelRoleWithBodyWithResponse(ctx, id, "application/json", bytes.NewReader(body))
}

func (c *Client) ListFolders(ctx context.Context, cursor string, pageSize int, path string) (*gen.FoldersListResp, error) {
	params := &gen.FoldersListParams{}
	if strings.TrimSpace(cursor) != "" {
		cur := strings.TrimSpace(cursor)
		params.Cursor = &cur
	}
	if pageSize > 0 {
		size := float32(pageSize)
		params.PageSize = &size
	}
	if strings.TrimSpace(path) != "" {
		p := strings.TrimSpace(path)
		params.Path = &p
	}
	return c.api.FoldersListWithResponse(ctx, params)
}

func (c *Client) CreateFolder(ctx context.Context, name string, parentFolderID *uuid.UUID, scope string, userID *uuid.UUID) (*gen.FoldersCreateResp, error) {
	body := gen.FoldersCreateBody{Name: name}
	if parentFolderID != nil {
		body.ParentFolderId = parentFolderID
	}
	if strings.TrimSpace(scope) != "" {
		s := gen.FoldersCreateBodyScope(strings.TrimSpace(scope))
		body.Scope = &s
	}
	if userID != nil {
		body.UserId = userID
	}
	return c.api.FoldersCreateWithResponse(ctx, body)
}

func (c *Client) DeleteFolder(ctx context.Context, id uuid.UUID) (*gen.FoldersDeleteResp, error) {
	return c.api.FoldersDeleteWithResponse(ctx, id)
}

func (c *Client) GetFolderPermissions(ctx context.Context, id uuid.UUID, userID *uuid.UUID) (*gen.FoldersGetPermissionsResp, error) {
	var params *gen.FoldersGetPermissionsParams
	if userID != nil {
		params = &gen.FoldersGetPermissionsParams{UserId: userID}
	}
	return c.api.FoldersGetPermissionsWithResponse(ctx, id, params)
}

func (c *Client) AddFolderPermissions(ctx context.Context, id uuid.UUID, body []byte) (*gen.FoldersAddPermissionsResp, error) {
	return c.api.FoldersAddPermissionsWithBodyWithResponse(ctx, id, "application/json", bytes.NewReader(body))
}

func (c *Client) UpdateFolderPermissions(ctx context.Context, id uuid.UUID, body []byte) (*gen.FoldersUpdatePermissionsResp, error) {
	return c.api.FoldersUpdatePermissionsWithBodyWithResponse(ctx, id, "application/json", bytes.NewReader(body))
}

func (c *Client) RevokeFolderPermissions(ctx context.Context, id uuid.UUID, body []byte) (*gen.FoldersRevokePermissionsResp, error) {
	return c.api.FoldersRevokePermissionsWithBodyWithResponse(ctx, id, "application/json", bytes.NewReader(body))
}

func (c *Client) ListLabels(ctx context.Context) (*gen.LabelsListResp, error) {
	return c.api.LabelsListWithResponse(ctx)
}

func (c *Client) GetLabel(ctx context.Context, name string) (*gen.LabelsGetResp, error) {
	return c.api.LabelsGetWithResponse(ctx, name)
}

func (c *Client) CreateLabel(ctx context.Context, name string, homepage, verified *bool, userID string) (*gen.LabelsCreateResp, error) {
	body := gen.LabelsCreateBody{
		Name: name,
	}
	if homepage != nil {
		body.Homepage = homepage
	}
	if verified != nil {
		body.Verified = verified
	}

	var params *gen.LabelsCreateParams
	if strings.TrimSpace(userID) != "" {
		u := strings.TrimSpace(userID)
		params = &gen.LabelsCreateParams{UserId: &u}
	}
	return c.api.LabelsCreateWithResponse(ctx, params, body)
}

func (c *Client) UpdateLabel(ctx context.Context, name string, newName *string, homepage, verified *bool, userID string) (*gen.LabelsUpdateResp, error) {
	body := gen.LabelsUpdateBody{}
	if newName != nil {
		body.Name = newName
	}
	if homepage != nil {
		body.Homepage = homepage
	}
	if verified != nil {
		body.Verified = verified
	}

	var params *gen.LabelsUpdateParams
	if strings.TrimSpace(userID) != "" {
		u := strings.TrimSpace(userID)
		params = &gen.LabelsUpdateParams{UserId: &u}
	}
	return c.api.LabelsUpdateWithResponse(ctx, name, params, body)
}

func (c *Client) DeleteLabel(ctx context.Context, name string, userID string) (*gen.LabelsDeleteResp, error) {
	var params *gen.LabelsDeleteParams
	if strings.TrimSpace(userID) != "" {
		u := strings.TrimSpace(userID)
		params = &gen.LabelsDeleteParams{UserId: &u}
	}
	return c.api.LabelsDeleteWithResponse(ctx, name, params)
}

func (c *Client) ListSchedules(ctx context.Context, cursor string, pageSize int, q string) (*gen.SchedulesListResp, error) {
	params := &gen.SchedulesListParams{}
	if strings.TrimSpace(cursor) != "" {
		cur := strings.TrimSpace(cursor)
		params.Cursor = &cur
	}
	if pageSize > 0 {
		size := float32(pageSize)
		params.PageSize = &size
	}
	if strings.TrimSpace(q) != "" {
		query := strings.TrimSpace(q)
		params.Q = &query
	}
	return c.api.SchedulesListWithResponse(ctx, params)
}

func (c *Client) GetSchedule(ctx context.Context, id uuid.UUID) (*gen.SchedulesGetResp, error) {
	return c.api.SchedulesGetWithResponse(ctx, id)
}

func (c *Client) DeleteSchedule(ctx context.Context, id uuid.UUID) (*gen.SchedulesDeleteResp, error) {
	return c.api.SchedulesDeleteWithResponse(ctx, id)
}

func (c *Client) PauseSchedule(ctx context.Context, id uuid.UUID) (*gen.SchedulesPauseResp, error) {
	return c.api.SchedulesPauseWithResponse(ctx, id)
}

func (c *Client) ResumeSchedule(ctx context.Context, id uuid.UUID) (*gen.SchedulesResumeResp, error) {
	return c.api.SchedulesResumeWithResponse(ctx, id)
}

func (c *Client) TriggerSchedule(ctx context.Context, id uuid.UUID) (*gen.SchedulesTriggerResp, error) {
	return c.api.SchedulesTriggerWithResponse(ctx, id)
}

func (c *Client) UpdateSchedule(ctx context.Context, id uuid.UUID) (*gen.SchedulesUpdateResp, error) {
	return c.api.SchedulesUpdateWithResponse(ctx, id)
}

func (c *Client) AIGenerateQuery(ctx context.Context, body []byte) (*gen.AiGenerateQueryResp, error) {
	return c.api.AiGenerateQueryWithBodyWithResponse(ctx, "application/json", bytes.NewReader(body))
}

func (c *Client) AIPickTopic(ctx context.Context, body []byte) (*gen.AiPickTopicResp, error) {
	return c.api.AiPickTopicWithBodyWithResponse(ctx, "application/json", bytes.NewReader(body))
}

func (c *Client) CreateSchedule(ctx context.Context, userID *uuid.UUID, body []byte) (*gen.SchedulesCreateResp, error) {
	var params *gen.SchedulesCreateParams
	if userID != nil {
		params = &gen.SchedulesCreateParams{UserId: userID}
	}
	return c.api.SchedulesCreateWithBodyWithResponse(ctx, params, "application/json", bytes.NewReader(body))
}

func (c *Client) GetScheduleRecipients(ctx context.Context, id uuid.UUID) (*gen.SchedulesRecipientsGetResp, error) {
	return c.api.SchedulesRecipientsGetWithResponse(ctx, id)
}

func (c *Client) AddScheduleRecipients(ctx context.Context, id uuid.UUID, body []byte) (*gen.SchedulesAddRecipientsResp, error) {
	return c.api.SchedulesAddRecipientsWithBodyWithResponse(ctx, id, "application/json", bytes.NewReader(body))
}

func (c *Client) RemoveScheduleRecipients(ctx context.Context, id uuid.UUID, body []byte) (*gen.SchedulesRemoveRecipientsResp, error) {
	return c.api.SchedulesRemoveRecipientsWithBodyWithResponse(ctx, id, "application/json", bytes.NewReader(body))
}

func (c *Client) TransferScheduleOwnership(ctx context.Context, id uuid.UUID, body []byte) (*gen.SchedulesTransferOwnershipResp, error) {
	return c.api.SchedulesTransferOwnershipWithBodyWithResponse(ctx, id, "application/json", bytes.NewReader(body))
}

func (c *Client) SubmitAgenticJob(ctx context.Context, body []byte) (*gen.AgenticJobSubmitResp, error) {
	return c.api.AgenticJobSubmitWithBodyWithResponse(ctx, "application/json", bytes.NewReader(body))
}

func (c *Client) GetAgenticJobStatus(ctx context.Context, jobID uuid.UUID) (*gen.AgenticJobStatusResp, error) {
	return c.api.AgenticJobStatusWithResponse(ctx, jobID)
}

func (c *Client) CancelAgenticJob(ctx context.Context, jobID uuid.UUID) (*gen.AgenticJobCancelResp, error) {
	return c.api.AgenticJobCancelWithResponse(ctx, jobID)
}

func (c *Client) GetAgenticJobResult(ctx context.Context, jobID uuid.UUID) (*gen.AgenticJobResultResp, error) {
	return c.api.AgenticJobResultWithResponse(ctx, jobID)
}

func (c *Client) DashboardDownload(ctx context.Context, identifier string, userID string, body []byte) (*gen.DashboardsDownloadResp, error) {
	var params *gen.DashboardsDownloadParams
	if strings.TrimSpace(userID) != "" {
		v := strings.TrimSpace(userID)
		params = &gen.DashboardsDownloadParams{UserId: &v}
	}
	return c.api.DashboardsDownloadWithBodyWithResponse(ctx, identifier, params, "application/json", bytes.NewReader(body))
}

func (c *Client) DashboardDownloadStatus(ctx context.Context, identifier string, jobID string, userID string) (*gen.DashboardsDownloadStatusResp, error) {
	var params *gen.DashboardsDownloadStatusParams
	if strings.TrimSpace(userID) != "" {
		v := strings.TrimSpace(userID)
		params = &gen.DashboardsDownloadStatusParams{UserId: &v}
	}
	return c.api.DashboardsDownloadStatusWithResponse(ctx, identifier, jobID, params)
}

func (c *Client) DashboardDownloadFile(ctx context.Context, identifier string, jobID string, userID string) (*gen.DashboardsDownloadFileResp, error) {
	var params *gen.DashboardsDownloadFileParams
	if strings.TrimSpace(userID) != "" {
		v := strings.TrimSpace(userID)
		params = &gen.DashboardsDownloadFileParams{UserId: &v}
	}
	return c.api.DashboardsDownloadFileWithResponse(ctx, identifier, jobID, params)
}

func (c *Client) GetDashboardFilters(ctx context.Context, identifier string, userID string) (*gen.DashboardsGetFiltersResp, error) {
	var params *gen.DashboardsGetFiltersParams
	if strings.TrimSpace(userID) != "" {
		v := strings.TrimSpace(userID)
		params = &gen.DashboardsGetFiltersParams{UserId: &v}
	}
	return c.api.DashboardsGetFiltersWithResponse(ctx, identifier, params)
}

func (c *Client) UpdateDashboardFilters(ctx context.Context, identifier string, userID string, body []byte) (*gen.DashboardsUpdateFiltersResp, error) {
	var params *gen.DashboardsUpdateFiltersParams
	if strings.TrimSpace(userID) != "" {
		v := strings.TrimSpace(userID)
		params = &gen.DashboardsUpdateFiltersParams{UserId: &v}
	}
	return c.api.DashboardsUpdateFiltersWithBodyWithResponse(ctx, identifier, params, "application/json", bytes.NewReader(body))
}

func (c *Client) GenerateEmbedSSOSession(ctx context.Context, body []byte) (*gen.EmbedSsoGenerateSessionResp, error) {
	return c.api.EmbedSsoGenerateSessionWithBodyWithResponse(ctx, "application/json", bytes.NewReader(body))
}

func (c *Client) ListUserAttributes(ctx context.Context) (*gen.UserAttributesListResp, error) {
	return c.api.UserAttributesListWithResponse(ctx)
}

func (c *Client) ExportUnstableDocument(ctx context.Context, identifier string) (*gen.UnstableDocumentsExportResp, error) {
	return c.api.UnstableDocumentsExportWithResponse(ctx, identifier)
}

func (c *Client) ImportUnstableDocument(ctx context.Context, body []byte) (*gen.UnstableDocumentsImportResp, error) {
	return c.api.UnstableDocumentsImportWithBodyWithResponse(ctx, "application/json", bytes.NewReader(body))
}

func ParseBody(body []byte) any {
	if len(body) == 0 {
		return map[string]any{}
	}
	var out any
	if err := json.Unmarshal(body, &out); err != nil {
		return map[string]any{"raw": string(body)}
	}
	return out
}
