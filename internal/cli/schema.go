package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/omni-co/omni-cli/internal/output"
)

type schemaFlag struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required,omitempty"`
	Description string `json:"description,omitempty"`
}

type schemaCommand struct {
	Name        string           `json:"name"`
	Aliases     []string         `json:"aliases,omitempty"`
	Summary     string           `json:"summary,omitempty"`
	Usage       string           `json:"usage,omitempty"`
	Flags       []schemaFlag     `json:"flags,omitempty"`
	Subcommands []*schemaCommand `json:"subcommands,omitempty"`
}

func runSchema(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("schema", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printSchemaUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}

	path := normalizeSchemaPath(fs.Args())
	root := commandSchemaRoot()
	target, ok := findSchemaNode(root, path)
	if !ok {
		return usageFail(rt, fmt.Sprintf("unknown command path for schema: %s", strings.Join(path, " ")))
	}

	payload := map[string]any{
		"path":    append([]string{"omni"}, path...),
		"command": target,
	}

	asJSON := true
	if rt.Plain {
		asJSON = false
	}
	if err := output.Print(payload, asJSON, rt.Plain); err != nil {
		return fail(rt, 1, codeAPIError, "failed to print schema", map[string]any{"error": err.Error()})
	}
	return 0
}

func printSchemaUsage() {
	fmt.Print(`omni schema:
  omni schema
  omni schema <command-path>
`)
}

func normalizeSchemaPath(parts []string) []string {
	path := make([]string, 0, len(parts))
	for _, raw := range parts {
		s := strings.ToLower(strings.TrimSpace(raw))
		if s == "" {
			continue
		}
		path = append(path, s)
	}
	if len(path) > 0 && path[0] == "omni" {
		path = path[1:]
	}
	return path
}

func findSchemaNode(root *schemaCommand, path []string) (*schemaCommand, bool) {
	node := root
	for _, step := range path {
		next := findSchemaChild(node, step)
		if next == nil {
			return nil, false
		}
		node = next
	}
	return node, true
}

func findSchemaChild(parent *schemaCommand, step string) *schemaCommand {
	for _, child := range parent.Subcommands {
		if child.Name == step {
			return child
		}
		for _, alias := range child.Aliases {
			if alias == step {
				return child
			}
		}
	}
	return nil
}

