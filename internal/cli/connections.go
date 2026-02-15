package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/omni-co/omni-cli/internal/client"
	"github.com/omni-co/omni-cli/internal/output"
)

func runConnections(rt *runtime, args []string) int {
	if len(args) == 0 {
		printConnectionsUsage()
		return 0
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "list":
		return runConnectionsList(rt, subArgs)
	case "create":
		return runConnectionsCreate(rt, subArgs)
	case "update":
		return runConnectionsUpdate(rt, subArgs)
	case "dbt":
		return runConnectionsDBT(rt, subArgs)
	case "schedules":
		return runConnectionsSchedules(rt, subArgs)
	case "environments":
		return runConnectionEnvironments(rt, subArgs)
	default:
		printConnectionsUsage()
		return usageFail(rt, fmt.Sprintf("unknown connections subcommand: %s", sub))
	}
}

func runConnectionsList(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("connections list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var name string
	fs.StringVar(&name, "name", "", "Filter by connection name")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printConnectionsUsage()
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

	resp, err := api.ListConnections(ctx, name)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "connections list request failed", map[string]any{"error": err.Error()})
	}
	if resp.JSON200 != nil {
		if err := output.Print(resp.JSON200, rt.JSON, rt.Plain); err != nil {
			return fail(rt, 1, codeAPIError, "failed to print connections list", map[string]any{"error": err.Error()})
		}
		return 0
	}

	return failFromHTTPStatus(rt, resp.StatusCode(), "connections list", resp.Body)
}

