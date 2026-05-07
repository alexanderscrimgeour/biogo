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
	params.GridWidth = 20
	params.GridHeight = 20
	params.StartingPopulation = 1

	g := grid.NewGrid(20, 20, 0)
	genome := simulation.MakeRandomGenome(params)

	startLoc := grid.Coord{X: 5, Y: 5}
	creature := simulation.NewCreature(grid.RESERVED_CELL_TYPES, startLoc, genome)
	g.Set(startLoc, grid.RESERVED_CELL_TYPES)

	pop := simulation.NewPopulation(params)
	pop.Creatures[grid.RESERVED_CELL_TYPES] = creature

	newLoc := grid.Coord{X: 6, Y: 5}
	pop.QueueForMove(creature, newLoc)
	pop.ProcessMoveQueue(g, params)

	if creature.Loc != newLoc {
		t.Errorf("creature did not move: got %v, want %v", creature.Loc, newLoc)
	}
	if g.At(startLoc) != 0 {
		t.Errorf("old location should be cleared, got %d", g.At(startLoc))
	}
	if g.At(newLoc) != grid.RESERVED_CELL_TYPES {
		t.Errorf("new location should hold creature id, got %d", g.At(newLoc))
	}
}

func TestProcessMoveQueueOccupiedTarget(t *testing.T) {
	params := defaultParams()
	g := grid.NewGrid(20, 20, 0)
	genome := simulation.MakeRandomGenome(params)

	startLoc := grid.Coord{X: 5, Y: 5}
	targetLoc := grid.Coord{X: 6, Y: 5}
	creature := simulation.NewCreature(grid.RESERVED_CELL_TYPES, startLoc, genome)
	g.Set(startLoc, grid.RESERVED_CELL_TYPES)
	g.Set(targetLoc, grid.WALL)

	pop := simulation.NewPopulation(params)
	pop.Creatures[grid.RESERVED_CELL_TYPES] = creature
	pop.QueueForMove(creature, targetLoc)
	pop.ProcessMoveQueue(g, params)

	if creature.Loc != startLoc {
		t.Errorf("creature should not move into wall cell, got %v", creature.Loc)
	}
}

func TestProcessMoveQueueConsumesFood(t *testing.T) {
	params := defaultParams()
	params.GridWidth = 20
	params.GridHeight = 20
	params.FoodEnergyFraction = 0.3
	g := grid.NewGrid(20, 20, 0)
	g.ZeroFill()

	genome := simulation.MakeRandomGenome(params)
	startLoc := grid.Coord{X: 5, Y: 5}
	foodLoc := grid.Coord{X: 6, Y: 5}

	creature := simulation.NewCreature(grid.RESERVED_CELL_TYPES, startLoc, genome)
	g.Set(startLoc, grid.RESERVED_CELL_TYPES)
	g.SpawnFood(0)
	g.Set(foodLoc, grid.FOOD)
	g.FoodLocations = append(g.FoodLocations, foodLoc)

	creature.Energy = float32(creature.Genome.MaxEnergy) * 0.5
	energyBefore := creature.Energy

	pop := simulation.NewPopulation(params)
	pop.Creatures[grid.RESERVED_CELL_TYPES] = creature
	pop.QueueForMove(creature, foodLoc)
	pop.ProcessMoveQueue(g, params)

	if creature.Loc != foodLoc {
		t.Errorf("creature should move to food location, got %v", creature.Loc)
	}
	if g.IsFood(foodLoc) {
		t.Error("food should be consumed after creature moves onto it")
	}
	if creature.Energy <= energyBefore {
		t.Error("creature energy should increase after eating food")
	}
}

// TestProcessDeathQueue verifies that a dead creature's corpse remains on the grid
// and in the population map with Alive=false, rather than being removed immediately.
func TestProcessDeathQueue(t *testing.T) {
	params := defaultParams()
	g := grid.NewGrid(20, 20, 0)
	g.ZeroFill()

	genome := simulation.MakeRandomGenome(params)
	loc := grid.Coord{X: 3, Y: 3}
	creature := simulation.NewCreature(grid.RESERVED_CELL_TYPES, loc, genome)
	g.Set(loc, grid.RESERVED_CELL_TYPES)

	pop := simulation.NewPopulation(params)
	pop.Creatures[grid.RESERVED_CELL_TYPES] = creature
	pop.QueueForDeath(creature)
	pop.ProcessDeathQueue(g, params)

	if creature.Alive {
		t.Error("creature should not be alive after ProcessDeathQueue")
	}
	if len(pop.Creatures) != 1 {
		t.Errorf("corpse should remain in population map, got %d creatures", len(pop.Creatures))
	}
	if g.IsEmptyAt(loc) {
		t.Error("corpse should remain on grid after death")
	}
}

