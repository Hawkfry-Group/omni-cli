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

func runSCIM(rt *runtime, args []string) int {
	if rt.Resolved.Profile.TokenType != "org" {
		return fail(rt, 1, codeAuthForbidden, "scim commands require an org API key", map[string]any{"token_type": rt.Resolved.Profile.TokenType})
	}
	if len(args) == 0 {
		printSCIMUsage()
		return 0
	}

	resource := args[0]
	subArgs := args[1:]

	switch resource {
	case "users":
		return runSCIMUsers(rt, subArgs)
	case "groups":
		return runSCIMGroups(rt, subArgs)
	case "embed-users":
		return runSCIMEmbedUsers(rt, subArgs)
	default:
		printSCIMUsage()
		return usageFail(rt, fmt.Sprintf("unknown scim resource: %s", resource))
	}
}

func runSCIMUsers(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni scim users <list|get|create|update|replace|delete> ...")
	}
	action := args[0]
	rest := args[1:]

	switch action {
	case "list":
		fs := flag.NewFlagSet("scim users list", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var count, startIndex int
		var filter string
		fs.IntVar(&count, "count", 20, "Max number of users")
		fs.IntVar(&startIndex, "start-index", 1, "Start index (1-based)")
		fs.StringVar(&filter, "filter", "", "SCIM filter expression")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printSCIMUsage()
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
		resp, err := api.ListSCIMUsers(ctx, count, startIndex, filter)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "scim users list request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "scim users list", resp.Body)
	case "get":
		if len(rest) != 1 {
			return usageFail(rt, "usage: omni scim users get <user-id>")
		}
		id, err := parseUUIDArg(rest[0], "user-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.GetSCIMUser(ctx, id)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "scim users get request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "scim users get", resp.Body)
	case "create":
		return runSCIMUsersFileMutation(rt, rest, "create")
	case "update":
		return runSCIMUsersFileMutation(rt, rest, "update")
	case "replace":
		return runSCIMUsersFileMutation(rt, rest, "replace")
	case "delete", "rm":
		if len(rest) != 1 {
			return usageFail(rt, "usage: omni scim users delete <user-id>")
		}
		id, err := parseUUIDArg(rest[0], "user-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.DeleteSCIMUser(ctx, id)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "scim users delete request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "scim users delete", resp.Body)
	default:
		return usageFail(rt, fmt.Sprintf("unknown scim users action: %s", action))
	}
}

