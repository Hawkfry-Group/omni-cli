package cli

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestFailFromHTTPStatusReturnsStructuredErrors(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		want    string
		wantErr string
	}{
		{name: "unauthorized", status: http.StatusUnauthorized, want: codeAuthUnauthorized, wantErr: "unauthorized: test context"},
		{name: "forbidden", status: http.StatusForbidden, want: codeAuthForbidden, wantErr: "permission denied: test context"},
		{name: "not found", status: http.StatusNotFound, want: codeAPIError, wantErr: "resource not found: test context"},
		{name: "other", status: http.StatusBadGateway, want: codeAPIError, wantErr: "unexpected API response: test context"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rt := &runtime{JSON: true}
			stdout, stderr, exit := captureRuntimeIO(t, func() int {
				return failFromHTTPStatus(rt, tc.status, "test context", []byte(`{"detail":"nope"}`))
			})
			if exit != 1 {
				t.Fatalf("expected exit code 1, got %d", exit)
			}
			if strings.TrimSpace(stdout) != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			if !strings.Contains(stderr, tc.want) || !strings.Contains(stderr, tc.wantErr) {
				t.Fatalf("expected stderr to contain %q and %q, got %q", tc.want, tc.wantErr, stderr)
			}
		})
	}
}

func TestSucceedOrFailPrintsSuccessPayload(t *testing.T) {
	rt := &runtime{JSON: true}
	stdout, stderr, exit := captureRuntimeIO(t, func() int {
		return succeedOrFail(rt, http.StatusOK, "api call", []byte(`{"ok":true}`))
	})
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d", exit)
	}
	if !strings.Contains(stdout, `"ok": true`) {
		t.Fatalf("expected JSON body on stdout, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}

func TestParseRateLimitAndEnvHelpers(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-RateLimit-Limit", "100")
	headers.Set("X-RateLimit-Remaining", "42")
	headers.Set("X-RateLimit-Reset", "1700000000")
	resp := &http.Response{Header: headers}
	rateLimit := parseRateLimit(resp)
	if rateLimit["limit"] != 100 || rateLimit["remaining"] != 42 || rateLimit["reset"] != 1700000000 {
		t.Fatalf("unexpected rate limit map: %#v", rateLimit)
	}
	if parseRateLimit(nil) != nil {
		t.Fatal("expected nil rate limit map for nil response")
	}

	t.Setenv("OMNI_VERBOSE", "yes")
	if !parseEnvBool("OMNI_VERBOSE") {
		t.Fatal("expected yes to parse as true")
	}
	t.Setenv("OMNI_VERBOSE", "0")
	if parseEnvBool("OMNI_VERBOSE") {
		t.Fatal("expected 0 to parse as false")
	}
}

func TestRequiredAuthForCommand(t *testing.T) {
	if got := requiredAuthForCommand("admin"); got != "org" {
		t.Fatalf("expected org auth for admin, got %q", got)
	}
	if got := requiredAuthForCommand("documents"); got != "" {
		t.Fatalf("expected no forced auth for documents, got %q", got)
	}
}

func captureRuntimeIO(t *testing.T, fn func() int) (stdout, stderr string, exit int) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}

	os.Stdout = outW
	os.Stderr = errW

	exit = fn()

	_ = outW.Close()
	_ = errW.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	outBytes, readErr := io.ReadAll(outR)
	if readErr != nil {
		t.Fatalf("read stdout: %v", readErr)
	}
	errBytes, readErr := io.ReadAll(errR)
	if readErr != nil {
		t.Fatalf("read stderr: %v", readErr)
	}

	return string(outBytes), string(errBytes), exit
}