// TestPredation verifies that a creature moving into a living creature's cell
// kills the prey and gains energy without moving.
func TestPredation(t *testing.T) {
	params := defaultParams()
	params.PreyEnergyFraction = 0.5
	g := grid.NewGrid(20, 20, 0)
	g.ZeroFill()

	genome := simulation.MakeRandomGenome(params)
	predatorLoc := grid.Coord{X: 5, Y: 5}
	preyLoc := grid.Coord{X: 6, Y: 5}

	predID := grid.RESERVED_CELL_TYPES
	preyID := grid.RESERVED_CELL_TYPES + 1

	predator := simulation.NewCreature(predID, predatorLoc, genome)
	prey := simulation.NewCreature(preyID, preyLoc, genome)
	prey.Energy = 100
	predator.Energy = float32(predator.Genome.MaxEnergy) * 0.3 // start low so energy gain is observable

	g.Set(predatorLoc, predID)
	g.Set(preyLoc, preyID)

	pop := simulation.NewPopulation(params)
	pop.Creatures[predID] = predator
	pop.Creatures[preyID] = prey

	predatorEnergyBefore := predator.Energy
	pop.QueueForMove(predator, preyLoc)
	pop.ProcessMoveQueue(g, params)

	if prey.Alive {
		t.Error("prey should be dead after predation")
	}
	if predator.Loc != predatorLoc {
		t.Errorf("predator should not move during predation, got %v", predator.Loc)
	}
	if predator.Energy <= predatorEnergyBefore {
		t.Error("predator should gain energy from predation")
	}
	if g.At(preyLoc) != preyID {
		t.Errorf("prey corpse should remain on grid at %v", preyLoc)
	}
}

// TestScavenging verifies that a creature moving into a corpse cell consumes it,
// gains energy, and moves into the vacated cell.
func TestScavenging(t *testing.T) {
	params := defaultParams()
	params.PreyEnergyFraction = 0.5
	g := grid.NewGrid(20, 20, 0)
	g.ZeroFill()

	genome := simulation.MakeRandomGenome(params)
	scavengerLoc := grid.Coord{X: 5, Y: 5}
	corpseLoc := grid.Coord{X: 6, Y: 5}

	scavID := grid.RESERVED_CELL_TYPES
	corpseID := grid.RESERVED_CELL_TYPES + 1

	scavenger := simulation.NewCreature(scavID, scavengerLoc, genome)
	scavenger.Energy = float32(scavenger.Genome.MaxEnergy) * 0.3 // start low so energy gain is observable
	corpse := simulation.NewCreature(corpseID, corpseLoc, genome)
	corpse.Alive = false
	corpse.Energy = 80

	g.Set(scavengerLoc, scavID)
	g.Set(corpseLoc, corpseID)

	pop := simulation.NewPopulation(params)
	pop.Creatures[scavID] = scavenger
	pop.Creatures[corpseID] = corpse

	energyBefore := scavenger.Energy
	pop.QueueForMove(scavenger, corpseLoc)
	pop.ProcessMoveQueue(g, params)

	if scavenger.Loc != corpseLoc {
		t.Errorf("scavenger should move into corpse cell, got %v", scavenger.Loc)
	}
	if scavenger.Energy <= energyBefore {
		t.Error("scavenger should gain energy from corpse")
	}
	if _, ok := pop.Creatures[corpseID]; ok {
		t.Error("consumed corpse should be removed from population map")
	}
}

