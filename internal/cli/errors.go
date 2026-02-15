package cli

import "github.com/omni-co/omni-cli/internal/output"

const (
	codeUsageError       = "USAGE_ERROR"
	codeConfigError      = "CONFIG_ERROR"
	codeConfigMissing    = "CONFIG_MISSING"
	codeAuthUnauthorized = "AUTH_UNAUTHORIZED"
	codeAuthForbidden    = "AUTH_FORBIDDEN"
	codeAuthError        = "AUTH_ERROR"
	codeAPIError         = "API_ERROR"
	codeNetworkError     = "NETWORK_ERROR"
	codeValidationError  = "VALIDATION_ERROR"
)

func fail(rt *runtime, exitCode int, code, message string, details any) int {
	output.PrintError(rt.JSON, code, message, details)
	return exitCode
}

func usageFail(rt *runtime, message string) int {
	return fail(rt, 2, codeUsageError, message, nil)
}
