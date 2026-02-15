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

	"github.com/google/uuid"
	"github.com/omni-co/omni-cli/internal/client"
)

func runSchedules(rt *runtime, args []string) int {
	if len(args) == 0 {
		printSchedulesUsage()
		return 0
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "list":
		return runSchedulesList(rt, subArgs)
	case "create":
		return runSchedulesCreate(rt, subArgs)
	case "get":
		return runSchedulesGet(rt, subArgs)
	case "update":
		return runSchedulesUpdate(rt, subArgs)
	case "delete", "rm":
		return runSchedulesDelete(rt, subArgs)
	case "pause":
		return runSchedulesPause(rt, subArgs)
	case "resume":
		return runSchedulesResume(rt, subArgs)
	case "trigger":
		return runSchedulesTrigger(rt, subArgs)
	case "recipients":
		return runSchedulesRecipients(rt, subArgs)
	case "transfer-ownership":
		return runSchedulesTransferOwnership(rt, subArgs)
	default:
		printSchedulesUsage()
		return usageFail(rt, fmt.Sprintf("unknown schedules subcommand: %s", sub))
	}
}

func runSchedulesList(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("schedules list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var cursor, q string
	var pageSize int
	fs.StringVar(&cursor, "cursor", "", "Pagination cursor")
	fs.IntVar(&pageSize, "page-size", 20, "Number of records per page")
	fs.StringVar(&q, "q", "", "Search query")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printSchedulesUsage()
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

	resp, err := api.ListSchedules(ctx, cursor, pageSize, q)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "schedules list request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "schedules list", resp.Body)
}

func runSchedulesCreate(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("schedules create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var filePath string
	var userIDArg string
	fs.StringVar(&filePath, "file", "", "Path to schedules create JSON body")
	fs.StringVar(&userIDArg, "user-id", "", "Optional owner membership UUID for org-scoped API keys")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printSchedulesUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if strings.TrimSpace(filePath) == "" {
		return usageFail(rt, "usage: omni schedules create --file <json-path> [--user-id <uuid>]")
	}
	userID, err := parseOptionalUUIDArg(userIDArg, "user-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}

	payload, err := readJSONFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read schedules create body", map[string]any{"error": err.Error()})
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := api.CreateSchedule(ctx, userID, payload)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "schedules create request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "schedules create", resp.Body)
}

func runSchedulesGet(rt *runtime, args []string) int {
	id, err := parseScheduleIDArg(args)
	if err != nil {
		return usageFail(rt, err.Error())
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.GetSchedule(ctx, id)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "schedules get request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "schedules get", resp.Body)
}

func runSchedulesDelete(rt *runtime, args []string) int {
	id, err := parseScheduleIDArg(args)
	if err != nil {
		return usageFail(rt, err.Error())
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.DeleteSchedule(ctx, id)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "schedules delete request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "schedules delete", resp.Body)
}

func runSchedulesUpdate(rt *runtime, args []string) int {
	id, err := parseScheduleIDArg(args)
	if err != nil {
		return usageFail(rt, err.Error())
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.UpdateSchedule(ctx, id)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "schedules update request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "schedules update", resp.Body)
}

func runSchedulesPause(rt *runtime, args []string) int {
	id, err := parseScheduleIDArg(args)
	if err != nil {
		return usageFail(rt, err.Error())
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.PauseSchedule(ctx, id)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "schedules pause request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "schedules pause", resp.Body)
}

func runSchedulesResume(rt *runtime, args []string) int {
	id, err := parseScheduleIDArg(args)
	if err != nil {
		return usageFail(rt, err.Error())
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.ResumeSchedule(ctx, id)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "schedules resume request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "schedules resume", resp.Body)
}

func runSchedulesTrigger(rt *runtime, args []string) int {
	id, err := parseScheduleIDArg(args)
	if err != nil {
		return usageFail(rt, err.Error())
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.TriggerSchedule(ctx, id)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "schedules trigger request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "schedules trigger", resp.Body)
}

func runSchedulesRecipients(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni schedules recipients <get|add|remove> ...")
	}
	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "get":
		return runSchedulesRecipientsGet(rt, subArgs)
	case "add":
		return runSchedulesRecipientsMutation(rt, subArgs, "add")
	case "remove", "rm":
		return runSchedulesRecipientsMutation(rt, subArgs, "remove")
	default:
		return usageFail(rt, fmt.Sprintf("unknown schedules recipients subcommand: %s", sub))
	}
}

func runSchedulesRecipientsGet(rt *runtime, args []string) int {
	id, err := parseScheduleIDArg(args)
	if err != nil {
		return usageFail(rt, err.Error())
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.GetScheduleRecipients(ctx, id)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "schedules recipients get request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "schedules recipients get", resp.Body)
}

func runSchedulesRecipientsMutation(rt *runtime, args []string, action string) int {
	fs := flag.NewFlagSet("schedules recipients "+action, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var filePath string
	fs.StringVar(&filePath, "file", "", "Path to recipients JSON body")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printSchedulesUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 || strings.TrimSpace(filePath) == "" {
		return usageFail(rt, fmt.Sprintf("usage: omni schedules recipients %s <schedule-id> --file <json-path>", action))
	}

	id, err := parseUUIDArg(fs.Arg(0), "schedule-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}
	payload, err := readJSONFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read recipients body", map[string]any{"error": err.Error()})
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if action == "add" {
		resp, reqErr := api.AddScheduleRecipients(ctx, id, payload)
		if reqErr != nil {
			return fail(rt, 1, codeNetworkError, "schedules recipients add request failed", map[string]any{"error": reqErr.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "schedules recipients add", resp.Body)
	}
	resp, reqErr := api.RemoveScheduleRecipients(ctx, id, payload)
	if reqErr != nil {
		return fail(rt, 1, codeNetworkError, "schedules recipients remove request failed", map[string]any{"error": reqErr.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "schedules recipients remove", resp.Body)
}

func runSchedulesTransferOwnership(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("schedules transfer-ownership", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var userIDArg string
	var filePath string
	fs.StringVar(&userIDArg, "user-id", "", "User membership UUID to transfer ownership to")
	fs.StringVar(&filePath, "file", "", "Path to transfer ownership JSON body (alternative to --user-id)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printSchedulesUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 {
		return usageFail(rt, "usage: omni schedules transfer-ownership <schedule-id> (--user-id <uuid> | --file <json-path>)")
	}
	if strings.TrimSpace(userIDArg) == "" && strings.TrimSpace(filePath) == "" {
		return usageFail(rt, "usage: omni schedules transfer-ownership <schedule-id> (--user-id <uuid> | --file <json-path>)")
	}
	if strings.TrimSpace(userIDArg) != "" && strings.TrimSpace(filePath) != "" {
		return usageFail(rt, "provide either --user-id or --file, not both")
	}

	id, err := parseUUIDArg(fs.Arg(0), "schedule-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}

	var payload []byte
	if strings.TrimSpace(filePath) != "" {
		payload, err = readJSONFile(filePath)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to read transfer ownership body", map[string]any{"error": err.Error()})
		}
	} else {
		userID, parseErr := parseUUIDArg(userIDArg, "user-id")
		if parseErr != nil {
			return usageFail(rt, parseErr.Error())
		}
		payload, err = json.Marshal(map[string]any{"userId": userID})
		if err != nil {
			return fail(rt, 1, codeValidationError, "failed to encode transfer ownership body", map[string]any{"error": err.Error()})
		}
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.TransferScheduleOwnership(ctx, id, payload)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "schedules transfer-ownership request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "schedules transfer-ownership", resp.Body)
}

func parseScheduleIDArg(args []string) (uuid.UUID, error) {
	if len(args) != 1 {
		return uuid.Nil, fmt.Errorf("usage: omni schedules <get|delete|pause|resume|trigger> <schedule-id>")
	}
	return parseUUIDArg(args[0], "schedule-id")
}

func printSchedulesUsage() {
	fmt.Print(`omni schedules commands:
  omni schedules list [--cursor <cursor>] [--page-size 20] [--q <query>]
  omni schedules create --file <json-path> [--user-id <uuid>]
  omni schedules get <schedule-id>
  omni schedules update <schedule-id>
  omni schedules delete <schedule-id>
  omni schedules pause <schedule-id>
  omni schedules resume <schedule-id>
  omni schedules trigger <schedule-id>
  omni schedules recipients get <schedule-id>
  omni schedules recipients add <schedule-id> --file <json-path>
  omni schedules recipients remove <schedule-id> --file <json-path>
  omni schedules transfer-ownership <schedule-id> (--user-id <uuid> | --file <json-path>)
`)
}
