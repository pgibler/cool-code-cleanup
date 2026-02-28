package ai

import (
	"cool-code-cleanup/internal/dependency"
	"cool-code-cleanup/internal/discovery"
)

type NopFallback struct{}

func (NopFallback) Infer(_ []discovery.Route) (dependency.Graph, error) {
	return dependency.Graph{
		Dependencies: map[string][]string{},
		Confidence:   "low",
		Rationale:    "fallback unavailable; no inferred dependencies",
	}, nil
}
