<me>@<laptop_name>:~/Code/cool-code-cleanup$ go run ./cmd/ccc cleanup --non-interactive --aggressive --auto-apply --show-progress
  changed: /home/<me>/Code/cool-code-cleanup/internal/cleanup/cleanup.go | Remove redundant guards | [remove_redundant_guards] Removed
  redundant default returns in directory handling within WalkDir callbacks, simplifying guard logic without changing behavior.
  changed: /home/<me>/Code/cool-code-cleanup/internal/cli/cli.go | Refactor code to follow DRY principles | [refactor_dry] Applied DRY
  refactors: 1) extracted a shared file-walking helper in cleanup to remove duplicated WalkDir logic; 2) centralized CSV parsing by
  exporting config.ParseCSV and using it from cli, removing duplicate implementation.
  changed: /home/<me>/Code/cool-code-cleanup/internal/cleanup/cleanup.go | Refactor code to follow DRY principles | [refactor_dry] Applied
  DRY refactors: 1) extracted a shared file-walking helper in cleanup to remove duplicated WalkDir logic; 2) centralized CSV parsing by
  exporting config.ParseCSV and using it from cli, removing duplicate implementation.
  changed: /home/<me>/Code/cool-code-cleanup/internal/config/config.go | Refactor code to follow DRY principles | [refactor_dry] Applied
  DRY refactors: 1) extracted a shared file-walking helper in cleanup to remove duplicated WalkDir logic; 2) centralized CSV parsing by
  exporting config.ParseCSV and using it from cli, removing duplicate implementation.
  changed: /home/<me>/Code/cool-code-cleanup/internal/report/report.go | Harden code with better error handling | [harden_error_handling]
  Hardened error handling by propagating and contextualizing errors, checking HTTP status codes, avoiding ignored marshal/request errors,
  and wrapping filesystem IO errors with file paths. Applied to OpenAI executor, cleanup execution, short-circuit patching, runner HTTP
  requests, and report writing.
  changed: /home/<me>/Code/cool-code-cleanup/internal/ai/openai_cleanup.go | Harden code with better error handling |
  [harden_error_handling] Hardened error handling by propagating and contextualizing errors, checking HTTP status codes, avoiding ignored
  marshal/request errors, and wrapping filesystem IO errors with file paths. Applied to OpenAI executor, cleanup execution, short-circuit
  patching, runner HTTP requests, and report writing.
  changed: /home/<me>/Code/cool-code-cleanup/internal/cleanup/cleanup.go | Harden code with better error handling |
  [harden_error_handling] Hardened error handling by propagating and contextualizing errors, checking HTTP status codes, avoiding ignored
  marshal/request errors, and wrapping filesystem IO errors with file paths. Applied to OpenAI executor, cleanup execution, short-circuit
  patching, runner HTTP requests, and report writing.
  changed: /home/<me>/Code/cool-code-cleanup/internal/runner/runner.go | Harden code with better error handling | [harden_error_handling]
  Hardened error handling by propagating and contextualizing errors, checking HTTP status codes, avoiding ignored marshal/request errors,
  and wrapping filesystem IO errors with file paths. Applied to OpenAI executor, cleanup execution, short-circuit patching, runner HTTP
  requests, and report writing.
  changed: /home/<me>/Code/cool-code-cleanup/internal/shortcircuit/shortcircuit.go | Harden code with better error handling |
  [harden_error_handling] Hardened error handling by propagating and contextualizing errors, checking HTTP status codes, avoiding ignored
  marshal/request errors, and wrapping filesystem IO errors with file paths. Applied to OpenAI executor, cleanup execution, short-circuit
  patching, runner HTTP requests, and report writing.
  changed: /home/<me>/Code/cool-code-cleanup/internal/ai/fallback.go | Standardize inconsistent naming styles | [standardize_naming]
  Standardized naming by renaming NoopFallback to NopFallback to align with Go's common 'Nop' convention. Updated type declaration and all
  references.
  changed: /home/<me>/Code/cool-code-cleanup/internal/mode/mode.go | Standardize inconsistent naming styles | [standardize_naming]
  Standardized naming by renaming NoopFallback to NopFallback to align with Go's common 'Nop' convention. Updated type declaration and all
  references.
  changed: /home/<me>/Code/cool-code-cleanup/internal/ai/openai_cleanup.go | Simplify complex logic while retaining functionality |
  [simplify_complex_logic] Simplified complex branching and control flow in multiple areas: clarified safety mode selection in OpenAI
  executor, reduced duplication in route filtering, and fixed/simplified toggle list cursor movement loop.
  changed: /home/<me>/Code/cool-code-cleanup/internal/mode/mode.go | Simplify complex logic while retaining functionality |
  [simplify_complex_logic] Simplified complex branching and control flow in multiple areas: clarified safety mode selection in OpenAI
  executor, reduced duplication in route filtering, and fixed/simplified toggle list cursor movement loop.
  changed: /home/<me>/Code/cool-code-cleanup/internal/tui/toggle_list.go | Simplify complex logic while retaining functionality |
  [simplify_complex_logic] Simplified complex branching and control flow in multiple areas: clarified safety mode selection in OpenAI
  executor, reduced duplication in route filtering, and fixed/simplified toggle list cursor movement loop.
  changed: /home/<me>/Code/cool-code-cleanup/internal/cleanup/cleanup.go | Detect expensive functions and offer ideas to improve
  performance | [detect_expensive_functions] Enhanced performance analysis in BuildPlan: detects potential expensive patterns (nested loops,
  heavy operations in loops, regex compilation, repeated I/O/JSON, string building) and generates actionable analysis suggestions per file.
  No code changes are applied; only suggestions are added to the plan.
  cleanup execution complete
  cleanup completed. Report written to .ccc/reports/20260228T134547Z.json