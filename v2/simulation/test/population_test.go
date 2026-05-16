package test

import (
	grid "biogo/v2/world"
	"biogo/v2/simulation"
	"testing"
)

func countCreatures(pop *simulation.Population) int {
	count := 0
	for _, c := range pop.Creatures {
		if c != nil {
			count++
		}
	}
	return count
}

func TestNewPopulation(t *testing.T) {
	p := defaultParams()
	pop := simulation.NewPopulation(p)
	if pop == nil {
		t.Fatal("NewPopulation returned nil")
	}
	if pop.Creatures == nil {
		t.Error("Creatures should be initialized")
	}
	if len(pop.Creatures) != 0 {
		t.Errorf("new population should have 0 creatures, got %d", len(pop.Creatures))
	}
}

func TestProcessMoveQueue(t *testing.T) {
	params := defaultParams()
	params.FoodInteractionRadius = 0.1

	w := grid.NewWorld(20, 20, 0)
	genome := simulation.MakeRandomGenome(params, 0)

	startPos := grid.Position{X: 5, Y: 5}
	id := w.AddCreature(startPos)
	creature := simulation.NewCreature(id, startPos, genome, params)

	pop := simulation.NewPopulation(params)
	pop.SetCreature(id, creature)

	newPos := grid.Position{X: 6, Y: 5}
	pop.QueueForMove(creature, newPos, 1.0)
	pop.ProcessMoveQueue(w)

	if creature.Loc != newPos {
		t.Errorf("creature did not move: got %v, want %v", creature.Loc, newPos)
	}
	pos, ok := w.GetCreaturePos(id)
	if !ok || pos != newPos {
		t.Errorf("world creature position not updated: got %v", pos)
	}
}

func TestProcessMoveQueueConsumesFood(t *testing.T) {
	params := defaultParams()
	params.FoodInteractionRadius = 2.0

	w := grid.NewWorld(20, 20, 0)
	genome := simulation.MakeRandomGenome(params, 0)
	genome.Mass = 100
	genome.MinMass = 10

	startPos := grid.Position{X: 5, Y: 5}
	destPos := grid.Position{X: 6, Y: 5}
	foodPos := grid.Position{X: 7, Y: 5}

	id := w.AddCreature(startPos)
	creature := simulation.NewCreature(id, startPos, genome, params)
	creature.Heading = 0 // east, so food at (7,5) is in the forward cone
	// Start at 50% of MaxEnergy so the creature is hungry enough to eat.
	creature.Energy = float32(creature.Mass) * params.EnergyPerMassUnit * 0.5

	w.AddPlant(foodPos, 10)

	pop := simulation.NewPopulation(params)
	pop.SetCreature(id, creature)
	pop.QueueForMove(creature, destPos, 1.0)
	pop.ProcessMoveQueue(w)

	if creature.Loc != destPos {
		t.Errorf("creature should move to destination, got %v", creature.Loc)
	}
	if creature.Stomach <= 0 {
		t.Error("creature should have eaten something (stomach > 0)")
	}
	// Plants may be fully consumed (PlantCount == 0) or only partially eaten
	// (a small creature's bite is less than FoodMass so the item persists with reduced mass).
	if w.PlantCount() > 0 && w.TotalPlantMass() >= 10 {
		t.Error("plant mass should decrease after creature eats from it")
	}
}

func TestProcessDeathQueue(t *testing.T) {
	params := defaultParams()
	w := grid.NewWorld(20, 20, 0)

	genome := simulation.MakeRandomGenome(params, 0)
	loc := grid.Position{X: 3, Y: 3}
	id := w.AddCreature(loc)
	creature := simulation.NewCreature(id, loc, genome, params)

	pop := simulation.NewPopulation(params)
	pop.SetCreature(id, creature)
	pop.AddAlive(id)
	pop.QueueForDeath(creature)
	pop.ProcessDeathQueue(w, params)

	if _, ok := pop.Get(id); ok {
		t.Error("dead creature should be removed from population")
	}
	if w.MeatCount() == 0 {
		t.Error("meat should be spawned at death location")
	}
}

