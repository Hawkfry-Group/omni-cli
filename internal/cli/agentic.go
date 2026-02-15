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

func runAgentic(rt *runtime, args []string) int {
	if len(args) == 0 {
		printAgenticUsage()
		return 0
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "submit":
		return runAgenticSubmit(rt, subArgs)
	case "status":
		return runAgenticStatus(rt, subArgs)
	case "cancel":
		return runAgenticCancel(rt, subArgs)
	case "result":
		return runAgenticResult(rt, subArgs)
	default:
		printAgenticUsage()
		return usageFail(rt, fmt.Sprintf("unknown agentic subcommand: %s", sub))
	}
}

func runAgenticSubmit(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("agentic submit", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var filePath string
	fs.StringVar(&filePath, "file", "", "Path to agentic submit JSON body")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printAgenticUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 0 || strings.TrimSpace(filePath) == "" {
		return usageFail(rt, "usage: omni agentic submit --file <json-path>")
	}

	payload, err := readJSONFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read agentic submit body", map[string]any{"error": err.Error()})
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := api.SubmitAgenticJob(ctx, payload)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "agentic submit request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "agentic submit", resp.Body)
}

func runAgenticStatus(rt *runtime, args []string) int {
	if len(args) != 1 {
		return usageFail(rt, "usage: omni agentic status <job-id>")
	}
	jobID, err := parseUUIDArg(args[0], "job-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.GetAgenticJobStatus(ctx, jobID)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "agentic status request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "agentic status", resp.Body)
}

func runAgenticCancel(rt *runtime, args []string) int {
	if len(args) != 1 {
		return usageFail(rt, "usage: omni agentic cancel <job-id>")
	}
	jobID, err := parseUUIDArg(args[0], "job-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.CancelAgenticJob(ctx, jobID)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "agentic cancel request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "agentic cancel", resp.Body)
}

func runAgenticResult(rt *runtime, args []string) int {
	if len(args) != 1 {
		return usageFail(rt, "usage: omni agentic result <job-id>")
	}
	jobID, err := parseUUIDArg(args[0], "job-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.GetAgenticJobResult(ctx, jobID)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "agentic result request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "agentic result", resp.Body)
}

func printAgenticUsage() {
	fmt.Print(`omni agentic commands:
  omni agentic submit --file <json-path>
  omni agentic status <job-id>
  omni agentic cancel <job-id>
  omni agentic result <job-id>
`)
}
