package test

import (
	"biogo/v2/grid"
	"biogo/v2/simulation"
	"testing"
)

func makeSimulation(p *simulation.Parameters) *simulation.Simulation {
	return simulation.New(p)
}

func TestPassedSurvivalCriteriaAllSurvive(t *testing.T) {
	p := defaultParams()
	p.StartingPopulation = 1
	p.MaxPopulation = 1
	p.GridWidth = 20
	p.GridHeight = 20
	p.MaxFood = 100 // grid is 20x20=400 cells; keep food well under cell count
	sim := makeSimulation(p)

	for _, c := range sim.Population.Creatures {
		if !simulation.PassedSurvivalCriteria(c, sim, simulation.AllSurvive) {
			t.Error("AllSurvive: all creatures should pass")
		}
	}
}

func TestPassedSurvivalCriteriaLeftSurvive(t *testing.T) {
	p := defaultParams()
	p.GridWidth = 20
	p.GridHeight = 20
	p.StartingPopulation = 0
	p.MaxPopulation = 0
	p.MinPopulation = 0
	sim := makeSimulation(p)

	genome := simulation.MakeRandomGenome(p)
	leftCreature := simulation.NewCreature(grid.RESERVED_CELL_TYPES, grid.Coord{X: 1, Y: 5}, genome)
	rightCreature := simulation.NewCreature(grid.RESERVED_CELL_TYPES+1, grid.Coord{X: 15, Y: 5}, genome)

	if !simulation.PassedSurvivalCriteria(leftCreature, sim, simulation.LeftSurvive) {
		t.Error("LeftSurvive: left-side creature should pass")
	}
	if simulation.PassedSurvivalCriteria(rightCreature, sim, simulation.LeftSurvive) {
		t.Error("LeftSurvive: right-side creature should not pass")
	}
}

func TestPassedSurvivalCriteriaRightSurvive(t *testing.T) {
	p := defaultParams()
	p.GridWidth = 20
	p.GridHeight = 20
	p.StartingPopulation = 0
	p.MaxPopulation = 0
	p.MinPopulation = 0
	sim := makeSimulation(p)

	genome := simulation.MakeRandomGenome(p)
	leftCreature := simulation.NewCreature(grid.RESERVED_CELL_TYPES, grid.Coord{X: 1, Y: 5}, genome)
	rightCreature := simulation.NewCreature(grid.RESERVED_CELL_TYPES+1, grid.Coord{X: 15, Y: 5}, genome)

	if simulation.PassedSurvivalCriteria(leftCreature, sim, simulation.RightSurvive) {
		t.Error("RightSurvive: left-side creature should not pass")
	}
	if !simulation.PassedSurvivalCriteria(rightCreature, sim, simulation.RightSurvive) {
		t.Error("RightSurvive: right-side creature should pass")
	}
}

func TestPassedSurvivalCriteriaCenter(t *testing.T) {
	p := defaultParams()
	p.GridWidth = 20
	p.GridHeight = 20
	p.StartingPopulation = 0
	p.MaxPopulation = 0
	p.MinPopulation = 0
	sim := makeSimulation(p)

	genome := simulation.MakeRandomGenome(p)
	// Radius is 50, grid is 20x20, so center is (10,10) and everything within 50 units passes
	centerCreature := simulation.NewCreature(grid.RESERVED_CELL_TYPES, grid.Coord{X: 10, Y: 10}, genome)
	if !simulation.PassedSurvivalCriteria(centerCreature, sim, simulation.Center) {
		t.Error("Center: creature at center should pass")
	}
}