func runConnectionsCreate(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("connections create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var filePath string
	fs.StringVar(&filePath, "file", "", "Path to connections create JSON body")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printConnectionsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 0 || strings.TrimSpace(filePath) == "" {
		return usageFail(rt, "usage: omni connections create --file <json-path>")
	}

	payload, err := readJSONFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read connections create body", map[string]any{"error": err.Error()})
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := api.CreateConnection(ctx, payload)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "connections create request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "connections create", resp.Body)
}

func runConnectionsUpdate(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("connections update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var filePath string
	fs.StringVar(&filePath, "file", "", "Path to connections update JSON body")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printConnectionsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 || strings.TrimSpace(filePath) == "" {
		return usageFail(rt, "usage: omni connections update <connection-id> --file <json-path>")
	}
	id, err := parseUUIDArg(fs.Arg(0), "connection-id")
	if err != nil {
		return usageFail(rt, err.Error())
	}
	payload, err := readJSONFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read connections update body", map[string]any{"error": err.Error()})
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := api.UpdateConnection(ctx, id, payload)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "connections update request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "connections update", resp.Body)
}

func runConnectionsDBT(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni connections dbt <get|update|delete> ...")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "get":
		if len(rest) != 1 {
			return usageFail(rt, "usage: omni connections dbt get <connection-id>")
		}
		connectionID, err := parseUUIDArg(rest[0], "connection-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.GetConnectionDBT(ctx, connectionID)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "connections dbt get request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "connections dbt get", resp.Body)
	case "delete":
		if len(rest) != 1 {
			return usageFail(rt, "usage: omni connections dbt delete <connection-id>")
		}
		connectionID, err := parseUUIDArg(rest[0], "connection-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.DeleteConnectionDBT(ctx, connectionID)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "connections dbt delete request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "connections dbt delete", resp.Body)
	case "update":
		fs := flag.NewFlagSet("connections dbt update", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var filePath string
		fs.StringVar(&filePath, "file", "", "Path to dbt update JSON body")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 1 || strings.TrimSpace(filePath) == "" {
			return usageFail(rt, "usage: omni connections dbt update <connection-id> --file <json-path>")
		}
		connectionID, err := parseUUIDArg(fs.Arg(0), "connection-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		payload, err := readJSONFile(filePath)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to read dbt update body", map[string]any{"error": err.Error()})
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		resp, err := api.UpdateConnectionDBT(ctx, connectionID, payload)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "connections dbt update request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "connections dbt update", resp.Body)
	default:
		return usageFail(rt, fmt.Sprintf("unknown connections dbt subcommand: %s", sub))
	}
}

func runConnectionsSchedules(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni connections schedules <list|create|get|update|delete> ...")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		if len(rest) != 1 {
			return usageFail(rt, "usage: omni connections schedules list <connection-id>")
		}
		connectionID, err := parseUUIDArg(rest[0], "connection-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.ListConnectionSchedules(ctx, connectionID)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "connections schedules list request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "connections schedules list", resp.Body)
	case "get":
		if len(rest) != 2 {
			return usageFail(rt, "usage: omni connections schedules get <connection-id> <schedule-id>")
		}
		connectionID, err := parseUUIDArg(rest[0], "connection-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		scheduleID, err := parseUUIDArg(rest[1], "schedule-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.GetConnectionSchedule(ctx, connectionID, scheduleID)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "connections schedules get request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "connections schedules get", resp.Body)
	case "delete":
		if len(rest) != 2 {
			return usageFail(rt, "usage: omni connections schedules delete <connection-id> <schedule-id>")
		}
		connectionID, err := parseUUIDArg(rest[0], "connection-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		scheduleID, err := parseUUIDArg(rest[1], "schedule-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.DeleteConnectionSchedule(ctx, connectionID, scheduleID)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "connections schedules delete request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "connections schedules delete", resp.Body)
	case "create":
		fs := flag.NewFlagSet("connections schedules create", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var filePath string
		fs.StringVar(&filePath, "file", "", "Path to schedule create JSON body")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 1 || strings.TrimSpace(filePath) == "" {
			return usageFail(rt, "usage: omni connections schedules create <connection-id> --file <json-path>")
		}
		connectionID, err := parseUUIDArg(fs.Arg(0), "connection-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		payload, err := readJSONFile(filePath)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to read schedule create body", map[string]any{"error": err.Error()})
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.CreateConnectionSchedule(ctx, connectionID, payload)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "connections schedules create request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "connections schedules create", resp.Body)
	case "update":
		fs := flag.NewFlagSet("connections schedules update", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var filePath string
		fs.StringVar(&filePath, "file", "", "Path to schedule update JSON body")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 2 || strings.TrimSpace(filePath) == "" {
			return usageFail(rt, "usage: omni connections schedules update <connection-id> <schedule-id> --file <json-path>")
		}
		connectionID, err := parseUUIDArg(fs.Arg(0), "connection-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		scheduleID, err := parseUUIDArg(fs.Arg(1), "schedule-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		payload, err := readJSONFile(filePath)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to read schedule update body", map[string]any{"error": err.Error()})
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.UpdateConnectionSchedule(ctx, connectionID, scheduleID, payload)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "connections schedules update request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "connections schedules update", resp.Body)
	default:
		return usageFail(rt, fmt.Sprintf("unknown connections schedules subcommand: %s", sub))
	}
}

func runConnectionEnvironments(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni connections environments <list|create|update|delete> ...")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		if len(rest) != 0 {
			return usageFail(rt, "usage: omni connections environments list")
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.ListConnectionEnvironments(ctx)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "connections environments list request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "connections environments list", resp.Body)
	case "create":
		fs := flag.NewFlagSet("connections environments create", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var filePath string
		fs.StringVar(&filePath, "file", "", "Path to environments create JSON body")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 0 || strings.TrimSpace(filePath) == "" {
			return usageFail(rt, "usage: omni connections environments create --file <json-path>")
		}
		payload, err := readJSONFile(filePath)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to read environments create body", map[string]any{"error": err.Error()})
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.CreateConnectionEnvironment(ctx, payload)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "connections environments create request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "connections environments create", resp.Body)
	case "update":
		fs := flag.NewFlagSet("connections environments update", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var filePath string
		fs.StringVar(&filePath, "file", "", "Path to environments update JSON body")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 1 || strings.TrimSpace(filePath) == "" {
			return usageFail(rt, "usage: omni connections environments update <environment-id> --file <json-path>")
		}
		environmentID, err := parseUUIDArg(fs.Arg(0), "environment-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		payload, err := readJSONFile(filePath)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to read environments update body", map[string]any{"error": err.Error()})
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.UpdateConnectionEnvironment(ctx, environmentID, payload)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "connections environments update request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "connections environments update", resp.Body)
	case "delete":
		if len(rest) != 1 {
			return usageFail(rt, "usage: omni connections environments delete <environment-id>")
		}
		environmentID, err := parseUUIDArg(rest[0], "environment-id")
		if err != nil {
			return usageFail(rt, err.Error())
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.DeleteConnectionEnvironment(ctx, environmentID)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "connections environments delete request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "connections environments delete", resp.Body)
	default:
		return usageFail(rt, fmt.Sprintf("unknown connections environments subcommand: %s", sub))
	}
}

func printConnectionsUsage() {
	fmt.Print(`omni connections commands:
  omni connections list [--name <filter>]
  omni connections create --file <json-path>
  omni connections update <connection-id> --file <json-path>
  omni connections dbt get <connection-id>
  omni connections dbt update <connection-id> --file <json-path>
  omni connections dbt delete <connection-id>
  omni connections schedules list <connection-id>
  omni connections schedules create <connection-id> --file <json-path>
  omni connections schedules get <connection-id> <schedule-id>
  omni connections schedules update <connection-id> <schedule-id> --file <json-path>
  omni connections schedules delete <connection-id> <schedule-id>
  omni connections environments list
  omni connections environments create --file <json-path>
  omni connections environments update <environment-id> --file <json-path>
  omni connections environments delete <environment-id>
`)
}
