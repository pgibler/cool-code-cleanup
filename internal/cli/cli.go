package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cool-code-cleanup/internal/app"
	"cool-code-cleanup/internal/config"
	modepkg "cool-code-cleanup/internal/mode"
	"cool-code-cleanup/internal/report"
)

func Run(args []string) error {
	if len(args) == 0 {
		printRootHelp()
		return nil
	}

	cmd := args[0]
	switch cmd {
	case "help", "-h", "--help":
		printRootHelp()
		return nil
	case "configure":
		return runCommand("configure", args[1:])
	case "profile":
		return runCommand("profile", args[1:])
	case "cleanup":
		return runCommand("cleanup", args[1:])
	default:
		return fmt.Errorf("unknown command %q\n\n%s", cmd, rootUsage())
	}
}

func runCommand(cmdName string, args []string) error {
	fs := flag.NewFlagSet(cmdName, flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	var cliOpts config.CLIOverrides
	cliOpts.ConfigPath = filepath.Join(".ccc", "config.json")
	cliOpts.Safe = true
	cliOpts.ReportPath = report.DefaultReportPath(time.Now())

	fs.StringVar(&cliOpts.ConfigPath, "config", cliOpts.ConfigPath, "Path to config file")
	fs.BoolVar(&cliOpts.Safe, "safe", true, "Enable safe mode")
	fs.BoolVar(&cliOpts.Aggressive, "aggressive", false, "Enable aggressive mode for riskier refactors")
	fs.BoolVar(&cliOpts.DryRun, "dry-run", false, "Plan changes without writing files")
	fs.BoolVar(&cliOpts.NonInteractive, "non-interactive", false, "Disable prompts and interactive UI")
	fs.StringVar(&cliOpts.ReportPath, "report-path", cliOpts.ReportPath, "Path to write JSON report")

	var profileFlags modepkg.ProfileFlags
	var cleanupFlags modepkg.CleanupFlags
	var includeCSV string
	var ignoreCSV string
	if cmdName == "profile" {
		fs.StringVar(&includeCSV, "include-routes", "", "Routes to include (comma-separated paths or METHOD path)")
		fs.StringVar(&ignoreCSV, "ignore-routes", "", "Routes to ignore (comma-separated paths or METHOD path)")
		fs.BoolVar(&profileFlags.DependencyShortCircuit, "dependency-short-circuit", true, "Enable dependency route short-circuiting enhancement")
		fs.StringVar(&profileFlags.EditPermissionMode, "edit-permission-mode", "", "Edit permission mode (per-edit|per-file)")
		fs.BoolVar(&profileFlags.AutoApply, "auto-apply", false, "Apply edits without per-file prompts if policy allows")
	}
	if cmdName == "cleanup" {
		fs.StringVar(&cleanupFlags.RulesPath, "rules", filepath.Join(".ccc", "rules", "cleanup.rules.json"), "Base cleanup rules file path")
		fs.StringVar(&cleanupFlags.RulesLocalPath, "rules-local", filepath.Join(".ccc", "rules", "cleanup.local.json"), "Optional local cleanup rules override path")
		fs.Func("enable-rule", "Enable cleanup rule by id (repeatable)", func(v string) error {
			cleanupFlags.EnableRules = append(cleanupFlags.EnableRules, strings.TrimSpace(v))
			return nil
		})
		fs.Func("disable-rule", "Disable cleanup rule by id (repeatable)", func(v string) error {
			cleanupFlags.DisableRules = append(cleanupFlags.DisableRules, strings.TrimSpace(v))
			return nil
		})
		fs.StringVar(&cleanupFlags.EditPermissionMode, "edit-permission-mode", "", "Edit permission mode (per-edit|per-file)")
		fs.BoolVar(&cleanupFlags.AutoApply, "auto-apply", false, "Apply edits without per-file prompts if policy allows")
	}

	fs.Usage = func() {
		fmt.Fprintln(os.Stdout, commandUsage(cmdName))
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	detectBoolFlagSet(fs, "safe", &cliOpts.SafeSet)
	detectBoolFlagSet(fs, "aggressive", &cliOpts.AggressiveSet)
	detectBoolFlagSet(fs, "dry-run", &cliOpts.DryRunSet)
	if cmdName == "profile" {
		profileFlags.IncludeRoutes = parseCSV(includeCSV)
		profileFlags.IgnoreRoutes = parseCSV(ignoreCSV)
	}

	effective, err := config.Resolve(cliOpts)
	rt := app.NewRuntime(cmdName, effective)
	projectRoot, _ := os.Getwd()
	rt.Report.ProjectRoot = projectRoot
	if err != nil {
		rt.AddStep("initialization", "failed", err.Error())
		if werr := report.Write(cliOpts.ReportPath, *rt.Report); werr != nil {
			return fmt.Errorf("write report failed: %w", werr)
		}
		return err
	}

	rt.AddStep("initialization", "completed", "configuration resolved")
	switch cmdName {
	case "configure":
		err = modepkg.RunConfigure(rt)
	case "profile":
		err = modepkg.RunProfile(rt, profileFlags)
	case "cleanup":
		err = modepkg.RunCleanup(rt, cleanupFlags)
	default:
		err = fmt.Errorf("unsupported mode %q", cmdName)
	}

	if err != nil {
		rt.Report.Errors = append(rt.Report.Errors, err.Error())
	}
	if werr := report.Write(cliOpts.ReportPath, *rt.Report); werr != nil {
		return fmt.Errorf("write report failed: %w", werr)
	}
	if err != nil {
		return err
	}
	fmt.Printf("%s completed. Report written to %s\n", cmdName, cliOpts.ReportPath)
	return nil
}

func detectBoolFlagSet(fs *flag.FlagSet, name string, target *bool) {
	*target = false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			*target = true
		}
	})
}

