package cleanup

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cool-code-cleanup/internal/rules"
)

type fakeProjectExec struct{}

func (fakeProjectExec) TransformProject(_ context.Context, _ string, files []ProjectFile, task Task, _ []rules.Rule, _ bool, _ bool) (ProjectTransformResult, error) {
	changed := map[string]string{}
	if task.RuleID != "remove_redundant_guards" {
		return ProjectTransformResult{Changed: false, ChangedFiles: changed}, nil
	}
	for _, f := range files {
		next := f.Content
		next = strings.ReplaceAll(next, "if true {", "{")
		if next != f.Content {
			changed[f.Path] = next
		}
	}
	return ProjectTransformResult{
		Changed:      len(changed) > 0,
		Summary:      "removed redundant guards",
		ChangedFiles: changed,
	}, nil
}

func TestProjectWidePhasesAndExecution(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.go")
	content := "package main\nfunc x(){ if true { println(\"ok\") } }\n"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	snapshot, err := BuildProjectSnapshot(dir)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if len(snapshot) == 0 {
		t.Fatalf("expected snapshot files")
	}

	selected := []rules.Rule{
		{
			ID:          "remove_redundant_guards",
			Enabled:     true,
			Title:       "Remove redundant guards",
			Description: "Remove redundant guard conditions.",
			Details:     "Simplify always true guards.",
		},
	}
	tasks := BuildTaskPlan(snapshot, selected)
	if len(tasks) == 0 {
		t.Fatalf("expected task plan")
	}

	plan, applied, taskResults, err := ExecuteTaskPlan(dir, snapshot, tasks, selected, false, true, false, fakeProjectExec{}, nil)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(plan.Edits) == 0 || len(applied) == 0 || len(taskResults) == 0 {
		t.Fatalf("expected edits and task results")
	}
}
