package cli

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteHelp(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.json")
	t.Setenv("OMNI_CONFIG", configPath)

	stdout, stderr, exit := captureExecute(t, []string{"--help"})
	if exit != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stdout, "omni - Omni CLI") {
		t.Fatalf("expected help output, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}

func TestExecuteUnknownCommandJSONError(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.json")
	t.Setenv("OMNI_CONFIG", configPath)

	stdout, stderr, exit := captureExecute(t, []string{"--json", "nope"})
	if exit != 2 {
		t.Fatalf("expected exit code 2, got %d", exit)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, `"code": "USAGE_ERROR"`) {
		t.Fatalf("expected USAGE_ERROR envelope, got %q", stderr)
	}
}

func TestExecuteProtectedCommandHelpWithoutConfig(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.json")
	t.Setenv("OMNI_CONFIG", configPath)

	stdout, stderr, exit := captureExecute(t, []string{"labels", "--help"})
	if exit != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stdout, "omni labels commands:") {
		t.Fatalf("expected labels usage output, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}

func TestExecuteUsersHelpWithoutConfig(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.json")
	t.Setenv("OMNI_CONFIG", configPath)

	stdout, stderr, exit := captureExecute(t, []string{"users", "--help"})
	if exit != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stdout, "omni users commands:") {
		t.Fatalf("expected users usage output, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}

func TestExecuteScimHelpWithoutConfig(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.json")
	t.Setenv("OMNI_CONFIG", configPath)

	stdout, stderr, exit := captureExecute(t, []string{"scim", "--help"})
	if exit != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stdout, "omni scim commands:") {
		t.Fatalf("expected scim usage output, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}

func TestExecuteAIHelpWithoutConfig(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.json")
	t.Setenv("OMNI_CONFIG", configPath)

	stdout, stderr, exit := captureExecute(t, []string{"ai", "--help"})
	if exit != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stdout, "omni ai commands:") {
		t.Fatalf("expected ai usage output, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}

func TestExecuteAPIHelpWithoutConfig(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.json")
	t.Setenv("OMNI_CONFIG", configPath)

	stdout, stderr, exit := captureExecute(t, []string{"api", "--help"})
	if exit != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stdout, "omni api commands:") {
		t.Fatalf("expected api usage output, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}

func TestExecuteUtilityCommandHelpWithoutConfig(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.json")
	t.Setenv("OMNI_CONFIG", configPath)

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "doctor", args: []string{"doctor", "--help"}, want: "omni doctor:"},
		{name: "query", args: []string{"query", "--help"}, want: "omni query commands:"},
		{name: "embed", args: []string{"embed", "--help"}, want: "omni embed commands:"},
		{name: "exit-codes", args: []string{"exit-codes", "--help"}, want: "omni exit-codes:"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, exit := captureExecute(t, tc.args)
			if exit != 0 {
				t.Fatalf("expected exit code 0, got %d (stderr=%q)", exit, stderr)
			}
			if !strings.Contains(stdout, tc.want) {
				t.Fatalf("expected help output %q, got %q", tc.want, stdout)
			}
			if strings.TrimSpace(stderr) != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}
		})
	}
}

func TestExecuteJSONPlainConflict(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.json")
	t.Setenv("OMNI_CONFIG", configPath)

	stdout, stderr, exit := captureExecute(t, []string{"--json", "--plain", "version"})
	if exit != 2 {
		t.Fatalf("expected exit code 2, got %d", exit)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "USAGE_ERROR") {
		t.Fatalf("expected USAGE_ERROR in stderr, got %q", stderr)
	}
}

func TestExecuteTrailingGlobalFlagAccepted(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.json")
	t.Setenv("OMNI_CONFIG", configPath)

	stdout, stderr, exit := captureExecute(t, []string{"completion", "bash", "--json"})
	if exit != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stdout, "# bash completion for omni") {
		t.Fatalf("expected bash completion output, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}

func TestExecuteAllowlistBlocksCommand(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.json")
	t.Setenv("OMNI_CONFIG", configPath)
	t.Setenv("OMNI_ENABLE_COMMANDS", "documents,query")

	stdout, stderr, exit := captureExecute(t, []string{"models", "list"})
	if exit != 1 {
		t.Fatalf("expected exit code 1, got %d", exit)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "AUTH_FORBIDDEN") {
		t.Fatalf("expected AUTH_FORBIDDEN in stderr, got %q", stderr)
	}
	if !strings.Contains(stderr, "allowed_commands") {
		t.Fatalf("expected allowlist details in stderr, got %q", stderr)
	}
}

func TestExecuteSchemaWithoutConfig(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.json")
	t.Setenv("OMNI_CONFIG", configPath)

	stdout, stderr, exit := captureExecute(t, []string{"schema"})
	if exit != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%q)", exit, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"name": "omni"`) {
		t.Fatalf("expected omni schema root, got %q", stdout)
	}
}

func TestExecuteSchemaPath(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.json")
	t.Setenv("OMNI_CONFIG", configPath)

	stdout, stderr, exit := captureExecute(t, []string{"schema", "documents", "permissions"})
	if exit != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%q)", exit, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"name": "permissions"`) {
		t.Fatalf("expected permissions schema, got %q", stdout)
	}
}

func TestExecuteSchemaUnknownPath(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.json")
	t.Setenv("OMNI_CONFIG", configPath)

	stdout, stderr, exit := captureExecute(t, []string{"schema", "nope"})
	if exit != 2 {
		t.Fatalf("expected exit code 2, got %d", exit)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "unknown command path for schema") {
		t.Fatalf("expected schema path usage error, got %q", stderr)
	}
}

func TestExecuteExitCodesJSONContract(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.json")
	t.Setenv("OMNI_CONFIG", configPath)

	stdout, stderr, exit := captureExecute(t, []string{"exit-codes"})
	if exit != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%q)", exit, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var payload struct {
		ExitCodes []struct {
			ExitCode int    `json:"exit_code"`
			Name     string `json:"name"`
		} `json:"exit_codes"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("unmarshal exit-codes output: %v\nbody=%q", err, stdout)
	}
	if len(payload.ExitCodes) == 0 {
		t.Fatalf("expected non-empty exit code contract, got %q", stdout)
	}
	if payload.ExitCodes[0].ExitCode != 0 || payload.ExitCodes[0].Name != "SUCCESS" {
		t.Fatalf("unexpected first exit code entry: %#v", payload.ExitCodes[0])
	}
}

func TestExecuteAllowlistStillAllowsVersion(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.json")
	t.Setenv("OMNI_CONFIG", configPath)
	t.Setenv("OMNI_ENABLE_COMMANDS", "documents")

	stdout, stderr, exit := captureExecute(t, []string{"version"})
	if exit != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%q)", exit, stderr)
	}
	if strings.TrimSpace(stdout) != "test" {
		t.Fatalf("expected version output 'test', got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}

func TestExecuteNoInputSetupDisablesPrompts(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.json")
	t.Setenv("OMNI_CONFIG", configPath)

	stdout, stderr, exit := captureExecute(t, []string{"--no-input", "setup"})
	if exit != 2 {
		t.Fatalf("expected exit code 2, got %d", exit)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if strings.Contains(stderr, "Omni CLI setup") {
		t.Fatalf("expected no prompt banner in stderr, got %q", stderr)
	}
	if !strings.Contains(stderr, "missing Omni URL") {
		t.Fatalf("expected missing URL usage error, got %q", stderr)
	}
}

func TestAllowlistParser(t *testing.T) {
	parsed := parseAllowlistedCommands(" documents, query , ,models ")
	if len(parsed) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(parsed))
	}
	if !isCommandAllowlisted("documents", parsed) {
		t.Fatal("expected documents to be allowlisted")
	}
	if isCommandAllowlisted("admin", parsed) {
		t.Fatal("expected admin to be blocked")
	}
	if !isCommandAllowlisted("version", parsed) {
		t.Fatal("expected version to always be allowed")
	}
}

func captureExecute(t *testing.T, args []string) (stdout, stderr string, exit int) {
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

	exit = Execute(args, "test")

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
