package test

import (
	"biogo/v2/grid"
	"biogo/v2/simulation"
	"testing"
)

func TestNewPopulation(t *testing.T) {
	p := defaultParams()
	pop := simulation.NewPopulation(p)
	if pop == nil {
		t.Fatal("NewPopulation returned nil")
	}
	if pop.Creatures == nil {
		t.Error("Creatures map should be initialized")
	}
	if len(pop.Creatures) != 0 {
		t.Errorf("new population should have 0 creatures, got %d", len(pop.Creatures))
	}
}

func TestProcessMoveQueue(t *testing.T) {
	params := defaultParams()
	params.FoodInteractionRadius = 0.1
	params.PredationRadius = 0.1

	w := grid.NewWorld(20, 20, 0)
	genome := simulation.MakeRandomGenome(params)

	startPos := grid.Position{X: 5, Y: 5}
	creature := simulation.NewCreature(1, startPos, genome)
	w.AddCreature(1, startPos)

	pop := simulation.NewPopulation(params)
	pop.Creatures[1] = creature

	newPos := grid.Position{X: 6, Y: 5}
	pop.QueueForMove(creature, newPos)
	pop.ProcessMoveQueue(w, params)

	if creature.Loc != newPos {
		t.Errorf("creature did not move: got %v, want %v", creature.Loc, newPos)
	}
	pos, ok := w.GetCreaturePos(1)
	if !ok || pos != newPos {
		t.Errorf("world creature position not updated: got %v", pos)
	}
}

func TestProcessMoveQueueConsumesFood(t *testing.T) {
	params := defaultParams()
	params.FoodEnergyFraction = 0.3
	params.FoodInteractionRadius = 2.0
	params.PredationRadius = 0.1

	w := grid.NewWorld(20, 20, 0)
	genome := simulation.MakeRandomGenome(params)

	startPos := grid.Position{X: 5, Y: 5}
	foodPos := grid.Position{X: 6, Y: 5}

	creature := simulation.NewCreature(1, startPos, genome)
	creature.Energy = float32(creature.Genome.MaxEnergy) * 0.5
	energyBefore := creature.Energy

	w.AddCreature(1, startPos)
	w.AddFood(foodPos)

	pop := simulation.NewPopulation(params)
	pop.Creatures[1] = creature
	pop.QueueForMove(creature, foodPos)
	pop.ProcessMoveQueue(w, params)

	if creature.Loc != foodPos {
		t.Errorf("creature should move to food location, got %v", creature.Loc)
	}
	if w.FoodCount() != 0 {
		t.Error("food should be consumed after creature moves onto it")
	}
	if creature.Energy <= energyBefore {
		t.Error("creature energy should increase after eating food")
	}
}

func TestProcessDeathQueue(t *testing.T) {
	params := defaultParams()
	w := grid.NewWorld(20, 20, 0)

	genome := simulation.MakeRandomGenome(params)
	loc := grid.Position{X: 3, Y: 3}
	creature := simulation.NewCreature(1, loc, genome)
	w.AddCreature(1, loc)

	pop := simulation.NewPopulation(params)
	pop.Creatures[1] = creature
	pop.QueueForDeath(creature)
	pop.ProcessDeathQueue(w, params)

	if creature.Alive {
		t.Error("creature should not be alive after ProcessDeathQueue")
	}
	if len(pop.Creatures) != 1 {
		t.Errorf("corpse should remain in population map, got %d creatures", len(pop.Creatures))
	}
}

func TestPredation(t *testing.T) {
	params := defaultParams()
	params.PreyEnergyFraction = 0.5
	params.PredationRadius = 3.0
	params.FoodInteractionRadius = 0.1

	w := grid.NewWorld(20, 20, 0)
	genome := simulation.MakeRandomGenome(params)

	predatorPos := grid.Position{X: 5, Y: 5}
	preyPos := grid.Position{X: 6, Y: 5} // within PredationRadius of new predator position

	predator := simulation.NewCreature(1, predatorPos, genome)
	prey := simulation.NewCreature(2, preyPos, genome)
	prey.Energy = 100
	predator.Energy = float32(predator.Genome.MaxEnergy) * 0.3

	w.AddCreature(1, predatorPos)
	w.AddCreature(2, preyPos)

	pop := simulation.NewPopulation(params)
	pop.Creatures[1] = predator
	pop.Creatures[2] = prey

	energyBefore := predator.Energy
	// Move predator to a position within PredationRadius of prey
	pop.QueueForMove(predator, grid.Position{X: 5.5, Y: 5})
	pop.ProcessMoveQueue(w, params)

	if prey.Alive {
		t.Error("prey should be dead after predation")
	}
	if predator.Energy <= energyBefore {
		t.Error("predator should gain energy from predation")
	}
}