// TestDeathSpawnsMeatMatchingMass verifies that total meat mass spawned matches the creature's body mass.
func TestDeathSpawnsMeatMatchingMass(t *testing.T) {
	params := defaultParams()
	w := grid.NewWorld(20, 20, 0)

	genome := simulation.MakeRandomGenome(params, 0)
	genome.Mass = 120
	loc := grid.Position{X: 10, Y: 10}
	id := w.AddCreature(loc)
	creature := simulation.NewAdultCreature(id, loc, genome, params)
	deathMass := creature.Mass

	pop := simulation.NewPopulation(params)
	pop.SetCreature(id, creature)
	pop.AddAlive(id)
	pop.QueueForDeath(creature)
	pop.ProcessDeathQueue(w, params)

	if w.TotalMeatMass() < float64(deathMass)-0.01 || w.TotalMeatMass() > float64(deathMass)+0.01 {
		t.Errorf("total meat mass %.2f should match creature mass %.2f", w.TotalMeatMass(), deathMass)
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
	genome := simulation.MakeRandomGenome(params, 0)
	dead := simulation.NewCreature(1, grid.Position{X: 1, Y: 1}, genome, params)
	dead.Alive = false
	pop.SetCreature(1, dead)
	if pop.OldestGenome() != nil {
		t.Error("OldestGenome should return nil when all creatures are dead")
	}
}

func TestOldestGenomeReturnsOldest(t *testing.T) {
	params := defaultParams()
	pop := simulation.NewPopulation(params)
	genome := simulation.MakeRandomGenome(params, 0)

	young := simulation.NewCreature(1, grid.Position{X: 1, Y: 1}, genome, params)
	young.Age = 10
	old := simulation.NewCreature(2, grid.Position{X: 2, Y: 2}, genome, params)
	old.Age = 100

	pop.SetCreature(1, young)
	pop.SetCreature(2, old)
	pop.AddAlive(1)
	pop.AddAlive(2)

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
		genome := simulation.MakeRandomGenome(params, 0)
		pop.SetCreature(id, simulation.NewCreature(id, grid.Position{X: float32(i), Y: 0}, genome, params))
	}

	diversity := pop.GeneticDiversity()
	if diversity < 0 || diversity > 1 {
		t.Errorf("GeneticDiversity out of [0,1]: %f", diversity)
	}
}

func TestReproductionCreatesOffspring(t *testing.T) {
	params := defaultParams()
	params.MaxPopulation = 100

	w := grid.NewWorld(50, 50, 0)
	genome := simulation.MakeRandomGenome(params, 0)
	genome.Mass = 200
	genome.MinMass = 10

	parentPos := grid.Position{X: 25, Y: 25}
	parentID := w.AddCreature(parentPos)
	parent := simulation.NewAdultCreature(parentID, parentPos, genome, params)
	// Start at full energy so reproduction threshold is met.
	parent.Energy = float32(parent.Mass) * params.EnergyPerMassUnit

	pop := simulation.NewPopulation(params)
	pop.SetCreature(parentID, parent)

	pop.QueueForReproduction(parent)
	pop.ProcessReproductionQueue(w, params)

	if countCreatures(pop) != 2 {
		t.Fatalf("expected 2 creatures after reproduction, got %d", countCreatures(pop))
	}
}

func TestReproductionHalvesParentMass(t *testing.T) {
	params := defaultParams()
	params.MaxPopulation = 100

	w := grid.NewWorld(50, 50, 0)
	genome := simulation.MakeRandomGenome(params, 0)
	genome.Mass = 100
	genome.MinMass = 10
	genome.MassSplitRatio = 255 // maximum split → 50%

	parentPos := grid.Position{X: 25, Y: 25}
	parentID := w.AddCreature(parentPos)
	parent := simulation.NewAdultCreature(parentID, parentPos, genome, params)
	parent.Energy = float32(parent.Mass) * params.EnergyPerMassUnit

	pop := simulation.NewPopulation(params)
	pop.SetCreature(parentID, parent)

	pop.QueueForReproduction(parent)
	pop.ProcessReproductionQueue(w, params)

	wantMass := float32(genome.Mass) / 2
	if parent.Mass != wantMass {
		t.Errorf("parent Mass after reproduction: got %f, want %f", parent.Mass, wantMass)
	}
}

