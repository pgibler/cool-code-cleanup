package mode

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"cool-code-cleanup/internal/ai"
	"cool-code-cleanup/internal/app"
	"cool-code-cleanup/internal/cleanup"
	"cool-code-cleanup/internal/config"
	"cool-code-cleanup/internal/dependency"
	"cool-code-cleanup/internal/discovery"
	"cool-code-cleanup/internal/gitflow"
	"cool-code-cleanup/internal/permission"
	"cool-code-cleanup/internal/profile"
	"cool-code-cleanup/internal/rules"
	"cool-code-cleanup/internal/runner"
	"cool-code-cleanup/internal/shortcircuit"
	"cool-code-cleanup/internal/tui"
)

type ProfileFlags struct {
	IncludeRoutes          []string
	IgnoreRoutes           []string
	DependencyShortCircuit bool
	EditPermissionMode     string
	AutoApply              bool
	CreateBranch           bool
	CreateBranchSet        bool
	CommitChanges          bool
	CommitChangesSet       bool
}

type CleanupFlags struct {
	RulesPath          string
	RulesLocalPath     string
	EnableRules        []string
	DisableRules       []string
	EditPermissionMode string
	AutoApply          bool
	CreateBranch       bool
	CreateBranchSet    bool
	CommitChanges      bool
	CommitChangesSet   bool
}

var CleanupExecutorFactory = func(cfg config.Config) (cleanup.RuleExecutor, error) {
	return ai.NewOpenAIExecutorFromConfig(cfg)
}

func RunConfigure(rt *app.Runtime) error {
	if rt.Effective.NonInteractive {
		return fmt.Errorf("configure requires interactive input; rerun without --non-interactive")
	}
	io := tui.NewIO(os.Stdin, os.Stdout)
	rt.AddStep("configure", "in_progress", "collecting settings")
	cfg := rt.Effective.Config

	model, err := io.Prompt("OpenAI model (default gpt-5): ")
	if err != nil {
		return err
	}
	if strings.TrimSpace(model) != "" {
		cfg.OpenAI.Model = model
	}
	apiEnv, err := io.Prompt("API key env var name (default OPENAI_API_KEY): ")
	if err != nil {
		return err
	}
	if strings.TrimSpace(apiEnv) != "" {
		cfg.OpenAI.APIKeyEnv = apiEnv
	}
	editMode, err := io.Prompt("Edit permission mode [per-edit/per-file] (default per-file): ")
	if err != nil {
		return err
	}
	if editMode == "per-edit" || editMode == "per-file" {
		cfg.Profile.EditPermissionMode = editMode
		cfg.Cleanup.EditPermissionMode = editMode
	}

	if err := config.Save(rt.Effective.ConfigPath, cfg); err != nil {
		rt.AddStep("configure", "failed", err.Error())
		return err
	}
	rt.Effective.Config = cfg
	rt.AddStep("configure", "completed", "saved config")
	return nil
}

