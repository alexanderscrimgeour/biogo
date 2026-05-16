package test

import (
	grid "biogo/v2/world"
	"biogo/v2/simulation"
	"math"
	"testing"
)

func TestDoNothingReducesMetabolicCost(t *testing.T) {
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
		if v != nil {
			c = v
			break
		}
	}
	c.Genome.MetabolicRate = 127
	c.Energy = float32(c.Mass) * p.EnergyPerMassUnit // start at full MaxEnergy
	energyBefore := c.Energy

	// Wire OSC1 (=1.0 at step 0 with OscPeriod=1) into REST so it always fires.
	c.Genome.OscPeriod = 1
	doNothingGene := simulation.Gene{
		SourceType: simulation.SENSOR,
		SourceID:   simulation.OSC1,
		SinkType:   simulation.ACTION,
		SinkID:     simulation.REST,
		Weight:     255,
	}
	c.Genome.Brain = []simulation.Gene{doNothingGene}
	c.Genome.SynapticDensity = 1
	c.CreateNeuralNet()

	rate := c.MetabolicRate(p, sim.World.TemperatureAt(c.Loc.Y))
	sim.Update()

	// Resting refunds the full metabolic drain and charges 10% — net cost = rate * 0.1.
	expectedEnergy := energyBefore - rate*0.1
	tolerance := float32(0.01)
	if math.Abs(float64(c.Energy-expectedEnergy)) > float64(tolerance) {
		t.Errorf("REST should charge half metabolic rate: before=%f expected=%f got=%f (rate=%f)",
			energyBefore, expectedEnergy, c.Energy, rate)
	}
}

func TestDoNothingIsEnabled(t *testing.T) {
	if !simulation.IsActionEnabled(simulation.REST) {
		t.Errorf("REST (action %d) should be enabled", simulation.REST)
	}
}

func TestPassivePredation_TakesBiteFromNearbyMeat(t *testing.T) {
	params := defaultParams()
	params.FoodInteractionRadius = 3.0
	params.MaxFood = 0

	w := grid.NewWorld(20, 20, 0)

	predGenome := simulation.MakeRandomGenome(params, 0)
	predGenome.Mass = 128
	predGenome.MinMass = 10
	predGenome.FieldOfView = 180
	predGenome.StomachSize = 255

	predPos := grid.Position{X: 5, Y: 5}
	predID := w.AddCreature(predPos)
	pred := simulation.NewAdultCreature(predID, predPos, predGenome, params)
	pred.Heading = 0 // east

	// Place a meat item east of the predator within interaction radius.
	meatPos := grid.Position{X: 7, Y: 5}
	meatMassBefore := float32(50)
	w.AddMeat(meatPos, meatMassBefore)

	pop := simulation.NewPopulation(params)
	pop.SetCreature(predID, pred)

	newPos := grid.Position{X: 6, Y: 5}
	pop.QueueForMove(pred, newPos, 1.0)
	pop.ProcessMoveQueue(w)

	meatMassAfter := w.TotalMeatMass()
	if float32(meatMassAfter) >= meatMassBefore {
		t.Errorf("meat should lose mass after being eaten: before=%f after=%f", meatMassBefore, meatMassAfter)
	}
	if pred.Stomach <= 0 {
		t.Errorf("predator stomach should be filled after eating meat: got %f", pred.Stomach)
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
