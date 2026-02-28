package discovery

import (
	"path/filepath"
	"testing"
)

func TestDiscoverNodeGoDjango(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "testdata"))
	routes, err := Discover(root)
	if err != nil {
		t.Fatalf("discover failed: %v", err)
	}
	if len(routes) < 6 {
		t.Fatalf("expected at least 6 routes across fixtures, got %d", len(routes))
	}
}
