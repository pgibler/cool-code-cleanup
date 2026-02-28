package app

import (
	"time"

	"cool-code-cleanup/internal/config"
	"cool-code-cleanup/internal/report"
)

type Runtime struct {
	Mode      string
	Effective config.Effective
	Report    *report.RunReport
	StartTime time.Time
}

func NewRuntime(mode string, eff config.Effective) *Runtime {
	now := time.Now().UTC()
	return &Runtime{
		Mode:      mode,
		Effective: eff,
		StartTime: now,
		Report: &report.RunReport{
			RunID:           now.Format("20060102T150405.000000000"),
			TimestampUTC:    now.Format(time.RFC3339),
			Mode:            mode,
			ReportPath:      eff.ReportPath,
			EffectiveConfig: eff.Config,
			SourceChains:    eff.SourceChains,
			Steps:           []report.Step{},
		},
	}
}

func (r *Runtime) AddStep(name, status, message string) {
	now := time.Now().UTC().Format(time.RFC3339)
	r.Report.Steps = append(r.Report.Steps, report.Step{
		Name:      name,
		Status:    status,
		Message:   message,
		StartedAt: now,
		EndedAt:   now,
	})
}