func runSCIMUsersFileMutation(rt *runtime, args []string, action string) int {
	fs := flag.NewFlagSet("scim users "+action, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var filePath string
	fs.StringVar(&filePath, "file", "", "Path to JSON request body")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printSCIMUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if action == "create" {
		if fs.NArg() != 0 || strings.TrimSpace(filePath) == "" {
			return usageFail(rt, "usage: omni scim users create --file <json-path>")
		}
	} else if fs.NArg() != 1 || strings.TrimSpace(filePath) == "" {
		return usageFail(rt, fmt.Sprintf("usage: omni scim users %s <user-id> --file <json-path>", action))
	}

	payload, err := readJSONFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read scim users payload", map[string]any{"error": err.Error()})
	}
	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if action == "create" {
		resp, reqErr := api.CreateSCIMUser(ctx, payload)
		if reqErr != nil {
			return fail(rt, 1, codeNetworkError, "scim users create request failed", map[string]any{"error": reqErr.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "scim users create", resp.Body)
	}

	id, err := parseUUIDArg(fs.Arg(0), "user-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}
	if action == "update" {
		resp, reqErr := api.UpdateSCIMUser(ctx, id, payload)
		if reqErr != nil {
			return fail(rt, 1, codeNetworkError, "scim users update request failed", map[string]any{"error": reqErr.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "scim users update", resp.Body)
	}
	resp, reqErr := api.ReplaceSCIMUser(ctx, id, payload)
	if reqErr != nil {
		return fail(rt, 1, codeNetworkError, "scim users replace request failed", map[string]any{"error": reqErr.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "scim users replace", resp.Body)
}

func runSCIMGroups(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni scim groups <list|get|create|update|replace|delete> ...")
	}
	action := args[0]
	rest := args[1:]

	switch action {
	case "list":
		fs := flag.NewFlagSet("scim groups list", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var count, startIndex int
		fs.IntVar(&count, "count", 20, "Max number of groups")
		fs.IntVar(&startIndex, "start-index", 1, "Start index (1-based)")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printSCIMUsage()
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
		resp, err := api.ListSCIMGroups(ctx, count, startIndex)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "scim groups list request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "scim groups list", resp.Body)
	case "get":
		if len(rest) != 1 {
			return usageFail(rt, "usage: omni scim groups get <group-mini-uuid>")
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.GetSCIMGroup(ctx, rest[0])
		if err != nil {
			return fail(rt, 1, codeNetworkError, "scim groups get request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "scim groups get", resp.Body)
	case "create":
		return runSCIMGroupsFileMutation(rt, rest, "create")
	case "update":
		return runSCIMGroupsFileMutation(rt, rest, "update")
	case "replace":
		return runSCIMGroupsFileMutation(rt, rest, "replace")
	case "delete", "rm":
		if len(rest) != 1 {
			return usageFail(rt, "usage: omni scim groups delete <group-mini-uuid>")
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.DeleteSCIMGroup(ctx, rest[0])
		if err != nil {
			return fail(rt, 1, codeNetworkError, "scim groups delete request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "scim groups delete", resp.Body)
	default:
		return usageFail(rt, fmt.Sprintf("unknown scim groups action: %s", action))
	}
}

func runSCIMGroupsFileMutation(rt *runtime, args []string, action string) int {
	fs := flag.NewFlagSet("scim groups "+action, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var filePath string
	fs.StringVar(&filePath, "file", "", "Path to JSON request body")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printSCIMUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if action == "create" {
		if fs.NArg() != 0 || strings.TrimSpace(filePath) == "" {
			return usageFail(rt, "usage: omni scim groups create --file <json-path>")
		}
	} else if fs.NArg() != 1 || strings.TrimSpace(filePath) == "" {
		return usageFail(rt, fmt.Sprintf("usage: omni scim groups %s <group-mini-uuid> --file <json-path>", action))
	}

	payload, err := readJSONFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read scim groups payload", map[string]any{"error": err.Error()})
	}
	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if action == "create" {
		resp, reqErr := api.CreateSCIMGroup(ctx, payload)
		if reqErr != nil {
			return fail(rt, 1, codeNetworkError, "scim groups create request failed", map[string]any{"error": reqErr.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "scim groups create", resp.Body)
	}
	groupID := fs.Arg(0)
	if action == "update" {
		resp, reqErr := api.UpdateSCIMGroup(ctx, groupID, payload)
		if reqErr != nil {
			return fail(rt, 1, codeNetworkError, "scim groups update request failed", map[string]any{"error": reqErr.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "scim groups update", resp.Body)
	}
	resp, reqErr := api.ReplaceSCIMGroup(ctx, groupID, payload)
	if reqErr != nil {
		return fail(rt, 1, codeNetworkError, "scim groups replace request failed", map[string]any{"error": reqErr.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "scim groups replace", resp.Body)
}

func runSCIMEmbedUsers(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni scim embed-users <list|get|delete> ...")
	}
	action := args[0]
	rest := args[1:]

	switch action {
	case "list":
		fs := flag.NewFlagSet("scim embed-users list", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var count, startIndex int
		var filter string
		fs.IntVar(&count, "count", 20, "Max number of users")
		fs.IntVar(&startIndex, "start-index", 1, "Start index (1-based)")
		fs.StringVar(&filter, "filter", "", "SCIM filter expression")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printSCIMUsage()
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
		resp, err := api.ListSCIMEmbedUsers(ctx, count, startIndex, filter)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "scim embed-users list request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "scim embed-users list", resp.Body)
	case "get":
		if len(rest) != 1 {
			return usageFail(rt, "usage: omni scim embed-users get <user-id>")
		}
		id, err := parseUUIDArg(rest[0], "user-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.GetSCIMEmbedUser(ctx, id)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "scim embed-users get request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "scim embed-users get", resp.Body)
	case "delete", "rm":
		if len(rest) != 1 {
			return usageFail(rt, "usage: omni scim embed-users delete <user-id>")
		}
		id, err := parseUUIDArg(rest[0], "user-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.DeleteSCIMEmbedUser(ctx, id)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "scim embed-users delete request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "scim embed-users delete", resp.Body)
	default:
		return usageFail(rt, fmt.Sprintf("unknown scim embed-users action: %s", action))
	}
}

func printSCIMUsage() {
	fmt.Print(`omni scim commands:
  omni scim users list [--count 20] [--start-index 1] [--filter <expr>]
  omni scim users get <user-id>
  omni scim users create --file <json-path>
  omni scim users update <user-id> --file <json-path>
  omni scim users replace <user-id> --file <json-path>
  omni scim users delete <user-id>
  omni scim groups list [--count 20] [--start-index 1]
  omni scim groups get <group-mini-uuid>
  omni scim groups create --file <json-path>
  omni scim groups update <group-mini-uuid> --file <json-path>
  omni scim groups replace <group-mini-uuid> --file <json-path>
  omni scim groups delete <group-mini-uuid>
  omni scim embed-users list [--count 20] [--start-index 1] [--filter <expr>]
  omni scim embed-users get <user-id>
  omni scim embed-users delete <user-id>
`)
}