func RunProfile(rt *app.Runtime, flags ProfileFlags) error {
	io := tui.NewIO(os.Stdin, os.Stdout)
	mergeProfileFlags(&rt.Effective.Config, flags)

	// Step 1a: profiling options
	options := []tui.ToggleItem{
		{
			ID:                   "dependency_short_circuit",
			Label:                "Dependency short-circuiting",
			Details:              []string{"source chain: " + chain(rt.Effective.SourceChains["profile.dependency_short_circuit"])},
			Enabled:              rt.Effective.Config.Profile.DependencyShortCircuit,
			ClearDetailsOnToggle: true,
		},
		{
			ID:                   "safe_mode",
			Label:                "Safe mode",
			Details:              []string{"source chain: " + chain(rt.Effective.SourceChains["modes.safe"])},
			Enabled:              rt.Effective.Config.Modes.Safe,
			ClearDetailsOnToggle: true,
		},
		{
			ID:                   "aggressive_mode",
			Label:                "Aggressive mode",
			Details:              []string{"source chain: " + chain(rt.Effective.SourceChains["modes.aggressive"])},
			Enabled:              rt.Effective.Config.Modes.Aggressive,
			ClearDetailsOnToggle: true,
		},
		{
			ID:                   "dry_run",
			Label:                "Dry run",
			Details:              []string{"source chain: " + chain(rt.Effective.SourceChains["modes.dry_run"])},
			Enabled:              rt.Effective.Config.Modes.DryRun,
			ClearDetailsOnToggle: true,
		},
	}
	list := tui.NewToggleList(options)
	screen := tui.StepScreen{
		Mode:        "Profile",
		StepName:    "Step 1a: Profiling options",
		Description: "Review effective options and sources. Toggle settings then accept.",
		Actions: []tui.Action{
			{Key: "accept", Label: "Accept", Selected: true},
			{Key: "back", Label: "Back"},
			{Key: "cancel", Label: "Cancel"},
		},
	}
	if !rt.Effective.NonInteractive {
		ok, canceled, err := io.RunToggleStep(screen, &list)
		if err != nil {
			return err
		}
		if canceled {
			rt.AddStep("step_1a_options", "canceled", "user canceled")
			return nil
		}
		if !ok {
			rt.AddStep("step_1a_options", "failed", "back not supported from first step")
			return nil
		}
	}
	rt.AddStep("step_1a_options", "completed", "profiling options accepted")
	for _, item := range list.Items {
		switch item.ID {
		case "dependency_short_circuit":
			rt.Effective.Config.Profile.DependencyShortCircuit = item.Enabled
		case "safe_mode":
			rt.Effective.Config.Modes.Safe = item.Enabled
		case "aggressive_mode":
			rt.Effective.Config.Modes.Aggressive = item.Enabled
		case "dry_run":
			rt.Effective.Config.Modes.DryRun = item.Enabled
		}
	}

	root, _ := os.Getwd()
	routes, err := discovery.Discover(root)
	if err != nil {
		rt.AddStep("route_discovery", "failed", err.Error())
		return err
	}
	filtered := filterRoutes(routes, rt.Effective.Config.Profile.IncludeRoutes, rt.Effective.Config.Profile.IgnoreRoutes)
	depGraph, err := dependency.Detect(filtered, ai.NoopFallback{})
	if err != nil {
		rt.AddStep("dependency_detection", "failed", err.Error())
		return err
	}
	rt.AddStep("route_discovery", "completed", fmt.Sprintf("discovered %d routes", len(filtered)))
	rt.AddStep("dependency_detection", "completed", depGraph.Rationale)
	if len(depGraph.Dependencies) == 0 {
		msg := "No route dependencies detected. Proceed to the next step?"
		fmt.Fprintln(os.Stdout, msg)
		if !rt.Effective.NonInteractive {
			resp, err := io.Prompt("[Y/n]: ")
			if err != nil {
				return err
			}
			if strings.EqualFold(strings.TrimSpace(resp), "n") || strings.EqualFold(strings.TrimSpace(resp), "no") {
				rt.AddStep("dependency_confirmation", "canceled", "user canceled after no-dependency notice")
				return nil
			}
		}
	}
	if missing := missingDependencies(filtered, depGraph.Dependencies); len(missing) > 0 {
		fmt.Fprintln(os.Stdout, "Some required dependency routes are disabled or missing from selection:")
		for _, m := range missing {
			fmt.Fprintln(os.Stdout, " - "+m)
		}
		if !rt.Effective.NonInteractive {
			resp, err := io.Prompt("Enable required dependencies and continue? [Y/n]: ")
			if err != nil {
				return err
			}
			if strings.EqualFold(strings.TrimSpace(resp), "n") || strings.EqualFold(strings.TrimSpace(resp), "no") {
				rt.AddStep("dependency_confirmation", "canceled", "user quit due to missing dependencies")
				return nil
			}
		}
	}

	// Step 1b: dependency route short-circuiting
	if len(depGraph.Dependencies) > 0 && rt.Effective.Config.Profile.DependencyShortCircuit {
		envVar := rt.Effective.Config.Profile.ShortCircuitEnvVar
		if strings.TrimSpace(envVar) == "" {
			envVar = "CoolCodeCleanupShortCircuit"
		}
		if !rt.Effective.NonInteractive {
			resp, err := io.Prompt(fmt.Sprintf("Short-circuit env var name (default %s): ", envVar))
			if err != nil {
				return err
			}
			if strings.TrimSpace(resp) != "" {
				envVar = resp
			}
			rt.Effective.Config.Profile.ShortCircuitEnvVar = envVar

			saveCfg, err := io.Prompt("Save short-circuit env var to config? [y/N]: ")
			if err != nil {
				return err
			}
			if strings.EqualFold(strings.TrimSpace(saveCfg), "y") || strings.EqualFold(strings.TrimSpace(saveCfg), "yes") {
				rt.Effective.Config.Profile.SaveShortCircuitToConfig = true
				if err := config.Save(rt.Effective.ConfigPath, rt.Effective.Config); err != nil {
					return err
				}
			}
			updateEnv, err := io.Prompt("Update .env with short-circuit value=true? [y/N]: ")
			if err != nil {
				return err
			}
			if strings.EqualFold(strings.TrimSpace(updateEnv), "y") || strings.EqualFold(strings.TrimSpace(updateEnv), "yes") {
				if err := upsertEnv(filepath.Join(root, ".env"), envVar, "true"); err != nil {
					return err
				}
				rt.Effective.Config.Profile.UpdateEnvFile = true
			}
		}

		candidates := shortcircuit.Candidates(filtered, depGraph.Dependencies)
		patchItems := make([]tui.ToggleItem, 0, len(candidates))
		for _, c := range candidates {
			patchItems = append(patchItems, tui.ToggleItem{
				ID:      c.RouteID,
				Label:   c.Description,
				Details: []string{c.File},
				Enabled: true,
			})
		}
		if len(patchItems) > 0 && !rt.Effective.NonInteractive {
			pl := tui.NewToggleList(patchItems)
			ps := tui.StepScreen{
				Mode:        "Profile",
				StepName:    "Step 1b: Dependency route short-circuiting enhancement",
				Description: "Select dependency routes to patch with short-circuit markers.",
				Actions: []tui.Action{
					{Key: "accept", Label: "Accept", Selected: true},
					{Key: "cancel", Label: "Cancel"},
				},
			}
			_, canceled, err := io.RunToggleStep(ps, &pl)
			if err != nil {
				return err
			}
			if canceled {
				rt.AddStep("step_1b_short_circuit", "canceled", "user canceled")
				return nil
			}
			selectedIDs := map[string]bool{}
			for _, i := range pl.Items {
				if i.Enabled {
					selectedIDs[i.ID] = true
				}
			}
			var selected []shortcircuit.PatchCandidate
			for _, c := range candidates {
				if selectedIDs[c.RouteID] {
					selected = append(selected, c)
				}
			}
			applied, err := shortcircuit.Apply(selected, envVar, rt.Effective.Config.Modes.DryRun)
			if err != nil {
				return err
			}
			for _, a := range applied {
				rt.Report.AppliedChanges = append(rt.Report.AppliedChanges, a)
			}
			rt.AddStep("step_1b_short_circuit", "completed", fmt.Sprintf("patched %d dependency routes", countShortCircuitApplied(applied)))
		} else {
			rt.AddStep("step_1b_short_circuit", "completed", "no candidate dependency routes required patching")
		}
	}

	// Step 2: enable/disable routes
	routeItems := make([]tui.ToggleItem, 0, len(filtered))
	for _, r := range filtered {
		disabledReason := ""
		if dependents := dependentRoutes(depGraph.Dependencies, r.ID); len(dependents) > 0 {
			disabledReason = fmt.Sprintf("required by %d enabled route(s)", len(dependents))
		}
		routeItems = append(routeItems, tui.ToggleItem{
			ID:             r.ID,
			Label:          fmt.Sprintf("%s %s", r.Method, r.Path),
			Details:        depGraph.Dependencies[r.ID],
			Enabled:        true,
			DisabledReason: disabledReason,
		})
	}
	routeList := tui.NewToggleList(routeItems)
	routeScreen := tui.StepScreen{
		Mode:        "Profile",
		StepName:    "Step 2: Enable/disable routes to profile",
		Description: "Select which discovered routes to execute.",
		Actions: []tui.Action{
			{Key: "accept", Label: "Accept", Selected: true},
			{Key: "cancel", Label: "Cancel"},
		},
	}
	if !rt.Effective.NonInteractive {
		_, canceled, err := io.RunToggleStep(routeScreen, &routeList)
		if err != nil {
			return err
		}
		if canceled {
			rt.AddStep("step_2_route_selection", "canceled", "user canceled")
			return nil
		}
	}
	rt.AddStep("step_2_route_selection", "completed", "route selection completed")

	selected := selectedRoutes(filtered, routeList)
	paramPlans := profile.AnalyzeParameters(selected)
	rt.AddStep("step_3_parameter_analysis", "completed", fmt.Sprintf("generated plans for %d routes", len(paramPlans)))

	// Step 4: profiling execution
	var invocations []runner.Invocation
	if len(selected) > 0 {
		proc, cmd := runner.Start(root)
		if proc != nil {
			defer proc.Stop()
			_ = runner.WaitForHealth("http://127.0.0.1:8000/health", 2*time.Second)
		}
		fmt.Fprintf(os.Stdout, "App startup command: %s\n", cmd)
		invocations = runner.Execute("http://127.0.0.1:8000", selected, paramPlans, depGraph.Dependencies)
		for _, inv := range invocations {
			fmt.Fprintln(os.Stdout, runner.FormatInvocation(inv))
		}
	}
	rt.AddStep("step_4_profiling", "completed", fmt.Sprintf("executed %d invocations", len(invocations)))

	// Step 5: cleanup proposal
	defaultRules := rules.DefaultRules().Rules
	var selectedRules []rules.Rule
	for _, r := range defaultRules {
		if r.Enabled {
			selectedRules = append(selectedRules, r)
		}
	}
	cplan, err := cleanup.BuildPlan(root, selectedRules, rt.Effective.Config.Modes.Safe, rt.Effective.Config.Modes.Aggressive)
	if err != nil {
		return err
	}
	perm := permission.Engine{
		Mode:           rt.Effective.Config.Profile.EditPermissionMode,
		AutoApply:      rt.Effective.Config.Profile.AutoApply,
		NonInteractive: rt.Effective.NonInteractive,
	}
	fileGroups := map[string][]cleanup.Edit{}
	for _, e := range cplan.Edits {
		fileGroups[e.File] = append(fileGroups[e.File], e)
	}
	var approvedPlan cleanup.Plan
	for file, edits := range fileGroups {
		ok, err := perm.ApproveFile(io, file, len(edits))
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		for _, e := range edits {
			ok, err := perm.ApproveEdit(io, e.File, e.Description)
			if err != nil {
				return err
			}
			if ok {
				approvedPlan.Edits = append(approvedPlan.Edits, e)
			}
		}
	}
	applied, err := cleanup.ApplyPlan(approvedPlan, rt.Effective.Config.Modes.Safe, rt.Effective.Config.Modes.Aggressive, rt.Effective.Config.Modes.DryRun)
	if err != nil {
		return err
	}
	rt.AddStep("step_5_cleanup", "completed", fmt.Sprintf("applied %d edits", countApplied(applied)))

	rt.Report.Routes = map[string]any{
		"discovered":   filtered,
		"selected":     selected,
		"dependencies": depGraph.Dependencies,
	}
	for _, inv := range invocations {
		rt.Report.ProfilingRuns = append(rt.Report.ProfilingRuns, inv)
	}
	for _, e := range approvedPlan.Edits {
		rt.Report.CleanupPlan = append(rt.Report.CleanupPlan, e)
	}
	for _, e := range applied {
		rt.Report.AppliedChanges = append(rt.Report.AppliedChanges, e)
	}

	if rt.Effective.Config.Git.AutoOfferBranchAndCommit && !rt.Effective.Config.Modes.DryRun {
		createBranch, commitChanges, err := decideGitActions(rt.Effective.NonInteractive, flags.CreateBranchSet, flags.CreateBranch, flags.CommitChangesSet, flags.CommitChanges, io)
		if err != nil {
			return err
		}
		gitMeta := map[string]any{}
		if createBranch {
			res := gitflow.CreateBranch("profile")
			gitMeta["create_branch"] = res
			if res.Error != "" {
				rt.Report.Git = gitMeta
				rt.AddStep("final_git_step", "failed", res.Error)
				return nil
			}
		}
		if commitChanges {
			res := gitflow.CommitChanges("profile")
			gitMeta["commit_changes"] = res
			rt.Report.Git = gitMeta
			if res.Error != "" {
				rt.AddStep("final_git_step", "failed", res.Error)
			} else {
				rt.AddStep("final_git_step", "completed", fmt.Sprintf("branch=%s commit=%s", res.Branch, res.Commit))
			}
		} else {
			rt.Report.Git = gitMeta
		}
	}
	return nil
}

