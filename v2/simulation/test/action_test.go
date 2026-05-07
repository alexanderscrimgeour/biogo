package test

import (
	"biogo/v2/simulation"
	"testing"
)

func TestIsActionEnabled(t *testing.T) {
	for a := byte(0); a < simulation.ACTION_COUNT; a++ {
		if !simulation.IsActionEnabled(a) {
			t.Errorf("action %d should be enabled", a)
		}
	}
	// Anything at or above ACTION_COUNT should be disabled.
	if simulation.IsActionEnabled(simulation.ACTION_COUNT) {
		t.Errorf("action %d (ACTION_COUNT) should be disabled", simulation.ACTION_COUNT)
	}
}
