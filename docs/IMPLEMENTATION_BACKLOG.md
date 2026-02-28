# Cool Code Cleanup Prioritized Implementation Backlog

## Priority Model

- P0: required to ship any usable v1 slice.
- P1: required for full v1 behavior from spec.
- P2: quality/completeness enhancements.

## Epic 0: Project Bootstrap and Dev Foundation (P0)

Goal: create runnable CLI skeleton in Go with command structure and baseline project layout.

### CCC-001: Initialize Go module and repo layout

Priority: P0  
Depends on: none

Scope:

- Initialize `go.mod`.
- Add `cmd/ccc/main.go`.
- Add internal package layout for config, tui, modes, report, git.
- Add `.ccc/` runtime directory conventions in code docs.

Acceptance Criteria:

- `go run ./cmd/ccc --help` executes successfully.
- Repo contains clear package boundaries for v1 features.

### CCC-002: Add root command and subcommands

Priority: P0  
Depends on: CCC-001

Scope:

- Implement `configure`, `profile`, `cleanup` command stubs.
- Add global flags (`--config`, `--safe`, `--aggressive`, `--dry-run`, `--non-interactive`, `--report-path`).

Acceptance Criteria:

- `ccc configure --help`, `ccc profile --help`, `ccc cleanup --help` all work.
- Global flags are visible and parse into a shared runtime options object.

### CCC-003: Add baseline run report writer

Priority: P0  
Depends on: CCC-002

Scope:

- Create report path generator `.ccc/reports/<timestamp>.json`.
- Write minimal run metadata on command exit (success/failure).

Acceptance Criteria:

- Running any command writes a JSON report file.
- Report path can be overridden by `--report-path`.

## Epic 1: Settings and Configuration Engine (P0)

Goal: deterministic settings resolution with provenance.

### CCC-010: Implement config schema and loader

Priority: P0  
Depends on: CCC-001

Scope:

- Create typed config structs matching engineering spec.
- Load `.ccc/config.json` when present.
- Validate enum fields (`edit_permission_mode`).

Acceptance Criteria:

- Valid config loads with defaults for missing fields.
- Invalid enum values produce actionable errors.

### CCC-011: Implement precedence merger + source chain tracking

Priority: P0  
Depends on: CCC-010, CCC-002

Scope:

- Merge settings from defaults, config, env, CLI.
- Preserve source chain for each value (e.g. `config -> env -> cli`).

Acceptance Criteria:

- Effective settings match precedence rules.
- Source chain is available to TUI and report output.

### CCC-012: Implement `ccc configure`

Priority: P1  
Depends on: CCC-010, CCC-011

Scope:

- Interactive prompts to set API key/model/defaults.
- Save project-local `.ccc/config.json`.

Acceptance Criteria:

- `ccc configure` creates/updates `.ccc/config.json`.
- User can choose env-var reference vs plaintext key storage.

## Epic 2: Shared TUI Framework (P0)

Goal: consistent multi-step screen layout and interaction model.

### CCC-020: Build unified step renderer

Priority: P0  
Depends on: CCC-002

Scope:

- Implement shared layout sections:
  - header (step)
  - description
  - content pane
  - action bar (`Accept`, `Back`, `Cancel`)

Acceptance Criteria:

- All mode steps use same renderer contract.
- Layout consistency is visibly enforced.

### CCC-021: Add interactive list/toggle components

Priority: P0  
Depends on: CCC-020

Scope:

- Keyboard nav: up/down, space toggle, enter confirm.
- Support disabled rows with inline reasons.

Acceptance Criteria:

- Route/option toggles work in TUI.
- Disabled dependency items show explanatory message.

## Epic 3: Profile Mode Engine (P1)

Goal: complete profile flow with route execution and cleanup proposal.

### CCC-030: Implement Step 1a (profiling options)

Priority: P1  
Depends on: CCC-011, CCC-020, CCC-021

Scope:

- Render effective settings and source attribution.
- Allow user toggling and proceed/cancel behavior.

Acceptance Criteria:

- Screen shows each option value and source chain.
- Updated selections flow into runtime context.

### CCC-031: Route discovery adapters (Node, Go, Django)

Priority: P1  
Depends on: CCC-002

Scope:

- Implement static scanners for supported frameworks.
- Return normalized route metadata contract.

Acceptance Criteria:

- Scanners detect representative route samples in fixture projects.
- Discovery output includes method/path/file/handler.

### CCC-032: Deterministic dependency detector + AI fallback hook

Priority: P1  
Depends on: CCC-031

Scope:

- Build deterministic dependency graph pass.
- Define AI fallback interface and confidence metadata.

Acceptance Criteria:

- Dependency graph built for known auth/payment fixture scenarios.
- Missing-confidence routes are routed through fallback interface.

### CCC-033: Implement Step 1b short-circuit enhancement flow

Priority: P1  
Depends on: CCC-032, CCC-021

Scope:

- Conditional prompt for short-circuit env var.
- Route-level toggle UI for patch candidates.
- Auto patch application when accepted.

Acceptance Criteria:

- Step appears only when required by dependency analysis.
- Accepted patches are generated and tracked in plan/report.

### CCC-034: Implement Step 2 route selection with dependency constraints

Priority: P1  
Depends on: CCC-032, CCC-021

Scope:

- Route selection list.
- Disable illegal toggles when dependency constraints exist.

Acceptance Criteria:

- User cannot disable prerequisite route while dependents enabled.
- Constraint explanation is displayed inline.

### CCC-035: Implement Step 3 parameter analysis

Priority: P1  
Depends on: CCC-034

Scope:

