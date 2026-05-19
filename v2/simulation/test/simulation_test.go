package test

import (
	"biogo/v2/simulation"
	"testing"
)

func smallParams() *simulation.Parameters {
	p := simulation.DefaultParams()
	p.World.Width = 50
	p.World.Height = 50
	p.Population.Initial = 10
	p.Population.Max = 20
	p.Population.Min = 5
	p.Food.SpawnInterval = 10
	p.Food.MaxFoliage = 200
	p.Food.MaxFungi = 0
	p.Food.FoliageFountainCount = 2
	return p
}

func TestNewSimulation(t *testing.T) {
	p := smallParams()
	sim := simulation.New(p)
	if sim == nil {
		t.Fatal("New returned nil")
	}
	if sim.World == nil {
		t.Error("simulation World should be initialized")
	}
	if sim.Population == nil {
		t.Error("simulation Population should be initialized")
	}
	if len(sim.Population.Creatures) != p.Population.Initial {
		t.Errorf("expected %d creatures, got %d", p.Population.Initial, len(sim.Population.Creatures))
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
	if len(views) != p.Population.Initial {
		t.Errorf("CreatureViews returned %d views, want %d", len(views), p.Population.Initial)
	}
	for _, v := range views {
		if v.X < 0 || v.X >= p.World.Width {
			t.Errorf("creature X=%f out of world bounds [0,%.0f)", v.X, p.World.Width)
		}
		if v.Y < 0 || v.Y >= p.World.Height {
			t.Errorf("creature Y=%f out of world bounds [0,%.0f)", v.Y, p.World.Height)
		}
	}
}

func TestSimulationInterfaceMethods(t *testing.T) {
	p := smallParams()
	sim := simulation.New(p)

	if sim.WorldWidth() != p.World.Width {
		t.Errorf("WorldWidth() = %.0f, want %.0f", sim.WorldWidth(), p.World.Width)
	}
	if sim.WorldHeight() != p.World.Height {
		t.Errorf("WorldHeight() = %.0f, want %.0f", sim.WorldHeight(), p.World.Height)
	}
	if sim.PopulationCount() != p.Population.Initial {
		t.Errorf("PopulationCount() = %d, want %d", sim.PopulationCount(), p.Population.Initial)
	}
}

func TestSimulationFoodSpawns(t *testing.T) {
	p := smallParams()
	p.Food.SpawnInterval = 1
	sim := simulation.New(p)

	sim.Update()
	if sim.PlantCount() == 0 {
		t.Error("expected plants to spawn after first update with FoodSpawnInterval=1")
	}
}

func TestSimulationFoodViews(t *testing.T) {
	p := smallParams()
	p.Food.SpawnInterval = 1
	sim := simulation.New(p)
	sim.Update()

	snap := sim.GetSnapshot()
	plantCount := 0
	for _, v := range snap.Food {
		if v.Type == simulation.FoodTypePlant {
			plantCount++
		}
	}
	if plantCount != sim.World.PlantCount() {
		t.Errorf("snapshot plant count %d != world PlantCount %d", plantCount, sim.World.PlantCount())
	}
	for _, v := range snap.Food {
		if v.X < 0 || v.X >= p.World.Width || v.Y < 0 || v.Y >= p.World.Height {
			t.Errorf("food item at (%f,%f) is out of world bounds", v.X, v.Y)
		}
	}
}

func TestSimulationMeatSpawnedOnDeath(t *testing.T) {
	p := smallParams()
	// High metabolic rate kills creatures quickly; disable food spawning.
	p.Metabolism.BaseBMR = 10000
	p.Food.SpawnInterval = 999999
	sim := simulation.New(p)

	sim.Update()

	snap := sim.GetSnapshot()
	for _, mv := range snap.Food {
		if mv.Type != simulation.FoodTypeMeat {
			continue
		}
		if mv.X < 0 || mv.X >= p.World.Width || mv.Y < 0 || mv.Y >= p.World.Height {
			t.Errorf("meat at (%f,%f) is out of world bounds", mv.X, mv.Y)
		}
		if mv.Radius <= 0 {
			t.Errorf("meat radius should be positive, got %f", mv.Radius)
		}
	}
}

func TestSimulationMinPopulationMaintained(t *testing.T) {
	p := smallParams()
	p.Population.Initial = 1
	p.Population.Max = 10
	p.Population.Min = 5
	// High metabolic rate to kill creatures quickly
	p.Metabolism.BaseBMR = 1000
	p.Food.SpawnInterval = 999999
	sim := simulation.New(p)

	for i := 0; i < 20; i++ {
		sim.Update()
	}

	if sim.PopulationCount() < p.Population.Min {
		t.Errorf("population %d dropped below MinPopulation %d", sim.PopulationCount(), p.Population.Min)
	}
}

func TestJuvenilePhaseBlocksReproduction(t *testing.T) {
	p := smallParams()
	p.Creature.MaxJuvenilePeriod = 10000 // very long juvenile phase
	p.Metabolism.BaseBMR = 0
	p.Metabolism.MoveCost = 0
	p.Reproduction.EnergyThreshold = 0.1
	p.Population.Min = 0
	p.Food.MaxFoliage = 0
	p.Food.MaxFungi = 0

	sim := simulation.New(p)

	for _, c := range sim.Population.Creatures {
		c.Genome.JuvenilePeriod = 255
		c.Energy = float32(c.Mass) * p.Metabolism.EnergyCapacityPerMass
		c.Age = 0 // reset so the full juvenile phase must elapse before reproduction
	}

	initialCount := sim.PopulationCount()

	for i := 0; i < 10; i++ {
		sim.Update()
	}

	if sim.PopulationCount() > initialCount {
		t.Errorf("juvenile creatures should not reproduce: started with %d, now have %d",
			initialCount, sim.PopulationCount())
	}
}

func TestAdultCreaturesCanReproduce(t *testing.T) {
	p := smallParams()
	p.Creature.MaxJuvenilePeriod = 0 // no juvenile phase — all creatures are immediately adults
	p.Metabolism.BaseBMR = 0
	p.Metabolism.MoveCost = 0
	p.Reproduction.EnergyThreshold = 0.1
	p.Population.Min = 0
	p.Population.Max = 200
	p.Population.Initial = 10
	p.Food.MaxFoliage = 0
	p.Food.MaxFungi = 0

	sim := simulation.New(p)

	for _, c := range sim.Population.Creatures {
		c.Genome.JuvenilePeriod = 255
		c.Genome.ReproductionType = 0   // asexual
		c.Genome.MassSplitRatio = 128   // ~25% split
		c.Energy = float32(c.Mass) * p.Metabolism.EnergyCapacityPerMass
		// Wire ENERGY sensor directly to REPRODUCE action so it fires unconditionally.
		c.Genome.Brain = append(c.Genome.Brain, simulation.Gene{
			SourceType: simulation.SENSOR,
			SourceID:   simulation.ENERGY,
			SinkType:   simulation.ACTION,
			SinkID:     simulation.REPRODUCE,
			Weight:     255,
		})
		c.CreateNeuralNet()
	}

	initialCount := sim.PopulationCount()

	for i := 0; i < 20; i++ {
		sim.Update()
	}

	if sim.PopulationCount() <= initialCount {
		t.Errorf("adults with sufficient energy should reproduce: started with %d, still have %d",
			initialCount, sim.PopulationCount())
	}
}
