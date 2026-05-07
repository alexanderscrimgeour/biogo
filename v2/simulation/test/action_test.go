package test

import (
	"biogo/v2/simulation"
	"testing"
)

func TestDoNothingRefundsMetabolicCost(t *testing.T) {
	p := defaultParams()
	p.MinMetabolicRate = 5
	p.MaxMetabolicRate = 5
	p.MoveCost = 0
	p.MinPopulation = 0
	p.MaxFood = 0
	p.StartingPopulation = 1

	sim := simulation.New(p)

	var c *simulation.Creature
	for _, v := range sim.Population.Creatures {
		c = v
		break
	}
	c.Genome.MetabolicRate = 127 // mid-range, maps to exactly 5.0 with min==max==5
	c.Energy = float32(c.Genome.MaxEnergy)
	energyBefore := c.Energy

	// Force DO_NOTHING to fire by wiring a sensor directly to it in the neural net.
	doNothingGene := &simulation.Gene{
		SourceType: simulation.SENSOR,
		SourceID:   simulation.ENERGY, // returns ~1.0 at full energy
		SinkType:   simulation.ACTION,
		SinkID:     simulation.DO_NOTHING,
		Weight:     255, // max positive weight → level ≈ 1.0, prob2Bool always fires
	}
	c.Genome.Brain = []*simulation.Gene{doNothingGene}
	c.Genome.BrainLength = 1
	c.CreateNeuralNet()

	sim.Update()

	// After one tick with DO_NOTHING firing, energy should be unchanged (metabolic cost refunded).
	if c.Energy != energyBefore {
		t.Errorf("DO_NOTHING should conserve energy: before=%f after=%f", energyBefore, c.Energy)
	}
}

func TestDoNothingIsEnabled(t *testing.T) {
	if !simulation.IsActionEnabled(simulation.DO_NOTHING) {
		t.Errorf("DO_NOTHING (action %d) should be enabled", simulation.DO_NOTHING)
	}
}

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