func TestScavenging(t *testing.T) {
	params := defaultParams()
	params.PreyEnergyFraction = 0.5
	params.PredationRadius = 3.0
	params.FoodInteractionRadius = 0.1

	w := grid.NewWorld(20, 20, 0)
	genome := simulation.MakeRandomGenome(params)

	scavengerPos := grid.Position{X: 5, Y: 5}
	corpsePos := grid.Position{X: 6, Y: 5}

	scavenger := simulation.NewCreature(1, scavengerPos, genome)
	scavenger.Energy = float32(scavenger.Genome.MaxEnergy) * 0.3
	corpse := simulation.NewCreature(2, corpsePos, genome)
	corpse.Alive = false
	corpse.Energy = 80

	w.AddCreature(1, scavengerPos)
	w.AddCreature(2, corpsePos)

	pop := simulation.NewPopulation(params)
	pop.Creatures[1] = scavenger
	pop.Creatures[2] = corpse

	energyBefore := scavenger.Energy
	pop.QueueForMove(scavenger, grid.Position{X: 5.5, Y: 5})
	pop.ProcessMoveQueue(w, params)

	if scavenger.Energy <= energyBefore {
		t.Error("scavenger should gain energy from corpse")
	}
	if _, ok := pop.Creatures[2]; ok {
		t.Error("consumed corpse should be removed from population map")
	}
}

func TestProcessCorpseDecay(t *testing.T) {
	params := defaultParams()
	params.CorpseDecayRate = 50
	w := grid.NewWorld(20, 20, 0)

	genome := simulation.MakeRandomGenome(params)
	loc := grid.Position{X: 3, Y: 3}
	corpse := simulation.NewCreature(1, loc, genome)
	corpse.Alive = false
	corpse.Energy = 60
	w.AddCreature(1, loc)

	pop := simulation.NewPopulation(params)
	pop.Creatures[1] = corpse

	pop.ProcessCorpseDecay(w, params)

	if corpse.Energy >= 60 {
		t.Error("corpse energy should decrease after decay")
	}
	if _, ok := pop.Creatures[1]; !ok {
		t.Error("corpse with remaining energy should still be in map")
	}

	pop.ProcessCorpseDecay(w, params)
	if _, ok := pop.Creatures[1]; ok {
		t.Error("fully decayed corpse should be removed from population map")
	}
}

func TestPredationGainBasedOnSize(t *testing.T) {
	params := defaultParams()
	params.PreyEnergyFraction = 1.0
	params.PredationRadius = 3.0
	params.FoodInteractionRadius = 0.1

	smallGenome := simulation.MakeRandomGenome(params)
	smallGenome.Size = 10
	largeGenome := simulation.MakeRandomGenome(params)
	largeGenome.Size = 200
	predatorGenome := simulation.MakeRandomGenome(params)
	predatorGenome.MaxEnergy = 255

	w := grid.NewWorld(20, 20, 0)

	smallPredator := simulation.NewCreature(1, grid.Position{X: 1, Y: 5}, predatorGenome)
	smallPredator.Energy = 1
	smallPrey := simulation.NewCreature(2, grid.Position{X: 2, Y: 5}, smallGenome)
	smallPrey.Energy = 999
	smallPrey.Age = params.MaxJuvenilePeriod + 1

	largePredator := simulation.NewCreature(3, grid.Position{X: 8, Y: 5}, predatorGenome)
	largePredator.Energy = 1
	largePrey := simulation.NewCreature(4, grid.Position{X: 9, Y: 5}, largeGenome)
	largePrey.Energy = 999
	largePrey.Age = params.MaxJuvenilePeriod + 1

	for _, c := range []*simulation.Creature{smallPredator, smallPrey, largePredator, largePrey} {
		w.AddCreature(c.Id, c.Loc)
	}

	pop := simulation.NewPopulation(params)
	pop.Creatures[1] = smallPredator
	pop.Creatures[2] = smallPrey
	pop.Creatures[3] = largePredator
	pop.Creatures[4] = largePrey

	pop.QueueForMove(smallPredator, grid.Position{X: 1.5, Y: 5})
	pop.QueueForMove(largePredator, grid.Position{X: 8.5, Y: 5})
	pop.ProcessMoveQueue(w, params)

	smallGain := smallPredator.Energy - 1
	largeGain := largePredator.Energy - 1

	if largeGain <= smallGain {
		t.Errorf("larger prey should yield more energy than smaller: gains %f vs %f",
			largeGain, smallGain)
	}
}

