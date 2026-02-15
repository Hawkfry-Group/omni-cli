package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/omni-co/omni-cli/internal/client"
	"github.com/omni-co/omni-cli/internal/output"
)

func runAdmin(rt *runtime, args []string) int {
	if rt.Resolved.Profile.TokenType != "org" {
		return fail(rt, 1, codeAuthForbidden, "admin commands require an org API key", map[string]any{"token_type": rt.Resolved.Profile.TokenType})
	}

	if len(args) < 2 {
		printAdminUsage()
		return 0
	}

	resource := args[0]
	action := args[1]
	rest := args[2:]

	switch resource {
	case "users":
		if action != "list" {
			printAdminUsage()
			return usageFail(rt, fmt.Sprintf("unknown admin users action: %s", action))
		}
		return runAdminUsersList(rt, rest)
	case "groups":
		if action != "list" {
			printAdminUsage()
			return usageFail(rt, fmt.Sprintf("unknown admin groups action: %s", action))
		}
		return runAdminGroupsList(rt, rest)
	default:
		printAdminUsage()
		return usageFail(rt, fmt.Sprintf("unknown admin resource: %s", resource))
	}
}

func runAdminUsersList(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("admin users list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var count, startIndex int
	fs.IntVar(&count, "count", 20, "Max number of users")
	fs.IntVar(&startIndex, "start-index", 1, "Start index (1-based)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printAdminUsage()
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

	resp, err := api.ListSCIMUsers(ctx, count, startIndex, "")
	if err != nil {
		return fail(rt, 1, codeNetworkError, "admin users list request failed", map[string]any{"error": err.Error()})
	}
	if resp.JSON200 != nil {
		if err := output.Print(resp.JSON200, rt.JSON, rt.Plain); err != nil {
			return fail(rt, 1, codeAPIError, "failed to print users list", map[string]any{"error": err.Error()})
		}
		return 0
	}
	return failFromHTTPStatus(rt, resp.StatusCode(), "admin users list", resp.Body)
}

func runAdminGroupsList(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("admin groups list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var count, startIndex int
	fs.IntVar(&count, "count", 20, "Max number of groups")
	fs.IntVar(&startIndex, "start-index", 1, "Start index (1-based)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printAdminUsage()
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
		return fail(rt, 1, codeNetworkError, "admin groups list request failed", map[string]any{"error": err.Error()})
	}
	if resp.JSON200 != nil {
		if err := output.Print(resp.JSON200, rt.JSON, rt.Plain); err != nil {
			return fail(rt, 1, codeAPIError, "failed to print groups list", map[string]any{"error": err.Error()})
		}
		return 0
	}
	return failFromHTTPStatus(rt, resp.StatusCode(), "admin groups list", resp.Body)
}

func printAdminUsage() {
	fmt.Print(`omni admin commands:
  omni admin users list [--count 20] [--start-index 1]
  omni admin groups list [--count 20] [--start-index 1]
`)
}