func RunCleanup(rt *app.Runtime, flags CleanupFlags) error {
	io := tui.NewIO(os.Stdin, os.Stdout)
	// Cleanup mode is analysis/cleanup only. Route dependency and short-circuit flows are profile-only.
	rt.Effective.Config.Profile.DependencyShortCircuit = false
	rt.AddStep("cleanup_scope", "completed", "dependency and short-circuit checks are skipped in cleanup mode")

	if flags.EditPermissionMode == "per-edit" || flags.EditPermissionMode == "per-file" {
		rt.Effective.Config.Cleanup.EditPermissionMode = flags.EditPermissionMode
	}
	rt.Effective.Config.Cleanup.AutoApply = flags.AutoApply

	rulesPath := flags.RulesPath
	if strings.TrimSpace(rulesPath) == "" {
		rulesPath = filepath.Join(".ccc", "rules", "cleanup.rules.json")
	}
	localRulesPath := flags.RulesLocalPath
	if strings.TrimSpace(localRulesPath) == "" {
		localRulesPath = filepath.Join(".ccc", "rules", "cleanup.local.json")
	}
	if err := rules.EnsureBaseFile(rulesPath); err != nil {
		return err
	}
	loaded, warnings, err := rules.Load(rulesPath, localRulesPath)
	if err != nil {
		return err
	}
	loaded = rules.ApplyCLIOverrides(loaded, flags.EnableRules, flags.DisableRules)
	rt.Report.Warnings = append(rt.Report.Warnings, warnings...)

	items := []tui.ToggleItem{}
	for _, r := range loaded {
		items = append(items, tui.ToggleItem{
			ID:                   r.ID,
			Label:                r.Title,
			Details:              []string{"source chain: " + chain(r.SourceChain)},
			Enabled:              r.Enabled,
			ClearDetailsOnToggle: true,
		})
	}
	list := tui.NewToggleList(items)
	screen := tui.StepScreen{
		Mode:        "Cleanup",
		StepName:    "Step 1: Codebase analysis rules",
		Description: "Review and toggle cleanup rules loaded from rules files.",
		Actions: []tui.Action{
			{Key: "accept", Label: "Accept", Selected: true},
			{Key: "cancel", Label: "Cancel"},
		},
	}
	if !rt.Effective.NonInteractive {
		_, canceled, err := io.RunToggleStep(screen, &list)
		if err != nil {
			return err
		}
		if canceled {
			rt.AddStep("cleanup_step_1", "canceled", "user canceled")
			return nil
		}
	}
	selectedRules := make([]rules.Rule, 0, len(loaded))
	enabledByID := map[string]bool{}
	for _, item := range list.Items {
		enabledByID[item.ID] = item.Enabled
	}
	for _, r := range loaded {
		r.Enabled = enabledByID[r.ID]
		if r.Enabled {
			selectedRules = append(selectedRules, r.Rule)
		}
	}
	rt.Report.Rules = map[string]any{
		"rules_file":        rulesPath,
		"rules_local_file":  localRulesPath,
		"loaded_rules":      loaded,
		"selected_rule_ids": ruleIDs(selectedRules),
	}
	rt.AddStep("cleanup_step_1", "completed", fmt.Sprintf("rules accepted (%d enabled)", len(selectedRules)))

	root, _ := os.Getwd()
	executor, err := CleanupExecutorFactory(rt.Effective.Config)
	if err != nil {
		return err
	}
	dryRun := rt.Effective.Config.Modes.DryRun
	if !dryRun && !rt.Effective.NonInteractive && !rt.Effective.Config.Cleanup.AutoApply {
		resp, err := io.Prompt("Apply AI-generated cleanup changes to files? [y/N]: ")
		if err != nil {
			return err
		}
		resp = strings.ToLower(strings.TrimSpace(resp))
		if resp != "y" && resp != "yes" {
			dryRun = true
		}
	}
	plan, applied, err := cleanup.ApplyRules(root, selectedRules, rt.Effective.Config.Modes.Safe, rt.Effective.Config.Modes.Aggressive, dryRun, executor)
	if err != nil {
		return err
	}
	rt.AddStep("cleanup_step_2", "completed", fmt.Sprintf("applied %d AI rule edits", countApplied(applied)))
	for _, e := range plan.Edits {
		rt.Report.CleanupPlan = append(rt.Report.CleanupPlan, e)
	}
	for _, e := range applied {
		rt.Report.AppliedChanges = append(rt.Report.AppliedChanges, e)
	}
	if rt.Effective.Config.Git.AutoOfferBranchAndCommit && !rt.Effective.Config.Modes.DryRun {
		createBranch, commitChanges, err := decideGitActions(rt.Effective.NonInteractive, flags.CreateBranchSet, flags.CreateBranch, flags.CommitChangesSet, flags.CommitChanges, io)
		if err != nil {
			return err
		}
		gitMeta := map[string]any{}
		if createBranch {
			res := gitflow.CreateBranch("cleanup")
			gitMeta["create_branch"] = res
			if res.Error != "" {
				rt.Report.Git = gitMeta
				rt.AddStep("final_git_step", "failed", res.Error)
				return nil
			}
		}
		if commitChanges {
			res := gitflow.CommitChanges("cleanup")
			gitMeta["commit_changes"] = res
			rt.Report.Git = gitMeta
			if res.Error != "" {
				rt.AddStep("final_git_step", "failed", res.Error)
			} else {
				rt.AddStep("final_git_step", "completed", fmt.Sprintf("branch=%s commit=%s", res.Branch, res.Commit))
			}
		} else {
			rt.Report.Git = gitMeta
		}
	}
	return nil
}

