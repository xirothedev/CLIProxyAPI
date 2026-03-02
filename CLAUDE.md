# CLAUDE

## Project summary

CLIProxyAPI is a Go service that proxies multiple CLI-authenticated providers behind compatible HTTP APIs:
- OpenAI/Claude-compatible endpoints under `/v1`
- Gemini-compatible endpoints under `/v1beta`
- Optional management control plane under `/v0/management`

Repository also provides `sdk/cliproxy` for embedding the same runtime behavior into other Go programs.

## Canonical source files

Use these as primary truth before editing:

- Runtime entry:
  - `cmd/server/main.go`
  - `internal/cmd/run.go`
- API routing and server behavior:
  - `internal/api/server.go`
  - `internal/api/handlers/management/*.go`
- Config schema and loading:
  - `internal/config/config.go`
  - `internal/config/sdk_config.go`
  - `config.example.yaml`
- Logging behavior:
  - `internal/logging/global_logger.go`
  - `internal/logging/log_dir_cleaner.go`
- Storage backends:
  - `internal/store/postgresstore.go`
  - `internal/store/gitstore.go`
  - `internal/store/objectstore.go`
- Docs/style references:
  - `README.md`
  - `README_CN.md`
  - `docs/sdk-usage.md`

## Common tasks in this repo

- Docs-only updates
  - Keep wording implementation-accurate
  - Prefer source references over assumptions
- Config/management clarifications
  - Verify key names exactly against `config.example.yaml` + config structs
- Routing or endpoint changes
  - Validate against `internal/api/server.go` and management handler methods
- Storage-related behavior
  - Confirm backend-specific bootstrap/sync semantics in `internal/store/*`

## Preferred verification steps

Default quick check (aligned with PR workflow):

```bash
go build -o /tmp/cli-proxy-api-check ./cmd/server
```

For docs-only tasks:
- Ensure no runtime files changed unintentionally
- Re-read markdown for concise sectioning (H1/H2 + bullets)

## Documentation update rules

- Keep sections short and operational.
- Do not add speculative promises or future behavior.
- Match existing tone from `README.md` and `docs/*.md`.
- Avoid README churn unless explicitly requested.
- Mention bilingual mirroring only when relevant to the specific task.

## Scope guardrails

- Prefer minimal, localized diffs.
- Do not refactor unrelated code while fixing docs or small issues.
- Respect repository restrictions and CI guards:
  - `internal/translator/**` is path-guarded in PR workflow.
- For risky/destructive actions (deletions, history rewrites, force pushes), stop and ask first.
