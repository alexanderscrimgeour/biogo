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
	MinEnergy                byte // energy floor used in SATIATION sensor
	MaxMass                  byte
	MinSpawnCognitiveBreadth byte // brain gene count at birth
	MaxSpawnCognitiveBreadth byte
	MinCognitiveBreadth      byte // brain gene count bounds during mutation
	MaxCognitiveBreadth      byte
	MinSynapticDensity       byte
	MaxSynapticDensity       byte
	MinSightDistance         byte
	MaxSightDistance         byte
	MinFieldOfView           byte
	MaxFieldOfView           byte
	ResponseCurveKFactor     float32

	// Mutation
	BaseMutationRate  float32
	SpawnMutationRate float32

	// Age / lifecycle
	BaseMaxAge        int
	MinJuvenilePeriod int
	MaxJuvenilePeriod int

	// Food system
	MaxFood               int
	FoodSpawnInterval     int
	FoodMass              float32 // mass of each food item consumed
	FoodInteractionRadius float64

	// Gaussian fountain spawning
	FountainCount      int     // number of drifting spawn points (3–5)
	FountainDriftSpeed float64 // world units per tick each fountain moves
	FountainRadius     float64 // Gaussian sigma for food placement around a fountain

	// Stomach / digestion
	MinStomachSize float32 // stomach capacity at StomachSize gene = 0
	MaxStomachSize float32 // stomach capacity at StomachSize gene = 255
	DigestionRate  float32 // stomach mass converted to energy per tick

	// Energy / metabolism
	BaseBMR                float32 // basal metabolic rate at MaxMass (Kleiber-scaled down for smaller mass)
	EnergyPerMassUnit      float32 // MaxEnergy = currentMass * EnergyPerMassUnit
	MoveCost               float32
	MaxSpeedPerStep        float64
	MaxRotationPerStep     float64
	MaxGrowthRatePerTick   float32 // peak mass units gained per tick (von Bertalanffy)
	GrowthEnergyCostFactor float32

	// Reproduction
	ReproductionEnergyThreshold float32
	ReproductionEfficiency      float32 // energy cost = offspringMass * EnergyPerMassUnit * Efficiency

	// Predation
	BaseBiteSize          float32 // mass consumed per eating interaction for a max-mass creature; scales linearly with body mass
	CorpseDecayRate       float32
	MinPredationMassRatio float32 // attacker must be >= this fraction of prey mass to initiate an attack
	AttackEnergyCost      float32 // energy drained from attacker per successful bite on a live creature

	// Learning
	MinNeuroplasticity   float32 // learning rate at Neuroplasticity gene = 0
	MaxNeuroplasticity   float32 // learning rate at Neuroplasticity gene = 255
	MinLearningThreshold float32 // minimum dopamine-correlation signal at LearningThreshold gene = 0
	MaxLearningThreshold float32 // minimum dopamine-correlation signal at LearningThreshold gene = 255

	// Misc
	SavedGenomeProportion  float64
	PopulationSensorRadius float64
}

func DefaultParams() *Parameters {
	p := &Parameters{
		GridWidth:                   1000,
		GridHeight:                  600,
		MaxPopulation:               20000,
		MinPopulation:               100,
		StartingPopulation:          1000,
		MinEnergy:                   2,
		MaxMass:                     255,
		MinSpawnCognitiveBreadth:    10,
		MaxSpawnCognitiveBreadth:    40,
		MinCognitiveBreadth:         5,
		MaxCognitiveBreadth:         64,
		MinSynapticDensity:          20,
		MaxSynapticDensity:          100,
		MinSightDistance:            5,
		MaxSightDistance:            50,
		MinFieldOfView:              10,
		MaxFieldOfView:              180,
		ResponseCurveKFactor:        2,
		BaseMutationRate:            0.005,
		SpawnMutationRate:           0.5,
		BaseMaxAge:                  25000,
		MinJuvenilePeriod:           300,
		MaxJuvenilePeriod:           1000,
		MaxFood:                     25000,
		FoodSpawnInterval:           50,
		FoodMass:                    10.0,
		FoodInteractionRadius:       3.0,
		FountainCount:               4,
		FountainDriftSpeed:          0.3,
		FountainRadius:              50.0,
		MinStomachSize:              5.0,
		MaxStomachSize:              100.0,
		DigestionRate:               0.5,
		BaseBMR:                     0.1,
		EnergyPerMassUnit:           1.0,
		MoveCost:                    0.01,
		MaxSpeedPerStep:             2.0,
		MaxRotationPerStep:          math.Pi / 4,
		MaxGrowthRatePerTick:        1.0,
		GrowthEnergyCostFactor:      0.2,
		ReproductionEnergyThreshold: 0.85,
		ReproductionEfficiency:      0.9,
		BaseBiteSize:                100.0,
		CorpseDecayRate:             0.05,
		MinPredationMassRatio:       0.25,
		AttackEnergyCost:            0.1,
		MinNeuroplasticity:          0.001,
		MaxNeuroplasticity:          0.05,
		MinLearningThreshold:        0.05,
		MaxLearningThreshold:        0.5,
		SavedGenomeProportion:       0.01,
		PopulationSensorRadius:      25,
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
		{p.MinSpawnCognitiveBreadth, p.MaxSpawnCognitiveBreadth, "SpawnCognitiveBreadth"},
		{p.MinCognitiveBreadth, p.MaxCognitiveBreadth, "CognitiveBreadth"},
		{p.MinSynapticDensity, p.MaxSynapticDensity, "SynapticDensity"},
		{p.MinSightDistance, p.MaxSightDistance, "SightDistance"},
		{p.MinFieldOfView, p.MaxFieldOfView, "FieldOfView"},
	} {
		if pair.min >= pair.max {
			return fmt.Errorf("Min%s (%d) >= Max%s (%d)", pair.name, pair.min, pair.name, pair.max)
		}
	}
	return nil
}
