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

func runDoctor(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var includeAdmin bool
	var timeoutSec int
	fs.BoolVar(&includeAdmin, "include-admin", rt.Resolved.Profile.TokenType == "org", "Include admin SCIM capability checks")
	fs.IntVar(&timeoutSec, "timeout-seconds", 20, "Validation timeout in seconds")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printDoctorUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	summary, validationErr := collectValidation(ctx, api, rt.Resolved.Profile.TokenType, includeAdmin)
	if validationErr != nil {
		return fail(rt, 1, validationErr.Code, validationErr.Message, validationErr.Details)
	}

	status := "ok"
	exitCode := 0
	if summary.Query.Status == "fail" || summary.Base.Status == "fail" {
		status = "degraded"
		exitCode = 1
	}
	if includeAdmin && summary.Admin.Status == "fail" {
		status = "degraded"
		exitCode = 1
	}

	result := map[string]any{
		"status":       status,
		"profile":      rt.Resolved.ProfileName,
		"base_url":     rt.Resolved.Profile.BaseURL,
		"token_type":   rt.Resolved.Profile.TokenType,
		"capabilities": summary.Capabilities,
		"checks": map[string]any{
			"base":  summary.Base,
			"query": summary.Query,
			"admin": summary.Admin,
		},
		"rate_limit": summary.RateLimit,
	}

	if err := output.Print(result, rt.JSON, rt.Plain); err != nil {
		return fail(rt, 1, codeAPIError, "failed to print output", map[string]any{"error": err.Error()})
	}
	return exitCode
}

func printDoctorUsage() {
	fmt.Print(`omni doctor:
  omni doctor [--include-admin] [--timeout-seconds 20]
`)
}
