# Changelog

## Unreleased

- Added secure token storage with keychain/config backends.
- Added setup wizard improvements with hidden token input and validation.
- Added `doctor` command for capability checks.
- Added `completion` command for bash/zsh/fish.
- Added structured JSON error envelope contract.
- Added core resource commands: `documents`, `models`, `connections`, and `admin` list commands.
- Added additional resource commands: `folders list`, `labels list|get|create|update|delete`, and `schedules list|get|delete|pause|resume|trigger`.
- Added retries with backoff for idempotent API requests.
- Added `schema` command for machine-readable command discovery.
- Added `exit-codes` command for stable automation exit-code contract discovery.
- Added support for global flags before or after the command path.
- Added CI workflow, goreleaser config, and initial unit tests.
