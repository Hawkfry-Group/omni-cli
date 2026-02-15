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
	"github.com/omni-co/omni-cli/internal/output"
)

func runLabels(rt *runtime, args []string) int {
	if len(args) == 0 {
		printLabelsUsage()
		return 0
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "list":
		return runLabelsList(rt, subArgs)
	case "get":
		return runLabelsGet(rt, subArgs)
	case "create":
		return runLabelsCreate(rt, subArgs)
	case "update":
		return runLabelsUpdate(rt, subArgs)
	case "delete", "rm":
		return runLabelsDelete(rt, subArgs)
	default:
		printLabelsUsage()
		return usageFail(rt, fmt.Sprintf("unknown labels subcommand: %s", sub))
	}
}

func runLabelsList(rt *runtime, args []string) int {
	if len(args) != 0 {
		return usageFail(rt, "usage: omni labels list")
	}
	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.ListLabels(ctx)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "labels list request failed", map[string]any{"error": err.Error()})
	}
	if resp.JSON200 != nil {
		if err := output.Print(resp.JSON200, rt.JSON, rt.Plain); err != nil {
			return fail(rt, 1, codeAPIError, "failed to print labels", map[string]any{"error": err.Error()})
		}
		return 0
	}
	return failFromHTTPStatus(rt, resp.StatusCode(), "labels list", resp.Body)
}

func runLabelsGet(rt *runtime, args []string) int {
	if len(args) != 1 {
		return usageFail(rt, "usage: omni labels get <name>")
	}
	name := strings.TrimSpace(args[0])
	if name == "" {
		return usageFail(rt, "label name is required")
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.GetLabel(ctx, name)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "labels get request failed", map[string]any{"error": err.Error()})
	}
	if resp.JSON200 != nil {
		if err := output.Print(resp.JSON200, rt.JSON, rt.Plain); err != nil {
			return fail(rt, 1, codeAPIError, "failed to print label", map[string]any{"error": err.Error()})
		}
		return 0
	}
	return failFromHTTPStatus(rt, resp.StatusCode(), "labels get", resp.Body)
}

func runLabelsCreate(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("labels create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var homepageStr, verifiedStr, userID string
	fs.StringVar(&homepageStr, "homepage", "", "Optional: true/false")
	fs.StringVar(&verifiedStr, "verified", "", "Optional: true/false")
	fs.StringVar(&userID, "user-id", "", "Optional target user ID (org key)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printLabelsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 {
		return usageFail(rt, "usage: omni labels create <name> [--homepage true|false] [--verified true|false] [--user-id <id>]")
	}

	homepage, err := parseOptionalBool(homepageStr)
	if err != nil {
		return usageFail(rt, err.Error())
	}
	verified, err := parseOptionalBool(verifiedStr)
	if err != nil {
		return usageFail(rt, err.Error())
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.CreateLabel(ctx, fs.Arg(0), homepage, verified, userID)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "labels create request failed", map[string]any{"error": err.Error()})
	}
	if resp.JSON201 != nil {
		if err := output.Print(resp.JSON201, rt.JSON, rt.Plain); err != nil {
			return fail(rt, 1, codeAPIError, "failed to print label", map[string]any{"error": err.Error()})
		}
		return 0
	}
	return failFromHTTPStatus(rt, resp.StatusCode(), "labels create", resp.Body)
}

func runLabelsUpdate(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("labels update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var newName, homepageStr, verifiedStr, userID string
	fs.StringVar(&newName, "new-name", "", "Optional new label name")
	fs.StringVar(&homepageStr, "homepage", "", "Optional: true/false")
	fs.StringVar(&verifiedStr, "verified", "", "Optional: true/false")
	fs.StringVar(&userID, "user-id", "", "Optional target user ID (org key)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printLabelsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 {
		return usageFail(rt, "usage: omni labels update <name> [--new-name <name>] [--homepage true|false] [--verified true|false] [--user-id <id>]")
	}

	homepage, err := parseOptionalBool(homepageStr)
	if err != nil {
		return usageFail(rt, err.Error())
	}
	verified, err := parseOptionalBool(verifiedStr)
	if err != nil {
		return usageFail(rt, err.Error())
	}

	var newNamePtr *string
	if strings.TrimSpace(newName) != "" {
		n := strings.TrimSpace(newName)
		newNamePtr = &n
	}
	if newNamePtr == nil && homepage == nil && verified == nil {
		return usageFail(rt, "no changes provided; set --new-name, --homepage, or --verified")
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.UpdateLabel(ctx, fs.Arg(0), newNamePtr, homepage, verified, userID)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "labels update request failed", map[string]any{"error": err.Error()})
	}
	if resp.JSON200 != nil {
		if err := output.Print(resp.JSON200, rt.JSON, rt.Plain); err != nil {
			return fail(rt, 1, codeAPIError, "failed to print label", map[string]any{"error": err.Error()})
		}
		return 0
	}
	return failFromHTTPStatus(rt, resp.StatusCode(), "labels update", resp.Body)
}

func runLabelsDelete(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("labels delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var userID string
	fs.StringVar(&userID, "user-id", "", "Optional target user ID (org key)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printLabelsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 {
		return usageFail(rt, "usage: omni labels delete <name> [--user-id <id>]")
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.DeleteLabel(ctx, fs.Arg(0), userID)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "labels delete request failed", map[string]any{"error": err.Error()})
	}
	if resp.StatusCode() >= 200 && resp.StatusCode() < 300 {
		if err := output.Print(client.ParseBody(resp.Body), rt.JSON, rt.Plain); err != nil {
			return fail(rt, 1, codeAPIError, "failed to print delete result", map[string]any{"error": err.Error()})
		}
		return 0
	}
	return failFromHTTPStatus(rt, resp.StatusCode(), "labels delete", resp.Body)
}

func printLabelsUsage() {
	fmt.Print(`omni labels commands:
  omni labels list
  omni labels get <name>
  omni labels create <name> [--homepage true|false] [--verified true|false] [--user-id <id>]
  omni labels update <name> [--new-name <name>] [--homepage true|false] [--verified true|false] [--user-id <id>]
  omni labels delete <name> [--user-id <id>]
`)
}
