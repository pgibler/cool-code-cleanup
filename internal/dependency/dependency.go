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

	if len(routes) == 0 || fallback == nil {
		return g, nil
	}

	fg, err := fallback.Infer(routes)
	if err != nil {
		return g, err
	}

	before := dependencyEdgeCount(g.Dependencies)
	mergeDependencies(g.Dependencies, fg.Dependencies)
	added := dependencyEdgeCount(g.Dependencies) - before
	if added == 0 {
		return g, nil
	}

	rationale := strings.TrimSpace(fg.Rationale)
	if rationale == "" {
		rationale = "ai inference"
	}
	if before > 0 {
		g.Confidence = "medium"
		g.Rationale = "deterministic heuristics + " + rationale
		return g, nil
	}
	g.Confidence = "low"
	g.Rationale = rationale
	return g, nil
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

func mergeDependencies(base map[string][]string, overlay map[string][]string) {
	for routeID, deps := range overlay {
		for _, dep := range deps {
			base[routeID] = appendIfMissing(base[routeID], dep)
		}
	}
}

func dependencyEdgeCount(deps map[string][]string) int {
	n := 0
	for _, items := range deps {
		n += len(items)
	}
	return n
}