func commandSchemaRoot() *schemaCommand {
	root := schemaCmd("omni", "Omni CLI", "omni [global flags] <command> [command flags]",
		schemaCmd("schema", "Print machine-readable command schema", "omni schema [command-path]"),
		schemaCmd("exit-codes", "Print stable automation exit codes", "omni exit-codes"),
		schemaCmd("doctor", "Check connectivity, auth, and capabilities", "omni doctor"),
		schemaCmd("completion", "Generate shell completion scripts", "omni completion <bash|zsh|fish>",
			schemaCmd("bash", "Generate bash completion script", ""),
			schemaCmd("zsh", "Generate zsh completion script", ""),
			schemaCmd("fish", "Generate fish completion script", ""),
		),
		schemaCmd("setup", "Configure Omni URL and auth profile", "omni setup"),
		schemaCmd("auth", "Manage profiles and tokens", "omni auth <subcommand>",
			schemaCmd("add", "Add auth profile", ""),
			schemaCmd("list", "List auth profiles", ""),
			schemaCmdAlias("remove", []string{"rm"}, "Remove auth profile", ""),
			schemaCmd("use", "Set active profile", ""),
			schemaCmdAlias("show", []string{"whoami"}, "Show resolved auth profile", ""),
		),
		schemaCmd("documents", "List and inspect documents", "omni documents <subcommand>",
			schemaCmd("list", "List documents", ""),
			schemaCmd("get", "Get document by id or slug", ""),
			schemaCmd("create", "Create a document", ""),
			schemaCmdAlias("delete", []string{"rm"}, "Delete a document", ""),
			schemaCmd("rename", "Rename a document", ""),
			schemaCmd("move", "Move a document", ""),
			schemaCmd("draft", "Manage drafts",
				"",
				schemaCmd("create", "Create draft", ""),
				schemaCmd("discard", "Discard draft", ""),
			),
			schemaCmd("duplicate", "Duplicate document", ""),
			schemaCmd("favorite", "Manage favorites",
				"",
				schemaCmd("add", "Add favorite", ""),
				schemaCmd("remove", "Remove favorite", ""),
			),
			schemaCmd("access", "List access records",
				"",
				schemaCmd("list", "List access records", ""),
			),
			schemaCmdAlias("permissions", []string{"perm"}, "Manage document permissions",
				"",
				schemaCmd("get", "Get permissions", ""),
				schemaCmd("add", "Add permissions", ""),
				schemaCmd("update", "Update permissions", ""),
				schemaCmd("revoke", "Revoke permissions", ""),
				schemaCmd("settings", "Update permission settings", ""),
			),
			schemaCmd("label", "Manage a single label",
				"",
				schemaCmd("add", "Add label", ""),
				schemaCmd("remove", "Remove label", ""),
			),
			schemaCmd("labels", "Manage document labels",
				"",
				schemaCmd("bulk-update", "Bulk update labels", ""),
			),
			schemaCmd("queries", "List document queries", ""),
			schemaCmd("transfer-ownership", "Transfer document ownership", ""),
		),
		schemaCmd("models", "Manage models", "omni models <subcommand>",
			schemaCmd("list", "List models", ""),
			schemaCmd("get", "Get model", ""),
			schemaCmd("create", "Create model", ""),
			schemaCmd("refresh", "Refresh model", ""),
			schemaCmd("validate", "Validate model", ""),
			schemaCmd("branch", "Manage model branches",
				"",
				schemaCmd("delete", "Delete branch", ""),
				schemaCmd("merge", "Merge branch", ""),
			),
			schemaCmd("cache-reset", "Reset model cache", ""),
			schemaCmd("topics", "Manage model topics",
				"",
				schemaCmd("list", "List topics", ""),
				schemaCmd("get", "Get topic", ""),
				schemaCmd("update", "Update topic", ""),
				schemaCmd("delete", "Delete topic", ""),
			),
			schemaCmd("views", "Manage model views",
				"",
				schemaCmd("list", "List views", ""),
				schemaCmd("update", "Update view", ""),
				schemaCmd("delete", "Delete view", ""),
			),
			schemaCmd("fields", "Manage model fields",
				"",
				schemaCmd("create", "Create field", ""),
				schemaCmd("update", "Update field", ""),
				schemaCmd("delete", "Delete field", ""),
			),
			schemaCmd("git", "Manage model git config",
				"",
				schemaCmd("get", "Get git config", ""),
				schemaCmd("create", "Create git config", ""),
				schemaCmd("update", "Update git config", ""),
				schemaCmd("delete", "Delete git config", ""),
				schemaCmd("sync", "Sync git config", ""),
			),
			schemaCmd("migrate", "Run model migration", ""),
			schemaCmd("content-validator", "Manage content validator config",
				"",
				schemaCmd("get", "Get content validator config", ""),
				schemaCmd("replace", "Replace content validator config", ""),
			),
			schemaCmd("yaml", "Manage model YAML",
				"",
				schemaCmd("get", "Get model YAML", ""),
				schemaCmd("create", "Create model YAML", ""),
				schemaCmd("delete", "Delete model YAML", ""),
			),
		),
		schemaCmd("connections", "Manage connections and environments", "omni connections <subcommand>",
			schemaCmd("list", "List connections", ""),
			schemaCmd("create", "Create connection", ""),
			schemaCmd("update", "Update connection", ""),
			schemaCmd("dbt", "Manage dbt connection config",
				"",
				schemaCmd("get", "Get dbt config", ""),
				schemaCmd("update", "Update dbt config", ""),
				schemaCmd("delete", "Delete dbt config", ""),
			),
			schemaCmd("schedules", "Manage connection schedules",
				"",
				schemaCmd("list", "List schedules", ""),
				schemaCmd("create", "Create schedule", ""),
				schemaCmd("get", "Get schedule", ""),
				schemaCmd("update", "Update schedule", ""),
				schemaCmd("delete", "Delete schedule", ""),
			),
			schemaCmd("environments", "Manage connection environments",
				"",
				schemaCmd("list", "List environments", ""),
				schemaCmd("create", "Create environment", ""),
				schemaCmd("update", "Update environment", ""),
				schemaCmd("delete", "Delete environment", ""),
			),
		),
		schemaCmd("folders", "Manage folders", "omni folders <subcommand>",
			schemaCmd("list", "List folders", ""),
			schemaCmd("create", "Create folder", ""),
			schemaCmdAlias("delete", []string{"rm"}, "Delete folder", ""),
			schemaCmdAlias("permissions", []string{"perm"}, "Manage folder permissions",
				"",
				schemaCmd("get", "Get permissions", ""),
				schemaCmd("add", "Add permissions", ""),
				schemaCmd("update", "Update permissions", ""),
				schemaCmd("revoke", "Revoke permissions", ""),
			),
		),
		schemaCmd("labels", "Manage labels", "omni labels <subcommand>",
			schemaCmd("list", "List labels", ""),
			schemaCmd("get", "Get label", ""),
			schemaCmd("create", "Create label", ""),
			schemaCmd("update", "Update label", ""),
			schemaCmdAlias("delete", []string{"rm"}, "Delete label", ""),
		),
		schemaCmd("schedules", "Manage schedules", "omni schedules <subcommand>",
			schemaCmd("list", "List schedules", ""),
			schemaCmd("create", "Create schedule", ""),
			schemaCmd("get", "Get schedule", ""),
			schemaCmd("update", "Update schedule", ""),
			schemaCmdAlias("delete", []string{"rm"}, "Delete schedule", ""),
			schemaCmd("pause", "Pause schedule", ""),
			schemaCmd("resume", "Resume schedule", ""),
			schemaCmd("trigger", "Trigger schedule", ""),
			schemaCmd("recipients", "Manage schedule recipients",
				"",
				schemaCmd("get", "Get recipients", ""),
				schemaCmd("add", "Add recipients", ""),
				schemaCmd("remove", "Remove recipients", ""),
			),
			schemaCmd("transfer-ownership", "Transfer schedule ownership", ""),
		),
		schemaCmd("dashboards", "Manage dashboard downloads and filters", "omni dashboards <subcommand>",
			schemaCmd("download", "Submit dashboard download", ""),
			schemaCmd("download-status", "Check dashboard download status", ""),
			schemaCmd("download-file", "Fetch dashboard file metadata", ""),
			schemaCmd("filters", "Manage dashboard filters",
				"",
				schemaCmd("get", "Get dashboard filters", ""),
				schemaCmd("update", "Update dashboard filters", ""),
			),
		),
		schemaCmd("agentic", "Run asynchronous AI agentic jobs", "omni agentic <subcommand>",
			schemaCmd("submit", "Submit job", ""),
			schemaCmd("status", "Check job status", ""),
			schemaCmd("cancel", "Cancel job", ""),
			schemaCmd("result", "Get job result", ""),
		),
		schemaCmd("embed", "Generate embed SSO sessions", "omni embed <subcommand>",
			schemaCmd("sso", "Embed SSO commands",
				"",
				schemaCmd("generate-session", "Generate embed session", ""),
			),
		),
		schemaCmd("unstable", "Unstable API routes", "omni unstable <subcommand>",
			schemaCmd("documents", "Unstable document operations",
				"",
				schemaCmd("export", "Export unstable document", ""),
				schemaCmd("import", "Import unstable document", ""),
			),
		),
		schemaCmd("user-attributes", "List user attribute definitions", "omni user-attributes list",
			schemaCmd("list", "List user attributes", ""),
		),
		schemaCmd("admin", "Org-key admin commands", "omni admin <subcommand>",
			schemaCmd("users", "Admin users commands",
				"",
				schemaCmd("list", "List users", ""),
			),
			schemaCmd("groups", "Admin groups commands",
				"",
				schemaCmd("list", "List groups", ""),
			),
		),
		schemaCmd("users", "Org-key user management and role assignment", "omni users <subcommand>",
			schemaCmd("list-email-only", "List users (email only)", ""),
			schemaCmd("create-email-only", "Create one user (email only)", ""),
			schemaCmd("create-email-only-bulk", "Create users in bulk (email only)", ""),
			schemaCmd("roles", "Manage direct roles",
				"",
				schemaCmd("get", "Get direct roles", ""),
				schemaCmd("assign", "Assign direct roles", ""),
			),
			schemaCmd("group-roles", "Manage group roles",
				"",
				schemaCmd("get", "Get group roles", ""),
				schemaCmd("assign", "Assign group roles", ""),
			),
		),
		schemaCmd("scim", "Org-key SCIM users/groups management", "omni scim <subcommand>",
			schemaCmd("users", "SCIM user operations",
				"",
				schemaCmd("list", "List users", ""),
				schemaCmd("get", "Get user", ""),
				schemaCmd("create", "Create user", ""),
				schemaCmd("update", "Patch user", ""),
				schemaCmd("replace", "Replace user", ""),
				schemaCmd("delete", "Delete user", ""),
			),
			schemaCmd("groups", "SCIM group operations",
				"",
				schemaCmd("list", "List groups", ""),
				schemaCmd("get", "Get group", ""),
				schemaCmd("create", "Create group", ""),
				schemaCmd("update", "Patch group", ""),
				schemaCmd("replace", "Replace group", ""),
				schemaCmd("delete", "Delete group", ""),
			),
			schemaCmd("embed-users", "SCIM embed-user operations",
				"",
				schemaCmd("list", "List embed users", ""),
				schemaCmd("get", "Get embed user", ""),
				schemaCmd("delete", "Delete embed user", ""),
			),
		),
		schemaCmd("ai", "Omni AI topic/query generation", "omni ai <subcommand>",
			schemaCmd("generate-query", "Generate query from prompt", ""),
			schemaCmd("workbook", "Generate workbook from prompt", ""),
			schemaCmd("pick-topic", "Pick model topic from prompt", ""),
		),
		schemaCmd("api", "Raw authenticated Omni API calls", "omni api call",
			schemaCmd("call", "Raw API call", ""),
		),
		schemaCmd("query", "Run Omni queries", "omni query run",
			schemaCmd("run", "Run query payload", ""),
		),
		schemaCmd("jobs", "Check asynchronous job status", "omni jobs status",
			schemaCmd("status", "Get job status", ""),
		),
		schemaCmd("version", "Print version", "omni version"),
		schemaCmd("help", "Show help", "omni help"),
	)
	root.Flags = []schemaFlag{
		{Name: "--profile", Type: "string", Description: "Profile name to use"},
		{Name: "--url", Type: "string", Description: "Omni instance URL"},
		{Name: "--auth", Type: "string", Description: "Auth to use: pat or org"},
		{Name: "--token", Type: "string", Description: "Omni API token (PAT or org key)"},
		{Name: "--token-type", Type: "string", Description: "Token type: pat or org"},
		{Name: "--config", Type: "string", Description: "Config file path"},
		{Name: "--json", Type: "bool", Description: "Print JSON output"},
		{Name: "--plain", Type: "bool", Description: "Print stable plain-text output"},
		{Name: "--no-input", Type: "bool", Description: "Disable interactive prompts"},
		{Name: "--verbose", Type: "bool", Description: "Verbose logging"},
	}
	return root
}

func schemaCmd(name, summary, usage string, subcommands ...*schemaCommand) *schemaCommand {
	return &schemaCommand{
		Name:        name,
		Summary:     summary,
		Usage:       usage,
		Subcommands: subcommands,
	}
}

func schemaCmdAlias(name string, aliases []string, summary, usage string, subcommands ...*schemaCommand) *schemaCommand {
	cmd := schemaCmd(name, summary, usage, subcommands...)
	cmd.Aliases = aliases
	return cmd
}
