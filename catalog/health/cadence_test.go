package health

import "testing"

// Continuous is the zero value, so a check that omits SchedulingCadence
// defaults to the most frequent cadence — it runs in every window, including a
// Continuous filter, rather than being silently excluded.
func TestSchedulingCadenceZeroValueIsContinuous(t *testing.T) {
	var zero SchedulingCadence
	if zero != Continuous {
		t.Fatalf("zero value = %v, want Continuous", zero)
	}
	if !zero.Covers(Continuous) {
		t.Errorf("zero-value cadence should cover Continuous")
	}
}

// Cadences nest by frequency: Continuous > Prolog > MaintenanceWindow. A check
// covers a requested cadence when its own cadence is at least as frequent.
func TestSchedulingCadenceCovers(t *testing.T) {
	tests := []struct {
		name      string
		check     SchedulingCadence
		requested SchedulingCadence
		want      bool
	}{
		{"continuous covers continuous", Continuous, Continuous, true},
		{"continuous covers prolog", Continuous, Prolog, true},
		{"continuous covers maintenance", Continuous, MaintenanceWindow, true},
		{"prolog does not cover continuous", Prolog, Continuous, false},
		{"prolog covers prolog", Prolog, Prolog, true},
		{"prolog covers maintenance", Prolog, MaintenanceWindow, true},
		{"maintenance does not cover continuous", MaintenanceWindow, Continuous, false},
		{"maintenance does not cover prolog", MaintenanceWindow, Prolog, false},
		{"maintenance covers maintenance", MaintenanceWindow, MaintenanceWindow, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.check.Covers(tt.requested); got != tt.want {
				t.Errorf("%v.Covers(%v) = %v, want %v", tt.check, tt.requested, got, tt.want)
			}
		})
	}
}
