package dependency

import (
	"fmt"
	"testing"

	"cool-code-cleanup/internal/discovery"
)

func TestDetectAuthDependencyDeterministic(t *testing.T) {
	routes := []discovery.Route{
		{ID: "r1", Method: "POST", Path: "/auth/login"},
		{ID: "r2", Method: "GET", Path: "/account/private"},
	}
	g, err := Detect(routes, nil)
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if len(g.Dependencies["r2"]) == 0 || g.Dependencies["r2"][0] != "r1" {
		t.Fatalf("expected r2 to depend on r1, got %+v", g.Dependencies)
	}
}

func TestDetectMergesAIDependencies(t *testing.T) {
	routes := []discovery.Route{
		{ID: "r1", Method: "POST", Path: "/auth/login"},
		{ID: "r2", Method: "GET", Path: "/account/private"},
		{ID: "r3", Method: "GET", Path: "/payments/private"},
	}
	g, err := Detect(routes, fakeFallback{
		graph: Graph{
			Dependencies: map[string][]string{
				"r3": {"r2"},
			},
			Rationale: "ai inferred auth bootstrap",
		},
	})
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if len(g.Dependencies["r2"]) == 0 || g.Dependencies["r2"][0] != "r1" {
		t.Fatalf("expected deterministic dep r2->r1, got %+v", g.Dependencies)
	}
	if len(g.Dependencies["r3"]) < 2 {
		t.Fatalf("expected merged deps for r3, got %+v", g.Dependencies["r3"])
	}
}

func TestDetectReturnsDeterministicGraphOnFallbackError(t *testing.T) {
	routes := []discovery.Route{
		{ID: "r1", Method: "POST", Path: "/auth/login"},
		{ID: "r2", Method: "GET", Path: "/account/private"},
	}
	g, err := Detect(routes, fakeFallback{err: fmt.Errorf("timeout")})
	if err == nil {
		t.Fatalf("expected fallback error")
	}
	if len(g.Dependencies["r2"]) == 0 || g.Dependencies["r2"][0] != "r1" {
		t.Fatalf("expected deterministic deps preserved, got %+v", g.Dependencies)
	}
}

type fakeFallback struct {
	graph Graph
	err   error
}

func (f fakeFallback) Infer(_ []discovery.Route) (Graph, error) {
	if f.err != nil {
		return Graph{}, f.err
	}
	return f.graph, nil
}
