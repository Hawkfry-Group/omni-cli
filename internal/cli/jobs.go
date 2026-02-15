package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/omni-co/omni-cli/internal/client"
	"github.com/omni-co/omni-cli/internal/output"
)

func runJobs(rt *runtime, args []string) int {
	if len(args) == 0 {
		printJobsUsage()
		return 0
	}

	sub := args[0]
	if sub != "status" {
		printJobsUsage()
		return usageFail(rt, fmt.Sprintf("unknown jobs subcommand: %s", sub))
	}
	if len(args) != 2 {
		return usageFail(rt, "usage: omni jobs status <job-id>")
	}

	jobID := args[1]
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cli, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}

	resp, err := cli.GetJobStatus(ctx, jobID)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "job status request failed", map[string]any{"error": err.Error()})
	}
	if resp.JSON200 != nil {
		if err := output.Print(resp.JSON200, rt.JSON, rt.Plain); err != nil {
			return fail(rt, 1, codeAPIError, "failed to print job status", map[string]any{"error": err.Error()})
		}
		return 0
	}

	if resp.StatusCode() == 401 {
		return fail(rt, 1, codeAuthUnauthorized, "unauthorized", client.ParseBody(resp.Body))
	}
	if resp.StatusCode() == 403 {
		return fail(rt, 1, codeAuthForbidden, "permission denied for job status", client.ParseBody(resp.Body))
	}
	return fail(rt, 1, codeAPIError, "job status returned unexpected response", client.ParseBody(resp.Body))
}

func printJobsUsage() {
	fmt.Print(`omni jobs commands:
  omni jobs status <job-id>
`)
}
