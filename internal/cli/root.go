package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/omni-co/omni-cli/internal/auth"
	"github.com/omni-co/omni-cli/internal/config"
	"github.com/omni-co/omni-cli/internal/output"
	"github.com/omni-co/omni-cli/internal/secret"
)

type runtime struct {
	Version    string
	JSON       bool
	Plain      bool
	Verbose    bool
	NoInput    bool
	ConfigPath string
	Config     *config.Config
	Resolved   *auth.Resolved
	Keychain   secret.Store
	PATLogin   PATLoginFunc
}

func Execute(args []string, version string) int {
	global := flag.NewFlagSet("omni", flag.ContinueOnError)
	global.SetOutput(io.Discard)

	var profile string
	var url string
	var token string
	var authMode string
	var tokenType string
	var configPath string
	var asJSON bool
	var asPlain bool
	var verbose bool
	var noInput bool

	global.StringVar(&profile, "profile", "", "Profile name to use")
	global.StringVar(&url, "url", "", "Omni instance URL")
	global.StringVar(&token, "token", "", "Omni API token (PAT or org key)")
	global.StringVar(&authMode, "auth", "", "Auth to use: pat or org")
	global.StringVar(&tokenType, "token-type", "", "Token type: pat or org")
	global.StringVar(&configPath, "config", "", "Config file path")
	global.BoolVar(&asJSON, "json", false, "Print JSON output")
	global.BoolVar(&asPlain, "plain", false, "Print stable plain-text output")
	global.BoolVar(&verbose, "verbose", false, "Verbose logging")
	global.BoolVar(&noInput, "no-input", false, "Disable interactive prompts")

	if err := global.Parse(normalizeArgsForGlobalFlags(args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage()
			return 0
		}
		output.PrintError(asJSON, codeUsageError, err.Error(), nil)
		return 2
	}

	if configPath == "" {
		configPath = os.Getenv("OMNI_CONFIG")
	}
	if !asPlain {
		asPlain = parseEnvBool("OMNI_PLAIN")
	}
	if !noInput {
		noInput = parseEnvBool("OMNI_NO_INPUT")
	}
	if asJSON && asPlain {
		output.PrintError(asJSON, codeUsageError, "--json and --plain cannot be used together", nil)
		return 2
	}
	if configPath == "" {
		path, err := config.DefaultPath()
		if err != nil {
			output.PrintError(asJSON, codeConfigError, "failed to resolve config path", map[string]any{"error": err.Error()})
			return 1
		}
		configPath = path
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		output.PrintError(asJSON, codeConfigError, "failed to load config", map[string]any{"path": configPath, "error": err.Error()})
		return 1
	}

	rt := &runtime{
		Version:    version,
		JSON:       asJSON,
		Plain:      asPlain,
		Verbose:    verbose,
		NoInput:    noInput,
		ConfigPath: configPath,
		Config:     cfg,
		Keychain:   secret.NewKeychainStore(),
		PATLogin:   defaultPATLogin,
	}

	remaining := global.Args()
	if len(remaining) == 0 {
		printUsage()
		return 0
	}

	cmd := remaining[0]
	cmdArgs := remaining[1:]

	enabledCommands := parseAllowlistedCommands(os.Getenv("OMNI_ENABLE_COMMANDS"))
	if !isCommandAllowlisted(cmd, enabledCommands) {
		return fail(rt, 1, codeAuthForbidden, "command blocked by OMNI_ENABLE_COMMANDS allowlist", map[string]any{
			"command":          cmd,
			"allowed_commands": sortedCommandList(enabledCommands),
		})
	}

	switch cmd {
	case "help", "-h", "--help":
		printUsage()
		return 0
	case "version":
		fmt.Println(version)
		return 0
	case "schema":
		return runSchema(rt, cmdArgs)
	case "exit-codes":
		return runExitCodes(rt, cmdArgs)
	case "completion":
		return runCompletion(rt, cmdArgs)
	case "setup":
		return runSetup(rt, cmdArgs, setupDefaults{
			Profile: profile,
			URL:     url,
		})
	case "auth":
		return runAuth(rt, cmdArgs)
	case "doctor", "query", "jobs", "documents", "models", "connections", "folders", "labels", "schedules", "dashboards", "agentic", "embed", "unstable", "user-attributes", "admin", "users", "scim", "ai", "api":
		if len(cmdArgs) == 0 || wantsSubcommandHelp(cmdArgs) {
			printCommandUsage(cmd)
			return 0
		}

		resolved, err := auth.Resolve(cfg, auth.Options{
			ProfileFlag:   profile,
			URLFlag:       url,
			TokenFlag:     token,
			TokenTypeFlag: tokenType,
			AuthFlag:      authMode,
			ConfigPath:    configPath,
			TokenStore:    rt.Keychain,
			RequireAuth:   requiredAuthForCommand(cmd),
		})
		if err != nil {
			details := map[string]any{"error": err.Error()}
			if err.Error() == "missing Omni URL; set with --url or OMNI_URL or save in profile" {
				return fail(rt, 1, codeConfigMissing, "missing Omni URL configuration", details)
			}
			if strings.HasPrefix(err.Error(), "missing PAT") || strings.HasPrefix(err.Error(), "missing org API key") {
				return fail(rt, 1, codeConfigMissing, "missing Omni auth configuration", details)
			}
			if len(cfg.Profiles) == 0 {
				return fail(rt, 1, codeConfigMissing, "no profiles configured; run `omni setup`", details)
			}
			return fail(rt, 1, codeAuthError, "failed to resolve authentication", details)
		}
		rt.Resolved = resolved
		if cmd == "doctor" {
			return runDoctor(rt, cmdArgs)
		}
		if cmd == "documents" {
			return runDocuments(rt, cmdArgs)
		}
		if cmd == "models" {
			return runModels(rt, cmdArgs)
		}
		if cmd == "connections" {
			return runConnections(rt, cmdArgs)
		}
		if cmd == "folders" {
			return runFolders(rt, cmdArgs)
		}
		if cmd == "labels" {
			return runLabels(rt, cmdArgs)
		}
		if cmd == "schedules" {
			return runSchedules(rt, cmdArgs)
		}
		if cmd == "dashboards" {
			return runDashboards(rt, cmdArgs)
		}
		if cmd == "agentic" {
			return runAgentic(rt, cmdArgs)
		}
		if cmd == "embed" {
			return runEmbed(rt, cmdArgs)
		}
		if cmd == "unstable" {
			return runUnstable(rt, cmdArgs)
		}
		if cmd == "user-attributes" {
			return runUserAttributes(rt, cmdArgs)
		}
		if cmd == "admin" {
			return runAdmin(rt, cmdArgs)
		}
		if cmd == "users" {
			return runUsers(rt, cmdArgs)
		}
		if cmd == "scim" {
			return runSCIM(rt, cmdArgs)
		}
		if cmd == "ai" {
			return runAI(rt, cmdArgs)
		}
		if cmd == "api" {
			return runAPI(rt, cmdArgs)
		}
		if cmd == "query" {
			return runQuery(rt, cmdArgs)
		}
		return runJobs(rt, cmdArgs)
	default:
		output.PrintError(rt.JSON, codeUsageError, "unknown command", map[string]any{"command": cmd})
		if !rt.JSON {
			printUsage()
		}
		return 2
	}
}

