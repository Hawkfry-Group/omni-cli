package cli

import (
	"github.com/omni-co/omni-cli/internal/client"
	"github.com/omni-co/omni-cli/internal/output"
)

func failFromHTTPStatus(rt *runtime, status int, context string, body []byte) int {
	details := client.ParseBody(body)
	switch status {
	case 401:
		return fail(rt, 1, codeAuthUnauthorized, "unauthorized: "+context, details)
	case 403:
		return fail(rt, 1, codeAuthForbidden, "permission denied: "+context, details)
	case 404:
		return fail(rt, 1, codeAPIError, "resource not found: "+context, details)
	default:
		return fail(rt, 1, codeAPIError, "unexpected API response: "+context, details)
	}
}

func succeedOrFail(rt *runtime, status int, context string, body []byte) int {
	if status >= 200 && status < 300 {
		if err := output.Print(client.ParseBody(body), rt.JSON, rt.Plain); err != nil {
			return fail(rt, 1, codeAPIError, "failed to print response: "+context, map[string]any{"error": err.Error()})
		}
		return 0
	}
	return failFromHTTPStatus(rt, status, context, body)
}