func mergeProfileFlags(cfg *config.Config, flags ProfileFlags) {
	if len(flags.IncludeRoutes) > 0 {
		cfg.Profile.IncludeRoutes = slices.Clone(flags.IncludeRoutes)
	}
	if len(flags.IgnoreRoutes) > 0 {
		cfg.Profile.IgnoreRoutes = slices.Clone(flags.IgnoreRoutes)
	}
	if flags.EditPermissionMode == "per-edit" || flags.EditPermissionMode == "per-file" {
		cfg.Profile.EditPermissionMode = flags.EditPermissionMode
	}
	cfg.Profile.DependencyShortCircuit = flags.DependencyShortCircuit
	cfg.Profile.AutoApply = flags.AutoApply
}

func filterRoutes(routes []discovery.Route, include, ignore []string) []discovery.Route {
	ignored := map[string]bool{}
	for _, r := range ignore {
		ignored[strings.ToLower(r)] = true
	}
	included := map[string]bool{}
	for _, r := range include {
		included[strings.ToLower(r)] = true
	}
	var out []discovery.Route
	for _, r := range routes {
		key := strings.ToLower(r.Method + " " + r.Path)
		if len(included) > 0 && !included[key] && !included[strings.ToLower(r.Path)] {
			continue
		}
		if ignored[key] || ignored[strings.ToLower(r.Path)] {
			continue
		}
		out = append(out, r)
	}
	return out
}

