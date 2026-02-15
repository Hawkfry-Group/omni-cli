package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omni-co/omni-cli/internal/client"
)

func runFolders(rt *runtime, args []string) int {
	if len(args) == 0 {
		printFoldersUsage()
		return 0
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "list":
		return runFoldersList(rt, subArgs)
	case "create":
		return runFoldersCreate(rt, subArgs)
	case "delete", "rm":
		return runFoldersDelete(rt, subArgs)
	case "permissions", "perm":
		return runFoldersPermissions(rt, subArgs)
	default:
		printFoldersUsage()
		return usageFail(rt, fmt.Sprintf("unknown folders subcommand: %s", sub))
	}
}

func runFoldersList(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("folders list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var cursor, path string
	var pageSize int
	fs.StringVar(&cursor, "cursor", "", "Pagination cursor")
	fs.IntVar(&pageSize, "page-size", 20, "Number of records per page")
	fs.StringVar(&path, "path", "", "Filter by exact folder path")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printFoldersUsage()
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

	resp, err := api.ListFolders(ctx, cursor, pageSize, path)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "folders list request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "folders list", resp.Body)
}

func runFoldersCreate(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("folders create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var parentIDArg, userIDArg, scope string
	fs.StringVar(&parentIDArg, "parent-folder-id", "", "Optional parent folder UUID")
	fs.StringVar(&userIDArg, "user-id", "", "Optional user UUID (org key)")
	fs.StringVar(&scope, "scope", "", "Optional scope (organization|restricted)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printFoldersUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 {
		return usageFail(rt, "usage: omni folders create <name> [--parent-folder-id <uuid>] [--scope organization|restricted] [--user-id <uuid>]")
	}

	name := strings.TrimSpace(fs.Arg(0))
	if name == "" {
		return usageFail(rt, "folder name is required")
	}

	var parentIDPtr *uuid.UUID
	if strings.TrimSpace(parentIDArg) != "" {
		id, err := parseUUIDArg(parentIDArg, "parent-folder-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		parentIDPtr = &id
	}

	var userIDPtr *uuid.UUID
	if strings.TrimSpace(userIDArg) != "" {
		id, err := parseUUIDArg(userIDArg, "user-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		userIDPtr = &id
	}

	if scope != "" && scope != "organization" && scope != "restricted" {
		return usageFail(rt, "--scope must be organization or restricted")
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.CreateFolder(ctx, name, parentIDPtr, scope, userIDPtr)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "folders create request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "folders create", resp.Body)
}

func runFoldersDelete(rt *runtime, args []string) int {
	if len(args) != 1 {
		return usageFail(rt, "usage: omni folders delete <folder-id>")
	}
	folderID, err := parseUUIDArg(args[0], "folder-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.DeleteFolder(ctx, folderID)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "folders delete request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "folders delete", resp.Body)
}

func runFoldersPermissions(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni folders permissions <get|add|update|revoke> ...")
	}
	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "get":
		return runFoldersPermissionsGet(rt, subArgs)
	case "add":
		return runFoldersPermissionsMutation(rt, subArgs, "add")
	case "update":
		return runFoldersPermissionsMutation(rt, subArgs, "update")
	case "revoke":
		return runFoldersPermissionsMutation(rt, subArgs, "revoke")
	default:
		return usageFail(rt, fmt.Sprintf("unknown folders permissions subcommand: %s", sub))
	}
}

func runFoldersPermissionsGet(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("folders permissions get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var userIDArg string
	fs.StringVar(&userIDArg, "user-id", "", "Optional user UUID filter")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printFoldersUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 {
		return usageFail(rt, "usage: omni folders permissions get <folder-id> [--user-id <uuid>]")
	}

	folderID, err := parseUUIDArg(fs.Arg(0), "folder-id")
	if err != nil {
		return usageFail(rt, err.Error())
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

	resp, err := api.GetFolderPermissions(ctx, folderID, userID)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "folders permissions get request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "folders permissions get", resp.Body)
}

func runFoldersPermissionsMutation(rt *runtime, args []string, action string) int {
	fs := flag.NewFlagSet("folders permissions "+action, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var filePath string
	fs.StringVar(&filePath, "file", "", "Path to permissions JSON body")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printFoldersUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 || strings.TrimSpace(filePath) == "" {
		return usageFail(rt, fmt.Sprintf("usage: omni folders permissions %s <folder-id> --file <json-path>", action))
	}
	folderID, err := parseUUIDArg(fs.Arg(0), "folder-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}

	payload, err := readJSONFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read folder permissions body", map[string]any{"error": err.Error()})
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	switch action {
	case "add":
		resp, reqErr := api.AddFolderPermissions(ctx, folderID, payload)
		if reqErr != nil {
			return fail(rt, 1, codeNetworkError, "folders permissions add request failed", map[string]any{"error": reqErr.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "folders permissions add", resp.Body)
	case "update":
		resp, reqErr := api.UpdateFolderPermissions(ctx, folderID, payload)
		if reqErr != nil {
			return fail(rt, 1, codeNetworkError, "folders permissions update request failed", map[string]any{"error": reqErr.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "folders permissions update", resp.Body)
	default:
		resp, reqErr := api.RevokeFolderPermissions(ctx, folderID, payload)
		if reqErr != nil {
			return fail(rt, 1, codeNetworkError, "folders permissions revoke request failed", map[string]any{"error": reqErr.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "folders permissions revoke", resp.Body)
	}
}

func printFoldersUsage() {
	fmt.Print(`omni folders commands:
  omni folders list [--cursor <cursor>] [--page-size 20] [--path <folder-path>]
  omni folders create <name> [--parent-folder-id <uuid>] [--scope organization|restricted] [--user-id <uuid>]
  omni folders delete <folder-id>
  omni folders permissions get <folder-id> [--user-id <uuid>]
  omni folders permissions add <folder-id> --file <json-path>
  omni folders permissions update <folder-id> --file <json-path>
  omni folders permissions revoke <folder-id> --file <json-path>
`)
}
