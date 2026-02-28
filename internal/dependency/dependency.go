package dependency

import (
	"slices"
	"strings"

	"cool-code-cleanup/internal/discovery"
)

type Graph struct {
	Dependencies map[string][]string `json:"dependencies"`
	Confidence   string              `json:"confidence"`
	Rationale    string              `json:"rationale"`
}

type Fallback interface {
	Infer(routes []discovery.Route) (Graph, error)
}

func Detect(routes []discovery.Route, fallback Fallback) (Graph, error) {
	g := Graph{
		Dependencies: map[string][]string{},
		Confidence:   "high",
		Rationale:    "deterministic heuristics",
	}

	authRoutes := findAuthRoutes(routes)
	for _, r := range routes {
		if requiresAuth(r) {
			for _, auth := range authRoutes {
				g.Dependencies[r.ID] = appendIfMissing(g.Dependencies[r.ID], auth.ID)
			}
		}
	}

	if len(g.Dependencies) > 0 || len(routes) == 0 || fallback == nil {
		return g, nil
	}

	fg, err := fallback.Infer(routes)
	if err != nil {
		return g, nil
	}
	fg.Confidence = "low"
	if fg.Rationale == "" {
		fg.Rationale = "ai fallback inference"
	}
	return fg, nil
}

func findAuthRoutes(routes []discovery.Route) []discovery.Route {
	var auth []discovery.Route
	for _, r := range routes {
		p := strings.ToLower(r.Path)
		if strings.Contains(p, "login") || strings.Contains(p, "auth") || strings.Contains(p, "token") {
			auth = append(auth, r)
		}
	}
	return auth
}

func requiresAuth(r discovery.Route) bool {
	p := strings.ToLower(r.Path)
	if strings.Contains(p, "private") || strings.Contains(p, "secure") || strings.Contains(p, "account") || strings.Contains(p, "payment") {
		return true
	}
	for _, m := range r.Middleware {
		l := strings.ToLower(m)
		if strings.Contains(l, "auth") || strings.Contains(l, "jwt") {
			return true
		}
	}
	return false
}

func appendIfMissing(list []string, item string) []string {
	if !slices.Contains(list, item) {
		return append(list, item)
	}
	return list
}
