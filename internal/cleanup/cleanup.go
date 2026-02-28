package cleanup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"cool-code-cleanup/internal/rules"
)

type Edit struct {
	File        string `json:"file"`
	Description string `json:"description"`
	Before      string `json:"before,omitempty"`
	After       string `json:"after,omitempty"`
	Applied     bool   `json:"applied"`
}

type Plan struct {
	Edits []Edit `json:"edits"`
}

type ProjectFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type Task struct {
	ID          string   `json:"id"`
	RuleID      string   `json:"rule_id"`
	RuleTitle   string   `json:"rule_title"`
	Description string   `json:"description"`
	Files       []string `json:"files"`
}

type TaskResult struct {
	TaskID       string   `json:"task_id"`
	RuleID       string   `json:"rule_id"`
	ChangedFiles []string `json:"changed_files"`
	Applied      bool     `json:"applied"`
	Summary      string   `json:"summary"`
	Error        string   `json:"error,omitempty"`
}

type ProgressEvent struct {
	File        string
	RuleID      string
	RuleTitle   string
	Phase       string
	Description string
}

type ProjectTransformResult struct {
	Changed      bool              `json:"changed"`
	Summary      string            `json:"summary"`
	ChangedFiles map[string]string `json:"changed_files"`
}

type ProjectExecutor interface {
	TransformProject(ctx context.Context, projectRoot string, files []ProjectFile, task Task, selectedRules []rules.Rule, safe, aggressive bool) (ProjectTransformResult, error)
}

func BuildProjectSnapshot(projectRoot string) ([]ProjectFile, error) {
	var files []ProjectFile
	err := filepath.WalkDir(projectRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".ccc", "node_modules", "vendor", "dist", "build", "bin":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".go" && ext != ".js" && ext != ".ts" && ext != ".py" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files = append(files, ProjectFile{Path: path, Content: string(raw)})
		return nil
	})
	return files, err
}

func BuildTaskPlan(files []ProjectFile, selectedRules []rules.Rule) []Task {
	var tasks []Task
	for i, r := range selectedRules {
		targets := selectTaskFiles(files, r)
		if len(targets) == 0 {
			continue
		}
		tasks = append(tasks, Task{
			ID:          fmt.Sprintf("task-%03d-%s", i+1, sanitizeID(r.ID)),
			RuleID:      r.ID,
			RuleTitle:   r.Title,
			Description: r.Description,
			Files:       targets,
		})
	}
	return tasks
}