func TestReproductionChildStartsAtHalfMass(t *testing.T) {
	params := defaultParams()
	params.MaxPopulation = 100

	w := grid.NewWorld(50, 50, 0)
	genome := simulation.MakeRandomGenome(params, 0)
	genome.Mass = 100
	genome.MinMass = 10
	genome.MutationRate = 0     // suppress mutations so child inherits same Mass
	genome.MassSplitRatio = 255 // maximum split → 50%

	parentPos := grid.Position{X: 25, Y: 25}
	parentID := w.AddCreature(parentPos)
	parent := simulation.NewAdultCreature(parentID, parentPos, genome, params)
	parent.Energy = float32(parent.Mass) * params.EnergyPerMassUnit

	pop := simulation.NewPopulation(params)
	pop.SetCreature(parentID, parent)

	pop.QueueForReproduction(parent)
	pop.ProcessReproductionQueue(w, params)

	if countCreatures(pop) != 2 {
		t.Fatalf("expected 2 creatures after reproduction, got %d", countCreatures(pop))
	}
	var child *simulation.Creature
	for _, c := range pop.Creatures {
		if c == nil || c == parent {
			continue
		}
		child = c
		break
	}
	wantMass := float32(genome.Mass) / 2
	if child.Mass != wantMass {
		t.Errorf("child Mass at birth: got %f, want %f", child.Mass, wantMass)
	}
}

func TestReproductionSkipsWhenEnergyBelowThreshold(t *testing.T) {
	params := defaultParams()
	params.MaxPopulation = 100

	w := grid.NewWorld(50, 50, 0)
	genome := simulation.MakeRandomGenome(params, 0)
	genome.Mass = 200
	genome.MinMass = 10

	parentPos := grid.Position{X: 25, Y: 25}
	parentID := w.AddCreature(parentPos)
	parent := simulation.NewAdultCreature(parentID, parentPos, genome, params)
	// Set energy just below the reproduction threshold.
	parent.Energy = params.ReproductionEnergyThreshold*float32(genome.Mass)*params.EnergyPerMassUnit - 1

	pop := simulation.NewPopulation(params)
	pop.SetCreature(parentID, parent)

	pop.QueueForReproduction(parent)
	pop.ProcessReproductionQueue(w, params)

	if countCreatures(pop) != 1 {
		t.Errorf("reproduction should be skipped below energy threshold, got %d creatures", countCreatures(pop))
	}
}

func TestReproductionSkipsWhenMinMassConstraintViolated(t *testing.T) {
	params := defaultParams()
	params.MaxPopulation = 100

	w := grid.NewWorld(50, 50, 0)
	genome := simulation.MakeRandomGenome(params, 0)
	genome.Mass = 10
	genome.MinMass = 6 // 6*2=12 >= 10: violates MinMass < Mass/2

	parentPos := grid.Position{X: 25, Y: 25}
	parentID := w.AddCreature(parentPos)
	parent := simulation.NewAdultCreature(parentID, parentPos, genome, params)
	parent.Energy = float32(parent.Mass) * params.EnergyPerMassUnit

	pop := simulation.NewPopulation(params)
	pop.SetCreature(parentID, parent)

	pop.QueueForReproduction(parent)
	pop.ProcessReproductionQueue(w, params)

	if countCreatures(pop) != 1 {
		t.Errorf("reproduction should be skipped when MinMass violates constraint, got %d creatures", countCreatures(pop))
	}
}

func TestGeneticDiversitySingleCreature(t *testing.T) {
	params := defaultParams()
	pop := simulation.NewPopulation(params)
	genome := simulation.MakeRandomGenome(params, 0)
	pop.SetCreature(1, simulation.NewCreature(1, grid.Position{}, genome, params))

	diversity := pop.GeneticDiversity()
	if diversity != 0 {
		t.Errorf("single creature diversity should be 0, got %f", diversity)
	}
}

func TestAliveCount(t *testing.T) {
	params := defaultParams()
	pop := simulation.NewPopulation(params)
	genome := simulation.MakeRandomGenome(params, 0)

	alive := simulation.NewCreature(1, grid.Position{X: 1, Y: 1}, genome, params)
	dead := simulation.NewCreature(2, grid.Position{X: 2, Y: 2}, genome, params)
	dead.Alive = false

	pop.SetCreature(1, alive)
	pop.SetCreature(2, dead)
	pop.AddAlive(1)

	if pop.AliveCount() != 1 {
		t.Errorf("AliveCount should be 1, got %d", pop.AliveCount())
	}
}
