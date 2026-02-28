# Cool Code Cleanup (CCC) Engineering Spec v1

## 1. Purpose

`ccc` is a cross-platform CLI tool (Go) for:

- profiling API route usage and cleaning unused code (`ccc profile`)
- analyzing and improving code quality (`ccc cleanup`)
- configuring OpenAI credentials/model and project behavior (`ccc configure`)

v1 scope is local codebases only.

## 2. Goals and Non-Goals

### 2.1 Goals

- Provide interactive TUI flows with a consistent layout for all steps.
- Support Node backends, Go backends, and Django for API route profiling.
- Detect route dependencies with deterministic logic first; use AI fallback.
- Allow safe and aggressive cleanup modes:
  - `--safe` defaulting to `true`
  - `--aggressive` for riskier refactors
- Support dry-run in profile and cleanup modes.
- Emit structured JSON report at `.ccc/reports/<timestamp>.json`.
- Support Linux, macOS, Windows.

### 2.2 Non-Goals (v1)

- Frontend profiling.
- Remote repo or hosted codebase support.
- CI-specific orchestration features beyond JSON report output.
- Advanced non-functional requirement guarantees (performance SLOs, etc.).

## 3. CLI Surface

## 3.1 Root Command

`ccc <command> [flags]`

Commands:

- `ccc configure`
- `ccc profile`
- `ccc cleanup`

## 3.2 Global Flags

- `--config <path>` optional config file path. Default: `.ccc/config.json`.
- `--safe` bool, default `true`.
- `--aggressive` bool, default `false`.
- `--dry-run` bool, default `false`.
- `--non-interactive` bool, default `false` (if true, no TUI toggles/prompts).
- `--report-path <path>` optional report override. Default `.ccc/reports/<timestamp>.json`.

Validation rules:

- If `--aggressive=true`, allow riskier transforms; `--safe` still applies baseline safety checks.
- If both `--non-interactive=true` and required decisions are missing, command exits with actionable error.

## 3.3 `configure` Command

Purpose: set project-local configuration in `.ccc/config.json`.

Primary settings:

- OpenAI API key
- OpenAI model
- Default short-circuit env var name (default `CoolCodeCleanupShortCircuit`)
- edit permission mode
- default mode flags (`safe`, `aggressive`, `dry_run`)

Security:

- API key can be stored in config only if user confirms.
- Recommended behavior: prefer env var reference in config (e.g. `OPENAI_API_KEY`) and not plaintext.

## 3.4 `profile` Command Flags

- `--include-routes <csv|repeatable>`
- `--ignore-routes <csv|repeatable>`
- `--dependency-short-circuit` bool
- `--edit-permission-mode <per-edit|per-file>`
- `--auto-apply` bool (skip per-file edit confirmation if policy allows)

## 3.5 `cleanup` Command Flags

- `--remove-redundant-guards` bool
- `--dry-refactor` bool (DRY principle refactors)
- `--harden-error-handling` bool
- `--gate-features-env` bool
- `--split-functions` bool
- `--standardize-naming` bool
- `--simplify-complex-logic` bool
- `--detect-expensive-functions` bool
- `--edit-permission-mode <per-edit|per-file>`
- `--auto-apply` bool

## 4. Configuration and Settings Resolution

## 4.1 Config Location

Project-local only: `.ccc/config.json`.

## 4.2 Effective Settings Precedence

1. CLI args
2. Environment variables
3. Config file JSON
4. Defaults

For each setting, retain full source chain in effective config metadata for UI display.

## 4.3 Config Schema (v1)

```json
{
  "openai": {
    "api_key_env": "OPENAI_API_KEY",
    "api_key_value": "",
    "model": "gpt-5"
  },
  "modes": {
    "safe": true,
    "aggressive": false,
    "dry_run": false
  },
  "profile": {
    "include_routes": [],
    "ignore_routes": [],
    "dependency_short_circuit": true,
    "short_circuit_env_var": "CoolCodeCleanupShortCircuit",
    "update_env_file": false,
    "save_short_circuit_to_config": true,
    "edit_permission_mode": "per-file",
    "auto_apply": false
  },
  "cleanup": {
    "remove_redundant_guards": true,
    "dry_refactor": true,
    "harden_error_handling": true,
    "gate_features_env": false,
    "split_functions": false,
    "standardize_naming": true,
    "simplify_complex_logic": true,
    "detect_expensive_functions": true,
    "edit_permission_mode": "per-file",
    "auto_apply": false
  },
  "git": {
    "auto_offer_branch_and_commit": true
  }
}
```

## 4.4 Env Var Mapping (initial)

