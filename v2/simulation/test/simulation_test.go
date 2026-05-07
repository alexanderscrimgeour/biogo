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
	p.MaxPopulation = 20
	p.MinPopulation = 5
	p.FoodSpawnInterval = 10
	p.FoodPerSpawn = 5
	p.MaxFood = 500 // grid is 50x50=2500 cells; keep food well under cell count
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
}

func TestSimulationRunsContinuously(t *testing.T) {
	p := smallParams()
	sim := simulation.New(p)

	for i := 0; i < 50; i++ {
		sim.Update()
	}

	if sim.Tick != 50 {
		t.Errorf("expected Tick=50 after 50 updates, got %d", sim.Tick)
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
	if sim.PopulationCount() != p.StartingPopulation {
		t.Errorf("PopulationCount() = %d, want %d", sim.PopulationCount(), p.StartingPopulation)
	}
}

func TestSimulationFoodSpawns(t *testing.T) {
	p := smallParams()
	p.FoodSpawnInterval = 1
	p.FoodPerSpawn = 3
	sim := simulation.New(p)

	// Tick 0 spawns food during the first step
	sim.Update()
	if sim.FoodCount() == 0 {
		t.Error("expected food to spawn after first update with FoodSpawnInterval=1")
	}
}

func TestSimulationFoodViews(t *testing.T) {
	p := smallParams()
	p.FoodSpawnInterval = 1
	p.FoodPerSpawn = 5
	sim := simulation.New(p)
	sim.Update()

	views := sim.FoodViews()
	if len(views) != sim.FoodCount() {
		t.Errorf("FoodViews len %d != FoodCount %d", len(views), sim.FoodCount())
	}
	for _, v := range views {
		if v.X < 0 || v.X >= p.GridWidth || v.Y < 0 || v.Y >= p.GridHeight {
			t.Errorf("food at (%d,%d) is out of grid bounds", v.X, v.Y)
		}
	}
}

func TestSimulationCorpseViews(t *testing.T) {
	p := smallParams()
	// High metabolic rate to kill a creature immediately, no food
	p.MetabolicRate = 10000
	p.FoodSpawnInterval = 999999
	p.CorpseDecayRate = 0.001 // decay very slowly so corpses persist
	sim := simulation.New(p)

	sim.Update()

	views := sim.CorpseViews()
	for _, v := range views {
		if v.X < 0 || v.X >= p.GridWidth || v.Y < 0 || v.Y >= p.GridHeight {
			t.Errorf("corpse at (%d,%d) is out of grid bounds", v.X, v.Y)
		}
		if v.EnergyFraction < 0 || v.EnergyFraction > 1 {
			t.Errorf("corpse EnergyFraction %f out of [0,1]", v.EnergyFraction)
		}
	}
}

func TestSimulationMinPopulationMaintained(t *testing.T) {
	p := smallParams()
	p.StartingPopulation = 1
	p.MaxPopulation = 10
	p.MinPopulation = 5
	// High metabolic rate to kill creatures quickly
	p.MetabolicRate = 1000
	p.FoodSpawnInterval = 999999
	sim := simulation.New(p)

	for i := 0; i < 20; i++ {
		sim.Update()
	}

	if sim.PopulationCount() < p.MinPopulation {
		t.Errorf("population %d dropped below MinPopulation %d", sim.PopulationCount(), p.MinPopulation)
	}
}

func TestJuvenilePhaseBlocksReproduction(t *testing.T) {
	p := smallParams()
	p.MaxJuvenilePeriod = 10000 // very long juvenile phase
	p.MetabolicRate = 0
	p.MoveCost = 0
	p.ReproductionEnergyThreshold = 0.1
	p.MinPopulation = 0 // prevent auto-spawning from inflating count
	p.MaxFood = 0

	sim := simulation.New(p)

	// Force all creatures to maximum juvenile period and full energy
	for _, c := range sim.Population.Creatures {
		c.Genome.JuvenilePeriod = 255 // effective period = 10000 ticks
		c.Energy = float32(c.Genome.MaxEnergy)
	}

	initialCount := sim.PopulationCount()

	for i := 0; i < 10; i++ {
		sim.Update()
	}

	if sim.PopulationCount() > initialCount {
		t.Errorf("juvenile creatures should not reproduce: started with %d, now have %d", initialCount, sim.PopulationCount())
	}
}

func TestAdultCreaturesCanReproduce(t *testing.T) {
	p := smallParams()
	p.MaxJuvenilePeriod = 0 // no juvenile phase — all creatures are immediately adults
	p.MetabolicRate = 0
	p.MoveCost = 0
	p.ReproductionEnergyThreshold = 0.1
	p.ReproductionEnergyCost = 0.05
	p.MinPopulation = 0
	p.MaxPopulation = 200
	p.StartingPopulation = 10
	p.MaxFood = 0

	sim := simulation.New(p)

	// Fill energy so reproduction threshold is met immediately
	for _, c := range sim.Population.Creatures {
		c.Genome.JuvenilePeriod = 255 // byte value doesn't matter when MaxJuvenilePeriod = 0
		c.Energy = float32(c.Genome.MaxEnergy)
	}

	initialCount := sim.PopulationCount()

	for i := 0; i < 20; i++ {
		sim.Update()
	}

	if sim.PopulationCount() <= initialCount {
		t.Errorf("adults with sufficient energy should reproduce: started with %d, still have %d", initialCount, sim.PopulationCount())
	}
}
