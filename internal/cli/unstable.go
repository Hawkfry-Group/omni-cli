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

func runUnstable(rt *runtime, args []string) int {
	if len(args) == 0 {
		printUnstableUsage()
		return 0
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "documents":
		return runUnstableDocuments(rt, rest)
	default:
		printUnstableUsage()
		return usageFail(rt, fmt.Sprintf("unknown unstable subcommand: %s", sub))
	}
}

func runUnstableDocuments(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni unstable documents <export|import> ...")
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "export":
		if len(rest) != 1 {
			return usageFail(rt, "usage: omni unstable documents export <identifier>")
		}
		identifier := strings.TrimSpace(rest[0])
		if identifier == "" {
			return usageFail(rt, "document identifier is required")
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.ExportUnstableDocument(ctx, identifier)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "unstable documents export request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "unstable documents export", resp.Body)
	case "import":
		fs := flag.NewFlagSet("unstable documents import", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var filePath string
		fs.StringVar(&filePath, "file", "", "Path to unstable document import JSON body")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printUnstableUsage()
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 0 || strings.TrimSpace(filePath) == "" {
			return usageFail(rt, "usage: omni unstable documents import --file <json-path>")
		}
		payload, err := readJSONFile(filePath)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to read unstable import body", map[string]any{"error": err.Error()})
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		resp, err := api.ImportUnstableDocument(ctx, payload)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "unstable documents import request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "unstable documents import", resp.Body)
	default:
		return usageFail(rt, fmt.Sprintf("unknown unstable documents subcommand: %s", sub))
	}
}

func printUnstableUsage() {
	fmt.Print(`omni unstable commands:
  omni unstable documents export <identifier>
  omni unstable documents import --file <json-path>
`)
}
