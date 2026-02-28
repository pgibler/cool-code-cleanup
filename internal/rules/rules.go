package rules

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const SchemaVersion = 1

type Rule struct {
	ID          string `json:"id"`
	Enabled     bool   `json:"enabled"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Details     string `json:"details"`
	RiskLevel   string `json:"risk_level,omitempty"`
}

type File struct {
	SchemaVersion int    `json:"schema_version"`
	Rules         []Rule `json:"rules"`
}

type LoadedRule struct {
	Rule
	SourceChain []string `json:"source_chain"`
}

func DefaultRules() File {
	return File{
		SchemaVersion: SchemaVersion,
		Rules: []Rule{
			{ID: "remove_redundant_guards", Enabled: true, Title: "Remove redundant guards", Description: "Remove redundant guard conditions that do not affect behavior.", Details: "Find branches such as always-true guards or duplicate checks that can be simplified safely.", RiskLevel: "safe"},
			{ID: "refactor_dry", Enabled: true, Title: "Refactor code to follow DRY principles", Description: "Reduce duplicated logic by consolidating repeated patterns.", Details: "Look for repeated code blocks that can be extracted into shared helpers without changing functionality.", RiskLevel: "safe"},
			{ID: "harden_error_handling", Enabled: true, Title: "Harden code with better error handling", Description: "Improve reliability through explicit error handling and propagation.", Details: "Identify ignored errors, weak error context, and missing failure paths; strengthen handling while preserving behavior.", RiskLevel: "safe"},
			{ID: "gate_features_env", Enabled: false, Title: "Gate features behind environment variables", Description: "Allow behavior to be controlled by environment flags.", Details: "Where appropriate, introduce env-guarded behavior for risky or optional features.", RiskLevel: "aggressive"},
			{ID: "split_functions", Enabled: false, Title: "Split up functions into reusable pieces", Description: "Break large functions into smaller, reusable units.", Details: "Identify long or multi-responsibility functions and extract cohesive sub-functions.", RiskLevel: "aggressive"},
			{ID: "standardize_naming", Enabled: true, Title: "Standardize inconsistent naming styles", Description: "Normalize inconsistent variable, function, and type naming.", Details: "Apply a consistent naming style within files/modules while preserving public API expectations.", RiskLevel: "safe"},
			{ID: "simplify_complex_logic", Enabled: true, Title: "Simplify complex logic while retaining functionality", Description: "Reduce complexity in branching and control flow.", Details: "Refactor overly complex logic into clearer structures and helper functions when needed.", RiskLevel: "safe"},
			{ID: "detect_expensive_functions", Enabled: true, Title: "Detect expensive functions and offer ideas to improve performance", Description: "Identify expensive code paths and suggest improvements.", Details: "Look for nested loops, repeated heavy operations, and hot paths; provide optimization suggestions.", RiskLevel: "safe"},
		},
	}
}

func EnsureBaseFile(path string) error {
	clean := filepath.Clean(path)
	if _, err := os.Stat(clean); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat rules file %s: %w", clean, err)
	}
	if err := os.MkdirAll(filepath.Dir(clean), 0o755); err != nil {
		return fmt.Errorf("create rules dir: %w", err)
	}
	out, err := json.MarshalIndent(DefaultRules(), "", "  ")
	if err != nil {
		return fmt.Errorf("encode default rules: %w", err)
	}
	return os.WriteFile(clean, append(out, '\n'), 0o644)
}

func Load(basePath, localPath string) ([]LoadedRule, []string, error) {
	base, err := loadFile(basePath)
	if err != nil {
		return nil, nil, err
	}
	local, localExists, err := loadOptionalFile(localPath)
	if err != nil {
		return nil, nil, err
	}

	merged := map[string]LoadedRule{}
	var order []string
	var warnings []string

	for _, r := range base.Rules {
		if warn := validateRule(r); warn != "" {
			warnings = append(warnings, warn)
			continue
		}
		if _, seen := merged[r.ID]; seen {
			warnings = append(warnings, "duplicate base rule id: "+r.ID)
			continue
		}
		merged[r.ID] = LoadedRule{Rule: r, SourceChain: []string{"base"}}
		order = append(order, r.ID)
	}
	if localExists {
		for _, r := range local.Rules {
			if warn := validateRule(r); warn != "" {
				warnings = append(warnings, warn)
				continue
			}
			if existing, ok := merged[r.ID]; ok {
				existing.Rule = mergeRule(existing.Rule, r)
				if !slices.Contains(existing.SourceChain, "local") {
					existing.SourceChain = append(existing.SourceChain, "local")
				}
				merged[r.ID] = existing
				continue
			}
			merged[r.ID] = LoadedRule{Rule: r, SourceChain: []string{"local"}}
			order = append(order, r.ID)
		}
	}

	var out []LoadedRule
	for _, id := range order {
		out = append(out, merged[id])
	}
	return out, warnings, nil
}

func ApplyCLIOverrides(rules []LoadedRule, enableIDs, disableIDs []string) []LoadedRule {
	enable := normalizeSet(enableIDs)
	disable := normalizeSet(disableIDs)
	for i := range rules {
		id := strings.ToLower(strings.TrimSpace(rules[i].ID))
		if _, ok := enable[id]; ok {
			rules[i].Enabled = true
			rules[i].SourceChain = appendIfMissing(rules[i].SourceChain, "cli")
		}
		if _, ok := disable[id]; ok {
			rules[i].Enabled = false
			rules[i].SourceChain = appendIfMissing(rules[i].SourceChain, "cli")
		}
	}
	return rules
}

func loadFile(path string) (File, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return File{}, fmt.Errorf("read rules file %s: %w", path, err)
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return File{}, fmt.Errorf("parse rules file %s: %w", path, err)
	}
	if f.SchemaVersion != SchemaVersion {
		return File{}, fmt.Errorf("unsupported rules schema_version=%d (expected %d)", f.SchemaVersion, SchemaVersion)
	}
	return f, nil
}

func loadOptionalFile(path string) (File, bool, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return File{}, false, nil
		}
		return File{}, false, fmt.Errorf("read local rules file %s: %w", path, err)
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return File{}, false, fmt.Errorf("parse local rules file %s: %w", path, err)
	}
	if f.SchemaVersion != SchemaVersion {
		return File{}, false, fmt.Errorf("unsupported local rules schema_version=%d (expected %d)", f.SchemaVersion, SchemaVersion)
	}
	return f, true, nil
}

func mergeRule(base Rule, override Rule) Rule {
	out := base
	if strings.TrimSpace(override.Title) != "" {
		out.Title = override.Title
	}
	if strings.TrimSpace(override.Description) != "" {
		out.Description = override.Description
	}
	if strings.TrimSpace(override.Details) != "" {
		out.Details = override.Details
	}
	if strings.TrimSpace(override.RiskLevel) != "" {
		out.RiskLevel = override.RiskLevel
	}
	out.Enabled = override.Enabled
	return out
}

func validateRule(r Rule) string {
	if strings.TrimSpace(r.ID) == "" {
		return "rule missing id"
	}
	if strings.TrimSpace(r.Title) == "" {
		return "rule " + r.ID + " missing title"
	}
	if strings.TrimSpace(r.Description) == "" {
		return "rule " + r.ID + " missing description"
	}
	if strings.TrimSpace(r.Details) == "" {
		return "rule " + r.ID + " missing details"
	}
	if r.RiskLevel != "" && r.RiskLevel != "safe" && r.RiskLevel != "aggressive" {
		return "rule " + r.ID + " has invalid risk_level"
	}
	return ""
}

func normalizeSet(items []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, item := range items {
		item = strings.ToLower(strings.TrimSpace(item))
		if item != "" {
			out[item] = struct{}{}
		}
	}
	return out
}

func appendIfMissing(list []string, value string) []string {
	if !slices.Contains(list, value) {
		return append(list, value)
	}
	return list
}
