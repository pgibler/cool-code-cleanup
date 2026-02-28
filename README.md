# Cool Code Cleanup (CCC)

`ccc` is a Go CLI that profiles API route usage and performs AI-assisted code cleanup with interactive, step-based terminal flows.

## Current Capabilities

- `ccc profile`
  - Discovers API routes (Node, Go, Django)
  - Detects route dependencies (deterministic-first, AI fallback interface)
  - Supports short-circuit enhancement flow for dependency routes
  - Generates parameter plans and executes profiling route runs
  - Proposes and applies cleanup changes (or simulates in dry-run)

- `ccc cleanup`
  - Runs configurable cleanup options across the codebase
  - Applies edits with permission policies (`per-edit` or `per-file`)
  - Supports safe/aggressive behavior controls

- `ccc configure`
  - Writes project-local settings to `.ccc/config.json`

- Reporting
  - Writes structured JSON reports to `.ccc/reports/<timestamp>.json`

## Requirements

- Go (version from `go.mod`)
- Linux, macOS, or Windows

## Quick Start

```bash
go run ./cmd/ccc --help
go run ./cmd/ccc profile --help
go run ./cmd/ccc cleanup --help
go run ./cmd/ccc configure --help
```

## Common Flags

- `--config` path to config file (default `.ccc/config.json`)
- `--safe` default `true`
- `--aggressive` default `false`
- `--dry-run` simulate changes without writing files
- `--non-interactive` run without interactive prompts
- `--report-path` override JSON report path

## Development

Run tests:

```bash
go test ./...
```

CI:

- GitHub Actions workflow at `.github/workflows/ci.yml`
- Runs `go test ./...` on push to `main` and on pull requests

## Project Docs

- Engineering spec: `docs/ENGINEERING_SPEC.md`
- Prioritized backlog: `docs/IMPLEMENTATION_BACKLOG.md`

