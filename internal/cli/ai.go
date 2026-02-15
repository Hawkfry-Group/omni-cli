package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/omni-co/omni-cli/internal/client"
)

func runAI(rt *runtime, args []string) int {
	if len(args) == 0 {
		printAIUsage()
		return 0
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "generate-query":
		return runAIGenerateQuery(rt, subArgs, false)
	case "workbook":
		return runAIGenerateQuery(rt, subArgs, true)
	case "pick-topic":
		return runAIPickTopic(rt, subArgs)
	default:
		printAIUsage()
		return usageFail(rt, fmt.Sprintf("unknown ai subcommand: %s", sub))
	}
}

func runAIGenerateQuery(rt *runtime, args []string, forceWorkbook bool) int {
	cmdName := "ai generate-query"
	if forceWorkbook {
		cmdName = "ai workbook"
	}
	fs := flag.NewFlagSet(cmdName, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var modelID string
	var prompt string
	var currentTopicName string
	var workbookURL bool
	var branchID string
	var structured bool
	var contextQueryFile string

	fs.StringVar(&modelID, "model-id", "", "Model UUID")
	fs.StringVar(&prompt, "prompt", "", "Natural language prompt")
	fs.StringVar(&currentTopicName, "current-topic-name", "", "Optional topic name")
	fs.BoolVar(&workbookURL, "workbook-url", false, "Return workbook URL")
	fs.StringVar(&branchID, "branch-id", "", "Optional branch ID")
	fs.BoolVar(&structured, "structured", false, "Return structured query format")
	fs.StringVar(&contextQueryFile, "context-query-file", "", "Path to JSON query object for context")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printAIUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 0 {
		return usageFail(rt, "usage: omni ai generate-query --model-id <uuid> --prompt <text> [--current-topic-name <name>] [--workbook-url] [--branch-id <id>] [--structured] [--context-query-file <json-path>]")
	}

	modelID = strings.TrimSpace(modelID)
	prompt = strings.TrimSpace(prompt)
	if modelID == "" || prompt == "" {
		if forceWorkbook {
			return usageFail(rt, "usage: omni ai workbook --model-id <uuid> --prompt <text> [--current-topic-name <name>] [--branch-id <id>] [--structured] [--context-query-file <json-path>]")
		}
		return usageFail(rt, "usage: omni ai generate-query --model-id <uuid> --prompt <text> [--current-topic-name <name>] [--workbook-url] [--branch-id <id>] [--structured] [--context-query-file <json-path>]")
	}

	payload := map[string]any{
		"modelId": modelID,
		"prompt":  prompt,
	}
	if strings.TrimSpace(currentTopicName) != "" {
		payload["currentTopicName"] = strings.TrimSpace(currentTopicName)
	}
	if strings.TrimSpace(branchID) != "" {
		payload["branchId"] = strings.TrimSpace(branchID)
	}
	if structured {
		payload["structured"] = true
	}

	if forceWorkbook {
		payload["workbookUrl"] = true
	} else if workbookURL {
		payload["workbookUrl"] = true
	}

	if strings.TrimSpace(contextQueryFile) != "" {
		raw, err := readJSONFile(contextQueryFile)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to read context query file", map[string]any{"error": err.Error()})
		}
		var contextQuery any
		if err := json.Unmarshal(raw, &contextQuery); err != nil {
			return fail(rt, 1, codeValidationError, "failed to parse context query JSON", map[string]any{"error": err.Error()})
		}
		payload["contextQuery"] = contextQuery
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fail(rt, 1, codeValidationError, "failed to encode ai request", map[string]any{"error": err.Error()})
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := api.AIGenerateQuery(ctx, body)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "ai generate-query request failed", map[string]any{"error": err.Error()})
	}
	action := "ai generate-query"
	if forceWorkbook {
		action = "ai workbook"
	}
	return succeedOrFail(rt, resp.StatusCode(), action, resp.Body)
}

func runAIPickTopic(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("ai pick-topic", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var modelID string
	var prompt string
	var currentTopicName string
	var branchID string
	var potentialTopicsCSV string

	fs.StringVar(&modelID, "model-id", "", "Model UUID")
	fs.StringVar(&prompt, "prompt", "", "Natural language prompt")
	fs.StringVar(&currentTopicName, "current-topic-name", "", "Optional current topic name")
	fs.StringVar(&branchID, "branch-id", "", "Optional branch ID")
	fs.StringVar(&potentialTopicsCSV, "potential-topic-names", "", "Optional comma-separated topic names")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printAIUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 0 {
		return usageFail(rt, "usage: omni ai pick-topic --model-id <uuid> --prompt <text> [--current-topic-name <name>] [--branch-id <id>] [--potential-topic-names t1,t2]")
	}
	modelID = strings.TrimSpace(modelID)
	prompt = strings.TrimSpace(prompt)
	if modelID == "" || prompt == "" {
		return usageFail(rt, "usage: omni ai pick-topic --model-id <uuid> --prompt <text> [--current-topic-name <name>] [--branch-id <id>] [--potential-topic-names t1,t2]")
	}

	payload := map[string]any{
		"modelId": modelID,
		"prompt":  prompt,
	}
	if strings.TrimSpace(currentTopicName) != "" {
		payload["currentTopicName"] = strings.TrimSpace(currentTopicName)
	}
	if strings.TrimSpace(branchID) != "" {
		payload["branchId"] = strings.TrimSpace(branchID)
	}
	if strings.TrimSpace(potentialTopicsCSV) != "" {
		rawItems := strings.Split(potentialTopicsCSV, ",")
		items := make([]string, 0, len(rawItems))
		for _, item := range rawItems {
			v := strings.TrimSpace(item)
			if v != "" {
				items = append(items, v)
			}
		}
		if len(items) > 0 {
			payload["potentialTopicNames"] = items
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fail(rt, 1, codeValidationError, "failed to encode ai request", map[string]any{"error": err.Error()})
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.AIPickTopic(ctx, body)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "ai pick-topic request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "ai pick-topic", resp.Body)
}

func printAIUsage() {
	fmt.Print(`omni ai commands:
  omni ai generate-query --model-id <uuid> --prompt <text> [--current-topic-name <name>] [--workbook-url] [--branch-id <id>] [--structured] [--context-query-file <json-path>]
  omni ai workbook --model-id <uuid> --prompt <text> [--current-topic-name <name>] [--branch-id <id>] [--structured] [--context-query-file <json-path>]
  omni ai pick-topic --model-id <uuid> --prompt <text> [--current-topic-name <name>] [--branch-id <id>] [--potential-topic-names t1,t2]
`)
}