func ExecuteTaskPlan(projectRoot string, snapshot []ProjectFile, tasks []Task, selectedRules []rules.Rule, safe, aggressive, dryRun bool, executor ProjectExecutor, onProgress func(ProgressEvent)) (Plan, []Edit, []TaskResult, error) {
	if executor == nil {
		return Plan{}, nil, nil, fmt.Errorf("cleanup project executor is required")
	}

	current := map[string]string{}
	for _, f := range snapshot {
		current[f.Path] = f.Content
	}

	var plan Plan
	var applied []Edit
	var results []TaskResult

	for _, task := range tasks {
		taskFiles := filesForTask(snapshot, task, current)
		if onProgress != nil {
			onProgress(ProgressEvent{
				RuleID:      task.RuleID,
				RuleTitle:   task.RuleTitle,
				Phase:       "running",
				Description: fmt.Sprintf("executing task %s", task.ID),
			})
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		result, err := executor.TransformProject(ctx, projectRoot, taskFiles, task, selectedRules, safe, aggressive)
		cancel()
		if err != nil {
			results = append(results, TaskResult{
				TaskID:  task.ID,
				RuleID:  task.RuleID,
				Applied: false,
				Error:   err.Error(),
			})
			if onProgress != nil {
				onProgress(ProgressEvent{
					RuleID:      task.RuleID,
					RuleTitle:   task.RuleTitle,
					Phase:       "error",
					Description: err.Error(),
				})
			}
			return plan, applied, results, fmt.Errorf("cleanup task %s failed: %w", task.ID, err)
		}
		if !result.Changed || len(result.ChangedFiles) == 0 {
			results = append(results, TaskResult{
				TaskID:  task.ID,
				RuleID:  task.RuleID,
				Applied: false,
				Summary: "no changes",
			})
			if onProgress != nil {
				onProgress(ProgressEvent{
					RuleID:      task.RuleID,
					RuleTitle:   task.RuleTitle,
					Phase:       "no_change",
					Description: "no changes",
				})
			}
			continue
		}

		changedPaths := make([]string, 0, len(result.ChangedFiles))
		for path, next := range result.ChangedFiles {
			prev, ok := current[path]
			if !ok || next == prev {
				continue
			}
			current[path] = next
			changedPaths = append(changedPaths, path)
			edit := Edit{
				File:        path,
				Description: fmt.Sprintf("[%s] %s", task.RuleID, nonEmpty(result.Summary, "AI project cleanup change")),
				Before:      "project-level AI task",
				After:       "updated content",
				Applied:     !dryRun,
			}
			plan.Edits = append(plan.Edits, edit)
			applied = append(applied, edit)
			if onProgress != nil {
				onProgress(ProgressEvent{
					File:        path,
					RuleID:      task.RuleID,
					RuleTitle:   task.RuleTitle,
					Phase:       "changed",
					Description: edit.Description,
				})
			}
		}
		if len(changedPaths) == 0 {
			continue
		}
		results = append(results, TaskResult{
			TaskID:       task.ID,
			RuleID:       task.RuleID,
			ChangedFiles: changedPaths,
			Applied:      !dryRun,
			Summary:      nonEmpty(result.Summary, "task applied"),
		})
		if !dryRun {
			for _, path := range changedPaths {
				if err := os.WriteFile(path, []byte(current[path]), 0o644); err != nil {
					return plan, applied, results, err
				}
			}
		}
	}

	return plan, applied, results, nil
}

// BuildPlan is a compatibility planner used by profile mode's cleanup proposal step.
// Cleanup mode itself uses the project-wide task execution pipeline.
func BuildPlan(projectRoot string, selectedRules []rules.Rule, safe, aggressive bool) (Plan, error) {
	cap := capabilitiesFromRules(selectedRules)
	var plan Plan
	err := filepath.WalkDir(projectRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".ccc", "node_modules", "vendor", "dist", "build", "bin":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".go" && ext != ".js" && ext != ".ts" && ext != ".py" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(raw)
		if cap.refactorDRY || cap.simplifyComplexLogic {
			if normalizeWhitespace(content) != content {
				plan.Edits = append(plan.Edits, Edit{
					File:        path,
					Description: "Normalize whitespace and collapse excessive blank lines",
					Before:      "text formatting",
					After:       "normalized",
					Applied:     false,
				})
			}
		}
		if cap.removeRedundantGuards && aggressive && !safe {
			if regexp.MustCompile(`(?m)^\s*if\s+(true|\(true\))\s*\{`).MatchString(content) {
				plan.Edits = append(plan.Edits, Edit{
					File:        path,
					Description: "Remove redundant always-true guard conditions",
					Before:      "if true",
					After:       "bare block",
					Applied:     false,
				})
			}
		}
		if cap.detectExpensiveFunctions && strings.Count(content, "for ") > 2 {
			plan.Edits = append(plan.Edits, Edit{
				File:        path,
				Description: "Potential expensive nested loops detected (analysis suggestion)",
				Applied:     false,
			})
		}
		return nil
	})
	return plan, err
}

// ApplyPlan is a compatibility applier used by profile mode's cleanup proposal step.
func ApplyPlan(plan Plan, safe, aggressive, dryRun bool) ([]Edit, error) {
	applied := make([]Edit, 0, len(plan.Edits))
	for _, edit := range plan.Edits {
		if strings.Contains(strings.ToLower(edit.Description), "analysis suggestion") {
			applied = append(applied, edit)
			continue
		}
		raw, err := os.ReadFile(edit.File)
		if err != nil {
			return applied, err
		}
		orig := string(raw)
		next := orig
		switch edit.Description {
		case "Normalize whitespace and collapse excessive blank lines":
			next = normalizeWhitespace(orig)
		case "Remove redundant always-true guard conditions":
			if aggressive && !safe {
				next = regexp.MustCompile(`if\s+\(?true\)?\s*\{`).ReplaceAllString(orig, "{")
			}
		}
		if next != orig {
			edit.Applied = true
			if !dryRun {
				if err := os.WriteFile(edit.File, []byte(next), 0o644); err != nil {
					return applied, err
				}
			}
		}
		applied = append(applied, edit)
	}
	return applied, nil
}

