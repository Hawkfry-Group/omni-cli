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

func runModels(rt *runtime, args []string) int {
	if len(args) == 0 {
		printModelsUsage()
		return 0
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "list":
		return runModelsList(rt, subArgs)
	case "get":
		return runModelsGet(rt, subArgs)
	case "create":
		return runModelsCreate(rt, subArgs)
	case "refresh":
		return runModelsRefresh(rt, subArgs)
	case "validate":
		return runModelsValidate(rt, subArgs)
	case "branch":
		return runModelsBranch(rt, subArgs)
	case "cache-reset":
		return runModelsCacheReset(rt, subArgs)
	case "topics":
		return runModelsTopics(rt, subArgs)
	case "views":
		return runModelsViews(rt, subArgs)
	case "fields":
		return runModelsFields(rt, subArgs)
	case "git":
		return runModelsGit(rt, subArgs)
	case "migrate":
		return runModelsMigrate(rt, subArgs)
	case "content-validator":
		return runModelsContentValidator(rt, subArgs)
	case "yaml":
		return runModelsYAML(rt, subArgs)
	default:
		printModelsUsage()
		return usageFail(rt, fmt.Sprintf("unknown models subcommand: %s", sub))
	}
}

func runModelsList(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("models list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var cursor string
	var pageSize int
	var name string
	fs.StringVar(&cursor, "cursor", "", "Pagination cursor")
	fs.IntVar(&pageSize, "page-size", 20, "Number of records per page")
	fs.StringVar(&name, "name", "", "Filter by model name")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printModelsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.ListModels(ctx, cursor, pageSize, name)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "models list request failed", map[string]any{"error": err.Error()})
	}
	if resp.JSON200 != nil {
		if err := output.Print(resp.JSON200, rt.JSON, rt.Plain); err != nil {
			return fail(rt, 1, codeAPIError, "failed to print models list", map[string]any{"error": err.Error()})
		}
		return 0
	}

	return failFromHTTPStatus(rt, resp.StatusCode(), "models list", resp.Body)
}

func printModelsUsage() {
	fmt.Print(`omni models commands:
  omni models list [--cursor <cursor>] [--page-size 20] [--name <filter>]
  omni models get <model-id>
  omni models create --file <json-path>
  omni models refresh <model-id>
  omni models validate <model-id> [--branch-id <uuid>] [--limit <n>]
  omni models branch delete <model-id> <branch-name>
  omni models branch merge <model-id> <branch-name> --file <json-path>
  omni models cache-reset <model-id> <policy-name> --file <json-path>
  omni models topics list <model-id> [--branch-id <uuid>]
  omni models topics get <model-id> <topic-name> [--branch-id <uuid>]
  omni models topics update <model-id> <topic-name> --file <json-path> [--branch-id <uuid>]
  omni models topics delete <model-id> <topic-name> [--branch-id <uuid>]
  omni models views list <model-id> [--branch-id <uuid>]
  omni models views update <model-id> <view-name> --file <json-path> [--branch-id <uuid>]
  omni models views delete <model-id> <view-name> [--branch-id <uuid>]
  omni models fields create <model-id> --file <json-path>
  omni models fields update <model-id> <view-name> <field-name> --file <json-path> [--branch-id <uuid>]
  omni models fields delete <model-id> <view-name> <field-name> [--branch-id <uuid>] [--topic-context <name>]
  omni models git get <model-id> [--include webhookSecret]
  omni models git create <model-id> --file <json-path>
  omni models git update <model-id> --file <json-path>
  omni models git delete <model-id>
  omni models git sync <model-id> --file <json-path>
  omni models migrate <model-id> --file <json-path>
  omni models content-validator get <model-id> [--branch-id <id>] [--user-id <id>] [--include-personal-folders true|false]
  omni models content-validator replace <model-id> --file <json-path> [--user-id <id>]
  omni models yaml get <model-id> [--branch-id <uuid>] [--file-name <name>] [--mode <mode>] [--include-checksums true|false]
  omni models yaml create <model-id> --file <json-path>
  omni models yaml delete <model-id> --file-name <name> [--branch-id <uuid>] [--mode <mode>] [--commit-message <msg>]
`)
}
