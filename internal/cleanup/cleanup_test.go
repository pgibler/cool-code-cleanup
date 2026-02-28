package cleanup

import (
	"os"
	"path/filepath"
	"testing"

	"cool-code-cleanup/internal/config"
)

func TestBuildPlanSafeVsAggressive(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.go")
	content := "package main\n\nfunc x(){\n\tif true {\n\t\tprintln(\"ok\")\n\t}\n}\n"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	opts := config.DefaultConfig().Cleanup
	opts.RemoveRedundantGuards = true

	safePlan, err := BuildPlan(dir, opts, true, false)
	if err != nil {
		t.Fatalf("safe plan: %v", err)
	}
	aggressivePlan, err := BuildPlan(dir, opts, false, true)
	if err != nil {
		t.Fatalf("aggressive plan: %v", err)
	}
	if len(aggressivePlan.Edits) <= len(safePlan.Edits) {
		t.Fatalf("expected aggressive plan to include more edits; safe=%d aggressive=%d", len(safePlan.Edits), len(aggressivePlan.Edits))
	}
}