func selectTaskFiles(files []ProjectFile, r rules.Rule) []string {
	text := strings.ToLower(r.ID + " " + r.Title + " " + r.Description + " " + r.Details)
	var out []string
	for _, f := range files {
		c := strings.ToLower(f.Content)
		include := false
		switch {
		case strings.Contains(text, "redundant guard"):
			include = strings.Contains(c, "if true") || strings.Contains(c, "if (true)")
		case strings.Contains(text, "dry"), strings.Contains(text, "duplicate"):
			include = strings.Count(c, "func ") > 1 || strings.Count(c, "function ") > 1
		case strings.Contains(text, "error handling"):
			include = strings.Contains(c, " err") || strings.Contains(c, "catch")
		case strings.Contains(text, "environment variable"), strings.Contains(text, "gate features"):
			include = strings.Contains(c, "os.Getenv") || strings.Contains(c, "process.env")
		case strings.Contains(text, "split") && strings.Contains(text, "function"):
			include = strings.Count(c, "\n") > 80
		case strings.Contains(text, "naming"):
			include = true
		case strings.Contains(text, "simplify complex"), strings.Contains(text, "reduce complexity"):
			include = strings.Count(c, "if ") > 3 || strings.Count(c, "switch ") > 0
		case strings.Contains(text, "expensive"), strings.Contains(text, "performance"), strings.Contains(text, "hot path"):
			include = strings.Count(c, "for ") > 1
		default:
			include = true
		}
		if include {
			out = append(out, f.Path)
		}
	}
	if len(out) == 0 {
		for _, f := range files {
			out = append(out, f.Path)
		}
	}
	return capTaskFiles(out, 24)
}

type capabilities struct {
	removeRedundantGuards    bool
	refactorDRY              bool
	hardenErrorHandling      bool
	gateFeaturesEnv          bool
	splitFunctions           bool
	standardizeNaming        bool
	simplifyComplexLogic     bool
	detectExpensiveFunctions bool
}

func capabilitiesFromRules(selected []rules.Rule) capabilities {
	var cap capabilities
	for _, r := range selected {
		text := strings.ToLower(r.ID + " " + r.Title + " " + r.Description + " " + r.Details)
		if strings.Contains(text, "redundant guard") {
			cap.removeRedundantGuards = true
		}
		if strings.Contains(text, "dry") || strings.Contains(text, "duplicate") {
			cap.refactorDRY = true
		}
		if strings.Contains(text, "error handling") {
			cap.hardenErrorHandling = true
		}
		if strings.Contains(text, "environment variable") || strings.Contains(text, "env-guard") || strings.Contains(text, "gate features") {
			cap.gateFeaturesEnv = true
		}
		if strings.Contains(text, "split") && strings.Contains(text, "function") {
			cap.splitFunctions = true
		}
		if strings.Contains(text, "naming") {
			cap.standardizeNaming = true
		}
		if strings.Contains(text, "simplify complex") || strings.Contains(text, "reduce complexity") {
			cap.simplifyComplexLogic = true
		}
		if strings.Contains(text, "expensive") || strings.Contains(text, "performance") || strings.Contains(text, "hot path") {
			cap.detectExpensiveFunctions = true
		}
	}
	return cap
}

func filesForTask(snapshot []ProjectFile, task Task, current map[string]string) []ProjectFile {
	taskSet := map[string]bool{}
	for _, p := range task.Files {
		taskSet[p] = true
	}
	var files []ProjectFile
	for _, f := range snapshot {
		if !taskSet[f.Path] {
			continue
		}
		files = append(files, ProjectFile{
			Path:    f.Path,
			Content: current[f.Path],
		})
	}
	return files
}

func capTaskFiles(files []string, max int) []string {
	if len(files) <= max {
		return files
	}
	return slices.Clone(files[:max])
}

func normalizeWhitespace(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	blankCount := 0
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == "" {
			blankCount++
			if blankCount > 1 {
				continue
			}
		} else {
			blankCount = 0
		}
		out = append(out, line)
	}
	next := strings.Join(out, "\n")
	if !strings.HasSuffix(next, "\n") {
		next += "\n"
	}
	return next
}

func sanitizeID(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "task"
	}
	return s
}

func nonEmpty(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}
