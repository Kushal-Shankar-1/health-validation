package catalog_test

import (
	"slices"
	"testing"

	"github.com/dsx-ai-factory/health-validation/catalog"
	"github.com/dsx-ai-factory/health-validation/catalog/health"
)

// cadenceCheck builds a minimal registrable check with the given id and cadence.
func cadenceCheck(id string, cadence health.SchedulingCadence) health.CheckDefinition {
	return health.CheckDefinition{
		ID:                id,
		Category:          health.Node,
		SchedulingCadence: cadence,
		RequiredSignals:   []string{"x"},
		Thresholds:        map[string]health.ThresholdSpec{},
		Rules:             []health.Rule{{Expression: "x > 0", Severity: health.Critical, Message: "x"}},
	}
}

func checkIDs(defs []*health.CheckDefinition) []string {
	out := make([]string, len(defs))
	for i, d := range defs {
		out[i] = d.ID
	}
	return out
}

// Filtering by cadence returns every check whose declared cadence is at least
// as frequent as the requested one, because cadences nest
// (Continuous > Prolog > MaintenanceWindow).
func TestListFiltersByCadenceHierarchy(t *testing.T) {
	cat := catalog.NewEmpty()
	cat.MustRegister(cadenceCheck("cont", health.Continuous))
	cat.MustRegister(cadenceCheck("prolog", health.Prolog))
	cat.MustRegister(cadenceCheck("maint", health.MaintenanceWindow))

	tests := []struct {
		name    string
		want    []string // sorted: List sorts by ID
		cadence health.SchedulingCadence
	}{
		{name: "continuous returns only continuous", cadence: health.Continuous, want: []string{"cont"}},
		{name: "prolog returns continuous and prolog", cadence: health.Prolog, want: []string{"cont", "prolog"}},
		{name: "maintenance returns every check", cadence: health.MaintenanceWindow, want: []string{"cont", "maint", "prolog"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cadence := tt.cadence
			got := checkIDs(cat.List(catalog.Filter{Cadence: &cadence}))
			if !slices.Equal(got, tt.want) {
				t.Errorf("List(cadence=%v) = %v, want %v", tt.cadence, got, tt.want)
			}
		})
	}

	// A nil cadence filter is ignored: every check is returned.
	if got := len(cat.List(catalog.Filter{})); got != 3 {
		t.Errorf("List(no filter) returned %d checks, want 3", got)
	}
}
