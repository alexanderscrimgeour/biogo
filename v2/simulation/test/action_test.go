package test

import (
	"biogo/v2/grid"
	"biogo/v2/simulation"
	"math"
	"testing"
)

func TestDoNothingHalvesMetabolicCost(t *testing.T) {
	p := defaultParams()
	p.BaseBMR = 5
	p.MoveCost = 0
	p.MinPopulation = 0
	p.MaxPopulation = 1
	p.MaxFood = 0
	p.StartingPopulation = 1

	sim := simulation.New(p)

	var c *simulation.Creature
	for _, v := range sim.Population.Creatures {
		c = v
		break
	}
	c.Genome.MetabolicRate = 127
	c.Energy = c.Mass * p.EnergyPerMassUnit // start at full MaxEnergy
	energyBefore := c.Energy

	// Wire OSC1 (=1.0 at step 0 with OscPeriod=1) into DO_NOTHING so it always fires.
	c.Genome.OscPeriod = 1
	doNothingGene := &simulation.Gene{
		SourceType: simulation.SENSOR,
		SourceID:   simulation.OSC1,
		SinkType:   simulation.ACTION,
		SinkID:     simulation.DO_NOTHING,
		Weight:     255,
	}
	c.Genome.Brain = []*simulation.Gene{doNothingGene}
	c.Genome.BrainLength = 1
	c.CreateNeuralNet()

	rate := c.MetabolicRate(p)
	sim.Update()

	// Resting refunds the full metabolic drain and charges half — net cost = rate/2.
	expectedEnergy := energyBefore - rate/2
	tolerance := float32(0.01)
	if math.Abs(float64(c.Energy-expectedEnergy)) > float64(tolerance) {
		t.Errorf("DO_NOTHING should charge half metabolic rate: before=%f expected=%f got=%f (rate=%f)",
			energyBefore, expectedEnergy, c.Energy, rate)
	}
}

func TestDoNothingIsEnabled(t *testing.T) {
	if !simulation.IsActionEnabled(simulation.DO_NOTHING) {
		t.Errorf("DO_NOTHING (action %d) should be enabled", simulation.DO_NOTHING)
	}
}

func TestEatAction_KillsTargetInFOV(t *testing.T) {
	p := defaultParams()
	p.MinPopulation = 0
	p.MaxPopulation = 0
	p.MaxFood = 0
	p.StartingPopulation = 0
	p.PredationRadius = 2.0

	sim := simulation.New(p)

	const predID = 9000
	predGenome := simulation.MakeRandomGenome(p)
	predGenome.FieldOfView = 90
	predGenome.OscPeriod = 1
	predGenome.Responsiveness = 128
	eatGene := &simulation.Gene{
		SourceType: simulation.SENSOR,
		SourceID:   simulation.OSC1,
		SinkType:   simulation.ACTION,
		SinkID:     simulation.EAT,
		Weight:     255,
	}
	predGenome.Brain = []*simulation.Gene{eatGene}
	predGenome.BrainLength = 1
	pred := simulation.NewAdultCreature(predID, grid.Position{X: 50, Y: 50}, predGenome, p)
	pred.Heading = 0
	// Set energy below MaxEnergy so that the energy gain from eating is measurable.
	pred.Energy = pred.Mass * p.EnergyPerMassUnit * 0.5
	sim.Population.Creatures[pred.Id] = pred
	sim.World.AddCreature(pred.Id, pred.Loc)

	const preyID = 9001
	preyGenome := simulation.MakeRandomGenome(p)
	preyGenome.Mass = 200
	preyGenome.MinMass = 10
	preyGenome.Responsiveness = 128
	preyGenome.Brain = []*simulation.Gene{}
	preyGenome.BrainLength = 0
	prey := simulation.NewAdultCreature(preyID, grid.Position{X: 51, Y: 50}, preyGenome, p)
	sim.Population.Creatures[prey.Id] = prey
	sim.World.AddCreature(prey.Id, prey.Loc)

	energyBefore := pred.Energy
	sim.Update()

	if prey.Alive {
		t.Error("prey should be dead after EAT action fired")
	}
	if pred.Energy <= energyBefore {
		t.Errorf("predator energy should increase after eating: before=%f after=%f", energyBefore, pred.Energy)
	}
}

func TestIsActionEnabled(t *testing.T) {
	for a := byte(0); a < simulation.ACTION_COUNT; a++ {
		if !simulation.IsActionEnabled(a) {
			t.Errorf("action %d should be enabled", a)
		}
	}
	if simulation.IsActionEnabled(simulation.ACTION_COUNT) {
		t.Errorf("action %d (ACTION_COUNT) should be disabled", simulation.ACTION_COUNT)
	}
}