// TestProcessCorpseDecay verifies that corpse energy decreases each tick and
// fully decayed corpses are removed from the grid and population map.
func TestProcessCorpseDecay(t *testing.T) {
	params := defaultParams()
	params.CorpseDecayRate = 50
	g := grid.NewGrid(20, 20, 0)
	g.ZeroFill()

	genome := simulation.MakeRandomGenome(params)
	loc := grid.Coord{X: 3, Y: 3}
	corpseID := grid.RESERVED_CELL_TYPES
	corpse := simulation.NewCreature(corpseID, loc, genome)
	corpse.Alive = false
	corpse.Energy = 60
	g.Set(loc, corpseID)

	pop := simulation.NewPopulation(params)
	pop.Creatures[corpseID] = corpse

	pop.ProcessCorpseDecay(g, params)

	if corpse.Energy >= 60 {
		t.Error("corpse energy should decrease after decay")
	}
	if _, ok := pop.Creatures[corpseID]; !ok {
		t.Error("corpse with remaining energy should still be in map")
	}

	// Decay again to fully consume the corpse (60 - 50 = 10, then 10 - 50 <= 0)
	pop.ProcessCorpseDecay(g, params)

	if _, ok := pop.Creatures[corpseID]; ok {
		t.Error("fully decayed corpse should be removed from population map")
	}
	if !g.IsEmptyAt(loc) {
		t.Error("fully decayed corpse should be cleared from grid")
	}
}

// TestPredationGainBasedOnSize verifies that predation gain depends on the prey's Size
// genome field, not the prey's current energy level.
func TestPredationGainBasedOnSize(t *testing.T) {
	params := defaultParams()
	params.PreyEnergyFraction = 1.0
	g := grid.NewGrid(20, 20, 0)
	g.ZeroFill()

	smallGenome := simulation.MakeRandomGenome(params)
	smallGenome.Size = 10
	largeGenome := simulation.MakeRandomGenome(params)
	largeGenome.Size = 200

	predID := grid.RESERVED_CELL_TYPES
	smallPreyID := grid.RESERVED_CELL_TYPES + 1
	largePreyID := grid.RESERVED_CELL_TYPES + 2

	predatorGenome := simulation.MakeRandomGenome(params)
	predatorGenome.MaxEnergy = 255

	predatorLoc := grid.Coord{X: 5, Y: 5}
	smallPreyLoc := grid.Coord{X: 6, Y: 5}
	largePredatorLoc := grid.Coord{X: 8, Y: 5}
	largePreyLoc := grid.Coord{X: 9, Y: 5}

	smallPredator := simulation.NewCreature(predID, predatorLoc, predatorGenome)
	smallPredator.Energy = 1
	smallPrey := simulation.NewCreature(smallPreyID, smallPreyLoc, smallGenome)
	smallPrey.Energy = 999
	smallPrey.Age = params.MaxJuvenilePeriod + 1 // adult: CurrentSize == genome.Size

	largePredator := simulation.NewCreature(largePreyID, largePredatorLoc, predatorGenome)
	largePredator.Energy = 1
	largePrey := simulation.NewCreature(largePreyID+1, largePreyLoc, largeGenome)
	largePrey.Energy = 999
	largePrey.Age = params.MaxJuvenilePeriod + 1 // adult: CurrentSize == genome.Size

	g.Set(predatorLoc, predID)
	g.Set(smallPreyLoc, smallPreyID)
	g.Set(largePredatorLoc, largePreyID)
	g.Set(largePreyLoc, largePreyID+1)

	pop := simulation.NewPopulation(params)
	pop.Creatures[predID] = smallPredator
	pop.Creatures[smallPreyID] = smallPrey
	pop.Creatures[largePreyID] = largePredator
	pop.Creatures[largePreyID+1] = largePrey

	pop.QueueForMove(smallPredator, smallPreyLoc)
	pop.QueueForMove(largePredator, largePreyLoc)
	pop.ProcessMoveQueue(g, params)

	smallGain := smallPredator.Energy - 1
	largeGain := largePredator.Energy - 1

	if largeGain <= smallGain {
		t.Errorf("larger prey (size=%d) should yield more energy than smaller prey (size=%d): got gains %f vs %f",
			largeGenome.Size, smallGenome.Size, largeGain, smallGain)
	}
}

