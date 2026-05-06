package test

import (
	"biogo/v2/simulation"
	"testing"
)

func TestIsActionEnabled(t *testing.T) {
	// All actions below ACTION_COUNT should be enabled
	for a := byte(0); a < simulation.ACTION_COUNT; a++ {
		if !simulation.IsActionEnabled(a) {
			t.Errorf("action %d should be enabled", a)
		}
	}
	// Actions at or above ACTION_COUNT should be disabled
	disabled := []byte{simulation.REPRODUCE, simulation.EAT}
	for _, a := range disabled {
		if simulation.IsActionEnabled(a) {
			t.Errorf("action %d should be disabled", a)
		}
	}
}
