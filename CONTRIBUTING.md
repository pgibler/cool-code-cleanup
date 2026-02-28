# Contributing

Thanks for contributing to Cool Code Cleanup (`ccc`).

## Prerequisites

- Go (version from `go.mod`)
- Git

## Local Setup

```bash
git clone https://github.com/pgibler/cool-code-cleanup.git
cd cool-code-cleanup
```

Run the CLI help:

```bash
go run ./cmd/ccc --help
```

## Development Workflow

1. Create a branch from `main`.
2. Make focused changes.
3. Run tests.
4. Update docs/spec/backlog when behavior changes.
5. Open a PR using the PR template.

## Testing

Run all tests:

```bash
go test ./...
```

## Project Structure

- `cmd/ccc`: CLI entrypoint
- `internal/cli`: command parsing and mode dispatch
- `internal/mode`: profile/cleanup/configure step orchestration
- `internal/config`: config schema, precedence merge, save/load
- `internal/tui`: shared step layout and interactive components
- `internal/discovery`: route discovery adapters
- `internal/dependency`: dependency detection
- `internal/runner`: app startup and route invocation
- `internal/cleanup`: cleanup planning and application
- `internal/report`: JSON report output

## Coding Notes

- Keep implementations minimal and spec-aligned.
- Preserve cross-platform behavior (Linux/macOS/Windows).
- Avoid destructive git operations in scripts or docs.

## Pull Requests

PRs should include:

- clear summary of what changed
- local test evidence
- notes on any known limitations

