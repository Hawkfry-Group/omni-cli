# Agent Command Map

This CLI is command-first for AI agents (same model as `gogcli`): the agent should pick an explicit command, not call a natural-language router command.

## Rules for agents

1. Always start from explicit intent and choose one concrete command.
2. Start by inspecting command shape with `omni schema [command-path]`.
3. Prefer `--json` for machine parsing.
4. Use `omni api call` only when no typed command exists.
5. Keep multi-step tasks explicit as a sequence of commands.
6. Branch on process exit code; inspect mapping with `omni exit-codes --json`.

## Intent to command mapping

- Inspect auth/session context: `omni auth whoami --json`
- List models to pick a model id: `omni models list --json`
- Pick a model topic for a prompt: `omni ai pick-topic --model-id <uuid> --prompt "<task>" --json`
- Generate query from natural language: `omni ai generate-query --model-id <uuid> --prompt "<task>" --json`
- Generate workbook URL directly: `omni ai workbook --model-id <uuid> --prompt "<task>" --json`
- List folders (including personal folders): `omni folders list --json`
- Create document/workbook from query payload: `omni documents create --file query.json --json`
- Move a document to a folder: `omni documents move <document-id> --folder-id <folder-id> --json`
- Update or inspect permissions: `omni documents permissions get|add|update|revoke ... --json`
- Run a query payload directly: `omni query run --file query.json --json`

## Example agent flow

Task: "Create a KPI workbook and store it in my personal folder."

1. `omni models list --json`
2. `omni ai workbook --model-id <uuid> --prompt "Create a KPI workbook for monthly revenue" --json`
3. If a document id is required for move/permissions, resolve it from `documents` list/get.
4. `omni folders list --json` and choose the personal folder id.
5. `omni documents move <document-id> --folder-id <personal-folder-id> --json`

## Fallback endpoint coverage

When a new Omni endpoint ships before typed CLI coverage, use:

- `omni api call --method <GET|POST|PATCH|DELETE> --path /api/v1/... [--body-file payload.json] --json`