// TestCorpseEnergySetOnDeath verifies that ProcessDeathQueue initializes corpse energy
// from the genome Size, not from the creature's remaining energy at time of death.
func TestCorpseEnergySetOnDeath(t *testing.T) {
	params := defaultParams()
	g := grid.NewGrid(20, 20, 0)
	g.ZeroFill()

	genome := simulation.MakeRandomGenome(params)
	genome.Size = 120
	loc := grid.Coord{X: 3, Y: 3}
	creature := simulation.NewCreature(grid.RESERVED_CELL_TYPES, loc, genome)
	creature.Energy = 5 // very low — should not affect corpse food value
	creature.Age = params.MaxJuvenilePeriod + 1 // adult: CurrentSize == genome.Size
	g.Set(loc, grid.RESERVED_CELL_TYPES)

	pop := simulation.NewPopulation(params)
	pop.Creatures[grid.RESERVED_CELL_TYPES] = creature
	pop.QueueForDeath(creature)
	pop.ProcessDeathQueue(g, params)

	if creature.Energy != float32(genome.Size) {
		t.Errorf("corpse energy should equal genome.Size (%d), got %f", genome.Size, creature.Energy)
	}
}

// TestPredationSetsCorpseEnergyFromSize verifies that a creature killed by predation
// has its corpse energy set to its genome Size.
func TestPredationSetsCorpseEnergyFromSize(t *testing.T) {
	params := defaultParams()
	g := grid.NewGrid(20, 20, 0)
	g.ZeroFill()

	preyGenome := simulation.MakeRandomGenome(params)
	preyGenome.Size = 80

	predID := grid.RESERVED_CELL_TYPES
	preyID := grid.RESERVED_CELL_TYPES + 1

	predator := simulation.NewCreature(predID, grid.Coord{X: 5, Y: 5}, simulation.MakeRandomGenome(params))
	prey := simulation.NewCreature(preyID, grid.Coord{X: 6, Y: 5}, preyGenome)
	prey.Energy = 999 // high pre-death energy — should not be the corpse value
	prey.Age = params.MaxJuvenilePeriod + 1 // adult: CurrentSize == genome.Size

	g.Set(predator.Loc, predID)
	g.Set(prey.Loc, preyID)

	pop := simulation.NewPopulation(params)
	pop.Creatures[predID] = predator
	pop.Creatures[preyID] = prey

	pop.QueueForMove(predator, prey.Loc)
	pop.ProcessMoveQueue(g, params)

	if prey.Energy != float32(preyGenome.Size) {
		t.Errorf("corpse energy after predation should equal genome.Size (%d), got %f", preyGenome.Size, prey.Energy)
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
	dead := simulation.NewCreature(grid.RESERVED_CELL_TYPES, grid.Coord{X: 1, Y: 1}, genome)
	dead.Alive = false
	pop.Creatures[grid.RESERVED_CELL_TYPES] = dead
	if pop.OldestGenome() != nil {
		t.Error("OldestGenome should return nil when all creatures are dead")
	}
}

func TestOldestGenomeReturnsOldest(t *testing.T) {
	params := defaultParams()
	pop := simulation.NewPopulation(params)
	genome := simulation.MakeRandomGenome(params)

	youngID := grid.RESERVED_CELL_TYPES
	oldID := grid.RESERVED_CELL_TYPES + 1

	young := simulation.NewCreature(youngID, grid.Coord{X: 1, Y: 1}, genome)
	young.Age = 10
	old := simulation.NewCreature(oldID, grid.Coord{X: 2, Y: 2}, genome)
	old.Age = 100

	pop.Creatures[youngID] = young
	pop.Creatures[oldID] = old

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
		id := grid.RESERVED_CELL_TYPES + i
		genome := simulation.MakeRandomGenome(params)
		loc := grid.Coord{X: i, Y: 0}
		pop.Creatures[id] = simulation.NewCreature(id, loc, genome)
	}

	diversity := pop.GeneticDiversity()
	if diversity < 0 || diversity > 1 {
		t.Errorf("GeneticDiversity out of [0,1]: %f", diversity)
	}
}

