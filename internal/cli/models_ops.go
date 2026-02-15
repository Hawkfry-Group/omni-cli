package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omni-co/omni-cli/internal/client"
)

func runModelsGet(rt *runtime, args []string) int {
	if len(args) != 1 {
		return usageFail(rt, "usage: omni models get <model-id>")
	}
	modelID, err := parseUUIDArg(args[0], "model-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}
	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	resp, err := api.GetModelByID(ctx, modelID)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "models get request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "models get", resp.Body)
}

func runModelsCreate(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("models create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var filePath string
	fs.StringVar(&filePath, "file", "", "Path to JSON request body")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printModelsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 0 || strings.TrimSpace(filePath) == "" {
		return usageFail(rt, "usage: omni models create --file <json-path>")
	}
	payload, err := readJSONFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read models create payload", map[string]any{"error": err.Error()})
	}
	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := api.CreateModel(ctx, payload)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "models create request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "models create", resp.Body)
}

func runModelsRefresh(rt *runtime, args []string) int {
	if len(args) != 1 {
		return usageFail(rt, "usage: omni models refresh <model-id>")
	}
	modelID, err := parseUUIDArg(args[0], "model-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}
	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	resp, err := api.RefreshModel(ctx, modelID)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "models refresh request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "models refresh", resp.Body)
}

func runModelsValidate(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("models validate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var branchIDArg string
	var limit int
	fs.StringVar(&branchIDArg, "branch-id", "", "Branch UUID")
	fs.IntVar(&limit, "limit", 0, "Maximum number of validation issues")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printModelsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 {
		return usageFail(rt, "usage: omni models validate <model-id> [--branch-id <uuid>] [--limit <n>]")
	}
	modelID, err := parseUUIDArg(fs.Arg(0), "model-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}
	branchID, err := parseOptionalUUIDArg(branchIDArg, "branch-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}
	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	resp, err := api.ValidateModel(ctx, modelID, branchID, limit)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "models validate request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "models validate", resp.Body)
}

func runModelsBranch(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni models branch <delete|merge> ...")
	}
	action := args[0]
	rest := args[1:]
	switch action {
	case "delete":
		if len(rest) != 2 {
			return usageFail(rt, "usage: omni models branch delete <model-id> <branch-name>")
		}
		modelID, err := parseUUIDArg(rest[0], "model-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.DeleteModelBranch(ctx, modelID, rest[1])
		if err != nil {
			return fail(rt, 1, codeNetworkError, "models branch delete request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "models branch delete", resp.Body)
	case "merge":
		fs := flag.NewFlagSet("models branch merge", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var filePath string
		fs.StringVar(&filePath, "file", "", "Path to JSON request body")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printModelsUsage()
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 2 || strings.TrimSpace(filePath) == "" {
			return usageFail(rt, "usage: omni models branch merge <model-id> <branch-name> --file <json-path>")
		}
		modelID, err := parseUUIDArg(fs.Arg(0), "model-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		payload, err := readJSONFile(filePath)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to read models branch merge payload", map[string]any{"error": err.Error()})
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		resp, err := api.MergeModelBranch(ctx, modelID, fs.Arg(1), payload)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "models branch merge request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "models branch merge", resp.Body)
	default:
		return usageFail(rt, fmt.Sprintf("unknown models branch action: %s", action))
	}
}

func runModelsCacheReset(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("models cache-reset", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var filePath string
	fs.StringVar(&filePath, "file", "", "Path to JSON request body")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printModelsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 2 || strings.TrimSpace(filePath) == "" {
		return usageFail(rt, "usage: omni models cache-reset <model-id> <policy-name> --file <json-path>")
	}
	modelID, err := parseUUIDArg(fs.Arg(0), "model-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}
	policyName := strings.TrimSpace(fs.Arg(1))
	if policyName == "" {
		return usageFail(rt, "policy-name is required")
	}
	payload, err := readJSONFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read models cache-reset payload", map[string]any{"error": err.Error()})
	}
	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	resp, err := api.ResetModelCache(ctx, modelID, policyName, payload)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "models cache-reset request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "models cache-reset", resp.Body)
}

func runModelsTopics(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni models topics <list|get|update|delete> ...")
	}
	action := args[0]
	rest := args[1:]
	switch action {
	case "list":
		fs := flag.NewFlagSet("models topics list", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var branchIDArg string
		fs.StringVar(&branchIDArg, "branch-id", "", "Branch UUID")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printModelsUsage()
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 1 {
			return usageFail(rt, "usage: omni models topics list <model-id> [--branch-id <uuid>]")
		}
		modelID, err := parseUUIDArg(fs.Arg(0), "model-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		branchID, err := parseOptionalUUIDArg(branchIDArg, "branch-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.ListModelTopics(ctx, modelID, branchID)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "models topics list request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "models topics list", resp.Body)
	case "get":
		fs := flag.NewFlagSet("models topics get", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var branchIDArg string
		fs.StringVar(&branchIDArg, "branch-id", "", "Branch UUID")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printModelsUsage()
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 2 {
			return usageFail(rt, "usage: omni models topics get <model-id> <topic-name> [--branch-id <uuid>]")
		}
		modelID, err := parseUUIDArg(fs.Arg(0), "model-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		branchID, err := parseOptionalUUIDArg(branchIDArg, "branch-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.GetModelTopic(ctx, modelID, fs.Arg(1), branchID)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "models topics get request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "models topics get", resp.Body)
	case "update":
		return runModelsTopicsMutation(rt, rest, "update")
	case "delete":
		return runModelsTopicsMutation(rt, rest, "delete")
	default:
		return usageFail(rt, fmt.Sprintf("unknown models topics action: %s", action))
	}
}

func runModelsTopicsMutation(rt *runtime, args []string, action string) int {
	fs := flag.NewFlagSet("models topics "+action, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var branchIDArg string
	var filePath string
	fs.StringVar(&branchIDArg, "branch-id", "", "Branch UUID")
	if action == "update" {
		fs.StringVar(&filePath, "file", "", "Path to JSON request body")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printModelsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if action == "update" {
		if fs.NArg() != 2 || strings.TrimSpace(filePath) == "" {
			return usageFail(rt, "usage: omni models topics update <model-id> <topic-name> --file <json-path> [--branch-id <uuid>]")
		}
	} else if fs.NArg() != 2 {
		return usageFail(rt, "usage: omni models topics delete <model-id> <topic-name> [--branch-id <uuid>]")
	}
	modelID, err := parseUUIDArg(fs.Arg(0), "model-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}
	branchID, err := parseOptionalUUIDArg(branchIDArg, "branch-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if action == "update" {
		payload, readErr := readJSONFile(filePath)
		if readErr != nil {
			return fail(rt, 1, codeConfigError, "failed to read models topics payload", map[string]any{"error": readErr.Error()})
		}
		resp, reqErr := api.UpdateModelTopic(ctx, modelID, fs.Arg(1), branchID, payload)
		if reqErr != nil {
			return fail(rt, 1, codeNetworkError, "models topics update request failed", map[string]any{"error": reqErr.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "models topics update", resp.Body)
	}
	resp, reqErr := api.DeleteModelTopic(ctx, modelID, fs.Arg(1), branchID)
	if reqErr != nil {
		return fail(rt, 1, codeNetworkError, "models topics delete request failed", map[string]any{"error": reqErr.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "models topics delete", resp.Body)
}

func runModelsViews(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni models views <list|update|delete> ...")
	}
	action := args[0]
	rest := args[1:]
	switch action {
	case "list":
		fs := flag.NewFlagSet("models views list", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var branchIDArg string
		fs.StringVar(&branchIDArg, "branch-id", "", "Branch UUID")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printModelsUsage()
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 1 {
			return usageFail(rt, "usage: omni models views list <model-id> [--branch-id <uuid>]")
		}
		modelID, err := parseUUIDArg(fs.Arg(0), "model-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		branchID, err := parseOptionalUUIDArg(branchIDArg, "branch-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.ListModelViews(ctx, modelID, branchID)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "models views list request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "models views list", resp.Body)
	case "update":
		return runModelsViewsMutation(rt, rest, "update")
	case "delete":
		return runModelsViewsMutation(rt, rest, "delete")
	default:
		return usageFail(rt, fmt.Sprintf("unknown models views action: %s", action))
	}
}

func runModelsViewsMutation(rt *runtime, args []string, action string) int {
	fs := flag.NewFlagSet("models views "+action, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var branchIDArg string
	var filePath string
	fs.StringVar(&branchIDArg, "branch-id", "", "Branch UUID")
	if action == "update" {
		fs.StringVar(&filePath, "file", "", "Path to JSON request body")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printModelsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if action == "update" {
		if fs.NArg() != 2 || strings.TrimSpace(filePath) == "" {
			return usageFail(rt, "usage: omni models views update <model-id> <view-name> --file <json-path> [--branch-id <uuid>]")
		}
	} else if fs.NArg() != 2 {
		return usageFail(rt, "usage: omni models views delete <model-id> <view-name> [--branch-id <uuid>]")
	}
	modelID, err := parseUUIDArg(fs.Arg(0), "model-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}
	branchID, err := parseOptionalUUIDArg(branchIDArg, "branch-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}
	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if action == "update" {
		payload, readErr := readJSONFile(filePath)
		if readErr != nil {
			return fail(rt, 1, codeConfigError, "failed to read models views payload", map[string]any{"error": readErr.Error()})
		}
		resp, reqErr := api.UpdateModelView(ctx, modelID, fs.Arg(1), branchID, payload)
		if reqErr != nil {
			return fail(rt, 1, codeNetworkError, "models views update request failed", map[string]any{"error": reqErr.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "models views update", resp.Body)
	}
	resp, reqErr := api.DeleteModelView(ctx, modelID, fs.Arg(1), branchID)
	if reqErr != nil {
		return fail(rt, 1, codeNetworkError, "models views delete request failed", map[string]any{"error": reqErr.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "models views delete", resp.Body)
}

func runModelsFields(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni models fields <create|update|delete> ...")
	}
	action := args[0]
	rest := args[1:]
	switch action {
	case "create":
		fs := flag.NewFlagSet("models fields create", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var filePath string
		fs.StringVar(&filePath, "file", "", "Path to JSON request body")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printModelsUsage()
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 1 || strings.TrimSpace(filePath) == "" {
			return usageFail(rt, "usage: omni models fields create <model-id> --file <json-path>")
		}
		modelID, err := parseUUIDArg(fs.Arg(0), "model-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		payload, err := readJSONFile(filePath)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to read models fields payload", map[string]any{"error": err.Error()})
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.CreateModelField(ctx, modelID, payload)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "models fields create request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "models fields create", resp.Body)
	case "update":
		return runModelsFieldsMutation(rt, rest, "update")
	case "delete":
		return runModelsFieldsMutation(rt, rest, "delete")
	default:
		return usageFail(rt, fmt.Sprintf("unknown models fields action: %s", action))
	}
}

func runModelsFieldsMutation(rt *runtime, args []string, action string) int {
	fs := flag.NewFlagSet("models fields "+action, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var branchIDArg, filePath, topicContext string
	fs.StringVar(&branchIDArg, "branch-id", "", "Branch UUID")
	fs.StringVar(&topicContext, "topic-context", "", "Topic context for field deletion")
	if action == "update" {
		fs.StringVar(&filePath, "file", "", "Path to JSON request body")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printModelsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if action == "update" {
		if fs.NArg() != 3 || strings.TrimSpace(filePath) == "" {
			return usageFail(rt, "usage: omni models fields update <model-id> <view-name> <field-name> --file <json-path> [--branch-id <uuid>]")
		}
	} else if fs.NArg() != 3 {
		return usageFail(rt, "usage: omni models fields delete <model-id> <view-name> <field-name> [--branch-id <uuid>] [--topic-context <name>]")
	}
	modelID, err := parseUUIDArg(fs.Arg(0), "model-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}
	branchID, err := parseOptionalUUIDArg(branchIDArg, "branch-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}
	viewName := fs.Arg(1)
	fieldName := fs.Arg(2)
	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if action == "update" {
		payload, readErr := readJSONFile(filePath)
		if readErr != nil {
			return fail(rt, 1, codeConfigError, "failed to read models fields payload", map[string]any{"error": readErr.Error()})
		}
		resp, reqErr := api.UpdateModelField(ctx, modelID, viewName, fieldName, branchID, payload)
		if reqErr != nil {
			return fail(rt, 1, codeNetworkError, "models fields update request failed", map[string]any{"error": reqErr.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "models fields update", resp.Body)
	}
	resp, reqErr := api.DeleteModelField(ctx, modelID, viewName, fieldName, branchID, topicContext)
	if reqErr != nil {
		return fail(rt, 1, codeNetworkError, "models fields delete request failed", map[string]any{"error": reqErr.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "models fields delete", resp.Body)
}

func runModelsGit(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni models git <get|create|update|delete|sync> ...")
	}
	action := args[0]
	rest := args[1:]
	switch action {
	case "get":
		fs := flag.NewFlagSet("models git get", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var include string
		fs.StringVar(&include, "include", "", "Optional include fields")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printModelsUsage()
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 1 {
			return usageFail(rt, "usage: omni models git get <model-id> [--include webhookSecret]")
		}
		modelID, err := parseUUIDArg(fs.Arg(0), "model-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.GetModelGit(ctx, modelID, include)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "models git get request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "models git get", resp.Body)
	case "delete":
		if len(rest) != 1 {
			return usageFail(rt, "usage: omni models git delete <model-id>")
		}
		modelID, err := parseUUIDArg(rest[0], "model-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.DeleteModelGit(ctx, modelID)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "models git delete request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "models git delete", resp.Body)
	case "create", "update", "sync":
		return runModelsGitMutation(rt, rest, action)
	default:
		return usageFail(rt, fmt.Sprintf("unknown models git action: %s", action))
	}
}

func runModelsGitMutation(rt *runtime, args []string, action string) int {
	fs := flag.NewFlagSet("models git "+action, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var filePath string
	fs.StringVar(&filePath, "file", "", "Path to JSON request body")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printModelsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 || strings.TrimSpace(filePath) == "" {
		return usageFail(rt, fmt.Sprintf("usage: omni models git %s <model-id> --file <json-path>", action))
	}
	modelID, err := parseUUIDArg(fs.Arg(0), "model-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}
	payload, err := readJSONFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read models git payload", map[string]any{"error": err.Error()})
	}
	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	switch action {
	case "create":
		resp, reqErr := api.CreateModelGit(ctx, modelID, payload)
		if reqErr != nil {
			return fail(rt, 1, codeNetworkError, "models git create request failed", map[string]any{"error": reqErr.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "models git create", resp.Body)
	case "update":
		resp, reqErr := api.UpdateModelGit(ctx, modelID, payload)
		if reqErr != nil {
			return fail(rt, 1, codeNetworkError, "models git update request failed", map[string]any{"error": reqErr.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "models git update", resp.Body)
	default:
		resp, reqErr := api.SyncModelGit(ctx, modelID, payload)
		if reqErr != nil {
			return fail(rt, 1, codeNetworkError, "models git sync request failed", map[string]any{"error": reqErr.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "models git sync", resp.Body)
	}
}

func runModelsMigrate(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("models migrate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var filePath string
	fs.StringVar(&filePath, "file", "", "Path to JSON request body")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printModelsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 || strings.TrimSpace(filePath) == "" {
		return usageFail(rt, "usage: omni models migrate <model-id> --file <json-path>")
	}
	modelID, err := parseUUIDArg(fs.Arg(0), "model-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}
	payload, err := readJSONFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read models migrate payload", map[string]any{"error": err.Error()})
	}
	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	resp, err := api.MigrateModel(ctx, modelID, payload)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "models migrate request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "models migrate", resp.Body)
}

func runModelsContentValidator(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni models content-validator <get|replace> ...")
	}
	action := args[0]
	rest := args[1:]
	switch action {
	case "get":
		fs := flag.NewFlagSet("models content-validator get", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var branchID, userID, includePersonalArg string
		fs.StringVar(&branchID, "branch-id", "", "Branch ID")
		fs.StringVar(&userID, "user-id", "", "User ID")
		fs.StringVar(&includePersonalArg, "include-personal-folders", "", "true|false")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printModelsUsage()
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 1 {
			return usageFail(rt, "usage: omni models content-validator get <model-id> [--branch-id <id>] [--user-id <id>] [--include-personal-folders true|false]")
		}
		modelID, err := parseUUIDArg(fs.Arg(0), "model-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		includePersonal, err := parseOptionalBool(includePersonalArg)
		if err != nil {
			return usageFail(rt, err.Error())
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.GetModelContentValidator(ctx, modelID, branchID, userID, includePersonal)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "models content-validator get request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "models content-validator get", resp.Body)
	case "replace":
		fs := flag.NewFlagSet("models content-validator replace", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var filePath, userID string
		fs.StringVar(&filePath, "file", "", "Path to JSON request body")
		fs.StringVar(&userID, "user-id", "", "Target user ID")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printModelsUsage()
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 1 || strings.TrimSpace(filePath) == "" {
			return usageFail(rt, "usage: omni models content-validator replace <model-id> --file <json-path> [--user-id <id>]")
		}
		modelID, err := parseUUIDArg(fs.Arg(0), "model-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		payload, err := readJSONFile(filePath)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to read models content-validator payload", map[string]any{"error": err.Error()})
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.ReplaceModelContentValidator(ctx, modelID, userID, payload)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "models content-validator replace request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "models content-validator replace", resp.Body)
	default:
		return usageFail(rt, fmt.Sprintf("unknown models content-validator action: %s", action))
	}
}

func runModelsYAML(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni models yaml <get|create> ...")
	}
	action := args[0]
	rest := args[1:]
	switch action {
	case "get":
		fs := flag.NewFlagSet("models yaml get", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var branchIDArg, fileName, mode, includeChecksumsArg string
		fs.StringVar(&branchIDArg, "branch-id", "", "Branch UUID")
		fs.StringVar(&fileName, "file-name", "", "YAML file name")
		fs.StringVar(&mode, "mode", "", "YAML mode")
		fs.StringVar(&includeChecksumsArg, "include-checksums", "", "true|false")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printModelsUsage()
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 1 {
			return usageFail(rt, "usage: omni models yaml get <model-id> [--branch-id <uuid>] [--file-name <name>] [--mode <mode>] [--include-checksums true|false]")
		}
		modelID, err := parseUUIDArg(fs.Arg(0), "model-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		branchID, err := parseOptionalUUIDArg(branchIDArg, "branch-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		includeChecksums, err := parseOptionalBool(includeChecksumsArg)
		if err != nil {
			return usageFail(rt, err.Error())
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.GetModelYAML(ctx, modelID, branchID, fileName, mode, includeChecksums)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "models yaml get request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "models yaml get", resp.Body)
	case "create":
		fs := flag.NewFlagSet("models yaml create", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var filePath string
		fs.StringVar(&filePath, "file", "", "Path to JSON request body")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printModelsUsage()
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 1 || strings.TrimSpace(filePath) == "" {
			return usageFail(rt, "usage: omni models yaml create <model-id> --file <json-path>")
		}
		modelID, err := parseUUIDArg(fs.Arg(0), "model-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		payload, err := readJSONFile(filePath)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to read models yaml payload", map[string]any{"error": err.Error()})
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.CreateModelYAML(ctx, modelID, payload)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "models yaml create request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "models yaml create", resp.Body)
	case "delete":
		fs := flag.NewFlagSet("models yaml delete", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var branchIDArg, fileName, mode, commitMessage string
		fs.StringVar(&branchIDArg, "branch-id", "", "Branch UUID")
		fs.StringVar(&fileName, "file-name", "", "YAML file name to delete")
		fs.StringVar(&mode, "mode", "", "YAML mode")
		fs.StringVar(&commitMessage, "commit-message", "", "Commit message")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printModelsUsage()
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 1 || strings.TrimSpace(fileName) == "" {
			return usageFail(rt, "usage: omni models yaml delete <model-id> --file-name <name> [--branch-id <uuid>] [--mode <mode>] [--commit-message <msg>]")
		}
		modelID, err := parseUUIDArg(fs.Arg(0), "model-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		branchID, err := parseOptionalUUIDArg(branchIDArg, "branch-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.DeleteModelYAML(ctx, modelID, fileName, branchID, mode, commitMessage)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "models yaml delete request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "models yaml delete", resp.Body)
	default:
		return usageFail(rt, fmt.Sprintf("unknown models yaml action: %s", action))
	}
}

func parseModelID(v string) (uuid.UUID, error) {
	return parseUUIDArg(v, "model-id")
}
