package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/omni-co/omni-cli/internal/client"
)

func runDocuments(rt *runtime, args []string) int {
	if len(args) == 0 {
		printDocumentsUsage()
		return 0
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "list":
		return runDocumentsList(rt, subArgs)
	case "get":
		return runDocumentsGet(rt, subArgs)
	case "create":
		return runDocumentsCreate(rt, subArgs)
	case "delete", "rm":
		return runDocumentsDelete(rt, subArgs)
	case "rename":
		return runDocumentsRename(rt, subArgs)
	case "move":
		return runDocumentsMove(rt, subArgs)
	case "draft":
		return runDocumentsDraft(rt, subArgs)
	case "duplicate":
		return runDocumentsDuplicate(rt, subArgs)
	case "favorite":
		return runDocumentsFavorite(rt, subArgs)
	case "access":
		return runDocumentsAccess(rt, subArgs)
	case "permissions", "perm":
		return runDocumentsPermissions(rt, subArgs)
	case "label", "labels":
		return runDocumentsLabels(rt, subArgs)
	case "queries":
		return runDocumentsQueries(rt, subArgs)
	case "transfer-ownership":
		return runDocumentsTransferOwnership(rt, subArgs)
	default:
		printDocumentsUsage()
		return usageFail(rt, fmt.Sprintf("unknown documents subcommand: %s", sub))
	}
}