// TestReproductionPrefersSpotBehind verifies that offspring are placed 2 cells behind
// the parent (opposite of LastMoveDir) when that cell is free.
func TestReproductionPrefersSpotBehind(t *testing.T) {
	params := defaultParams()
	params.MaxPopulation = 100
	params.ReproductionEnergyCost = 0.1
	g := grid.NewGrid(20, 20, 0)
	g.ZeroFill()

	genome := simulation.MakeRandomGenome(params)
	genome.MaxEnergy = 255

	parentLoc := grid.Coord{X: 10, Y: 10}
	parentID := grid.RESERVED_CELL_TYPES
	parent := simulation.NewCreature(parentID, parentLoc, genome)
	parent.Energy = float32(genome.MaxEnergy)
	parent.LastMoveDir = grid.Dir{X: 1, Y: 0} // moving east; behind is west
	g.Set(parentLoc, parentID)

	pop := simulation.NewPopulation(params)
	pop.Creatures[parentID] = parent

	nextID := parentID + 1
	pop.QueueForReproduction(parent)
	pop.ProcessReproductionQueue(g, params, func() int {
		id := nextID
		nextID++
		return id
	})

	if len(pop.Creatures) != 2 {
		t.Fatalf("expected 2 creatures after reproduction, got %d", len(pop.Creatures))
	}

	expectedLoc := grid.Coord{X: 8, Y: 10} // 2 steps west of {10,10}
	var child *simulation.Creature
	for id, c := range pop.Creatures {
		if id != parentID {
			child = c
		}
	}
	if child == nil {
		t.Fatal("child not found in population")
	}
	if child.Loc != expectedLoc {
		t.Errorf("child should spawn at %v (2 behind parent), got %v", expectedLoc, child.Loc)
	}
}

// TestReproductionFallsBackToAdjacent verifies that offspring fall back to an adjacent
// cell when the preferred spot 2 behind is occupied.
func TestReproductionFallsBackToAdjacent(t *testing.T) {
	params := defaultParams()
	params.MaxPopulation = 100
	params.ReproductionEnergyCost = 0.1
	g := grid.NewGrid(20, 20, 0)
	g.ZeroFill()

	genome := simulation.MakeRandomGenome(params)
	genome.MaxEnergy = 255

	parentLoc := grid.Coord{X: 10, Y: 10}
	parentID := grid.RESERVED_CELL_TYPES
	parent := simulation.NewCreature(parentID, parentLoc, genome)
	parent.Energy = float32(genome.MaxEnergy)
	parent.LastMoveDir = grid.Dir{X: 1, Y: 0} // moving east
	g.Set(parentLoc, parentID)

	// Block the preferred spot 2 behind (west)
	blockedLoc := grid.Coord{X: 8, Y: 10}
	g.Set(blockedLoc, grid.WALL)

	pop := simulation.NewPopulation(params)
	pop.Creatures[parentID] = parent

	nextID := parentID + 1
	pop.QueueForReproduction(parent)
	pop.ProcessReproductionQueue(g, params, func() int {
		id := nextID
		nextID++
		return id
	})

	if len(pop.Creatures) != 2 {
		t.Fatalf("expected 2 creatures after reproduction, got %d", len(pop.Creatures))
	}

	var child *simulation.Creature
	for id, c := range pop.Creatures {
		if id != parentID {
			child = c
		}
	}
	if child == nil {
		t.Fatal("child not found in population")
	}
	if child.Loc == blockedLoc {
		t.Error("child should not spawn at blocked location")
	}
}

func TestGeneticDiversitySingleCreature(t *testing.T) {
	params := defaultParams()
	pop := simulation.NewPopulation(params)
	genome := simulation.MakeRandomGenome(params)
	pop.Creatures[grid.RESERVED_CELL_TYPES] = simulation.NewCreature(grid.RESERVED_CELL_TYPES, grid.Coord{}, genome)

	diversity := pop.GeneticDiversity()
	if diversity != 0 {
		t.Errorf("single creature diversity should be 0, got %f", diversity)
	}
}

func TestAliveCount(t *testing.T) {
	params := defaultParams()
	pop := simulation.NewPopulation(params)
	genome := simulation.MakeRandomGenome(params)

	aliveID := grid.RESERVED_CELL_TYPES
	deadID := grid.RESERVED_CELL_TYPES + 1

	alive := simulation.NewCreature(aliveID, grid.Coord{X: 1, Y: 1}, genome)
	dead := simulation.NewCreature(deadID, grid.Coord{X: 2, Y: 2}, genome)
	dead.Alive = false

	pop.Creatures[aliveID] = alive
	pop.Creatures[deadID] = dead

	if pop.AliveCount() != 1 {
		t.Errorf("AliveCount should be 1, got %d", pop.AliveCount())
	}
}