- Valid + invalid test parameter generation per route.
- TUI review/accept/cancel step.

Acceptance Criteria:

- Parameter plans exist for each selected route.
- Output feeds invocation runner.

### CCC-036: Implement Step 4 profiling execution runner

Priority: P1  
Depends on: CCC-035

Scope:

- Start app process automatically.
- Health-check readiness.
- Invoke dependency routes first, then target routes.
- Print invocation + status + success checkmark.

Acceptance Criteria:

- Runner executes ordered route plan against fixtures.
- Failures are surfaced with per-invocation context.

### CCC-037: Implement Step 5 post-profile cleanup proposal

Priority: P1  
Depends on: CCC-036

Scope:

- Map execution evidence to removable symbol candidates.
- Present per-file cleanup list and line references.
- Apply or simulate edits depending on dry-run.

Acceptance Criteria:

- Proposal output contains file-level grouped changes.
- Accept path applies edits with permission policy.

## Epic 4: Cleanup Mode Engine (P1)

Goal: full cleanup mode from analysis through modifications.

### CCC-040: Implement cleanup options analysis step

Priority: P1  
Depends on: CCC-011, CCC-020, CCC-021

Scope:

- Load cleanup toggles from effective settings.
- Interactive option toggles and confirmation.

Acceptance Criteria:

- Cleanup options shown and editable in TUI.
- Accept/cancel works as expected.

### CCC-041: Implement cleanup planner and edit executor

Priority: P1  
Depends on: CCC-040

Scope:

- Build file-by-file change plans for enabled options.
- Apply changes using permission mode (`per-edit`/`per-file`).

Acceptance Criteria:

- Applies only approved edits in interactive mode.
- Produces no writes under dry-run.

### CCC-042: Implement safe/aggressive rule gates

Priority: P1  
Depends on: CCC-041

Scope:

- Enforce safe defaults.
- Enable broader transformations under aggressive mode.

Acceptance Criteria:

- Safe mode blocks risky transforms by default.
- Aggressive mode enables configured risky transforms.

## Epic 5: Patch, Permission, and Git Flow (P1)

Goal: complete user-governed modification lifecycle.

### CCC-050: Implement permission gate engine

Priority: P1  
Depends on: CCC-021

Scope:

- Prompt policy for `per-edit` and `per-file`.
- `auto_apply` behavior integration.

Acceptance Criteria:

- Prompt cadence matches selected permission mode.
- Actions logged to report.

### CCC-051: Implement final branch + commit step

Priority: P1  
Depends on: CCC-037 or CCC-041

Scope:

- Offer to create branch and commit.
- Generate commit message from mode + summary.

Acceptance Criteria:

- Branch + commit succeeds when user accepts.
- Decline path exits cleanly without git mutation.

## Epic 6: JSON Reporting and Observability (P1)

Goal: robust structured output for every run.

### CCC-060: Expand report schema to full v1 structure

Priority: P1  
Depends on: CCC-003, CCC-011

Scope:

- Add top-level fields from spec:
  - run metadata
  - effective settings
  - steps/durations
  - route/dependency info
  - cleanup plan and applied changes
  - git actions
  - warnings/errors

Acceptance Criteria:

- Report validates against internal schema contract.
- Partial-failure runs still emit report.

## Epic 7: Testing and Fixtures (P1)

Goal: confidence across supported stacks and flows.

### CCC-070: Add fixture projects for Node, Go, Django

Priority: P1  
Depends on: CCC-031

Scope:

- Add minimal sample API projects with known routes/dependencies.

Acceptance Criteria:

- Fixtures run in local test harness.
- Route detection assertions are stable.

### CCC-071: Integration tests for profile flow

Priority: P1  
Depends on: CCC-036, CCC-070

Scope:

- End-to-end tests through options, discovery, dependency, invocation, proposal.

Acceptance Criteria:

- Golden test outputs for route order and report sections.

### CCC-072: Integration tests for cleanup flow

Priority: P1  
Depends on: CCC-041

Scope:

- End-to-end cleanup tests in safe and aggressive modes.

Acceptance Criteria:

- Safe mode and aggressive mode produce expected distinct plans.

## Epic 8: Polish and Hardening (P2)

Goal: improve UX and maintainability after core behavior works.

### CCC-080: Improve non-interactive mode behavior

Priority: P2  
Depends on: P1 epics

Scope:

- Enforce required flags in CI/headless runs.
- Better diagnostics for missing decisions.

### CCC-081: Add richer diagnostics and remediation hints

Priority: P2  
Depends on: P1 epics

Scope:

- Improve step-specific error explanations.
- Add retry guidance and likely fixes.

## Execution Order (Recommended)

1. CCC-001, CCC-002, CCC-003
2. CCC-010, CCC-011
3. CCC-020, CCC-021
4. CCC-030, CCC-031, CCC-032
5. CCC-034, CCC-035, CCC-036
6. CCC-037, CCC-040, CCC-041, CCC-042
7. CCC-050, CCC-051, CCC-060
8. CCC-070, CCC-071, CCC-072
9. CCC-012, CCC-033, P2 backlog

## Start-Now Slice (Implementation Immediately)

Sprint Slice A (first buildable baseline):

- CCC-001 Initialize module/layout.
- CCC-002 Root/subcommands + global flags.
- CCC-010 Config schema + loader.
- CCC-011 Precedence merger with source chains.
- CCC-003 Minimal report writer.

Definition of Done for Slice A:

- `ccc configure|profile|cleanup --help` work.
- Effective config resolution works with source attribution.
- Running `ccc profile` and `ccc cleanup` emits baseline JSON report.
