package cli

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func runAPI(rt *runtime, args []string) int {
	if len(args) == 0 {
		printAPIUsage()
		return 0
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "call":
		return runAPICall(rt, subArgs)
	default:
		printAPIUsage()
		return usageFail(rt, fmt.Sprintf("unknown api subcommand: %s", sub))
	}
}

func runAPICall(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("api call", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var method string
	var path string
	var bodyFile string
	var body string
	var timeoutSec int
	var headers headerArgs

	fs.StringVar(&method, "method", "GET", "HTTP method")
	fs.StringVar(&path, "path", "", "Path to call, e.g. /api/v1/documents or /documents")
	fs.StringVar(&bodyFile, "body-file", "", "Path to request body file")
	fs.StringVar(&body, "body", "", "Inline request body")
	fs.IntVar(&timeoutSec, "timeout-seconds", 30, "HTTP timeout in seconds")
	fs.Var(&headers, "header", "Additional request header in Key:Value format (repeatable)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printAPIUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}
	if fs.NArg() != 0 {
		return usageFail(rt, "usage: omni api call --method GET --path /api/v1/documents [--body-file payload.json] [--header Key:Value]")
	}
	if strings.TrimSpace(path) == "" {
		return usageFail(rt, "--path is required")
	}
	if strings.TrimSpace(bodyFile) != "" && strings.TrimSpace(body) != "" {
		return usageFail(rt, "provide either --body-file or --body, not both")
	}

	var payload []byte
	var err error
	if strings.TrimSpace(bodyFile) != "" {
		payload, err = readJSONFile(bodyFile)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to read request body file", map[string]any{"error": err.Error()})
		}
	}
	if strings.TrimSpace(body) != "" {
		payload = []byte(body)
	}

	reqURL := buildAPIURL(rt.Resolved.Profile.BaseURL, path)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(strings.TrimSpace(method)), reqURL, bytes.NewReader(payload))
	if err != nil {
		return fail(rt, 1, codeValidationError, "failed to build request", map[string]any{"error": err.Error()})
	}
	req.Header.Set("Authorization", "Bearer "+rt.Resolved.Profile.Token)
	req.Header.Set("Accept", "application/json")
	for k, v := range headers.Values() {
		req.Header.Set(k, v)
	}
	if len(payload) > 0 && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	httpClient := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "api request failed", map[string]any{"error": err.Error()})
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fail(rt, 1, codeNetworkError, "failed to read API response body", map[string]any{"error": err.Error()})
	}
	return succeedOrFail(rt, resp.StatusCode, "api call", respBody)
}

func buildAPIURL(baseURL, path string) string {
	baseURL = strings.TrimSpace(baseURL)
	baseURL = strings.TrimRight(baseURL, "/")
	baseURL = strings.TrimSuffix(baseURL, "/api/v1")
	baseURL = strings.TrimSuffix(baseURL, "/api")

	path = strings.TrimSpace(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !strings.HasPrefix(path, "/api/") {
		if strings.HasPrefix(path, "/v1/") || path == "/v1" {
			path = "/api" + path
		} else {
			path = "/api/v1" + path
		}
	}
	return baseURL + path
}

type headerArgs []string

func (h *headerArgs) String() string {
	return strings.Join(*h, ",")
}

func (h *headerArgs) Set(value string) error {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid header %q: expected Key:Value", value)
	}
	key := strings.TrimSpace(parts[0])
	val := strings.TrimSpace(parts[1])
	if key == "" {
		return fmt.Errorf("invalid header %q: missing header key", value)
	}
	*h = append(*h, key+":"+val)
	return nil
}

func (h headerArgs) Values() map[string]string {
	out := make(map[string]string, len(h))
	for _, item := range h {
		parts := strings.SplitN(item, ":", 2)
		if len(parts) == 2 {
			out[parts[0]] = parts[1]
		}
	}
	return out
}

func printAPIUsage() {
	fmt.Print(`omni api commands:
  omni api call --method GET --path /api/v1/documents
  omni api call --method POST --path /api/v1/documents --body-file payload.json
`)
}
