package dependency

import (
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