func requiredAuthForCommand(cmd string) string {
	switch cmd {
	case "admin", "users", "scim":
		return "org"
	default:
		return ""
	}
}

func printUsage() {
	fmt.Print(`omni - Omni CLI

Usage:
  omni [global flags] <command> [command flags]

Commands:
  schema    Print machine-readable command schema
  exit-codes  Print stable automation exit codes
  doctor    Check connectivity, auth, and capabilities
  completion  Generate shell completion scripts
  setup     Configure Omni URL and auth profile
  auth      Manage profiles and tokens
  documents List and inspect documents
  models    Manage models
  connections Manage connections and environments
  folders   List folders
  labels    Manage labels
  schedules Manage schedules
  dashboards Manage dashboard downloads and filters
  agentic   Run asynchronous AI agentic jobs
  embed     Generate embed SSO sessions
  unstable  Unstable API routes
  user-attributes List user attribute definitions
  admin     Org-key-only admin commands
  users     Org-key user management and model-role assignments
  scim      Org-key SCIM users/groups management
  ai        Omni AI topic/query generation
  api       Raw authenticated Omni API calls
  query     Run Omni queries
  jobs      Check asynchronous job status
  version   Print version

Global flags:
  --profile <name>
  --url <https://instance.omniapp.co>
  --auth <pat|org>
  --token <token>
  --token-type <pat|org>
  --config <path>
  --json
  --plain
  --no-input
  --verbose

Examples:
  omni schema
  omni schema documents permissions
  omni exit-codes --json
  omni doctor --json
  omni completion zsh
  omni setup
  omni setup --profile prod --url https://acme.omniapp.co --auth-mode both --org-key $OMNI_ORG_KEY --default-auth pat
  omni documents list --page-size 20
  omni documents rename wk_abc123 --name "Q1 Dashboard"
  omni documents permissions add wk_abc123 --file permits.json
  omni models list --name marketing
  omni connections list
  omni connections environments list
  omni folders list --page-size 20
  omni folders permissions get 550e8400-e29b-41d4-a716-446655440000
  omni folders create "Team Reports" --scope organization
  omni labels list
  omni schedules create --file schedule.json
  omni schedules recipients add 550e8400-e29b-41d4-a716-446655440000 --file recipients.json
  omni schedules list --page-size 20
  omni dashboards filters get wk_abc123
  omni agentic submit --file agentic-job.json
  omni embed sso generate-session --file embed-session.json
  omni unstable documents export wk_abc123
  omni user-attributes list
  omni admin users list
  omni users list-email-only --page-size 20
  omni scim users list --count 20
  omni ai generate-query --model-id <model-uuid> --prompt "Revenue by month"
  omni ai workbook --model-id <model-uuid> --prompt "Top 10 customers by revenue"
  omni api call --method GET --path /api/v1/documents
  omni auth add --name prod --url https://acme.omniapp.co --auth-mode org --org-key $OMNI_ORG_KEY --token-store auto
  omni auth use prod
  omni query run --file query.json --wait --result-type json
  omni jobs status 12345
`)
}