- `CCC_OPENAI_API_KEY`
- `CCC_OPENAI_MODEL`
- `CCC_SAFE`
- `CCC_AGGRESSIVE`
- `CCC_DRY_RUN`
- `CCC_PROFILE_INCLUDE_ROUTES`
- `CCC_PROFILE_IGNORE_ROUTES`
- `CCC_PROFILE_SHORT_CIRCUIT`
- `CCC_PROFILE_SHORT_CIRCUIT_ENV_VAR`
- `CCC_EDIT_PERMISSION_MODE`

## 5. Unified TUI Layout

Every step screen must render:

1. Step header: `<Mode> - <Step Name>`
2. Step description: one concise paragraph
3. Main content pane
4. Action bar: `Accept`, `Back`, `Cancel` (and mode-specific actions)

Additional requirements:

- Keyboard navigation (up/down, space to toggle, enter to confirm).
- Consistent spacing and section order across all steps.
- Show validation errors inline without breaking layout.

## 6. Profile Mode Flow

## 6.1 Step 1a: Profiling Options

Behavior:

- Load effective settings with precedence chain.
- Show each option value plus source attribution:
  - single source: `source=cli` or `source=env` or `source=config` or `source=default`
  - multiple set points: chain view, e.g. `config -> env -> cli`
- Interactive toggles for:
  - routes include/ignore
  - dependency route short-circuiting
  - edit permission mode
  - safe/aggressive/dry-run
- User chooses `Accept` or `Cancel`.

## 6.2 Route Discovery and Dependency Detection

Route discovery targets:

- Node (Express/Fastify/Nest HTTP routes)
- Go (`net/http`, common routers)
- Django URL patterns + view bindings

Dependency detection pipeline:

1. Deterministic pass:
  - auth middleware/token issuance route mapping
  - shared precondition signals (session/token/csrf/payment setup)
  - route metadata/annotations if present
2. AI fallback:
  - infer dependency graph when deterministic confidence is low
  - return confidence + rationale text

If no dependencies:

- Prompt: `No route dependencies detected. Proceed to the next step?`

If dependencies exist and required dependency routes are disabled:

- Show missing required dependency routes.
- User actions:
  - `Enable Required Dependencies`
  - `Quit`

If external service requirements detected:

- Offer short-circuit flow using env var (default `CoolCodeCleanupShortCircuit`).

## 6.3 Step 1b: Dependency Route Short-Circuiting Enhancement

Run only when dependency routes are detected and at least one route needs short-circuit behavior.

Substep A: Name short-circuit env var (conditional)

- Skip if short-circuit env var already found in:
  - CLI options
  - env vars
  - config
  - common in-code patterns
- Otherwise prompt for:
  - env var name (default `CoolCodeCleanupShortCircuit`)
  - whether to update `.env`
  - whether to save to `.ccc/config.json`

Substep B: Patch dependency routes for short-circuiting

- Show interactive list of routes requiring short-circuit logic.
- User toggles which routes to patch.
- If accepted, generate/apply patches automatically.
- Respect edit permission mode (`per-edit` or `per-file`) unless `auto_apply=true`.

## 6.4 Step 2: Enable/Disable Routes to Profile

- Display discovered routes in interactive list.
- Each route shows dependency routes beneath it.
- Toggle behavior:
  - cannot disable a dependency route while dependent route remains enabled
  - show explanation when blocked
- User chooses `Accept` or `Cancel`.

## 6.5 Step 3: Parameter Analysis

For each selected route, produce:

- valid parameter sets for meaningful path execution
- invalid parameter sets for error path invocation

User chooses `Accept` or `Cancel`.

## 6.6 Step 4: Begin Profiling

Execution:

1. Start app automatically (framework-aware launcher heuristics + optional configured command).
2. Invoke dependency routes first.
3. Invoke main routes next.
4. Log each invocation with:
  - route
  - request params/body
  - status/result
  - completion checkmark on success

On completion, proceed to cleanup proposal step.

## 6.7 Step 5: Code Cleanup Proposal (post-profile)

- Output candidate files with proposed removals:
  - unused functions/types/variables/imports
  - location references (line spans)
- User chooses `Accept` or `Cancel`.
- If accepted:
  - apply code cleanup edits (or simulate only in dry-run)
  - follow edit permission mode and safe/aggressive policy

## 6.8 Final Step: Git Offer

- Offer to create branch + commit with generated message.
- If user accepts:
  - create branch
  - commit applied changes
- If dry-run, skip commit creation unless user explicitly asks to include report-only commit.

## 7. Cleanup Mode Flow

