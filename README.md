# EARLY ALPHA
This repository is public early so the CLI contract can be reviewed in the open. Expect breaking changes while the command surface, auth flows, and automation guarantees are still settling.

# omni-cli

Automation-first Omni CLI for agents, workflows, and machine-to-machine use.

This repository is not trying to be a human-first terminal app. The primary goal is a stable command contract that other tools can call safely:

- JSON-first output for automation
- stable exit codes
- machine-readable command discovery via `omni schema`
- explicit auth/profile handling
- raw API access via `omni api call`

The CLI is built from a generated Omni OpenAPI client plus handwritten command and auth logic. Agent usage mapping is documented in [docs/agent-command-map.md](docs/agent-command-map.md).

## What The Repo Does Today

- wraps the Omni API in typed command groups such as `documents`, `models`, `connections`, `folders`, `labels`, `schedules`, `dashboards`, `query`, `users`, and `scim`
- supports PAT auth, org API keys, or both on one profile
- uses browser login for PAT setup
- supports keychain-backed secrets on macOS with config-file fallback
- provides `doctor`, `schema`, `exit-codes`, and raw `api call` for agent/runtime integration
- includes an interactive connection creation wizard for supported dialects, with `--file` still available for automation

## Automation Contract

- Success payloads are written to `stdout`.
- Error payloads are written to `stderr`.
- `--json` is the preferred mode for callers and agents.
- `--plain` provides stable TSV-style output for list-shaped responses.
- `omni exit-codes` prints the stable exit-code and error-code contract.
- `omni schema [command path]` prints machine-readable command metadata.
- `OMNI_ENABLE_COMMANDS` can restrict the allowed top-level command set for embedded/agent use.

Error shape in `--json` mode:

```json
{
  "error": {
    "code": "AUTH_UNAUTHORIZED",
    "message": "token unauthorized for Omni API",
    "details": {}
  }
}
```

## Auth Model

Profiles live in `~/.omni/config.json` by default.

- `pat`: PAT acquired through browser login
- `org`: org API key entered directly
- `both`: PAT plus org key on one profile

Runtime behavior:

- general commands use the profile's `default_auth`
- `admin`, `users`, and `scim` require org auth automatically
- `--auth pat|org` overrides the auth used for the current command
- `--token` and `--token-type` still work as legacy/manual overrides

PAT flow details are documented in [docs/pat-auth-flow.md](docs/pat-auth-flow.md).

## Setup

Interactive setup:

```bash
omni setup
omni setup --profile prod --url https://acme.omniapp.co --auth-mode both --org-key "$OMNI_ORG_KEY" --default-auth pat
```

Non-interactive setup:

```bash
omni setup --non-interactive --profile prod --url https://acme.omniapp.co --auth-mode org --org-key "$OMNI_ORG_KEY" --token-store auto
omni auth add --name prod --url https://acme.omniapp.co --auth-mode both --org-key "$OMNI_ORG_KEY" --default-auth pat --token-store keychain
omni auth use prod
omni auth show --profile prod --json
```

Supported environment variables:

- `OMNI_PROFILE`
- `OMNI_URL`
- `OMNI_AUTH`
- `OMNI_ORG_KEY`
- `OMNI_PAT` for manual override only; normal PAT setup is browser-based
- `OMNI_TOKEN`
- `OMNI_TOKEN_TYPE`
- `OMNI_CONFIG`
- `OMNI_PLAIN`
- `OMNI_NO_INPUT`
- `OMNI_ENABLE_COMMANDS`

## Flag Parsing

Global flags can be provided before or after the command path:

```bash
omni --json documents list
omni documents list --json
```

Command-local flags should be placed before positional arguments. For example:

```bash
omni documents rename --name "Q1 Dashboard" wk_abc123
omni folders create --scope organization "Team Reports"
omni labels create --homepage true finance
```

## Machine-Oriented Examples

```bash
omni schema
omni schema documents permissions
omni exit-codes --json
omni doctor --json
omni api call --method GET --path /api/v1/documents
omni query run --file query.json --result-type json --wait
omni user-attributes list --json
OMNI_ENABLE_COMMANDS=query,documents omni documents list --page-size 20 --json
```

## Command Surface

Top-level commands:

- `schema`
- `exit-codes`
- `doctor`
- `completion`
- `setup`
- `auth`
- `documents`
- `models`
- `connections`
- `folders`
- `labels`
- `schedules`
- `dashboards`
- `agentic`
- `embed`
- `unstable`
- `user-attributes`
- `admin`
- `users`
- `scim`
- `ai`
- `api`
- `query`
- `jobs`
- `version`

