package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/omni-co/omni-cli/internal/client"
	"github.com/omni-co/omni-cli/internal/client/gen"
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
	return runConnectionsCreateWithPrompts(rt, args, !rt.NoInput && stdinIsTerminal(), bufio.NewReader(os.Stdin), promptSecretInput)
}

func runConnectionsCreateWithPrompts(rt *runtime, args []string, canPrompt bool, reader *bufio.Reader, promptSecret func(*bufio.Reader, string) (string, error)) int {
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
	if fs.NArg() != 0 {
		return usageFail(rt, "usage: omni connections create --file <json-path>")
	}

	var payload []byte
	var err error
	if strings.TrimSpace(filePath) != "" {
		payload, err = readJSONFile(filePath)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to read connections create body", map[string]any{"error": err.Error()})
		}
	} else {
		if !canPrompt {
			return usageFail(rt, "usage: omni connections create --file <json-path> (or run from a terminal for interactive setup)")
		}
		payload, err = promptConnectionCreatePayload(reader, promptSecret)
		if err != nil {
			return fail(rt, 1, codeUsageError, "interactive connection setup failed", map[string]any{"error": err.Error()})
		}
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

func promptConnectionCreatePayload(reader *bufio.Reader, promptSecret func(*bufio.Reader, string) (string, error)) ([]byte, error) {
	if reader == nil {
		reader = bufio.NewReader(os.Stdin)
	}
	if promptSecret == nil {
		promptSecret = promptSecretInput
	}

	fmt.Fprintln(os.Stderr, "Interactive connection setup")
	fmt.Fprintf(os.Stderr, "Press Enter to accept defaults. Supported interactive dialects: %s.\n", strings.Join(connectionWizardDialects(), ", "))

	dialect, err := promptInput(reader, "Dialect", string(gen.Postgres))
	if err != nil {
		return nil, err
	}
	normalizedDialect := normalizeConnectionDialect(dialect)
	if normalizedDialect == "" {
		return nil, fmt.Errorf("invalid dialect %q", strings.TrimSpace(dialect))
	}
	spec, ok := connectionWizardSpecForDialect(normalizedDialect)
	if !ok {
		return nil, fmt.Errorf("interactive setup does not yet support %q; use --file for this dialect", normalizedDialect)
	}

	name, err := promptRequiredInput(reader, "Display name", "")
	if err != nil {
		return nil, err
	}
	body := map[string]any{
		"dialect": string(normalizedDialect),
		"name":    name,
	}
	if err := populateConnectionWizardBody(reader, promptSecret, spec, body); err != nil {
		return nil, err
	}

	baseRoleInput, err := promptInput(reader, "Base role", string(gen.QUERIER))
	if err != nil {
		return nil, err
	}
	baseRole, err := parseConnectionBaseRole(baseRoleInput)
	if err != nil {
		return nil, err
	}
	systemTimezone, err := promptInput(reader, "Database timezone", "UTC")
	if err != nil {
		return nil, err
	}
	queryTimezone, err := promptInput(reader, "Query timezone (blank for no conversion)", "")
	if err != nil {
		return nil, err
	}
	queryTimeout, err := promptOptionalPositiveInt(reader, "Query timeout seconds", "900")
	if err != nil {
		return nil, err
	}
	includeSchemas, err := promptInput(reader, "Include schemas (optional)", "")
	if err != nil {
		return nil, err
	}
	scratchSchema, err := promptInput(reader, "Schema for table uploads (optional)", "")
	if err != nil {
		return nil, err
	}
	allowsUserSpecificTimezones, err := promptOptionalBool(reader, "Allow user-specific timezones", false)
	if err != nil {
		return nil, err
	}

	body["baseRole"] = string(baseRole)
	if tz := strings.TrimSpace(systemTimezone); tz != "" {
		body["systemTimezone"] = tz
	}
	if tz := strings.TrimSpace(queryTimezone); tz != "" {
		body["queryTimezone"] = tz
	}
	if queryTimeout > 0 {
		body["queryTimeoutSeconds"] = queryTimeout
	}
	if schemas := strings.TrimSpace(includeSchemas); schemas != "" {
		body["includeSchemas"] = schemas
	}
	if schema := strings.TrimSpace(scratchSchema); schema != "" {
		body["scratchSchema"] = schema
	}
	if allowsUserSpecificTimezones {
		body["allowsUserSpecificTimezones"] = true
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

type connectionWizardSpec struct {
	HostLabel                 string
	HostRequired              bool
	PortLabel                 string
	PortDefault               string
	PortRequired              bool
	DatabaseLabel             string
	DatabaseRequired          bool
	DefaultSchemaLabel        string
	DefaultSchemaRequired     bool
	UsernameLabel             string
	UsernameRequired          bool
	PasswordLabel             string
	PasswordRequired          bool
	RegionLabel               string
	RegionRequired            bool
	WarehouseLabel            string
	WarehouseRequired         bool
	SupportsTrustCertificate  bool
	SupportsPrivateKey        bool
	PrivateKeyLabel           string
	SupportsServiceAccountKey bool
	ServiceAccountKeyLabel    string
	SupportsMaxBillingBytes   bool
	SupportsOtherCatalogs     bool
}

func connectionWizardDialects() []string {
	return []string{
		string(gen.Postgres),
		string(gen.Redshift),
		string(gen.Mysql),
		string(gen.Mariadb),
		string(gen.Mssql),
		string(gen.Snowflake),
		string(gen.Databricks),
		string(gen.Bigquery),
		string(gen.Athena),
		string(gen.Motherduck),
		string(gen.Clickhouse),
		string(gen.Exasol),
		string(gen.Starrocks),
		string(gen.Trino),
	}
}

func connectionWizardSpecForDialect(dialect gen.ConnectionsCreateJSONBodyDialect) (connectionWizardSpec, bool) {
	switch dialect {
	case gen.Postgres:
		return connectionWizardSpec{
			HostLabel:        "Host",
			HostRequired:     true,
			PortLabel:        "Port",
			PortDefault:      "5432",
			PortRequired:     true,
			DatabaseLabel:    "Database",
			DatabaseRequired: true,
			UsernameLabel:    "Username",
			UsernameRequired: true,
			PasswordLabel:    "Password",
			PasswordRequired: true,
		}, true
	case gen.Redshift:
		return connectionWizardSpec{
			HostLabel:        "Host",
			HostRequired:     true,
			PortLabel:        "Port",
			PortDefault:      "5439",
			PortRequired:     true,
			DatabaseLabel:    "Database",
			DatabaseRequired: true,
			UsernameLabel:    "Username",
			UsernameRequired: true,
			PasswordLabel:    "Password",
			PasswordRequired: true,
		}, true
	case gen.Mysql, gen.Mariadb:
		return connectionWizardSpec{
			HostLabel:        "Host",
			HostRequired:     true,
			PortLabel:        "Port",
			PortDefault:      "3306",
			PortRequired:     true,
			DatabaseLabel:    "Database",
			DatabaseRequired: true,
			UsernameLabel:    "Username",
			UsernameRequired: true,
			PasswordLabel:    "Password",
			PasswordRequired: true,
		}, true
	case gen.Mssql:
		return connectionWizardSpec{
			HostLabel:                "Host",
			HostRequired:             true,
			PortLabel:                "Port",
			PortDefault:              "1433",
			PortRequired:             true,
			DatabaseLabel:            "Database",
			DatabaseRequired:         true,
			DefaultSchemaLabel:       "Default schema",
			DefaultSchemaRequired:    true,
			UsernameLabel:            "Username",
			UsernameRequired:         true,
			PasswordLabel:            "Password",
			PasswordRequired:         true,
			SupportsTrustCertificate: true,
		}, true
	case gen.Snowflake:
		return connectionWizardSpec{
			HostLabel:             "Account identifier",
			HostRequired:          true,
			DatabaseLabel:         "Database",
			DatabaseRequired:      true,
			UsernameLabel:         "Username",
			UsernameRequired:      true,
			PasswordLabel:         "Password",
			PasswordRequired:      false,
			WarehouseLabel:        "Warehouse",
			WarehouseRequired:     true,
			SupportsPrivateKey:    true,
			PrivateKeyLabel:       "Private key file path",
			SupportsOtherCatalogs: true,
		}, true
	case gen.Databricks:
		return connectionWizardSpec{
			HostLabel:             "Host",
			HostRequired:          true,
			DatabaseLabel:         "Default catalog",
			DatabaseRequired:      true,
			UsernameLabel:         "Username or client ID",
			UsernameRequired:      true,
			PasswordLabel:         "Token or client secret",
			PasswordRequired:      true,
			WarehouseLabel:        "HTTP path",
			WarehouseRequired:     true,
			SupportsOtherCatalogs: true,
		}, true
	case gen.Bigquery:
		return connectionWizardSpec{
			DatabaseLabel:             "Project ID",
			DatabaseRequired:          true,
			DefaultSchemaLabel:        "Default dataset (optional)",
			UsernameLabel:             "Client email",
			UsernameRequired:          true,
			RegionLabel:               "Region",
			RegionRequired:            true,
			SupportsServiceAccountKey: true,
			ServiceAccountKeyLabel:    "Service account JSON file path",
			SupportsMaxBillingBytes:   true,
			SupportsOtherCatalogs:     true,
		}, true
	case gen.Athena:
		return connectionWizardSpec{
			DatabaseLabel:         "Data catalog",
			DatabaseRequired:      true,
			UsernameLabel:         "AWS access key ID",
			UsernameRequired:      true,
			PasswordLabel:         "AWS secret access key",
			PasswordRequired:      true,
			RegionLabel:           "AWS region",
			RegionRequired:        true,
			SupportsOtherCatalogs: true,
		}, true
	case gen.Motherduck:
		return connectionWizardSpec{
			DatabaseLabel:         "Database (optional)",
			PasswordLabel:         "Service token",
			PasswordRequired:      true,
			SupportsOtherCatalogs: true,
		}, true
	case gen.Clickhouse, gen.Exasol, gen.Starrocks, gen.Trino:
		return connectionWizardSpec{
			HostLabel:                "Host",
			HostRequired:             true,
			PortLabel:                "Port",
			PortRequired:             true,
			DatabaseLabel:            "Database",
			DatabaseRequired:         true,
			UsernameLabel:            "Username",
			UsernameRequired:         true,
			PasswordLabel:            "Password",
			PasswordRequired:         true,
			SupportsTrustCertificate: dialect == gen.Clickhouse || dialect == gen.Exasol,
			SupportsOtherCatalogs:    dialect == gen.Trino,
		}, true
	default:
		return connectionWizardSpec{}, false
	}
}

func populateConnectionWizardBody(reader *bufio.Reader, promptSecret func(*bufio.Reader, string) (string, error), spec connectionWizardSpec, body map[string]any) error {
	serviceAccountDefaults := map[string]string{}
	if spec.SupportsServiceAccountKey {
		path, err := promptRequiredInput(reader, spec.ServiceAccountKeyLabel, "")
		if err != nil {
			return err
		}
		content, defaults, err := readConnectionSecretFile(path)
		if err != nil {
			return err
		}
		body["passwordUnencrypted"] = content
		serviceAccountDefaults = defaults
	}

	if spec.HostRequired || spec.HostLabel != "" {
		host, err := promptMaybeRequiredInput(reader, spec.HostLabel, "", spec.HostRequired)
		if err != nil {
			return err
		}
		if host != "" {
			body["host"] = host
		}
	}
	if spec.PortRequired || spec.PortLabel != "" {
		port, err := promptMaybeRequiredPositiveInt(reader, spec.PortLabel, spec.PortDefault, spec.PortRequired)
		if err != nil {
			return err
		}
		if port > 0 {
			body["port"] = port
		}
	}
	if spec.DatabaseRequired || spec.DatabaseLabel != "" {
		defaultValue := ""
		if spec.SupportsServiceAccountKey {
			defaultValue = serviceAccountDefaults["project_id"]
		}
		database, err := promptMaybeRequiredInput(reader, spec.DatabaseLabel, defaultValue, spec.DatabaseRequired)
		if err != nil {
			return err
		}
		if database != "" {
			body["database"] = database
		}
	}
	if spec.DefaultSchemaRequired || spec.DefaultSchemaLabel != "" {
		defaultSchema, err := promptMaybeRequiredInput(reader, spec.DefaultSchemaLabel, "", spec.DefaultSchemaRequired)
		if err != nil {
			return err
		}
		if defaultSchema != "" {
			body["defaultSchema"] = defaultSchema
		}
	}
	if spec.UsernameRequired || spec.UsernameLabel != "" {
		defaultValue := ""
		if spec.SupportsServiceAccountKey {
			defaultValue = serviceAccountDefaults["client_email"]
		}
		username, err := promptMaybeRequiredInput(reader, spec.UsernameLabel, defaultValue, spec.UsernameRequired)
		if err != nil {
			return err
		}
		if username != "" {
			body["username"] = username
		}
	}

	if spec.SupportsPrivateKey {
		authMethod, err := promptInput(reader, "Authentication method (password|private-key)", "password")
		if err != nil {
			return err
		}
		switch strings.ToLower(strings.TrimSpace(authMethod)) {
		case "", "password":
			password, err := promptConnectionSecretValue(promptSecret, reader, spec.PasswordLabel, true)
			if err != nil {
				return err
			}
			body["passwordUnencrypted"] = password
		case "private-key", "private_key", "keypair", "key-pair":
			privateKeyPath, err := promptRequiredInput(reader, spec.PrivateKeyLabel, "")
			if err != nil {
				return err
			}
			privateKey, err := readRequiredFile(privateKeyPath)
			if err != nil {
				return err
			}
			body["privateKey"] = privateKey
		default:
			return fmt.Errorf("invalid authentication method %q", strings.TrimSpace(authMethod))
		}
	} else if spec.PasswordRequired || spec.PasswordLabel != "" {
		password, err := promptConnectionSecretValue(promptSecret, reader, spec.PasswordLabel, spec.PasswordRequired)
		if err != nil {
			return err
		}
		if password != "" {
			body["passwordUnencrypted"] = password
		}
	}

	if spec.WarehouseRequired || spec.WarehouseLabel != "" {
		warehouse, err := promptMaybeRequiredInput(reader, spec.WarehouseLabel, "", spec.WarehouseRequired)
		if err != nil {
			return err
		}
		if warehouse != "" {
			body["warehouse"] = warehouse
		}
	}
	if spec.RegionRequired || spec.RegionLabel != "" {
		region, err := promptMaybeRequiredInput(reader, spec.RegionLabel, "", spec.RegionRequired)
		if err != nil {
			return err
		}
		if region != "" {
			body["region"] = region
		}
	}
	if spec.SupportsMaxBillingBytes {
		maxBillingBytes, err := promptInput(reader, "Max billing bytes (optional)", "")
		if err != nil {
			return err
		}
		if strings.TrimSpace(maxBillingBytes) != "" {
			body["maxBillingBytes"] = strings.TrimSpace(maxBillingBytes)
		}
	}
	if spec.SupportsOtherCatalogs {
		includeOtherCatalogs, err := promptInput(reader, "Include other catalogs/databases (optional)", "")
		if err != nil {
			return err
		}
		if strings.TrimSpace(includeOtherCatalogs) != "" {
			body["includeOtherCatalogs"] = strings.TrimSpace(includeOtherCatalogs)
		}
	}
	if spec.SupportsTrustCertificate {
		trustServerCertificate, err := promptOptionalBool(reader, "Trust server certificate", false)
		if err != nil {
			return err
		}
		if trustServerCertificate {
			body["trustServerCertificate"] = true
		}
	}
	return nil
}

func promptRequiredInput(reader *bufio.Reader, label, defaultValue string) (string, error) {
	value, err := promptInput(reader, label, defaultValue)
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", strings.ToLower(label))
	}
	return value, nil
}

func promptMaybeRequiredInput(reader *bufio.Reader, label, defaultValue string, required bool) (string, error) {
	if required {
		return promptRequiredInput(reader, label, defaultValue)
	}
	value, err := promptInput(reader, label, defaultValue)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func promptRequiredPositiveInt(reader *bufio.Reader, label, defaultValue string) (int, error) {
	value, err := promptRequiredInput(reader, label, defaultValue)
	if err != nil {
		return 0, err
	}
	return parsePositiveInt(value, label)
}

func promptMaybeRequiredPositiveInt(reader *bufio.Reader, label, defaultValue string, required bool) (int, error) {
	if required {
		return promptRequiredPositiveInt(reader, label, defaultValue)
	}
	return promptOptionalPositiveInt(reader, label, defaultValue)
}

func promptOptionalPositiveInt(reader *bufio.Reader, label, defaultValue string) (int, error) {
	value, err := promptInput(reader, label, defaultValue)
	if err != nil {
		return 0, err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	return parsePositiveInt(value, label)
}

func promptOptionalBool(reader *bufio.Reader, label string, defaultValue bool) (bool, error) {
	defaultText := "no"
	if defaultValue {
		defaultText = "yes"
	}
	value, err := promptInput(reader, label+" (yes|no)", defaultText)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return defaultValue, nil
	case "y", "yes", "true", "1":
		return true, nil
	case "n", "no", "false", "0":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be yes or no", strings.ToLower(label))
	}
}

func promptConnectionSecretValue(promptSecret func(*bufio.Reader, string) (string, error), reader *bufio.Reader, label string, required bool) (string, error) {
	value, err := promptSecret(reader, label)
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if required && value == "" {
		return "", fmt.Errorf("%s is required", strings.ToLower(label))
	}
	return value, nil
}

func parsePositiveInt(value, label string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", strings.ToLower(label))
	}
	return n, nil
}

func normalizeConnectionDialect(value string) gen.ConnectionsCreateJSONBodyDialect {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(gen.Postgres):
		return gen.Postgres
	case string(gen.Mysql):
		return gen.Mysql
	case string(gen.Mariadb):
		return gen.Mariadb
	case string(gen.Redshift):
		return gen.Redshift
	case string(gen.Snowflake):
		return gen.Snowflake
	case string(gen.Bigquery):
		return gen.Bigquery
	case string(gen.Athena):
		return gen.Athena
	case string(gen.Databricks):
		return gen.Databricks
	case string(gen.Clickhouse):
		return gen.Clickhouse
	case string(gen.Exasol):
		return gen.Exasol
	case string(gen.Motherduck):
		return gen.Motherduck
	case string(gen.Mssql):
		return gen.Mssql
	case string(gen.Starrocks):
		return gen.Starrocks
	case string(gen.Trino):
		return gen.Trino
	default:
		return ""
	}
}

func parseConnectionBaseRole(value string) (gen.ConnectionsCreateJSONBodyBaseRole, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")
	switch normalized {
	case string(gen.CONNECTIONADMIN):
		return gen.CONNECTIONADMIN, nil
	case string(gen.MODELER):
		return gen.MODELER, nil
	case string(gen.NOACCESS):
		return gen.NOACCESS, nil
	case string(gen.QUERIER):
		return gen.QUERIER, nil
	case string(gen.RESTRICTEDQUERIER):
		return gen.RESTRICTEDQUERIER, nil
	case string(gen.VIEWER):
		return gen.VIEWER, nil
	default:
		return "", fmt.Errorf("invalid base role %q", strings.TrimSpace(value))
	}
}

func readRequiredFile(path string) (string, error) {
	content, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(string(content)) == "" {
		return "", fmt.Errorf("file %q is empty", strings.TrimSpace(path))
	}
	return string(content), nil
}

func readConnectionSecretFile(path string) (string, map[string]string, error) {
	content, err := readRequiredFile(path)
	if err != nil {
		return "", nil, err
	}

	defaults := map[string]string{}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(content), &parsed); err == nil {
		if v, ok := parsed["project_id"].(string); ok {
			defaults["project_id"] = strings.TrimSpace(v)
		}
		if v, ok := parsed["client_email"].(string); ok {
			defaults["client_email"] = strings.TrimSpace(v)
		}
	}
	return content, defaults, nil
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
  omni connections create
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

create without --file starts an interactive setup wizard.
`)
}