## 7.1 Step 1: Codebase Analysis + Options

- Load settings with same precedence model.
- Present cleanup option toggles in TUI.
- User chooses `Accept` or `Cancel`.

## 7.2 Step 2: Code Cleanup Execution

- Analyze codebase according to enabled options.
- Build file-by-file change plan.
- Apply edits with permission model:
  - `per-edit`: prompt each edit chunk
  - `per-file`: prompt once per file
- Respect `safe` and `aggressive` mode flags.
- In dry-run, produce plan without writing.

## 7.3 Final Step: Git Offer

- Offer branch + commit for applied changes.

## 8. Safety and Edit Policies

## 8.1 Safe vs Aggressive

- `safe=true` (default):
  - behavior-preserving transforms only
  - conservative deletion threshold for “unused”
  - require high confidence for automated removal
- `aggressive=true`:
  - allow deeper refactors and broader simplification
  - still requires compilable/parsable output checks

If both are true:

- aggressive feature set enabled, while baseline safety checks still enforced.

## 8.2 Edit Permission Modes

- `per-edit`: ask before each edit chunk.
- `per-file`: ask once before all edits in file.

`auto_apply=true` bypasses prompts if policy allows non-interactive execution.

## 9. Route Profiling Engine (v1 Architecture)

## 9.1 Components

- `scanner`: discovers API routes by framework/language adapters.
- `dependency`: deterministic dependency graph builder.
- `ai-inference`: fallback dependency and parameter inference.
- `runner`: app process manager and HTTP invoker.
- `coverage-map`: maps runtime evidence to code symbols.
- `cleanup-planner`: identifies removable code and candidate edits.
- `patcher`: applies edits with policy gating.

## 9.2 Adapter Contracts

Each route adapter returns normalized metadata:

- route ID
- method/path
- handler symbol/file
- middleware chain
- likely prerequisites
- framework confidence score

## 9.3 App Startup

Startup strategy:

1. Use configured start command if present.
2. Fallback to framework heuristics:
  - Node: package scripts and common start commands
  - Go: `go run`/known entrypoints
  - Django: manage.py runserver with test settings
3. Health-check until ready or timeout.

## 10. AI Integration

Uses configured OpenAI model for:

- dependency inference fallback
- parameter generation
- cleanup patch generation/refinement

Requirements:

- deterministic-first architecture must run before AI calls.
- all AI-suggested edits go through parser/validator checks.
- prompt templates versioned in code.

## 11. JSON Report Output

Default output path:

- `.ccc/reports/<timestamp>.json` (UTC timestamp, filesystem-safe format).

Required top-level fields:

- `run_id`
- `timestamp_utc`
- `mode` (`profile` or `cleanup`)
- `project_root`
- `effective_settings` (values + source chain)
- `steps` (status, duration, errors)
- `routes` (discovered, selected, dependencies)
- `profiling_runs` (invocations, payload summaries, outcomes)
- `cleanup_plan` (by file, by edit)
- `applied_changes` (or simulated in dry-run)
- `git` (branch/commit actions and result)
- `warnings`
- `errors`

Report must be written even on partial failure when possible.

## 12. Error Handling (v1)

- Each step has recoverable and fatal error classes.
- Recoverable errors show actionable next steps and allow retry.
- Fatal errors terminate run with:
  - clear reason
  - step where failure happened
  - report path
  - non-zero exit code

## 13. Cross-Platform Notes

- Use Go standard library path/process APIs for Linux/macOS/Windows compatibility.
- Avoid shell-specific assumptions for command execution.
- Normalize line endings when patching files.

## 14. Initial Implementation Task Breakdown

1. Initialize Go module and CLI command skeleton (`configure`, `profile`, `cleanup`).
2. Implement config loader/merger with precedence and source-chain tracking.
3. Build shared TUI layout framework and reusable step renderer.
4. Implement route scanning adapters (Node, Go, Django).
5. Implement deterministic dependency detection and graph validation.
6. Integrate AI fallback for dependencies + parameter analysis.
7. Implement short-circuit enhancement workflow and code patch pipeline.
8. Implement app startup manager and route invoker runner.
9. Implement cleanup planner and edit engine with permission modes.
10. Add safe/aggressive rule gating.
11. Add dry-run behavior across profile/cleanup.
12. Add JSON report writer and step telemetry.
13. Add final git branch+commit flow.
14. Add integration tests with sample projects (Node, Go, Django).

## 15. Open Items for Future Versions

- Frontend profiling mode.
- Additional frameworks/languages.
- CI orchestration and policy packs.
- Rich HTML report rendering.
