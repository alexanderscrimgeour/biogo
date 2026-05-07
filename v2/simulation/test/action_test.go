package test

import (
	"biogo/v2/grid"
	"biogo/v2/simulation"
	"testing"
)

func TestDoNothingRefundsMetabolicCost(t *testing.T) {
	p := defaultParams()
	p.MinMetabolicRate = 5
	p.MaxMetabolicRate = 5
	p.MoveCost = 0
	p.MinPopulation = 0
	p.MaxPopulation = 1 // prevents reproduction energy drain during the test
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

	// Force DO_NOTHING to fire by wiring OSC1 (= 1.0 at step 0 with OscPeriod=1) into it.
	// OSC1 gives an exact 1.0 output, so prob2Bool always returns 1 (rand.Float64() < 1.0).
	c.Genome.OscPeriod = 1
	doNothingGene := &simulation.Gene{
		SourceType: simulation.SENSOR,
		SourceID:   simulation.OSC1,
		SinkType:   simulation.ACTION,
		SinkID:     simulation.DO_NOTHING,
		Weight:     255, // max positive weight → level = 1.0, prob2Bool always fires
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

func TestEatAction_KillsTargetInFOV(t *testing.T) {
	p := defaultParams()
	p.MinPopulation = 0
	p.MaxPopulation = 0 // prevents reproduction energy drain and ID-allocation side effects
	p.MaxFood = 0
	p.StartingPopulation = 0
	p.PredationRadius = 2.0

	sim := simulation.New(p)

	// Build predator with a deterministic brain: OSC1 (=1.0 at tick 0 with OscPeriod=1) → EAT.
	// Use high IDs to avoid conflicts with the sim's internal allocateID counter.
	const predID = 9000
	predGenome := simulation.MakeRandomGenome(p)
	predGenome.FieldOfView = 90
	predGenome.MaxEnergy = 200
	predGenome.OscPeriod = 1        // OSC1 = 1.0 at simStep=0
	predGenome.Responsiveness = 128 // non-zero prevents NaN in SET_RESPONSIVENESS
	eatGene := &simulation.Gene{
		SourceType: simulation.SENSOR,
		SourceID:   simulation.OSC1, // evaluates to 1.0 → EAT always fires
		SinkType:   simulation.ACTION,
		SinkID:     simulation.EAT,
		Weight:     255,
	}
	predGenome.Brain = []*simulation.Gene{eatGene}
	predGenome.BrainLength = 1
	pred := simulation.NewAdultCreature(predID, grid.Position{X: 50, Y: 50}, predGenome, p)
	pred.Heading = 0
	pred.Energy = 100 // below MaxEnergy so energy gain from eating is measurable
	sim.Population.Creatures[pred.Id] = pred
	sim.World.AddCreature(pred.Id, pred.Loc)

	// Build prey with an empty brain so it never moves or predates.
	const preyID = 9001
	preyGenome := simulation.MakeRandomGenome(p)
	preyGenome.Mass = 200
	preyGenome.MaxEnergy = 200
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
	// Anything at or above ACTION_COUNT should be disabled.
	if simulation.IsActionEnabled(simulation.ACTION_COUNT) {
		t.Errorf("action %d (ACTION_COUNT) should be disabled", simulation.ACTION_COUNT)
	}
}
