package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/omni-co/omni-cli/internal/client"
	"github.com/omni-co/omni-cli/internal/client/gen"
	"github.com/omni-co/omni-cli/internal/output"
)

func runQuery(rt *runtime, args []string) int {
	if len(args) == 0 {
		printQueryUsage()
		return 0
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "run":
		return runQueryRun(rt, subArgs)
	default:
		printQueryUsage()
		return usageFail(rt, fmt.Sprintf("unknown query subcommand: %s", sub))
	}
}

func runQueryRun(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("query run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var filePath string
	var resultType string
	var wait bool
	var timeoutSec int

	fs.StringVar(&filePath, "file", "", "Path to query request JSON")
	fs.StringVar(&resultType, "result-type", "json", "json|csv|xlsx|arrow")
	fs.BoolVar(&wait, "wait", true, "Wait for async job completion")
	fs.IntVar(&timeoutSec, "timeout-seconds", 30, "Wait timeout in seconds")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printQueryUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if filePath == "" {
		return usageFail(rt, "--file is required")
	}

	payloadBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read query file", map[string]any{"error": err.Error()})
	}

	var body gen.QueryRunBody
	if err := json.Unmarshal(payloadBytes, &body); err != nil {
		return fail(rt, 1, codeValidationError, "failed to parse query file", map[string]any{"error": err.Error()})
	}

	if err := setQueryResultType(&body, resultType); err != nil {
		return usageFail(rt, err.Error())
	}

	cli, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec+10)*time.Second)
	defer cancel()

	runResp, err := cli.RunQuery(ctx, body)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "run query request failed", map[string]any{"error": err.Error()})
	}

	jobIDs := make([]string, 0)
	if runResp.JSON200 != nil && runResp.JSON200.JobIds != nil {
		jobIDs = append(jobIDs, (*runResp.JSON200.JobIds)...)
	}
	if runResp.JSON408 != nil && runResp.JSON408.RemainingJobIds != nil {
		jobIDs = append(jobIDs, (*runResp.JSON408.RemainingJobIds)...)
	}

	if wait && len(jobIDs) > 0 {
		waitResp, waitErr := cli.WaitForQueryResults(ctx, jobIDs)
		if waitErr != nil {
			return fail(rt, 1, codeNetworkError, "query wait request failed", map[string]any{"error": waitErr.Error()})
		}
		if waitResp.JSON200 != nil {
			if err := output.Print(waitResp.JSON200, rt.JSON, rt.Plain); err != nil {
				return fail(rt, 1, codeAPIError, "failed to print query results", map[string]any{"error": err.Error()})
			}
			return 0
		}
		if waitResp.StatusCode() == 401 {
			return fail(rt, 1, codeAuthUnauthorized, "unauthorized", client.ParseBody(waitResp.Body))
		}
		if waitResp.StatusCode() == 403 {
			return fail(rt, 1, codeAuthForbidden, "permission denied for query wait", client.ParseBody(waitResp.Body))
		}
		return fail(rt, 1, codeAPIError, "query wait returned unexpected response", client.ParseBody(waitResp.Body))
	}

	if runResp.JSON200 != nil {
		if err := output.Print(runResp.JSON200, rt.JSON, rt.Plain); err != nil {
			return fail(rt, 1, codeAPIError, "failed to print query response", map[string]any{"error": err.Error()})
		}
		return 0
	}
	if runResp.JSON408 != nil {
		if err := output.Print(runResp.JSON408, rt.JSON, rt.Plain); err != nil {
			return fail(rt, 1, codeAPIError, "failed to print timeout response", map[string]any{"error": err.Error()})
		}
		return 0
	}

	if runResp.StatusCode() == 401 {
		return fail(rt, 1, codeAuthUnauthorized, "unauthorized", client.ParseBody(runResp.Body))
	}
	if runResp.StatusCode() == 403 {
		return fail(rt, 1, codeAuthForbidden, "permission denied for query", client.ParseBody(runResp.Body))
	}
	return fail(rt, 1, codeAPIError, "query run returned unexpected response", client.ParseBody(runResp.Body))
}

func setQueryResultType(body *gen.QueryRunBody, resultType string) error {
	rt := strings.ToLower(strings.TrimSpace(resultType))
	switch rt {
	case "", "arrow":
		body.ResultType = nil
	case "json":
		v := gen.QueryRunBodyResultTypeJson
		body.ResultType = &v
	case "csv":
		v := gen.QueryRunBodyResultTypeCsv
		body.ResultType = &v
	case "xlsx":
		v := gen.QueryRunBodyResultTypeXlsx
		body.ResultType = &v
	default:
		return fmt.Errorf("invalid --result-type %q; use json, csv, xlsx, or arrow", resultType)
	}
	return nil
}

func printQueryUsage() {
	fmt.Print(`omni query commands:
  omni query run --file query.json [--result-type json|csv|xlsx|arrow] [--wait]
`)
}
