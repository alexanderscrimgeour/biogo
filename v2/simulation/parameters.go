package simulation

import (
	"fmt"
	"math"
)

type Parameters struct {
	// World geometry
	GridWidth  float64
	GridHeight float64

	// Population
	MaxPopulation      int
	MinPopulation      int
	StartingPopulation int

	// Genome constraints — byte ranges the genome is clamped to at birth and mutation
	MinEnergy           byte // energy floor used in SATIATION sensor
	MaxMass             byte
	MinSpawnNeuronCount byte // brain gene count at birth
	MaxSpawnNeuronCount byte
	MinNeuronCount      byte // brain gene count bounds during mutation
	MaxNeuronCount      byte
	MinHiddenLayerCount byte
	MaxHiddenLayerCount byte
	MinSightDistance    byte
	MaxSightDistance    byte
	MinFieldOfView      byte
	MaxFieldOfView      byte
	ResponseCurveKFactor float32

	// Mutation
	MinMutationRate   float32
	SpawnMutationRate float32

	// Age / lifecycle
	BaseMaxAge        int
	MinJuvenilePeriod int
	MaxJuvenilePeriod int

	// Food system
	MaxFood               int
	FoodSpawnInterval     int
	FoodPerSpawn          int
	FoodCalories          float32 // fixed energy gained per food item consumed
	FoodPatchRadius       float64
	FoodPatchSize         int
	FoodInteractionRadius float64

	// Energy / metabolism
	BaseBMR             float32 // basal metabolic rate at MaxMass (Kleiber-scaled down for smaller mass)
	EnergyPerMassUnit   float32 // MaxEnergy = currentMass * EnergyPerMassUnit
	MoveCost            float32
	MaxSpeedPerStep     float64
	MaxRotationPerStep  float64
	MaxGrowthRatePerTick float32 // peak mass units gained per tick (von Bertalanffy)
	GrowthEnergyCostFactor float32

	// Reproduction
	ReproductionEnergyThreshold float32
	ReproductionEfficiency      float32 // energy cost = offspringMass * EnergyPerMassUnit * Efficiency

	// Predation
	PredationRadius float64
	CorpseDecayRate float32

	// Misc
	SavedGenomeProportion  float64
	PopulationSensorRadius float64
}

func DefaultParams() *Parameters {
	p := &Parameters{
		GridWidth:                   1000,
		GridHeight:                  600,
		MaxPopulation:               20000,
		MinPopulation:               10,
		StartingPopulation:          1000,
		MinEnergy:                   2,
		MaxMass:                     255,
		MinSpawnNeuronCount:         15,
		MaxSpawnNeuronCount:         50,
		MinNeuronCount:              10,
		MaxNeuronCount:              50,
		MinHiddenLayerCount:         3,
		MaxHiddenLayerCount:         8,
		MinSightDistance:            5,
		MaxSightDistance:            50,
		MinFieldOfView:              10,
		MaxFieldOfView:              180,
		ResponseCurveKFactor:        2,
		MinMutationRate:             0.0001,
		SpawnMutationRate:           0.01,
		BaseMaxAge:                  5000,
		MinJuvenilePeriod:           300,
		MaxJuvenilePeriod:           1000,
		MaxFood:                     30000,
		FoodSpawnInterval:           100,
		FoodPerSpawn:                1000,
		FoodCalories:                15.0,
		FoodPatchRadius:             20.0,
		FoodPatchSize:               200,
		FoodInteractionRadius:       3.0,
		BaseBMR:                     0.6,
		EnergyPerMassUnit:           1.0,
		MoveCost:                    0.05,
		MaxSpeedPerStep:             2.0,
		MaxRotationPerStep:          math.Pi / 4,
		MaxGrowthRatePerTick:        1.0,
		GrowthEnergyCostFactor:      0.2,
		ReproductionEnergyThreshold: 0.85,
		ReproductionEfficiency:      0.7,
		PredationRadius:             2.0,
		CorpseDecayRate:             0.05,
		SavedGenomeProportion:       0.1,
		PopulationSensorRadius:      6,
	}
	if err := p.Validate(); err != nil {
		panic(err)
	}
	return p
}

func (p *Parameters) Validate() error {
	if p.MinPopulation > p.StartingPopulation {
		return fmt.Errorf("MinPopulation (%d) > StartingPopulation (%d)", p.MinPopulation, p.StartingPopulation)
	}
	if p.StartingPopulation > p.MaxPopulation {
		return fmt.Errorf("StartingPopulation (%d) > MaxPopulation (%d)", p.StartingPopulation, p.MaxPopulation)
	}
	if p.BaseBMR < 0 {
		return fmt.Errorf("BaseBMR (%v) must be >= 0", p.BaseBMR)
	}
	if p.EnergyPerMassUnit <= 0 {
		return fmt.Errorf("EnergyPerMassUnit (%v) must be > 0", p.EnergyPerMassUnit)
	}
	if p.ReproductionEnergyThreshold <= 0 || p.ReproductionEnergyThreshold > 1 {
		return fmt.Errorf("ReproductionEnergyThreshold (%v) must be in (0, 1]", p.ReproductionEnergyThreshold)
	}
	if p.ReproductionEfficiency <= 0 || p.ReproductionEfficiency > 2 {
		return fmt.Errorf("ReproductionEfficiency (%v) must be in (0, 2]", p.ReproductionEfficiency)
	}
	if p.MinJuvenilePeriod > p.MaxJuvenilePeriod {
		return fmt.Errorf("MinJuvenilePeriod (%d) > MaxJuvenilePeriod (%d)", p.MinJuvenilePeriod, p.MaxJuvenilePeriod)
	}
	type bytePair struct {
		min, max byte
		name     string
	}
	for _, pair := range []bytePair{
		{p.MinSpawnNeuronCount, p.MaxSpawnNeuronCount, "SpawnNeuronCount"},
		{p.MinNeuronCount, p.MaxNeuronCount, "NeuronCount"},
		{p.MinHiddenLayerCount, p.MaxHiddenLayerCount, "HiddenLayerCount"},
		{p.MinSightDistance, p.MaxSightDistance, "SightDistance"},
		{p.MinFieldOfView, p.MaxFieldOfView, "FieldOfView"},
	} {
		if pair.min >= pair.max {
			return fmt.Errorf("Min%s (%d) >= Max%s (%d)", pair.name, pair.min, pair.name, pair.max)
		}
	}
	return nil
}
