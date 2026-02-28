package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Step struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	StartedAt string `json:"started_at,omitempty"`
	EndedAt   string `json:"ended_at,omitempty"`
}

type RunReport struct {
	RunID           string              `json:"run_id"`
	TimestampUTC    string              `json:"timestamp_utc"`
	Mode            string              `json:"mode"`
	ProjectRoot     string              `json:"project_root"`
	ReportPath      string              `json:"report_path"`
	EffectiveConfig any                 `json:"effective_settings,omitempty"`
	SourceChains    map[string][]string `json:"source_chains,omitempty"`
	Steps           []Step              `json:"steps"`
	Routes          any                 `json:"routes,omitempty"`
	Rules           any                 `json:"rules,omitempty"`
	ProfilingRuns   []any               `json:"profiling_runs,omitempty"`
	CleanupPlan     []any               `json:"cleanup_plan,omitempty"`
	AppliedChanges  []any               `json:"applied_changes,omitempty"`
	Git             any                 `json:"git,omitempty"`
	Warnings        []string            `json:"warnings,omitempty"`
	Errors          []string            `json:"errors,omitempty"`
}

func DefaultReportPath(now time.Time) string {
	ts := now.UTC().Format("20060102T150405Z")
	return filepath.Join(".ccc", "reports", ts+".json")
}

func Write(path string, r RunReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create report file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return fmt.Errorf("encode report: %w", err)
	}
	return nil
}