High-value subcommands:

- `omni auth add|list|remove|use|show|whoami`
- `omni documents list|get|create|delete|rename|move|draft create|draft discard|duplicate|favorite add|favorite remove|access list|permissions get|permissions add|permissions update|permissions revoke|permissions settings|label add|label remove|labels bulk-update|queries|transfer-ownership`
- `omni models list|get|create|refresh|validate|branch delete|branch merge|cache-reset|topics list|get|update|delete|views list|update|delete|fields create|update|delete|git get|create|update|delete|sync|migrate|content-validator get|replace|yaml get|create|delete`
- `omni connections list|create|update|dbt get|dbt update|dbt delete|schedules list|schedules create|schedules get|schedules update|schedules delete|environments list|environments create|environments update|environments delete`
- `omni folders list|create|delete|permissions get|permissions add|permissions update|permissions revoke`
- `omni labels list|get|create|update|delete`
- `omni schedules list|create|get|update|delete|pause|resume|trigger|recipients get|recipients add|recipients remove|transfer-ownership`
- `omni dashboards download|download-status|download-file|filters get|filters update`
- `omni agentic submit|status|cancel|result`
- `omni embed sso generate-session`
- `omni unstable documents export|import`
- `omni user-attributes list`
- `omni admin users list`
- `omni admin groups list`
- `omni users list-email-only|create-email-only|create-email-only-bulk|roles get|roles assign|group-roles get|group-roles assign`
- `omni scim users list|get|create|update|replace|delete|groups list|get|create|update|replace|delete|embed-users list|get|delete`
- `omni ai generate-query|workbook|pick-topic`
- `omni api call`
- `omni query run`
- `omni jobs status`

The bundled OpenAPI coverage report is tracked in:

- [docs/endpoint-coverage.md](docs/endpoint-coverage.md)
- [docs/endpoint-coverage.json](docs/endpoint-coverage.json)

## Connections Wizard

`omni connections create` can run interactively in a terminal, or accept `--file` JSON for automation.

Supported interactive dialects:

- `postgres`
- `redshift`
- `mysql`
- `mariadb`
- `mssql`
- `snowflake`
- `databricks`
- `bigquery`
- `athena`
- `motherduck`
- `clickhouse`
- `exasol`
- `starrocks`
- `trino`

Examples:

```bash
omni connections create
omni connections create --file connection.json
omni connections update 550e8400-e29b-41d4-a716-446655440000 --file connection-update.json
omni connections environments list
omni connections schedules list 550e8400-e29b-41d4-a716-446655440000
```

## Human Debugging Examples

These are still useful when developing the CLI, but they are secondary to the machine-facing contract.

```bash
omni completion zsh > ~/.zsh/completions/_omni
omni documents list --page-size 20
omni documents get wk_abc123
omni documents rename --name "Executive Dashboard" wk_abc123
omni documents permissions add wk_abc123 --file permits.json
omni models list --name marketing
omni folders permissions get 550e8400-e29b-41d4-a716-446655440000
omni labels create --homepage true finance
omni schedules create --file schedule.json
omni schedules recipients add 550e8400-e29b-41d4-a716-446655440000 --file recipients.json
omni dashboards filters update wk_abc123 --file dashboard-filters.json
omni agentic submit --file agentic-job.json
omni embed sso generate-session --file embed-session.json
omni unstable documents export wk_abc123
omni users list-email-only --page-size 20
omni scim users list --count 20
omni ai generate-query --model-id 550e8400-e29b-41d4-a716-446655440000 --prompt "Revenue by month"
omni jobs status 12345
```

## Repo Layout

- `cmd/omni`: entrypoint
- `internal/cli`: handwritten command handlers and runtime wiring
- `internal/auth`: auth resolution and profile behavior
- `internal/config`: config load/save and profile format
- `internal/client`: handwritten client wrappers
- `internal/client/gen`: generated OpenAPI client code
- `internal/output`: stdout/stderr and plain/json formatting helpers
- `internal/secret`: keychain/config credential storage abstraction
- `api/openapi.json`: checked-in Omni API spec
- `scripts/update_openapi.sh`: fetch and normalize an Omni OpenAPI spec
- `scripts/generate_client.sh`: regenerate the typed client

## Build And Release

```bash
make build
make test
make coverage-report
```

Refresh the API client:

```bash
./scripts/update_openapi.sh https://your-instance.omniapp.co/openapi.json
go generate ./internal/client/gen
```

Build release artifacts locally:

```bash
goreleaser release --snapshot --clean
```
