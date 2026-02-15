# omni-cli

Go CLI scaffold for Omni API.

## Status

This scaffold now uses generated typed Go client code from Omni OpenAPI.
Agent usage mapping is documented in `docs/agent-command-map.md`.

## Features in this scaffold

- Profile-based auth in `~/.omni/config.json`
- Supports both PAT and org API key tokens
- Token storage backends: `keychain` (macOS) or `config` fallback
- Config precedence: flag > env > config file
- `omni setup` wizard (interactive + non-interactive)
- `omni doctor` capability checks (base/query/admin)
- `omni completion` scripts for `bash`, `zsh`, `fish`
- Core resource commands: `documents`, `models`, `connections`, `folders`, `labels`, `schedules` (including create/update/delete/permissions flows)
- Org-admin command groups for `users` and `scim` management
- Omni AI command group for topic selection and natural-language query generation
- Agentic async job command group (`agentic`)
- Dashboard download/filter command group (`dashboards`)
- Embed SSO session command group (`embed`)
- Unstable API command group (`unstable`)
- User attribute definitions command group (`user-attributes`)
- Escape hatch for complete coverage: `omni api call` for authenticated raw endpoint calls
- Org-key admin commands: `admin users list`, `admin groups list`
- Query flow with optional async wait call
- Retry/backoff for idempotent API requests (`429`, `502`, `503`, `504`)
- JSON output mode for scripting
- OpenAPI-driven typed client generation (`oapi-codegen`)
- CI workflow and GoReleaser packaging config

## Folder layout

- `cmd/omni`: main entrypoint
- `internal/cli`: command handlers
- `internal/config`: config load/save
- `internal/auth`: profile and token resolution
- `internal/client`: Omni HTTP client wrapper
- `internal/client/gen`: generated OpenAPI client code
- `internal/output`: human/json output helpers
- `api/openapi.json`: source spec (replace with your instance spec)
- `scripts/update_openapi.sh`: fetch and validate instance OpenAPI
- `scripts/generate_client.sh`: normalize 3.1 schema and regenerate client

## Global flags

- `--profile`
- `--url`
- `--token`
- `--token-type pat|org`
- `--config`
- `--json`
- `--plain`
- `--no-input`
- `--verbose`

## Environment variables

- `OMNI_PROFILE`
- `OMNI_URL`
- `OMNI_TOKEN`
- `OMNI_TOKEN_TYPE`
- `OMNI_CONFIG`
- `OMNI_PLAIN`
- `OMNI_NO_INPUT`
- `OMNI_ENABLE_COMMANDS`

## Shell completion

```bash
# zsh
mkdir -p ~/.zsh/completions
omni completion zsh > ~/.zsh/completions/_omni

# bash
omni completion bash > ~/.local/share/bash-completion/completions/omni

# fish
omni completion fish > ~/.config/fish/completions/omni.fish
```

## Error contract

- Success payloads are printed to `stdout`.
- Error payloads are printed to `stderr`.
- `--plain` prints stable tab-delimited output for list-style responses.
- In `--json` mode, errors use:

```json
{
  "error": {
    "code": "AUTH_UNAUTHORIZED",
    "message": "token unauthorized for Omni API",
    "details": {}
  }
}
```

## Example usage

```bash
omni setup
omni setup --non-interactive --profile prod --url https://acme.omniapp.co --token "$OMNI_PAT" --token-type pat --token-store auto
omni --no-input setup --profile prod --url https://acme.omniapp.co --token "$OMNI_PAT" --token-type pat
omni auth add --name prod --url https://acme.omniapp.co --token "$OMNI_PAT" --token-type pat --token-store keychain
omni doctor --json
omni --plain documents list --page-size 20
OMNI_ENABLE_COMMANDS=query,documents omni documents list --page-size 20
omni completion zsh > ~/.zsh/completions/_omni
omni documents list --page-size 20
omni documents get wk_abc123
omni documents create --file document.json
omni documents rename wk_abc123 --name "Executive Dashboard"
omni documents permissions add wk_abc123 --file permits.json
omni models list --name marketing
omni models validate 550e8400-e29b-41d4-a716-446655440000
omni models topics list 550e8400-e29b-41d4-a716-446655440000
omni connections list
omni connections create --file connection.json
omni connections dbt update 550e8400-e29b-41d4-a716-446655440000 --file connection-dbt.json
omni connections schedules list 550e8400-e29b-41d4-a716-446655440000
omni connections environments list
omni folders list --page-size 20
omni folders create "Team Reports" --scope organization
omni folders permissions get 550e8400-e29b-41d4-a716-446655440000
omni labels list
omni labels create finance --homepage true
omni schedules list --page-size 20
omni schedules create --file schedule.json
omni schedules update 550e8400-e29b-41d4-a716-446655440000
omni schedules recipients add 550e8400-e29b-41d4-a716-446655440000 --file recipients.json
omni schedules trigger 550e8400-e29b-41d4-a716-446655440000
omni dashboards download wk_abc123 --file dashboard-download.json
omni dashboards filters update wk_abc123 --file dashboard-filters.json
omni agentic submit --file agentic-job.json
omni embed sso generate-session --file embed-session.json
omni unstable documents export wk_abc123
omni user-attributes list
omni admin users list --count 10
omni users list-email-only --page-size 20
omni scim users list --count 20
omni ai generate-query --model-id 550e8400-e29b-41d4-a716-446655440000 --prompt "Revenue by month"
omni ai workbook --model-id 550e8400-e29b-41d4-a716-446655440000 --prompt "Top 10 customers by revenue"
omni api call --method GET --path /api/v1/documents
omni auth whoami
omni query run --file query.json --result-type json --wait
omni jobs status 12345
```

## Command overview

- `omni setup`
- `omni doctor`
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

## Build

`go` is required locally.

```bash
make build
make test
```

## Refresh OpenAPI Client

```bash
./scripts/update_openapi.sh https://your-instance.omniapp.co/openapi.json
go generate ./internal/client/gen
make coverage-report
```

Coverage outputs:

- `docs/endpoint-coverage.md`
- `docs/endpoint-coverage.json`

## Release

```bash
goreleaser release --snapshot --clean
```