func parseEnvBool(name string) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return false
	}
	v, err := strconv.ParseBool(raw)
	if err == nil {
		return v
	}
	switch strings.ToLower(raw) {
	case "y", "yes", "on":
		return true
	default:
		return false
	}
}

func parseAllowlistedCommands(raw string) map[string]struct{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	out := make(map[string]struct{})
	for _, item := range strings.Split(raw, ",") {
		cmd := strings.ToLower(strings.TrimSpace(item))
		if cmd == "" {
			continue
		}
		out[cmd] = struct{}{}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func isCommandAllowlisted(cmd string, allowlisted map[string]struct{}) bool {
	if len(allowlisted) == 0 {
		return true
	}
	switch cmd {
	case "help", "-h", "--help", "version", "schema", "exit-codes":
		return true
	}
	_, ok := allowlisted[strings.ToLower(strings.TrimSpace(cmd))]
	return ok
}

func sortedCommandList(set map[string]struct{}) []string {
	if len(set) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func wantsSubcommandHelp(args []string) bool {
	for _, a := range args {
		switch a {
		case "help", "--help", "-h":
			return true
		}
	}
	return false
}

func printCommandUsage(cmd string) {
	switch cmd {
	case "doctor":
		printDoctorUsage()
	case "query":
		printQueryUsage()
	case "jobs":
		printJobsUsage()
	case "documents":
		printDocumentsUsage()
	case "models":
		printModelsUsage()
	case "connections":
		printConnectionsUsage()
	case "folders":
		printFoldersUsage()
	case "labels":
		printLabelsUsage()
	case "schedules":
		printSchedulesUsage()
	case "dashboards":
		printDashboardsUsage()
	case "agentic":
		printAgenticUsage()
	case "embed":
		printEmbedUsage()
	case "unstable":
		printUnstableUsage()
	case "user-attributes":
		printUserAttributesUsage()
	case "admin":
		printAdminUsage()
	case "users":
		printUsersUsage()
	case "scim":
		printSCIMUsage()
	case "ai":
		printAIUsage()
	case "api":
		printAPIUsage()
	default:
		printUsage()
	}
}