func TestCorpseEnergySetOnDeath(t *testing.T) {
	params := defaultParams()
	w := grid.NewWorld(20, 20, 0)

	genome := simulation.MakeRandomGenome(params)
	genome.Size = 120
	loc := grid.Position{X: 3, Y: 3}
	creature := simulation.NewCreature(1, loc, genome)
	creature.Energy = 5
	creature.Age = params.MaxJuvenilePeriod + 1
	w.AddCreature(1, loc)

	pop := simulation.NewPopulation(params)
	pop.Creatures[1] = creature
	pop.QueueForDeath(creature)
	pop.ProcessDeathQueue(w, params)

	if creature.Energy != float32(genome.Size) {
		t.Errorf("corpse energy should equal genome.Size (%d), got %f", genome.Size, creature.Energy)
	}
}

func TestOldestGenomeEmpty(t *testing.T) {
	params := defaultParams()
	pop := simulation.NewPopulation(params)
	if pop.OldestGenome() != nil {
		t.Error("OldestGenome should return nil for an empty population")
	}
}

func TestOldestGenomeDeadOnly(t *testing.T) {
	params := defaultParams()
	pop := simulation.NewPopulation(params)
	genome := simulation.MakeRandomGenome(params)
	dead := simulation.NewCreature(1, grid.Position{X: 1, Y: 1}, genome)
	dead.Alive = false
	pop.Creatures[1] = dead
	if pop.OldestGenome() != nil {
		t.Error("OldestGenome should return nil when all creatures are dead")
	}
}

func TestOldestGenomeReturnsOldest(t *testing.T) {
	params := defaultParams()
	pop := simulation.NewPopulation(params)
	genome := simulation.MakeRandomGenome(params)

	young := simulation.NewCreature(1, grid.Position{X: 1, Y: 1}, genome)
	young.Age = 10
	old := simulation.NewCreature(2, grid.Position{X: 2, Y: 2}, genome)
	old.Age = 100

	pop.Creatures[1] = young
	pop.Creatures[2] = old

	result := pop.OldestGenome()
	if result == nil {
		t.Fatal("OldestGenome should not return nil when alive creatures exist")
	}
	if result != old.Genome {
		t.Error("OldestGenome should return the genome of the creature with the highest age")
	}
}

func TestGeneticDiversity(t *testing.T) {
	params := defaultParams()
	pop := simulation.NewPopulation(params)

	for i := 0; i < 10; i++ {
		id := i + 1
		genome := simulation.MakeRandomGenome(params)
		pop.Creatures[id] = simulation.NewCreature(id, grid.Position{X: float64(i), Y: 0}, genome)
	}

	diversity := pop.GeneticDiversity()
	if diversity < 0 || diversity > 1 {
		t.Errorf("GeneticDiversity out of [0,1]: %f", diversity)
	}
}

func TestReproductionCreatesOffspring(t *testing.T) {
	params := defaultParams()
	params.MaxPopulation = 100
	params.ReproductionEnergyCost = 0.1

	w := grid.NewWorld(50, 50, 0)
	genome := simulation.MakeRandomGenome(params)
	genome.MaxEnergy = 255

	parentPos := grid.Position{X: 25, Y: 25}
	parent := simulation.NewCreature(1, parentPos, genome)
	parent.Energy = float32(genome.MaxEnergy)
	w.AddCreature(1, parentPos)

	pop := simulation.NewPopulation(params)
	pop.Creatures[1] = parent

	nextID := 2
	pop.QueueForReproduction(parent)
	pop.ProcessReproductionQueue(w, params, func() int {
		id := nextID
		nextID++
		return id
	})

	if len(pop.Creatures) != 2 {
		t.Fatalf("expected 2 creatures after reproduction, got %d", len(pop.Creatures))
	}
}

func TestGeneticDiversitySingleCreature(t *testing.T) {
	params := defaultParams()
	pop := simulation.NewPopulation(params)
	genome := simulation.MakeRandomGenome(params)
	pop.Creatures[1] = simulation.NewCreature(1, grid.Position{}, genome)

	diversity := pop.GeneticDiversity()
	if diversity != 0 {
		t.Errorf("single creature diversity should be 0, got %f", diversity)
	}
}

func TestAliveCount(t *testing.T) {
	params := defaultParams()
	pop := simulation.NewPopulation(params)
	genome := simulation.MakeRandomGenome(params)

	alive := simulation.NewCreature(1, grid.Position{X: 1, Y: 1}, genome)
	dead := simulation.NewCreature(2, grid.Position{X: 2, Y: 2}, genome)
	dead.Alive = false

	pop.Creatures[1] = alive
	pop.Creatures[2] = dead

	if pop.AliveCount() != 1 {
		t.Errorf("AliveCount should be 1, got %d", pop.AliveCount())
	}
}
