# AGENTS

## Repository purpose

CLIProxyAPI is a Go proxy for exposing CLI-authenticated model providers through OpenAI-, Gemini-, and Claude-compatible HTTP APIs.

This repository includes:
- Server runtime (`cmd/server`, `internal/**`)
- Reusable embedding SDK (`sdk/cliproxy`)
- Product/operator documentation (`README*`, `docs/*`)

## Important directories

- `cmd/`
  - entrypoints, mainly `cmd/server`
- `internal/`
  - runtime implementation, config loading, API server, handlers, storage, logging, TUI
- `sdk/`
  - reusable embedding surface and SDK-facing packages
- `docs/`
  - focused guides for SDK and usage patterns
- `.github/workflows/`
  - CI guardrails, PR checks, release, Docker publishing

## Safe working rules

- Keep changes tightly scoped to the requested task.
- Prefer editing existing files over creating new structure unless the task explicitly needs new docs/files.
- Avoid speculative refactors in `internal/**`.
- Reuse existing config names, endpoint names, and wording from source files.
- For behavioral changes, read the relevant runtime file before proposing edits.
- For docs-only tasks, do not drift into runtime cleanup.

## Build / verification

Current lightweight PR verification matches:

```bash
go build -o /tmp/cli-proxy-api-check ./cmd/server
```

Use this as the default quick verification for:
- docs changes
- config examples
- non-invasive repository updates

If changing runtime code, also spot-check the exact source path you touched.

## Documentation conventions

Follow the repository’s existing documentation style:
- Short H1/H2 sections
- Concise bullets
- Fenced code blocks only when they add operational value
- Keep claims source-driven and implementation-accurate

When relevant, preserve the repository’s bilingual documentation pattern:
- English doc first (`README.md`, `docs/*.md`)
- Chinese counterpart often exists (`README_CN.md`, `docs/*_CN.md`)

Do **not** assume every new doc requires a CN mirror unless explicitly requested.

## Repository constraints

There is an explicit PR guard on:
- `internal/translator/**`

Pull requests that modify this path fail the `translator-path-guard` workflow.

Treat `internal/translator/**` as restricted unless the task explicitly calls for coordinated maintenance outside the normal PR path.

## Practical guidance for automation

Before changing code:
- Read the source file you plan to edit
- Check whether a doc/config file already captures the intended behavior
- Keep endpoint names and config keys exact

Before finishing:
- Re-read changed markdown for style consistency
- Confirm scope stayed narrow
- Prefer quoting source references in summaries (file paths) rather than broad paraphrase
