# Endpoint Coverage Report

- Generated: `2026-02-15T01:34:49Z`
- Total OpenAPI endpoints: `130`
- Typed CLI-covered endpoints: `130`
- Missing typed coverage: `0`
- Typed coverage: `100.0%`
- Note: all missing typed endpoints are still callable via `omni api call`.

## Missing Typed Coverage (Gap List)

| Method | Path | Operation ID | Tags |
| --- | --- | --- | --- |

## Typed-Covered Endpoints

| Method | Path | Operation ID | Source Wrapper(s) |
| --- | --- | --- | --- |
| `GET` | `/api/scim/v2/Groups` | `scimGroupsList` | `ListSCIMGroups` |
| `POST` | `/api/scim/v2/Groups` | `scimGroupsCreate` | `CreateSCIMGroup` |
| `DELETE` | `/api/scim/v2/Groups/{miniUuid}` | `scimGroupsDelete` | `DeleteSCIMGroup` |
| `GET` | `/api/scim/v2/Groups/{miniUuid}` | `scimGroupsGet` | `GetSCIMGroup` |
| `PATCH` | `/api/scim/v2/Groups/{miniUuid}` | `scimGroupsUpdate` | `UpdateSCIMGroup` |
| `PUT` | `/api/scim/v2/Groups/{miniUuid}` | `scimGroupsReplace` | `ReplaceSCIMGroup` |
| `GET` | `/api/scim/v2/Users` | `scimUsersList` | `AdminProbe, ListSCIMUsers` |
| `POST` | `/api/scim/v2/Users` | `scimUsersCreate` | `CreateSCIMUser` |
| `DELETE` | `/api/scim/v2/Users/{id}` | `scimUsersDelete` | `DeleteSCIMUser` |
| `GET` | `/api/scim/v2/Users/{id}` | `scimUsersGet` | `GetSCIMUser` |
| `PATCH` | `/api/scim/v2/Users/{id}` | `scimUsersUpdate` | `UpdateSCIMUser` |
| `PUT` | `/api/scim/v2/Users/{id}` | `scimUsersReplace` | `ReplaceSCIMUser` |
| `GET` | `/api/scim/v2/embed/Users` | `scimEmbedUsersList` | `ListSCIMEmbedUsers` |
| `DELETE` | `/api/scim/v2/embed/Users/{id}` | `scimEmbedUsersDelete` | `DeleteSCIMEmbedUser` |
| `GET` | `/api/scim/v2/embed/Users/{id}` | `scimEmbedUsersGet` | `GetSCIMEmbedUser` |
| `POST` | `/api/unstable/documents/import` | `unstableDocumentsImport` | `ImportUnstableDocument` |
| `GET` | `/api/unstable/documents/{identifier}/export` | `unstableDocumentsExport` | `ExportUnstableDocument` |
| `POST` | `/api/v1/agentic/jobs` | `agenticJobSubmit` | `SubmitAgenticJob` |
| `GET` | `/api/v1/agentic/jobs/{jobId}` | `agenticJobStatus` | `GetAgenticJobStatus` |
| `POST` | `/api/v1/agentic/jobs/{jobId}/cancel` | `agenticJobCancel` | `CancelAgenticJob` |
| `GET` | `/api/v1/agentic/jobs/{jobId}/result` | `agenticJobResult` | `GetAgenticJobResult` |
| `POST` | `/api/v1/ai/generate-query` | `aiGenerateQuery` | `AIGenerateQuery` |
| `POST` | `/api/v1/ai/pick-topic` | `aiPickTopic` | `AIPickTopic` |
| `GET` | `/api/v1/connection-environments` | `connectionEnvironmentsList` | `ListConnectionEnvironments` |
| `POST` | `/api/v1/connection-environments` | `connectionEnvironmentsCreate` | `CreateConnectionEnvironment` |
| `DELETE` | `/api/v1/connection-environments/{id}` | `connectionEnvironmentsDelete` | `DeleteConnectionEnvironment` |
| `PUT` | `/api/v1/connection-environments/{id}` | `connectionEnvironmentsUpdate` | `UpdateConnectionEnvironment` |
| `GET` | `/api/v1/connections` | `connectionsList` | `ListConnections` |
| `POST` | `/api/v1/connections` | `connectionsCreate` | `CreateConnection` |
| `DELETE` | `/api/v1/connections/{connectionId}/dbt` | `connectionsDbtDelete` | `DeleteConnectionDBT` |
| `GET` | `/api/v1/connections/{connectionId}/dbt` | `connectionsDbtGet` | `GetConnectionDBT` |
| `PUT` | `/api/v1/connections/{connectionId}/dbt` | `connectionsDbtUpdate` | `UpdateConnectionDBT` |
| `GET` | `/api/v1/connections/{connectionId}/schedules` | `connectionsSchedulesList` | `ListConnectionSchedules` |
| `POST` | `/api/v1/connections/{connectionId}/schedules` | `connectionsSchedulesCreate` | `CreateConnectionSchedule` |
| `DELETE` | `/api/v1/connections/{connectionId}/schedules/{scheduleId}` | `connectionsSchedulesDelete` | `DeleteConnectionSchedule` |
| `GET` | `/api/v1/connections/{connectionId}/schedules/{scheduleId}` | `connectionsSchedulesGet` | `GetConnectionSchedule` |
| `PUT` | `/api/v1/connections/{connectionId}/schedules/{scheduleId}` | `connectionsSchedulesUpdate` | `UpdateConnectionSchedule` |
| `PATCH` | `/api/v1/connections/{id}` | `connectionsUpdate` | `UpdateConnection` |
| `GET` | `/api/v1/content` | `contentList` | `BaseProbe` |
| `POST` | `/api/v1/dashboards/{identifier}/download` | `dashboardsDownload` | `DashboardDownload` |
| `GET` | `/api/v1/dashboards/{identifier}/download/{jobId}` | `dashboardsDownloadFile` | `DashboardDownloadFile` |
| `GET` | `/api/v1/dashboards/{identifier}/download/{jobId}/status` | `dashboardsDownloadStatus` | `DashboardDownloadStatus` |
| `GET` | `/api/v1/dashboards/{identifier}/filters` | `dashboardsGetFilters` | `GetDashboardFilters` |
| `PATCH` | `/api/v1/dashboards/{identifier}/filters` | `dashboardsUpdateFilters` | `UpdateDashboardFilters` |
| `GET` | `/api/v1/documents` | `documentsList` | `ListDocuments` |
| `POST` | `/api/v1/documents` | `documentsCreate` | `CreateDocument` |
| `DELETE` | `/api/v1/documents/{identifier}` | `documentsDelete` | `DeleteDocument` |
| `GET` | `/api/v1/documents/{identifier}` | `documentsGet` | `GetDocument` |
| `PATCH` | `/api/v1/documents/{identifier}` | `documentsRename` | `RenameDocument` |
| `GET` | `/api/v1/documents/{identifier}/access-list` | `documentsAccessList` | `ListDocumentAccess` |
| `DELETE` | `/api/v1/documents/{identifier}/draft` | `documentsDiscardDraft` | `DiscardDocumentDraft` |
| `POST` | `/api/v1/documents/{identifier}/draft` | `documentsCreateDraft` | `CreateDocumentDraft` |
| `POST` | `/api/v1/documents/{identifier}/duplicate` | `documentsDuplicate` | `DuplicateDocument` |
| `DELETE` | `/api/v1/documents/{identifier}/favorite` | `documentsRemoveFavorite` | `RemoveDocumentFavorite` |
| `PUT` | `/api/v1/documents/{identifier}/favorite` | `documentsAddFavorite` | `AddDocumentFavorite` |
| `PATCH` | `/api/v1/documents/{identifier}/labels` | `documentsBulkUpdateLabels` | `BulkUpdateDocumentLabels` |
| `DELETE` | `/api/v1/documents/{identifier}/labels/{labelName}` | `documentsRemoveLabel` | `RemoveDocumentLabel` |
| `PUT` | `/api/v1/documents/{identifier}/labels/{labelName}` | `documentsAddLabel` | `AddDocumentLabel` |
| `POST` | `/api/v1/documents/{identifier}/move` | `documentsMove` | `MoveDocument` |
| `DELETE` | `/api/v1/documents/{identifier}/permissions` | `documentsRevokePermits` | `RevokeDocumentPermits` |
| `GET` | `/api/v1/documents/{identifier}/permissions` | `documentsGetPermissions` | `GetDocumentPermissions` |
| `PATCH` | `/api/v1/documents/{identifier}/permissions` | `documentsUpdatePermits` | `UpdateDocumentPermits` |
| `POST` | `/api/v1/documents/{identifier}/permissions` | `documentsAddPermits` | `AddDocumentPermits` |
| `PUT` | `/api/v1/documents/{identifier}/permissions` | `documentsUpdatePermissionSettings` | `UpdateDocumentPermissionSettings` |
| `GET` | `/api/v1/documents/{identifier}/queries` | `documentsGetQueries` | `ListDocumentQueries` |
| `PUT` | `/api/v1/documents/{identifier}/transfer-ownership` | `documentsTransferOwnership` | `TransferDocumentOwnership` |
| `POST` | `/api/v1/embed/sso/generate-session` | `embedSsoGenerateSession` | `GenerateEmbedSSOSession` |
| `GET` | `/api/v1/folders` | `foldersList` | `ListFolders` |
| `POST` | `/api/v1/folders` | `foldersCreate` | `CreateFolder` |
| `DELETE` | `/api/v1/folders/{folderId}` | `foldersDelete` | `DeleteFolder` |
| `DELETE` | `/api/v1/folders/{folderId}/permissions` | `foldersRevokePermissions` | `RevokeFolderPermissions` |
| `GET` | `/api/v1/folders/{folderId}/permissions` | `foldersGetPermissions` | `GetFolderPermissions` |
| `PATCH` | `/api/v1/folders/{folderId}/permissions` | `foldersUpdatePermissions` | `UpdateFolderPermissions` |
| `POST` | `/api/v1/folders/{folderId}/permissions` | `foldersAddPermissions` | `AddFolderPermissions` |
| `GET` | `/api/v1/jobs/{jobId}/status` | `jobsGetStatus` | `GetJobStatus` |
| `GET` | `/api/v1/labels` | `labelsList` | `ListLabels` |
| `POST` | `/api/v1/labels` | `labelsCreate` | `CreateLabel` |
| `DELETE` | `/api/v1/labels/{name}` | `labelsDelete` | `DeleteLabel` |
| `GET` | `/api/v1/labels/{name}` | `labelsGet` | `GetLabel` |
| `PUT` | `/api/v1/labels/{name}` | `labelsUpdate` | `UpdateLabel` |
| `GET` | `/api/v1/models` | `modelsList` | `GetModelByID, ListModels` |
| `POST` | `/api/v1/models` | `modelsCreate` | `CreateModel` |
| `DELETE` | `/api/v1/models/{modelId}/branch/{branchName}` | `modelsDeleteBranch` | `DeleteModelBranch` |
| `POST` | `/api/v1/models/{modelId}/branch/{branchName}/merge` | `modelsMergeBranch` | `MergeModelBranch` |
| `POST` | `/api/v1/models/{modelId}/cache_reset/{policyName}` | `modelsCacheReset` | `ResetModelCache` |
| `GET` | `/api/v1/models/{modelId}/content-validator` | `modelsContentValidatorGet` | `GetModelContentValidator` |
| `POST` | `/api/v1/models/{modelId}/content-validator` | `modelsContentValidatorReplace` | `ReplaceModelContentValidator` |
| `POST` | `/api/v1/models/{modelId}/field` | `modelsCreateField` | `CreateModelField` |
| `DELETE` | `/api/v1/models/{modelId}/git` | `modelsGitDelete` | `DeleteModelGit` |
| `GET` | `/api/v1/models/{modelId}/git` | `modelsGitGet` | `GetModelGit` |
| `PATCH` | `/api/v1/models/{modelId}/git` | `modelsGitUpdate` | `UpdateModelGit` |
| `POST` | `/api/v1/models/{modelId}/git` | `modelsGitCreate` | `CreateModelGit` |
| `POST` | `/api/v1/models/{modelId}/git/sync` | `modelsGitSync` | `SyncModelGit` |
| `POST` | `/api/v1/models/{modelId}/migrate` | `modelsMigrate` | `MigrateModel` |
| `POST` | `/api/v1/models/{modelId}/refresh` | `modelsRefresh` | `RefreshModel` |
| `GET` | `/api/v1/models/{modelId}/topic` | `modelsListTopics` | `ListModelTopics` |
| `DELETE` | `/api/v1/models/{modelId}/topic/{topicName}` | `modelsDeleteTopic` | `DeleteModelTopic` |
| `GET` | `/api/v1/models/{modelId}/topic/{topicName}` | `modelsGetTopic` | `GetModelTopic` |
| `PATCH` | `/api/v1/models/{modelId}/topic/{topicName}` | `modelsUpdateTopic` | `UpdateModelTopic` |
| `GET` | `/api/v1/models/{modelId}/validate` | `modelsValidate` | `ValidateModel` |
| `GET` | `/api/v1/models/{modelId}/view` | `modelsGetViews` | `ListModelViews` |
| `DELETE` | `/api/v1/models/{modelId}/view/{viewName}` | `modelsDeleteView` | `DeleteModelView` |
| `PATCH` | `/api/v1/models/{modelId}/view/{viewName}` | `modelsUpdateView` | `UpdateModelView` |
| `DELETE` | `/api/v1/models/{modelId}/view/{viewName}/field/{fieldName}` | `modelsDeleteField` | `DeleteModelField` |
| `PATCH` | `/api/v1/models/{modelId}/view/{viewName}/field/{fieldName}` | `modelsUpdateField` | `UpdateModelField` |
| `DELETE` | `/api/v1/models/{modelId}/yaml` | `modelsYamlDelete` | `DeleteModelYAML` |
| `GET` | `/api/v1/models/{modelId}/yaml` | `modelsYamlGet` | `GetModelYAML` |
| `POST` | `/api/v1/models/{modelId}/yaml` | `modelsYamlCreate` | `CreateModelYAML` |
| `POST` | `/api/v1/query/run` | `queryRun` | `QueryProbe, RunQuery` |
| `GET` | `/api/v1/query/wait` | `queryWait` | `WaitForQueryResults` |
| `GET` | `/api/v1/schedules` | `schedulesList` | `ListSchedules` |
| `POST` | `/api/v1/schedules` | `schedulesCreate` | `CreateSchedule` |
| `DELETE` | `/api/v1/schedules/{scheduleId}` | `schedulesDelete` | `DeleteSchedule` |
| `GET` | `/api/v1/schedules/{scheduleId}` | `schedulesGet` | `GetSchedule` |
| `PUT` | `/api/v1/schedules/{scheduleId}` | `schedulesUpdate` | `UpdateSchedule` |
| `PUT` | `/api/v1/schedules/{scheduleId}/add-recipients` | `schedulesAddRecipients` | `AddScheduleRecipients` |
| `PUT` | `/api/v1/schedules/{scheduleId}/pause` | `schedulesPause` | `PauseSchedule` |
| `GET` | `/api/v1/schedules/{scheduleId}/recipients` | `schedulesRecipientsGet` | `GetScheduleRecipients` |
| `PUT` | `/api/v1/schedules/{scheduleId}/remove-recipients` | `schedulesRemoveRecipients` | `RemoveScheduleRecipients` |
| `PUT` | `/api/v1/schedules/{scheduleId}/resume` | `schedulesResume` | `ResumeSchedule` |
| `PUT` | `/api/v1/schedules/{scheduleId}/transfer-ownership` | `schedulesTransferOwnership` | `TransferScheduleOwnership` |
| `POST` | `/api/v1/schedules/{scheduleId}/trigger` | `schedulesTrigger` | `TriggerSchedule` |
| `GET` | `/api/v1/user-attributes` | `userAttributesList` | `ListUserAttributes` |
| `GET` | `/api/v1/user-groups/{id}/model-roles` | `userGroupsGetModelRoles` | `GetUserGroupModelRoles` |
| `POST` | `/api/v1/user-groups/{id}/model-roles` | `userGroupsAssignModelRole` | `AssignUserGroupModelRole` |
| `GET` | `/api/v1/users/email-only` | `usersListEmailOnly` | `ListEmailOnlyUsers` |
| `POST` | `/api/v1/users/email-only` | `usersCreateEmailOnly` | `CreateEmailOnlyUser` |
| `POST` | `/api/v1/users/email-only/bulk` | `usersCreateEmailOnlyBulk` | `CreateEmailOnlyUsersBulk` |
| `GET` | `/api/v1/users/{id}/model-roles` | `usersGetModelRoles` | `GetUserModelRoles` |
| `POST` | `/api/v1/users/{id}/model-roles` | `usersAssignModelRole` | `AssignUserModelRole` |

