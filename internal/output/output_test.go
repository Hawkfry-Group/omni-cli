package output

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestPrintErrorJSONEnvelope(t *testing.T) {
	stderr := captureStderr(t, func() {
		PrintError(true, "AUTH_UNAUTHORIZED", "token unauthorized", map[string]any{"status": 401})
	})

	var env ErrorEnvelope
	if err := json.Unmarshal([]byte(stderr), &env); err != nil {
		t.Fatalf("expected valid json envelope, got error %v and body %q", err, stderr)
	}
	if env.Error.Code != "AUTH_UNAUTHORIZED" {
		t.Fatalf("expected code AUTH_UNAUTHORIZED, got %q", env.Error.Code)
	}
	if env.Error.Message != "token unauthorized" {
		t.Fatalf("expected message token unauthorized, got %q", env.Error.Message)
	}
	details, ok := env.Error.Details.(map[string]any)
	if !ok {
		t.Fatalf("expected details object, got %#v", env.Error.Details)
	}
	if details["status"] != float64(401) {
		t.Fatalf("expected details.status=401, got %#v", details["status"])
	}
}

func TestPrintErrorHuman(t *testing.T) {
	stderr := captureStderr(t, func() {
		PrintError(false, "CONFIG_MISSING", "missing token", nil)
	})
	if got := strings.TrimSpace(stderr); got != "CONFIG_MISSING: missing token" {
		t.Fatalf("unexpected human error format: %q", got)
	}
}

func TestPlainRecordsTSV(t *testing.T) {
	stdout := captureStdout(t, func() {
		err := Plain(os.Stdout, map[string]any{
			"records": []map[string]any{
				{"id": "a1", "name": "alpha", "active": true},
				{"id": "b2", "name": "beta", "active": false},
			},
		})
		if err != nil {
			t.Fatalf("Plain returned error: %v", err)
		}
	})

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 TSV lines, got %d: %q", len(lines), stdout)
	}
	if lines[0] != "active\tid\tname" {
		t.Fatalf("unexpected TSV header: %q", lines[0])
	}
	if lines[1] != "true\ta1\talpha" {
		t.Fatalf("unexpected first row: %q", lines[1])
	}
	if lines[2] != "false\tb2\tbeta" {
		t.Fatalf("unexpected second row: %q", lines[2])
	}
}

func TestPrintPlainMode(t *testing.T) {
	stdout := captureStdout(t, func() {
		if err := Print(map[string]any{"ok": true, "profile": "default"}, false, true); err != nil {
			t.Fatalf("Print returned error: %v", err)
		}
	})
	if strings.Contains(stdout, "{") {
		t.Fatalf("expected non-JSON plain output, got %q", stdout)
	}
	if !strings.Contains(stdout, "ok\ttrue") || !strings.Contains(stdout, "profile\tdefault") {
		t.Fatalf("unexpected plain output: %q", stdout)
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	os.Stderr = w

	defer func() {
		_ = w.Close()
		os.Stderr = oldStderr
	}()

	fn()
	_ = w.Close()

	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read captured stderr: %v", err)
	}
	return string(b)
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = w

	defer func() {
		_ = w.Close()
		os.Stdout = oldStdout
	}()

	fn()
	_ = w.Close()

	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	return string(b)
}
