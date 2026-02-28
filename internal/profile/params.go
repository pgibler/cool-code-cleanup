package profile

import (
	"fmt"

	"cool-code-cleanup/internal/discovery"
)

type ParameterPlan struct {
	RouteID string              `json:"route_id"`
	Valid   []map[string]string `json:"valid"`
	Invalid []map[string]string `json:"invalid"`
}

func AnalyzeParameters(routes []discovery.Route) []ParameterPlan {
	plans := make([]ParameterPlan, 0, len(routes))
	for i, r := range routes {
		plans = append(plans, ParameterPlan{
			RouteID: r.ID,
			Valid: []map[string]string{
				{"example": "valid", "sequence": fmt.Sprintf("%d", i+1)},
			},
			Invalid: []map[string]string{
				{"example": "invalid", "sequence": "0"},
			},
		})
	}
	return plans
}
