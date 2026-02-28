package cleanup

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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

func BuildPlan(projectRoot string, selected []rules.Rule, safe, aggressive bool) (Plan, error) {
	cap := capabilitiesFromRules(selected)
	var plan Plan
	err := filepath.WalkDir(projectRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == ".ccc" || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".go" && ext != ".js" && ext != ".ts" && ext != ".py" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)
		normalized := normalizeWhitespace(content, cap.refactorDRY || cap.simplifyComplexLogic)
		if normalized != content {
			plan.Edits = append(plan.Edits, Edit{
				File:        path,
				Description: "Normalize whitespace and collapse excessive blank lines",
				Before:      "text formatting",
				After:       "normalized",
				Applied:     false,
			})
		}

		if cap.removeRedundantGuards && aggressive && !safe {
			redundant := regexp.MustCompile(`(?m)^\s*if\s+(true|\(true\))\s*\{`)
			if redundant.MatchString(content) {
				plan.Edits = append(plan.Edits, Edit{
					File:        path,
					Description: "Remove redundant always-true guard conditions",
					Before:      "if true",
					After:       "bare block",
					Applied:     false,
				})
			}
		}

		if cap.detectExpensiveFunctions {
			if strings.Count(content, "for ") > 2 && strings.Count(content, "for ") != strings.Count(content, "\nfor ") {
				plan.Edits = append(plan.Edits, Edit{
					File:        path,
					Description: "Potential expensive nested loops detected (analysis suggestion)",
					Applied:     false,
				})
			}
		}
		return nil
	})
	return plan, err
}

func ApplyPlan(plan Plan, safe, aggressive, dryRun bool) ([]Edit, error) {
	applied := make([]Edit, 0, len(plan.Edits))
	for _, edit := range plan.Edits {
		if strings.Contains(strings.ToLower(edit.Description), "analysis suggestion") {
			applied = append(applied, edit)
			continue
		}
		data, err := os.ReadFile(edit.File)
		if err != nil {
			return applied, err
		}
		orig := string(data)
		next := orig
		switch edit.Description {
		case "Normalize whitespace and collapse excessive blank lines":
			next = normalizeWhitespace(orig, true)
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

func normalizeWhitespace(content string, enabled bool) string {
	if !enabled {
		return content
	}
	sc := bufio.NewScanner(strings.NewReader(content))
	var out []string
	blankCount := 0
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), " \t")
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
	return strings.Join(out, "\n") + "\n"
}

func DescribePlan(plan Plan) []string {
	if len(plan.Edits) == 0 {
		return []string{"No cleanup edits detected."}
	}
	lines := make([]string, 0, len(plan.Edits))
	for _, e := range plan.Edits {
		lines = append(lines, fmt.Sprintf("%s: %s", e.File, e.Description))
	}
	return lines
}
