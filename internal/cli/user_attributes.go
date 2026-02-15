package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/omni-co/omni-cli/internal/client"
)

func runUserAttributes(rt *runtime, args []string) int {
	if len(args) == 0 {
		printUserAttributesUsage()
		return 0
	}
	if len(args) != 1 || args[0] != "list" {
		printUserAttributesUsage()
		return usageFail(rt, "usage: omni user-attributes list")
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.ListUserAttributes(ctx)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "user-attributes list request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "user-attributes list", resp.Body)
}

func printUserAttributesUsage() {
	fmt.Print(`omni user-attributes commands:
  omni user-attributes list
`)
}