func dependentRoutes(deps map[string][]string, routeID string) []string {
	var out []string
	for route, reqs := range deps {
		for _, req := range reqs {
			if req == routeID {
				out = append(out, route)
				break
			}
		}
	}
	return out
}

func selectedRoutes(routes []discovery.Route, list tui.ToggleList) []discovery.Route {
	enabled := map[string]bool{}
	for _, item := range list.Items {
		if item.Enabled {
			enabled[item.ID] = true
		}
	}
	var out []discovery.Route
	for _, r := range routes {
		if enabled[r.ID] {
			out = append(out, r)
		}
	}
	return out
}

func countApplied(edits []cleanup.Edit) int {
	count := 0
	for _, e := range edits {
		if e.Applied {
			count++
		}
	}
	return count
}

func offerCreateBranch(io tui.IO) (bool, error) {
	resp, err := io.Prompt("Create branch? [y/N]: ")
	if err != nil {
		return false, err
	}
	resp = strings.ToLower(strings.TrimSpace(resp))
	return resp == "y" || resp == "yes", nil
}

func offerCommit(io tui.IO) (bool, error) {
	resp, err := io.Prompt("Commit changes? [y/N]: ")
	if err != nil {
		return false, err
	}
	resp = strings.ToLower(strings.TrimSpace(resp))
	return resp == "y" || resp == "yes", nil
}

