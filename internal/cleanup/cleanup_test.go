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

type flakyProjectExec struct{}

func (flakyProjectExec) TransformProject(_ context.Context, _ string, files []ProjectFile, task Task, _ []rules.Rule, _ bool, _ bool) (ProjectTransformResult, error) {
	if strings.Contains(task.RuleID, "fail") {
		return ProjectTransformResult{}, context.DeadlineExceeded
	}
	changed := map[string]string{}
	for _, f := range files {
		next := strings.ReplaceAll(f.Content, "if true {", "{")
		if next != f.Content {
			changed[f.Path] = next
		}
	}
	return ProjectTransformResult{
		Changed:      len(changed) > 0,
		Summary:      "changed",
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

func TestExecuteTaskPlanContinuesAfterTaskError(t *testing.T) {
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
	selected := []rules.Rule{
		{
			ID:          "fail_task",
			Enabled:     true,
			Title:       "Fail task",
			Description: "force failure",
			Details:     "force failure",
		},
		{
			ID:          "remove_redundant_guards",
			Enabled:     true,
			Title:       "Remove redundant guards",
			Description: "cleanup",
			Details:     "cleanup",
		},
	}
	tasks := BuildTaskPlan(snapshot, selected)
	plan, applied, results, err := ExecuteTaskPlan(dir, snapshot, tasks, selected, false, true, true, flakyProjectExec{}, nil)
	if err != nil {
		t.Fatalf("execute should continue after partial failures: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 task results, got %d", len(results))
	}
	if strings.TrimSpace(results[0].Error) == "" {
		t.Fatalf("expected first task to fail")
	}
	if len(plan.Edits) == 0 || len(applied) == 0 {
		t.Fatalf("expected successful task changes after failure")
	}
}
