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
)

func runDashboards(rt *runtime, args []string) int {
	if len(args) == 0 {
		printDashboardsUsage()
		return 0
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "download":
		return runDashboardsDownload(rt, subArgs)
	case "download-status":
		return runDashboardsDownloadStatus(rt, subArgs)
	case "download-file":
		return runDashboardsDownloadFile(rt, subArgs)
	case "filters":
		return runDashboardsFilters(rt, subArgs)
	default:
		printDashboardsUsage()
		return usageFail(rt, fmt.Sprintf("unknown dashboards subcommand: %s", sub))
	}
}

func runDashboardsDownload(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("dashboards download", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var filePath, userID string
	fs.StringVar(&filePath, "file", "", "Path to dashboard download JSON body")
	fs.StringVar(&userID, "user-id", "", "Optional target user membership ID")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printDashboardsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 1 || strings.TrimSpace(filePath) == "" {
		return usageFail(rt, "usage: omni dashboards download <identifier> --file <json-path> [--user-id <id>]")
	}
	identifier := strings.TrimSpace(fs.Arg(0))
	if identifier == "" {
		return usageFail(rt, "dashboard identifier is required")
	}

	payload, err := readJSONFile(filePath)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to read dashboards download body", map[string]any{"error": err.Error()})
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := api.DashboardDownload(ctx, identifier, userID, payload)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "dashboards download request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "dashboards download", resp.Body)
}

func runDashboardsDownloadStatus(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("dashboards download-status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var userID string
	fs.StringVar(&userID, "user-id", "", "Optional target user membership ID")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printDashboardsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 2 {
		return usageFail(rt, "usage: omni dashboards download-status <identifier> <job-id> [--user-id <id>]")
	}
	identifier := strings.TrimSpace(fs.Arg(0))
	jobID := strings.TrimSpace(fs.Arg(1))
	if identifier == "" || jobID == "" {
		return usageFail(rt, "dashboard identifier and job-id are required")
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.DashboardDownloadStatus(ctx, identifier, jobID, userID)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "dashboards download-status request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "dashboards download-status", resp.Body)
}

func runDashboardsDownloadFile(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("dashboards download-file", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var userID string
	fs.StringVar(&userID, "user-id", "", "Optional target user membership ID")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printDashboardsUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 2 {
		return usageFail(rt, "usage: omni dashboards download-file <identifier> <job-id> [--user-id <id>]")
	}
	identifier := strings.TrimSpace(fs.Arg(0))
	jobID := strings.TrimSpace(fs.Arg(1))
	if identifier == "" || jobID == "" {
		return usageFail(rt, "dashboard identifier and job-id are required")
	}

	api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := api.DashboardDownloadFile(ctx, identifier, jobID, userID)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "dashboards download-file request failed", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode(), "dashboards download-file", resp.Body)
}

func runDashboardsFilters(rt *runtime, args []string) int {
	if len(args) == 0 {
		return usageFail(rt, "usage: omni dashboards filters <get|update> ...")
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "get":
		fs := flag.NewFlagSet("dashboards filters get", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var userID string
		fs.StringVar(&userID, "user-id", "", "Optional target user membership ID")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 1 {
			return usageFail(rt, "usage: omni dashboards filters get <identifier> [--user-id <id>]")
		}
		identifier := strings.TrimSpace(fs.Arg(0))
		if identifier == "" {
			return usageFail(rt, "dashboard identifier is required")
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.GetDashboardFilters(ctx, identifier, userID)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "dashboards filters get request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "dashboards filters get", resp.Body)
	case "update":
		fs := flag.NewFlagSet("dashboards filters update", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var userID, filePath string
		fs.StringVar(&userID, "user-id", "", "Optional target user membership ID")
		fs.StringVar(&filePath, "file", "", "Path to dashboard filters update JSON body")
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return 0
			}
			return usageFail(rt, err.Error())
		}
		if fs.NArg() != 1 || strings.TrimSpace(filePath) == "" {
			return usageFail(rt, "usage: omni dashboards filters update <identifier> --file <json-path> [--user-id <id>]")
		}
		identifier := strings.TrimSpace(fs.Arg(0))
		if identifier == "" {
			return usageFail(rt, "dashboard identifier is required")
		}
		payload, err := readJSONFile(filePath)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to read dashboards filters update body", map[string]any{"error": err.Error()})
		}
		api, err := client.New(rt.Resolved.Profile.BaseURL, rt.Resolved.Profile.Token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resp, err := api.UpdateDashboardFilters(ctx, identifier, userID, payload)
		if err != nil {
			return fail(rt, 1, codeNetworkError, "dashboards filters update request failed", map[string]any{"error": err.Error()})
		}
		return succeedOrFail(rt, resp.StatusCode(), "dashboards filters update", resp.Body)
	default:
		return usageFail(rt, fmt.Sprintf("unknown dashboards filters subcommand: %s", sub))
	}
}

func printDashboardsUsage() {
	fmt.Print(`omni dashboards commands:
  omni dashboards download <identifier> --file <json-path> [--user-id <id>]
  omni dashboards download-status <identifier> <job-id> [--user-id <id>]
  omni dashboards download-file <identifier> <job-id> [--user-id <id>]
  omni dashboards filters get <identifier> [--user-id <id>]
  omni dashboards filters update <identifier> --file <json-path> [--user-id <id>]
`)
}
