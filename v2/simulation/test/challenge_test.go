package test

import (
	grid "biogo/v2/world"
	"biogo/v2/simulation"
	"testing"
)

func makeSimulation(p *simulation.Parameters) *simulation.Simulation {
	return simulation.New(p)
}

func TestPassedSurvivalCriteriaAllSurvive(t *testing.T) {
	p := defaultParams()
	p.Population.Initial = 1
	p.Population.Max = 1
	p.World.Width = 20
	p.World.Height = 20
	p.Food.MaxFoliage = 100
	p.Food.MaxFungi = 0
	sim := makeSimulation(p)

	for _, c := range sim.Population.Creatures {
		if !simulation.PassedSurvivalCriteria(c, sim, simulation.AllSurvive) {
			t.Error("AllSurvive: all creatures should pass")
		}
	}
}

func TestPassedSurvivalCriteriaLeftSurvive(t *testing.T) {
	p := defaultParams()
	p.World.Width = 20
	p.World.Height = 20
	p.Population.Initial = 0
	p.Population.Max = 0
	p.Population.Min = 0
	sim := makeSimulation(p)

	genome := simulation.MakeRandomGenome(p, 0)
	leftCreature := simulation.NewCreature(1, grid.Position{X: 1, Y: 5}, genome, p)
	rightCreature := simulation.NewCreature(2, grid.Position{X: 15, Y: 5}, genome, p)

	if !simulation.PassedSurvivalCriteria(leftCreature, sim, simulation.LeftSurvive) {
		t.Error("LeftSurvive: left-side creature should pass")
	}
	if simulation.PassedSurvivalCriteria(rightCreature, sim, simulation.LeftSurvive) {
		t.Error("LeftSurvive: right-side creature should not pass")
	}
}

func TestPassedSurvivalCriteriaRightSurvive(t *testing.T) {
	p := defaultParams()
	p.World.Width = 20
	p.World.Height = 20
	p.Population.Initial = 0
	p.Population.Max = 0
	p.Population.Min = 0
	sim := makeSimulation(p)

	genome := simulation.MakeRandomGenome(p, 0)
	leftCreature := simulation.NewCreature(1, grid.Position{X: 1, Y: 5}, genome, p)
	rightCreature := simulation.NewCreature(2, grid.Position{X: 15, Y: 5}, genome, p)

	if simulation.PassedSurvivalCriteria(leftCreature, sim, simulation.RightSurvive) {
		t.Error("RightSurvive: left-side creature should not pass")
	}
	if !simulation.PassedSurvivalCriteria(rightCreature, sim, simulation.RightSurvive) {
		t.Error("RightSurvive: right-side creature should pass")
	}
}

func TestPassedSurvivalCriteriaCenter(t *testing.T) {
	p := defaultParams()
	p.World.Width = 200
	p.World.Height = 200
	p.Population.Initial = 0
	p.Population.Max = 0
	p.Population.Min = 0
	sim := makeSimulation(p)

	genome := simulation.MakeRandomGenome(p, 0)
	centerCreature := simulation.NewCreature(1, grid.Position{X: 100, Y: 100}, genome, p)
	if !simulation.PassedSurvivalCriteria(centerCreature, sim, simulation.Center) {
		t.Error("Center: creature at center should pass")
	}
}
