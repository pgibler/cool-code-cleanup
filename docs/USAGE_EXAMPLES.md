# CCC Usage Examples

This guide shows common ways to run `ccc` with different flags.

## Add `ccc` to `PATH`

Build `ccc` and export the binary directory so you can run `ccc` directly.

```bash
# From the cool-code-cleanup repo
go build -o bin/ccc ./cmd/ccc

# Export for current shell session
export PATH="/home/gibler/Code/cool-code-cleanup/bin:$PATH"

# Verify
ccc --help
```

To make this persistent, add the `export PATH=...` line to your shell profile (for example `~/.bashrc` or `~/.zshrc`).

## Quickstart: Dry-Run

Use this to analyze another project without writing any file changes.

```bash
cd /path/to/your-project
ccc cleanup --dry-run
```

Non-interactive dry-run with a custom report path:

```bash
cd /path/to/your-project
ccc cleanup --dry-run --non-interactive --report-path .ccc/reports/dry-run.json
```

## Core Commands

```bash
ccc --help
ccc configure --help
ccc profile --help
ccc cleanup --help
```

## Global Flag Examples

Use an explicit project config file:

```bash
ccc cleanup --project-config .ccc/config.json
```

Use the alias for project config:

```bash
ccc cleanup --config .ccc/config.json
```

Use a custom global config:

```bash
ccc cleanup --global-config ~/.config/ccc/config.json
```

This is useful when you want to test authentication or model settings without changing the default global config.

Enable safe mode explicitly:

```bash
ccc cleanup --safe
```

Enable aggressive mode:

```bash
ccc cleanup --aggressive
```

Run non-interactively:

```bash
ccc cleanup --non-interactive
```

Write report to a specific file:

```bash
ccc cleanup --report-path .ccc/reports/manual-run.json
```

## Profile Mode Examples

Profile with dry-run:

```bash
ccc profile --dry-run
```

Profile only selected routes:

```bash
ccc profile --include-routes "/users,/health"
```

Ignore selected routes:

```bash
ccc profile --ignore-routes "/metrics,/internal/status"
```

Enable dependency short-circuit flow:

```bash
ccc profile --dependency-short-circuit
```

Set edit permission mode:

```bash
ccc profile --edit-permission-mode per-file
```

Auto-apply allowed edits:

```bash
ccc profile --auto-apply
```

Create branch and commit at final step:

```bash
ccc profile --create-branch --commit-changes
```

## Cleanup Mode Examples

Cleanup with dry-run:

```bash
ccc cleanup --dry-run
```

Cleanup with progress output:

```bash
ccc cleanup --show-progress
```

Use custom rules files:

```bash
ccc cleanup --rules .ccc/rules/cleanup.rules.json --rules-local .ccc/rules/cleanup.local.json
```

Enable and disable specific rules:

```bash
ccc cleanup --enable-rule remove_dead_code --disable-rule risky_large_refactor
```

Set edit permission mode:

```bash
ccc cleanup --edit-permission-mode per-edit
```

Auto-apply allowed edits:

```bash
ccc cleanup --auto-apply
```

This is best paired with a dry-run first when testing a new ruleset.

Create branch and commit at final step:

```bash
ccc cleanup --create-branch --commit-changes
```
