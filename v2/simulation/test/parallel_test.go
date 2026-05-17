package test

import (
	"biogo/v2/simulation"
	"testing"
)

func BenchmarkSimulationStep(b *testing.B) {
	p := simulation.DefaultParams()
	p.Population.Initial = 1000
	p.Population.Max = 5000
	p.Population.Min = 0
	sim := simulation.New(p)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.Update()
	}
}

func BenchmarkSimulationStep100(b *testing.B) {
	p := simulation.DefaultParams()
	p.Population.Initial = 100
	p.Population.Max = 500
	p.Population.Min = 0
	sim := simulation.New(p)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.Update()
	}
}

func TestAliveIDs(t *testing.T) {
	p := defaultParams()
	sim := simulation.New(p)

	ids := sim.Population.AliveIDs()
	if len(ids) != p.Population.Initial {
		t.Errorf("AliveIDs returned %d IDs, want %d", len(ids), p.Population.Initial)
	}

	seen := make(map[int]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			t.Errorf("duplicate ID %d in AliveIDs", id)
		}
		seen[id] = true
		c, ok := sim.Population.Get(id)
		if !ok {
			t.Errorf("ID %d from AliveIDs not found in Creatures", id)
		}
		if !c.Alive {
			t.Errorf("ID %d from AliveIDs is not alive", id)
		}
	}
}

func TestParallelStepMatchesExpectedBehaviour(t *testing.T) {
	p := defaultParams()
	p.Metabolism.BaseBMR = 10000
	p.Food.SpawnInterval = 999999
	p.Population.Min = 0

	sim := simulation.New(p)
	initialCount := sim.PopulationCount()
	if initialCount == 0 {
		t.Fatal("no creatures at start")
	}

	sim.Update()

	// With lethal metabolic rate all creatures must be dead or queued for death.
	if sim.PopulationCount() > 0 {
		t.Errorf("expected 0 alive after lethal metabolism tick, got %d", sim.PopulationCount())
	}
}