func runDocumentsList(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("documents list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var cursor string
	var pageSize int
	fs.StringVar(&cursor, "cursor", "", "Pagination cursor")
	fs.IntVar(&pageSize, "page-size", 20, "Number of records per page")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printDocumentsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.ListDocuments(ctx, cursor, pageSize)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "documents list request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "documents list", resp.Body)
}

func runDocumentsGet(rt *runtime, args []string) int {
	if len(args) != 1 {
		return usageFail(rt, "usage: omni documents get <identifier>")
	}
	identifier := args[0]

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.GetDocument(ctx, identifier)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "documents get request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "documents get", resp.Body)
}

func runDocumentsCreate(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("documents create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var filePath string
	fs.StringVar(&filePath, "file", "", "Path to documents create JSON body")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printDocumentsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if filePath == "" {
		return usageFail(rt, "usage: omni documents create --file <json-path>")
	}

	payload, err := readJSONFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read documents create body", map[string]any{"error": err.Error()})
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := api.CreateDocument(ctx, payload)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "documents create request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "documents create", resp.Body)
}

func runDocumentsDelete(rt *runtime, args []string) int {
	if len(args) != 1 {
		return usageFail(rt, "usage: omni documents delete <identifier>")
	}
	identifier := strings.TrimSpace(args[0])
	if identifier == "" {
		return usageFail(rt, "document identifier is required")
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.DeleteDocument(ctx, identifier)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "documents delete request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "documents delete", resp.Body)
}

func runDocumentsRename(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("documents rename", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var name string
	var clearExistingDraft bool
	fs.StringVar(&name, "name", "", "New document name")
	fs.BoolVar(&clearExistingDraft, "clear-existing-draft", false, "Clear existing draft before rename")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printDocumentsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 {
		return usageFail(rt, "usage: omni documents rename <identifier> --name <new-name> [--clear-existing-draft]")
	}
	identifier := strings.TrimSpace(fs.Arg(0))
	if identifier == "" {
		return usageFail(rt, "document identifier is required")
	}
	if strings.TrimSpace(name) == "" {
		return usageFail(rt, "--name is required")
	}

	body := map[string]any{
		"name": strings.TrimSpace(name),
	}
	if clearExistingDraft {
		body["clearExistingDraft"] = true
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fail(rt, 1, codeValidationError, "failed to encode documents rename body", map[string]any{"error": err.Error()})
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.RenameDocument(ctx, identifier, payload)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "documents rename request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "documents rename", resp.Body)
}

func runDocumentsMove(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("documents move", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var filePath string
	fs.StringVar(&filePath, "file", "", "Path to documents move JSON body")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printDocumentsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 || strings.TrimSpace(filePath) == "" {
		return usageFail(rt, "usage: omni documents move <identifier> --file <json-path>")
	}
	identifier := strings.TrimSpace(fs.Arg(0))
	if identifier == "" {
		return usageFail(rt, "document identifier is required")
	}

	payload, err := readJSONFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read documents move body", map[string]any{"error": err.Error()})
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.MoveDocument(ctx, identifier, payload)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "documents move request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "documents move", resp.Body)
}

func runDocumentsDraft(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni documents draft <create|discard> ...")
	}
	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "create":
		return runDocumentsDraftMutation(rt, subArgs, "create")
	case "discard":
		return runDocumentsDraftMutation(rt, subArgs, "discard")
	default:
		return usageFail(rt, fmt.Sprintf("unknown documents draft subcommand: %s", sub))
	}
}

func runDocumentsDraftMutation(rt *runtime, args []string, action string) int {
	fs := flag.NewFlagSet("documents draft "+action, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var filePath string
	fs.StringVar(&filePath, "file", "", "Path to draft JSON body")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printDocumentsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 {
		return usageFail(rt, fmt.Sprintf("usage: omni documents draft %s <identifier> [--file <json-path>]", action))
	}
	identifier := strings.TrimSpace(fs.Arg(0))
	if identifier == "" {
		return usageFail(rt, "document identifier is required")
	}

	payload := []byte("{}")
	var err error
	if strings.TrimSpace(filePath) != "" {
		payload, err = readJSONFile(filePath)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to read draft body", map[string]any{"error": err.Error()})
		}
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if action == "create" {
		resp, reqErr := api.CreateDocumentDraft(ctx, identifier, payload)
		if reqErr != nil {
			return fail(rt, 1, codeNetworkError, "documents draft create request failed", map[string]any{"error": reqErr.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "documents draft create", resp.Body)
	}
	resp, reqErr := api.DiscardDocumentDraft(ctx, identifier, payload)
	if reqErr != nil {
		return fail(rt, 1, codeNetworkError, "documents draft discard request failed", map[string]any{"error": reqErr.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "documents draft discard", resp.Body)
}

func runDocumentsDuplicate(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("documents duplicate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var filePath string
	var userIDArg string
	fs.StringVar(&filePath, "file", "", "Path to documents duplicate JSON body")
	fs.StringVar(&userIDArg, "user-id", "", "Target user membership UUID for org-scoped API keys")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printDocumentsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 || strings.TrimSpace(filePath) == "" {
		return usageFail(rt, "usage: omni documents duplicate <identifier> --file <json-path> [--user-id <uuid>]")
	}
	identifier := strings.TrimSpace(fs.Arg(0))
	if identifier == "" {
		return usageFail(rt, "document identifier is required")
	}
	userID, err := parseOptionalUUIDArg(userIDArg, "user-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}
	payload, err := readJSONFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read duplicate body", map[string]any{"error": err.Error()})
	}
	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	resp, err := api.DuplicateDocument(ctx, identifier, payload, userID)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "documents duplicate request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "documents duplicate", resp.Body)
}

func runDocumentsFavorite(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni documents favorite <add|remove> ...")
	}
	sub := args[0]
	subArgs := args[1:]

	fs := flag.NewFlagSet("documents favorite "+sub, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var userIDArg string
	fs.StringVar(&userIDArg, "user-id", "", "Target user membership UUID for org-scoped API keys")
	if err := fs.Parse(subArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printDocumentsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 {
		return usageFail(rt, fmt.Sprintf("usage: omni documents favorite %s <identifier> [--user-id <uuid>]", sub))
	}
	identifier := strings.TrimSpace(fs.Arg(0))
	if identifier == "" {
		return usageFail(rt, "document identifier is required")
	}
	userID, err := parseOptionalUUIDArg(userIDArg, "user-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}
	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	switch sub {
	case "add":
		resp, reqErr := api.AddDocumentFavorite(ctx, identifier, userID)
		if reqErr != nil {
			return fail(rt, 1, codeNetworkError, "documents favorite add request failed", map[string]any{"error": reqErr.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "documents favorite add", resp.Body)
	case "remove", "rm":
		resp, reqErr := api.RemoveDocumentFavorite(ctx, identifier, userID)
		if reqErr != nil {
			return fail(rt, 1, codeNetworkError, "documents favorite remove request failed", map[string]any{"error": reqErr.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "documents favorite remove", resp.Body)
	default:
		return usageFail(rt, fmt.Sprintf("unknown documents favorite subcommand: %s", sub))
	}
}

func runDocumentsAccess(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni documents access list <identifier> [--cursor <cursor>] [--page-size 20] [--type user|userGroup] [--access-source direct|folder]")
	}
	sub := args[0]
	subArgs := args[1:]
	switch sub {
	case "list":
		return runDocumentsAccessList(rt, subArgs)
	default:
		return usageFail(rt, fmt.Sprintf("unknown documents access subcommand: %s", sub))
	}
}

func runDocumentsAccessList(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("documents access list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var cursor string
	var pageSize int
	var principalType string
	var accessSource string
	var sortField string
	var sortDirection string

	fs.StringVar(&cursor, "cursor", "", "Pagination cursor")
	fs.IntVar(&pageSize, "page-size", 20, "Number of records per page")
	fs.StringVar(&principalType, "type", "", "Principal type filter: user|userGroup")
	fs.StringVar(&accessSource, "access-source", "", "Access source filter: direct|folder")
	fs.StringVar(&sortField, "sort-field", "", "Sort field")
	fs.StringVar(&sortDirection, "sort-direction", "", "Sort direction: asc|desc")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printDocumentsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 {
		return usageFail(rt, "usage: omni documents access list <identifier> [--cursor <cursor>] [--page-size 20] [--type user|userGroup] [--access-source direct|folder]")
	}
	if principalType != "" && principalType != "user" && principalType != "userGroup" {
		return usageFail(rt, "--type must be user or userGroup")
	}
	if accessSource != "" && accessSource != "direct" && accessSource != "folder" {
		return usageFail(rt, "--access-source must be direct or folder")
	}
	if sortDirection != "" && sortDirection != "asc" && sortDirection != "desc" {
		return usageFail(rt, "--sort-direction must be asc or desc")
	}

	identifier := strings.TrimSpace(fs.Arg(0))
	if identifier == "" {
		return usageFail(rt, "document identifier is required")
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.ListDocumentAccess(ctx, identifier, cursor, pageSize, principalType, accessSource, sortField, sortDirection)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "documents access list request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "documents access list", resp.Body)
}

func runDocumentsPermissions(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni documents permissions <get|add|update|revoke> ...")
	}
	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "get":
		return runDocumentsPermissionsGet(rt, subArgs)
	case "add":
		return runDocumentsPermissionsAdd(rt, subArgs)
	case "update":
		return runDocumentsPermissionsUpdate(rt, subArgs)
	case "revoke":
		return runDocumentsPermissionsRevoke(rt, subArgs)
	case "settings":
		return runDocumentsPermissionsSettings(rt, subArgs)
	default:
		return usageFail(rt, fmt.Sprintf("unknown documents permissions subcommand: %s", sub))
	}
}

func runDocumentsPermissionsGet(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("documents permissions get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var userIDArg string
	fs.StringVar(&userIDArg, "user-id", "", "User membership UUID")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printDocumentsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 || strings.TrimSpace(userIDArg) == "" {
		return usageFail(rt, "usage: omni documents permissions get <identifier> --user-id <uuid>")
	}

	identifier := strings.TrimSpace(fs.Arg(0))
	if identifier == "" {
		return usageFail(rt, "document identifier is required")
	}
	userID, err := parseUUIDArg(userIDArg, "user-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.GetDocumentPermissions(ctx, identifier, userID)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "documents permissions get request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "documents permissions get", resp.Body)
}

func runDocumentsPermissionsAdd(rt *runtime, args []string) int {
	return runDocumentsPermissionMutation(rt, args, "add")
}

func runDocumentsPermissionsUpdate(rt *runtime, args []string) int {
	return runDocumentsPermissionMutation(rt, args, "update")
}

func runDocumentsPermissionsRevoke(rt *runtime, args []string) int {
	return runDocumentsPermissionMutation(rt, args, "revoke")
}

func runDocumentsPermissionsSettings(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("documents permissions settings", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var filePath string
	fs.StringVar(&filePath, "file", "", "Path to permission settings JSON body")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printDocumentsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 || strings.TrimSpace(filePath) == "" {
		return usageFail(rt, "usage: omni documents permissions settings <identifier> --file <json-path>")
	}
	identifier := strings.TrimSpace(fs.Arg(0))
	if identifier == "" {
		return usageFail(rt, "document identifier is required")
	}
	payload, err := readJSONFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read permission settings body", map[string]any{"error": err.Error()})
	}
	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	resp, err := api.UpdateDocumentPermissionSettings(ctx, identifier, payload)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "documents permissions settings request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "documents permissions settings", resp.Body)
}

func runDocumentsPermissionMutation(rt *runtime, args []string, action string) int {
	fs := flag.NewFlagSet("documents permissions "+action, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var filePath string
	fs.StringVar(&filePath, "file", "", "Path to permissions JSON body")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printDocumentsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 || strings.TrimSpace(filePath) == "" {
		return usageFail(rt, fmt.Sprintf("usage: omni documents permissions %s <identifier> --file <json-path>", action))
	}
	identifier := strings.TrimSpace(fs.Arg(0))
	if identifier == "" {
		return usageFail(rt, "document identifier is required")
	}

	payload, err := readJSONFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read permissions body", map[string]any{"error": err.Error()})
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	switch action {
	case "add":
		resp, reqErr := api.AddDocumentPermits(ctx, identifier, payload)
		if reqErr != nil {
			return fail(rt, 1, codeNetworkError, "documents permissions add request failed", map[string]any{"error": reqErr.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "documents permissions add", resp.Body)
	case "update":
		resp, reqErr := api.UpdateDocumentPermits(ctx, identifier, payload)
		if reqErr != nil {
			return fail(rt, 1, codeNetworkError, "documents permissions update request failed", map[string]any{"error": reqErr.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "documents permissions update", resp.Body)
	default:
		resp, reqErr := api.RevokeDocumentPermits(ctx, identifier, payload)
		if reqErr != nil {
			return fail(rt, 1, codeNetworkError, "documents permissions revoke request failed", map[string]any{"error": reqErr.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "documents permissions revoke", resp.Body)
	}
}

func runDocumentsLabels(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni documents label <add|remove> <identifier> <label> [--user-id <uuid>]")
	}
	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "add":
		return runDocumentsLabelAdd(rt, subArgs)
	case "remove", "rm":
		return runDocumentsLabelRemove(rt, subArgs)
	case "bulk-update":
		return runDocumentsLabelsBulkUpdate(rt, subArgs)
	default:
		return usageFail(rt, fmt.Sprintf("unknown documents label subcommand: %s", sub))
	}
}

func runDocumentsLabelAdd(rt *runtime, args []string) int {
	return runDocumentsLabelMutation(rt, args, "add")
}

func runDocumentsLabelRemove(rt *runtime, args []string) int {
	return runDocumentsLabelMutation(rt, args, "remove")
}

func runDocumentsLabelsBulkUpdate(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("documents labels bulk-update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var filePath string
	var userIDArg string
	fs.StringVar(&filePath, "file", "", "Path to bulk label update JSON body")
	fs.StringVar(&userIDArg, "user-id", "", "Target user membership UUID for org-scoped API keys")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printDocumentsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 || strings.TrimSpace(filePath) == "" {
		return usageFail(rt, "usage: omni documents labels bulk-update <identifier> --file <json-path> [--user-id <uuid>]")
	}
	identifier := strings.TrimSpace(fs.Arg(0))
	if identifier == "" {
		return usageFail(rt, "document identifier is required")
	}
	userID, err := parseOptionalUUIDArg(userIDArg, "user-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}
	payload, err := readJSONFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read bulk labels body", map[string]any{"error": err.Error()})
	}
	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	resp, err := api.BulkUpdateDocumentLabels(ctx, identifier, payload, userID)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "documents labels bulk-update request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "documents labels bulk-update", resp.Body)
}

func runDocumentsLabelMutation(rt *runtime, args []string, action string) int {
	fs := flag.NewFlagSet("documents label "+action, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var userIDArg string
	fs.StringVar(&userIDArg, "user-id", "", "Target user membership UUID for org-scoped API keys")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printDocumentsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 2 {
		return usageFail(rt, fmt.Sprintf("usage: omni documents label %s <identifier> <label> [--user-id <uuid>]", action))
	}
	identifier := strings.TrimSpace(fs.Arg(0))
	label := strings.TrimSpace(fs.Arg(1))
	if identifier == "" || label == "" {
		return usageFail(rt, "document identifier and label are required")
	}
	userID, err := parseOptionalUUIDArg(userIDArg, "user-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if action == "add" {
		resp, reqErr := api.AddDocumentLabel(ctx, identifier, label, userID)
		if reqErr != nil {
			return fail(rt, 1, codeNetworkError, "documents label add request failed", map[string]any{"error": reqErr.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "documents label add", resp.Body)
	}

	resp, reqErr := api.RemoveDocumentLabel(ctx, identifier, label, userID)
	if reqErr != nil {
		return fail(rt, 1, codeNetworkError, "documents label remove request failed", map[string]any{"error": reqErr.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "documents label remove", resp.Body)
}

func runDocumentsQueries(rt *runtime, args []string) int {
	if len(args) != 1 {
		return usageFail(rt, "usage: omni documents queries <identifier>")
	}
	identifier := strings.TrimSpace(args[0])
	if identifier == "" {
		return usageFail(rt, "document identifier is required")
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.ListDocumentQueries(ctx, identifier)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "documents queries request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "documents queries", resp.Body)
}

func runDocumentsTransferOwnership(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("documents transfer-ownership", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var userIDArg string
	var filePath string
	fs.StringVar(&userIDArg, "user-id", "", "Membership UUID of new owner")
	fs.StringVar(&filePath, "file", "", "Path to transfer ownership JSON body (alternative to --user-id)")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printDocumentsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 {
		return usageFail(rt, "usage: omni documents transfer-ownership <identifier> (--user-id <uuid> | --file <json-path>)")
	}
	if strings.TrimSpace(userIDArg) == "" && strings.TrimSpace(filePath) == "" {
		return usageFail(rt, "usage: omni documents transfer-ownership <identifier> (--user-id <uuid> | --file <json-path>)")
	}
	if strings.TrimSpace(userIDArg) != "" && strings.TrimSpace(filePath) != "" {
		return usageFail(rt, "provide either --user-id or --file, not both")
	}
	identifier := strings.TrimSpace(fs.Arg(0))
	if identifier == "" {
		return usageFail(rt, "document identifier is required")
	}

	var payload []byte
	var err error
	if strings.TrimSpace(filePath) != "" {
		payload, err = readJSONFile(filePath)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to read transfer ownership body", map[string]any{"error": err.Error()})
		}
	} else {
		userID, parseErr := parseUUIDArg(userIDArg, "user-id")
		if parseErr != nil {
			return usageFail(rt, parseErr.Error())
		}
		payload, err = json.Marshal(map[string]any{"userId": userID})
		if err != nil {
			return fail(rt, 1, codeValidationError, "failed to encode transfer ownership body", map[string]any{"error": err.Error()})
		}
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.TransferDocumentOwnership(ctx, identifier, payload)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "documents transfer-ownership request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "documents transfer-ownership", resp.Body)
}

func printDocumentsUsage() {
	fmt.Print(`omni documents commands:
  omni documents list [--cursor <cursor>] [--page-size 20]
  omni documents get <identifier>
  omni documents create --file <json-path>
  omni documents delete <identifier>
  omni documents rename <identifier> --name <new-name> [--clear-existing-draft]
  omni documents move <identifier> --file <json-path>
  omni documents draft create <identifier> [--file <json-path>]
  omni documents draft discard <identifier> [--file <json-path>]
  omni documents duplicate <identifier> --file <json-path> [--user-id <uuid>]
  omni documents favorite add <identifier> [--user-id <uuid>]
  omni documents favorite remove <identifier> [--user-id <uuid>]
  omni documents access list <identifier> [--cursor <cursor>] [--page-size 20] [--type user|userGroup] [--access-source direct|folder]
  omni documents permissions get <identifier> --user-id <uuid>
  omni documents permissions add <identifier> --file <json-path>
  omni documents permissions update <identifier> --file <json-path>
  omni documents permissions revoke <identifier> --file <json-path>
  omni documents permissions settings <identifier> --file <json-path>
  omni documents label add <identifier> <label> [--user-id <uuid>]
  omni documents label remove <identifier> <label> [--user-id <uuid>]
  omni documents labels bulk-update <identifier> --file <json-path> [--user-id <uuid>]
  omni documents queries <identifier>
  omni documents transfer-ownership <identifier> (--user-id <uuid> | --file <json-path>)
`)
}
