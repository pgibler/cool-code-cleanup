package mode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cool-code-cleanup/internal/app"
	"cool-code-cleanup/internal/config"
)

func TestRunProfileNonInteractive(t *testing.T) {
	dir := makeTempFixture(t, filepath.Join("..", "testdata", "go_app"))
	withCWD(t, dir, func() {
		eff, err := config.Resolve(config.CLIOverrides{
			ConfigPath:     filepath.Join(dir, ".ccc", "config.json"),
			ReportPath:     filepath.Join(dir, ".ccc", "reports", "test.json"),
			NonInteractive: true,
			SafeSet:        true,
			Safe:           true,
			DryRunSet:      true,
			DryRun:         true,
		})
		if err != nil {
			t.Fatalf("resolve failed: %v", err)
		}
		rt := app.NewRuntime("profile", eff)
		if err := RunProfile(rt, ProfileFlags{DependencyShortCircuit: true}); err != nil {
			t.Fatalf("profile failed: %v", err)
		}
		if len(rt.Report.Steps) == 0 {
			t.Fatalf("expected steps in report")
		}
	})
}

func TestRunCleanupNonInteractive(t *testing.T) {
	dir := makeTempFixture(t, filepath.Join("..", "testdata", "node_app"))
	withCWD(t, dir, func() {
		eff, err := config.Resolve(config.CLIOverrides{
			ConfigPath:     filepath.Join(dir, ".ccc", "config.json"),
			ReportPath:     filepath.Join(dir, ".ccc", "reports", "test.json"),
			NonInteractive: true,
			SafeSet:        true,
			Safe:           true,
			DryRunSet:      true,
			DryRun:         true,
		})
		if err != nil {
			t.Fatalf("resolve failed: %v", err)
		}
		rt := app.NewRuntime("cleanup", eff)
		if err := RunCleanup(rt, CleanupFlags{}); err != nil {
			t.Fatalf("cleanup failed: %v", err)
		}
		if len(rt.Report.Steps) == 0 {
			t.Fatalf("expected steps in report")
		}
		for _, step := range rt.Report.Steps {
			name := strings.ToLower(step.Name)
			if strings.Contains(name, "step_1b_short_circuit") || strings.Contains(name, "dependency_detection") || strings.Contains(name, "dependency_confirmation") {
				t.Fatalf("cleanup mode should not run dependency/short-circuit flow step: %s", step.Name)
			}
		}
	})
}

func withCWD(t *testing.T, dir string, fn func()) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	defer func() { _ = os.Chdir(wd) }()
	fn()
}

func makeTempFixture(t *testing.T, src string) string {
	t.Helper()
	dst := t.TempDir()
	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
	return dst
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
