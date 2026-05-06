package test

import (
	"biogo/v2/simulation"
	"testing"
)

func smallParams() *simulation.Parameters {
	p := simulation.DefaultParams()
	p.GridWidth = 50
	p.GridHeight = 50
	p.StartingPopulation = 10
	p.MaxPopulation = 10
	p.MaxAge = 5
	p.MaxGenerations = 10
	p.Challenge = simulation.AllSurvive
	return p
}

func TestNewSimulation(t *testing.T) {
	p := smallParams()
	sim := simulation.New(p)
	if sim == nil {
		t.Fatal("New returned nil")
	}
	if sim.Grid == nil {
		t.Error("simulation Grid should be initialized")
	}
	if sim.Population == nil {
		t.Error("simulation Population should be initialized")
	}
	if len(sim.Population.Creatures) != p.StartingPopulation {
		t.Errorf("expected %d creatures, got %d", p.StartingPopulation, len(sim.Population.Creatures))
	}
	if sim.Tick != 0 {
		t.Errorf("initial Tick should be 0, got %d", sim.Tick)
	}
	if sim.Generation != 0 {
		t.Errorf("initial Generation should be 0, got %d", sim.Generation)
	}
}

func TestSimulationRunsFullGeneration(t *testing.T) {
	p := smallParams()
	sim := simulation.New(p)

	for i := 0; i <= p.MaxAge; i++ {
		sim.Update()
	}

	if sim.Generation != 1 {
		t.Errorf("expected Generation 1 after one cycle, got %d", sim.Generation)
	}
	if sim.Tick != 0 {
		t.Errorf("Tick should reset to 0 after new generation, got %d", sim.Tick)
	}
}

func TestSimulationCreatureViews(t *testing.T) {
	p := smallParams()
	sim := simulation.New(p)

	views := sim.CreatureViews()
	if len(views) != p.StartingPopulation {
		t.Errorf("CreatureViews returned %d views, want %d", len(views), p.StartingPopulation)
	}
	for _, v := range views {
		if v.X < 0 || v.X >= p.GridWidth {
			t.Errorf("creature X=%d out of grid bounds [0,%d)", v.X, p.GridWidth)
		}
		if v.Y < 0 || v.Y >= p.GridHeight {
			t.Errorf("creature Y=%d out of grid bounds [0,%d)", v.Y, p.GridHeight)
		}
	}
}

func TestSimulationInterfaceMethods(t *testing.T) {
	p := smallParams()
	sim := simulation.New(p)

	if sim.GridWidth() != p.GridWidth {
		t.Errorf("GridWidth() = %d, want %d", sim.GridWidth(), p.GridWidth)
	}
	if sim.GridHeight() != p.GridHeight {
		t.Errorf("GridHeight() = %d, want %d", sim.GridHeight(), p.GridHeight)
	}
	if sim.CurrentGeneration() != 0 {
		t.Errorf("CurrentGeneration() = %d, want 0", sim.CurrentGeneration())
	}
	if sim.PopulationCount() != p.StartingPopulation {
		t.Errorf("PopulationCount() = %d, want %d", sim.PopulationCount(), p.StartingPopulation)
	}
}
