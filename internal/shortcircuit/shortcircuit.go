package shortcircuit

import (
	"fmt"
	"os"
	"strings"

	"cool-code-cleanup/internal/discovery"
)

type PatchCandidate struct {
	RouteID     string `json:"route_id"`
	File        string `json:"file"`
	Description string `json:"description"`
	Applied     bool   `json:"applied"`
}

func Candidates(routes []discovery.Route, dependencies map[string][]string) []PatchCandidate {
	depSet := map[string]bool{}
	for _, reqs := range dependencies {
		for _, req := range reqs {
			depSet[req] = true
		}
	}
	var out []PatchCandidate
	for _, r := range routes {
		if !depSet[r.ID] {
			continue
		}
		path := strings.ToLower(r.Path)
		if strings.Contains(path, "auth") || strings.Contains(path, "payment") || strings.Contains(path, "otp") || strings.Contains(path, "email") || strings.Contains(path, "phone") {
			out = append(out, PatchCandidate{
				RouteID:     r.ID,
				File:        r.File,
				Description: fmt.Sprintf("Add short-circuit marker for %s %s", r.Method, r.Path),
			})
		}
	}
	return out
}

func Apply(candidates []PatchCandidate, envVar string, dryRun bool) ([]PatchCandidate, error) {
	out := make([]PatchCandidate, 0, len(candidates))
	for _, c := range candidates {
		data, err := os.ReadFile(c.File)
		if err != nil {
			return out, fmt.Errorf("read %s: %w", c.File, err)
		}
		content := string(data)
		marker := fmt.Sprintf("CCC short-circuit marker: set %s=true to bypass external dependencies", envVar)
		if strings.Contains(content, marker) {
			out = append(out, c)
			continue
		}
		next := "// " + marker + "\n" + content
		if !dryRun {
			if err := os.WriteFile(c.File, []byte(next), 0o644); err != nil {
				return out, fmt.Errorf("write %s: %w", c.File, err)
			}
		}
		c.Applied = true
		out = append(out, c)
	}
	return out, nil
}