func decideGitActions(nonInteractive bool, createSet, createValue, commitSet, commitValue bool, io tui.IO) (createBranch bool, commitChanges bool, err error) {
	if nonInteractive {
		return createSet && createValue, commitSet && commitValue, nil
	}
	createBranch = createValue
	if !createSet {
		createBranch, err = offerCreateBranch(io)
		if err != nil {
			return false, false, err
		}
	}
	commitChanges = commitValue
	if !commitSet {
		commitChanges, err = offerCommit(io)
		if err != nil {
			return false, false, err
		}
	}
	return createBranch, commitChanges, nil
}

func upsertEnv(path, key, value string) error {
	line := key + "=" + value
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(path, []byte(line+"\n"), 0o644)
		}
		return err
	}
	lines := strings.Split(string(data), "\n")
	found := false
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), key+"=") {
			lines[i] = line
			found = true
		}
	}
	if !found {
		lines = append(lines, line)
	}
	return os.WriteFile(path, []byte(strings.TrimRight(strings.Join(lines, "\n"), "\n")+"\n"), 0o644)
}

func countShortCircuitApplied(items []shortcircuit.PatchCandidate) int {
	n := 0
	for _, i := range items {
		if i.Applied {
			n++
		}
	}
	return n
}

func chain(items []string) string {
	if len(items) == 0 {
		return "default"
	}
	return strings.Join(items, " -> ")
}

func missingDependencies(routes []discovery.Route, deps map[string][]string) []string {
	available := map[string]bool{}
	for _, r := range routes {
		available[r.ID] = true
	}
	var missing []string
	for routeID, reqs := range deps {
		for _, req := range reqs {
			if !available[req] {
				missing = append(missing, fmt.Sprintf("route %s requires missing dependency %s", routeID, req))
			}
		}
	}
	return missing
}

func ruleIDs(items []rules.Rule) []string {
	out := make([]string, 0, len(items))
	for _, r := range items {
		out = append(out, r.ID)
	}
	return out
}
