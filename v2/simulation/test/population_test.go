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
	pop.ProcessMoveQueue(g)

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
	pop.ProcessMoveQueue(g)

	if creature.Loc != startLoc {
		t.Errorf("creature should not move into occupied cell, got %v", creature.Loc)
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