func printRootHelp() {
	fmt.Fprintln(os.Stdout, rootUsage())
}

func rootUsage() string {
	return strings.TrimSpace(`
Cool Code Cleanup (ccc)

Usage:
  ccc <command> [flags]

Commands:
  configure   Configure project-local settings
  profile     Profile API routes and propose cleanup
  cleanup     Analyze code and apply cleanup options
  help        Show this help

Run "ccc <command> --help" for command options.
`)
}

func commandUsage(mode string) string {
	headline := map[string]string{
		"configure": "Configure project-local settings",
		"profile":   "Profile API routes and propose cleanup",
		"cleanup":   "Analyze code and apply cleanup options",
	}

	base := `
%s

Usage:
  ccc %s [flags]

Global Flags:
  --config <path>            Path to config file (default .ccc/config.json)
  --safe                     Enable safe mode (default true)
  --aggressive               Enable aggressive mode (default false)
  --dry-run                  Plan changes without writing files
  --non-interactive          Disable prompts and interactive UI
  --report-path <path>       Path to write JSON report
`
	var extra string
	switch mode {
	case "profile":
		extra = `
Profile Flags:
  --include-routes <csv>     (profile) include routes to profile
  --ignore-routes <csv>      (profile) ignore routes from profiling
  --dependency-short-circuit Enable short-circuit enhancement
  --edit-permission-mode     Edit permission mode (per-edit|per-file)
  --auto-apply               Apply edits without prompts where allowed
`
	case "cleanup":
		extra = `
Cleanup Flags:
  --rules <path>             Base cleanup rules file path
  --rules-local <path>       Optional local cleanup rules override path
  --enable-rule <id>         Enable rule by id (repeatable)
  --disable-rule <id>        Disable rule by id (repeatable)
  --edit-permission-mode     Edit permission mode (per-edit|per-file)
  --auto-apply               Apply edits without prompts where allowed
`
	case "configure":
		extra = `
Configure Notes:
  Interactive prompts will write to project-local .ccc/config.json.
`
	}
	return fmt.Sprintf(strings.TrimSpace(base+extra+`
  --help                     Show help
`), headline[mode], mode)
}

func parseCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
