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

func runEmbed(rt *runtime, args []string) int {
	if len(args) == 0 {
		printEmbedUsage()
		return 0
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "sso":
		return runEmbedSSO(rt, rest)
	default:
		printEmbedUsage()
		return usageFail(rt, fmt.Sprintf("unknown embed subcommand: %s", sub))
	}
}

func runEmbedSSO(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni embed sso generate-session --file <json-path>")
	}
	sub := args[0]
	rest := args[1:]
	if sub != "generate-session" {
		return usageFail(rt, fmt.Sprintf("unknown embed sso subcommand: %s", sub))
	}

	fs := flag.NewFlagSet("embed sso generate-session", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var filePath string
	fs.StringVar(&filePath, "file", "", "Path to embed session JSON body")
	if err := fs.Parse(rest); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printEmbedUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 0 || strings.TrimSpace(filePath) == "" {
		return usageFail(rt, "usage: omni embed sso generate-session --file <json-path>")
	}

	payload, err := readJSONFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read embed session body", map[string]any{"error": err.Error()})
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.GenerateEmbedSSOSession(ctx, payload)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "embed sso generate-session request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "embed sso generate-session", resp.Body)
}

func printEmbedUsage() {
	fmt.Print(`omni embed commands:
  omni embed sso generate-session --file <json-path>
`)
}
