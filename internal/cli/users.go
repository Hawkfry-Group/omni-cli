package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/omni-co/omni-cli/internal/client"
)

func runUsers(rt *runtime, args []string) int {
	if rt.Resolved.Profile.TokenType != "org" {
		return fail(rt, 1, codeAuthForbidden, "users commands require an org API key", map[string]any{"token_type": rt.Resolved.Profile.TokenType})
	}
	if len(args) == 0 {
		printUsersUsage()
		return 0
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "list-email-only":
		return runUsersListEmailOnly(rt, rest)
	case "create-email-only":
		return runUsersCreateEmailOnly(rt, rest)
	case "create-email-only-bulk":
		return runUsersCreateEmailOnlyBulk(rt, rest)
	case "roles":
		return runUsersRoles(rt, rest)
	case "group-roles":
		return runUserGroupRoles(rt, rest)
	default:
		printUsersUsage()
		return usageFail(rt, fmt.Sprintf("unknown users subcommand: %s", sub))
	}
}

func runUsersListEmailOnly(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("users list-email-only", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var cursor, email, sortDirection string
	var pageSize int
	fs.StringVar(&cursor, "cursor", "", "Pagination cursor")
	fs.StringVar(&email, "email", "", "Filter by email")
	fs.StringVar(&sortDirection, "sort-direction", "", "Sort direction: asc|desc")
	fs.IntVar(&pageSize, "page-size", 20, "Page size (max 20)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsersUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if sortDirection != "" && sortDirection != "asc" && sortDirection != "desc" {
		return usageFail(rt, "--sort-direction must be asc or desc")
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.ListEmailOnlyUsers(ctx, cursor, pageSize, email, sortDirection)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "users list-email-only request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "users list-email-only", resp.Body)
}

func runUsersCreateEmailOnly(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("users create-email-only", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var filePath string
	fs.StringVar(&filePath, "file", "", "Path to JSON request body")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsersUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 0 || strings.TrimSpace(filePath) == "" {
		return usageFail(rt, "usage: omni users create-email-only --file <json-path>")
	}

	payload, err := readJSONFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read users create-email-only payload", map[string]any{"error": err.Error()})
	}
	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := api.CreateEmailOnlyUser(ctx, payload)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "users create-email-only request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "users create-email-only", resp.Body)
}

func runUsersCreateEmailOnlyBulk(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("users create-email-only-bulk", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var filePath string
	fs.StringVar(&filePath, "file", "", "Path to JSON request body")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsersUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 0 || strings.TrimSpace(filePath) == "" {
		return usageFail(rt, "usage: omni users create-email-only-bulk --file <json-path>")
	}

	payload, err := readJSONFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read users create-email-only-bulk payload", map[string]any{"error": err.Error()})
	}
	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := api.CreateEmailOnlyUsersBulk(ctx, payload)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "users create-email-only-bulk request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "users create-email-only-bulk", resp.Body)
}

func runUsersRoles(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni users roles <get|assign> ...")
	}
	action := args[0]
	rest := args[1:]

	switch action {
	case "get":
		fs := flag.NewFlagSet("users roles get", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var connectionIDArg, modelIDArg string
		fs.StringVar(&connectionIDArg, "connection-id", "", "Filter by connection UUID")
		fs.StringVar(&modelIDArg, "model-id", "", "Filter by model UUID")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printUsersUsage()
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 1 {
			return usageFail(rt, "usage: omni users roles get <user-id> [--connection-id <uuid>] [--model-id <uuid>]")
		}
		userID, err := parseUUIDArg(fs.Arg(0), "user-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		connectionID, err := parseOptionalUUIDArg(connectionIDArg, "connection-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		modelID, err := parseOptionalUUIDArg(modelIDArg, "model-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}

		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.GetUserModelRoles(ctx, userID, connectionID, modelID)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "users roles get request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "users roles get", resp.Body)
	case "assign":
		fs := flag.NewFlagSet("users roles assign", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var filePath string
		fs.StringVar(&filePath, "file", "", "Path to JSON request body")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printUsersUsage()
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 1 || strings.TrimSpace(filePath) == "" {
			return usageFail(rt, "usage: omni users roles assign <user-id> --file <json-path>")
		}
		userID, err := parseUUIDArg(fs.Arg(0), "user-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		payload, err := readJSONFile(filePath)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to read users roles payload", map[string]any{"error": err.Error()})
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.AssignUserModelRole(ctx, userID, payload)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "users roles assign request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "users roles assign", resp.Body)
	default:
		return usageFail(rt, fmt.Sprintf("unknown users roles action: %s", action))
	}
}

func runUserGroupRoles(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni users group-roles <get|assign> ...")
	}
	action := args[0]
	rest := args[1:]

	switch action {
	case "get":
		fs := flag.NewFlagSet("users group-roles get", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var connectionIDArg, modelIDArg string
		fs.StringVar(&connectionIDArg, "connection-id", "", "Filter by connection UUID")
		fs.StringVar(&modelIDArg, "model-id", "", "Filter by model UUID")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printUsersUsage()
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 1 {
			return usageFail(rt, "usage: omni users group-roles get <group-id> [--connection-id <uuid>] [--model-id <uuid>]")
		}
		connectionID, err := parseOptionalUUIDArg(connectionIDArg, "connection-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		modelID, err := parseOptionalUUIDArg(modelIDArg, "model-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		groupID := strings.TrimSpace(fs.Arg(0))
		if groupID == "" {
			return usageFail(rt, "group-id is required")
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.GetUserGroupModelRoles(ctx, groupID, connectionID, modelID)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "users group-roles get request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "users group-roles get", resp.Body)
	case "assign":
		fs := flag.NewFlagSet("users group-roles assign", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var filePath string
		fs.StringVar(&filePath, "file", "", "Path to JSON request body")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printUsersUsage()
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 1 || strings.TrimSpace(filePath) == "" {
			return usageFail(rt, "usage: omni users group-roles assign <group-id> --file <json-path>")
		}
		groupID := strings.TrimSpace(fs.Arg(0))
		if groupID == "" {
			return usageFail(rt, "group-id is required")
		}
		payload, err := readJSONFile(filePath)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to read users group-roles payload", map[string]any{"error": err.Error()})
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.AssignUserGroupModelRole(ctx, groupID, payload)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "users group-roles assign request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "users group-roles assign", resp.Body)
	default:
		return usageFail(rt, fmt.Sprintf("unknown users group-roles action: %s", action))
	}
}

func printUsersUsage() {
	fmt.Print(`omni users commands:
  omni users list-email-only [--cursor <cursor>] [--page-size 20] [--email <email>] [--sort-direction asc|desc]
  omni users create-email-only --file <json-path>
  omni users create-email-only-bulk --file <json-path>
  omni users roles get <user-id> [--connection-id <uuid>] [--model-id <uuid>]
  omni users roles assign <user-id> --file <json-path>
  omni users group-roles get <group-id> [--connection-id <uuid>] [--model-id <uuid>]
  omni users group-roles assign <group-id> --file <json-path>
`)
}
