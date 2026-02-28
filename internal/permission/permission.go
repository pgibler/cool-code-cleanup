package permission

import (
	"fmt"
	"strings"

	"cool-code-cleanup/internal/tui"
)

type Engine struct {
	Mode           string
	AutoApply      bool
	NonInteractive bool
}

func (e Engine) ApproveFile(io tui.IO, file string, changes int) (bool, error) {
	if e.AutoApply || e.NonInteractive {
		return true, nil
	}
	if e.Mode == "per-edit" {
		return true, nil
	}
	resp, err := io.Prompt(fmt.Sprintf("Approve file changes for %s (%d edits)? [y/N]: ", file, changes))
	if err != nil {
		return false, err
	}
	return isYes(resp), nil
}

func (e Engine) ApproveEdit(io tui.IO, file, desc string) (bool, error) {
	if e.AutoApply || e.NonInteractive {
		return true, nil
	}
	if e.Mode != "per-edit" {
		return true, nil
	}
	resp, err := io.Prompt(fmt.Sprintf("Approve edit in %s: %s [y/N]: ", file, desc))
	if err != nil {
		return false, err
	}
	return isYes(resp), nil
}

func isYes(s string) bool {
	v := strings.ToLower(strings.TrimSpace(s))
	return v == "y" || v == "yes"
}
