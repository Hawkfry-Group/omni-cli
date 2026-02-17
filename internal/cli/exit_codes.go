package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/omni-co/omni-cli/internal/output"
)

type exitCodeInfo struct {
	ExitCode    int    `json:"exit_code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type errorCodeInfo struct {
	Code        string `json:"code"`
	ExitCode    int    `json:"exit_code"`
	Description string `json:"description"`
}

func runExitCodes(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("exit-codes", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printExitCodesUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 0 {
		return usageFail(rt, "usage: omni exit-codes")
	}

	payload := map[string]any{
		"exit_codes": []exitCodeInfo{
			{ExitCode: 0, Name: "SUCCESS", Description: "Command completed successfully."},
			{ExitCode: 1, Name: "FAILURE", Description: "Runtime, API, auth, config, network, or validation failure."},
			{ExitCode: 2, Name: "USAGE", Description: "Invalid command usage or argument parsing failure."},
		},
		"error_codes": []errorCodeInfo{
			{Code: codeUsageError, ExitCode: 2, Description: "Invalid command usage."},
			{Code: codeConfigError, ExitCode: 1, Description: "Configuration file or runtime config error."},
			{Code: codeConfigMissing, ExitCode: 1, Description: "Required configuration is missing."},
			{Code: codeAuthUnauthorized, ExitCode: 1, Description: "Token is missing or unauthorized."},
			{Code: codeAuthForbidden, ExitCode: 1, Description: "Token does not have permission for this command."},
			{Code: codeAuthError, ExitCode: 1, Description: "Authentication resolution failed."},
			{Code: codeAPIError, ExitCode: 1, Description: "Unexpected API response or output formatting failure."},
			{Code: codeNetworkError, ExitCode: 1, Description: "Request transport failed or timed out."},
			{Code: codeValidationError, ExitCode: 1, Description: "Input payload validation failed."},
		},
	}

	asJSON := true
	if rt.Plain {
		asJSON = false
	}
	if err := output.Print(payload, asJSON, rt.Plain); err != nil {
		return fail(rt, 1, codeAPIError, "failed to print exit-code contract", map[string]any{"error": err.Error()})
	}
	return 0
}

func printExitCodesUsage() {
	fmt.Print(`omni exit-codes:
  omni exit-codes
`)
}
