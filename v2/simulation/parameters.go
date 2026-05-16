package simulation

import (
	"fmt"
)

type WorldParameters struct {
	Width              float64
	Height             float64
	CollisionRepulsion float64
}

type PopulationParameters struct {
	Max     int
	Min     int
	Initial int
}

type SpawnParameters struct {
	SavedGenomeProportion float64
}

type FoodParameters struct {
	Max                int
	BaseMass           float32
	SpawnInterval      int
	FountainCount      int
	FountainDriftSpeed float64
	FountainRadius     float64
	RandomFraction     float64
}

type CreatureParameters struct {
	BaseMaxAge        float64
	BaseBiteSize      float64
	BaseMaxForce      float64
	MinJuvenilePeriod int
	MaxJuvenilePeriod int
	MinMass           float64
	MaxMass           float64
	MinVisionRadius   float64
	MaxVisionRadius   float64
	MinFieldOfView    float64
	MaxFieldOfView    float64
}

type NeurologyParameters struct {
	MinEnergy            byte
	MinCognitiveBreadth  byte
	MaxCognitiveBreadth  byte
	MinSynapticDensity   byte
	MaxSynapticDensity   byte
	BaseMutationRate     float32
	ResponseCurveKFactor float32
	// Learning
	MinNeuroplasticity   float32
	MaxNeuroplasticity   float32
	MinLearningThreshold float32
	MaxLearningThreshold float32
}

type MetabolismParameters struct {
	MinStomachSize         float64
	MaxStomachSize         float64
	DigestionRate          float64
	BaseBMR                float32
	EnergyPerMassUnit      float32
	MoveCost               float32
	SpeedDamping           float64
	MaxGrowthRatePerTick   float32
	GrowthEnergyCostFactor float32
}

type ReproductionParameters struct {
	EnergyThreshold float32
	Efficiency      float32
	MatingRadius    float64
	MinSimilarity   float32
}

type PredationParameters struct {
	AttackEnergyCost float32
}

type RadiationParameters struct {
	ZoneWidth          float64
	MutationMultiplier float32
	DamagePerTick      float32
}

type EnvironmentParameters struct {
	ColdMetabolicMultiplier float32
	WarmMetabolicMultiplier float32
	ColdSpeedMultiplier     float32
	Radiation               RadiationParameters
}

type EvolutionParameters struct {
	Tier2Generation int
	Tier3Generation int
	Tier4Generation int
}

type Parameters struct {
	World        WorldParameters
	Population   PopulationParameters
	Food         FoodParameters
	Creature     CreatureParameters
	Neurology    NeurologyParameters
	Metabolism   MetabolismParameters
	Reproduction ReproductionParameters
	Predation    PredationParameters
	Environment  EnvironmentParameters
	Evolution    EvolutionParameters
	Spawn        SpawnParameters
}

func DefaultParams() *Parameters {
	p := &Parameters{
		World: WorldParameters{
			Width:              15000,
			Height:             15000,
			CollisionRepulsion: 0.5,
		},
		Population: PopulationParameters{
			Max:     35000,
			Min:     500,
			Initial: 1000,
		},
		Food: FoodParameters{
			Max:                100000,
			BaseMass:           25.0,
			SpawnInterval:      10,
			FountainCount:      20,
			FountainDriftSpeed: 10,
			FountainRadius:     400.0,
			RandomFraction:     0.05,
		},
		Creature: CreatureParameters{
			BaseMaxForce:      2.5,
			BaseMaxAge:        25000,
			BaseBiteSize:      100.0,
			MinJuvenilePeriod: 300,
			MaxJuvenilePeriod: 1000,
			MinMass:           3,
			MaxMass:           750,
			MinVisionRadius:   50,
			MaxVisionRadius:   250,
			MinFieldOfView:    10,
			MaxFieldOfView:    180,
		},
		Neurology: NeurologyParameters{
			MinCognitiveBreadth:  0,
			MaxCognitiveBreadth:  0,
			MinSynapticDensity:   10,
			MaxSynapticDensity:   100,
			BaseMutationRate:     0.005,
			ResponseCurveKFactor: 2,
			MinNeuroplasticity:   0.0005,
			MaxNeuroplasticity:   0.001,
			MinLearningThreshold: 0.05,
			MaxLearningThreshold: 0.5,
		},
		Metabolism: MetabolismParameters{
			DigestionRate:          0.2,
			BaseBMR:                0.05,
			EnergyPerMassUnit:      1.0,
			MoveCost:               0.01,
			SpeedDamping:           0.85,
			MaxGrowthRatePerTick:   1.0,
			GrowthEnergyCostFactor: 0.2,
		},
		Reproduction: ReproductionParameters{
			EnergyThreshold: 0.85,
			Efficiency:      0.9,
			MatingRadius:    16.0,
			MinSimilarity:   0.75,
		},
		Predation: PredationParameters{
			AttackEnergyCost: 1.0,
		},
		Environment: EnvironmentParameters{
			ColdMetabolicMultiplier: 0.8,
			WarmMetabolicMultiplier: 4.0,
			ColdSpeedMultiplier:     0.4,
			Radiation: RadiationParameters{
				ZoneWidth:          0.2,
				MutationMultiplier: 25.0,
				DamagePerTick:      0.1,
			},
		},
		Evolution: EvolutionParameters{
			Tier2Generation: 3,
			Tier3Generation: 10,
			Tier4Generation: 50,
		},
		Spawn: SpawnParameters{
			SavedGenomeProportion: 0.05,
		},
	}
	if err := p.Validate(); err != nil {
		panic(err)
	}
	return p
}

func (p *Parameters) Validate() error {
	if p.Population.Min > p.Population.Initial {
		return fmt.Errorf("MinPopulation (%d) > StartingPopulation (%d)", p.Population.Min, p.Population.Initial)
	}
	if p.Population.Initial > p.Population.Max {
		return fmt.Errorf("StartingPopulation (%d) > MaxPopulation (%d)", p.Population.Initial, p.Population.Max)
	}
	if p.Metabolism.BaseBMR < 0 {
		return fmt.Errorf("BaseBMR (%v) must be >= 0", p.Metabolism.BaseBMR)
	}
	if p.Metabolism.EnergyPerMassUnit <= 0 {
		return fmt.Errorf("EnergyPerMassUnit (%v) must be > 0", p.Metabolism.EnergyPerMassUnit)
	}

	if p.Reproduction.EnergyThreshold <= 0 || p.Reproduction.EnergyThreshold > 1 {
		return fmt.Errorf("ReproductionEnergyThreshold (%v) must be in (0, 1]", p.Reproduction.EnergyThreshold)
	}
	if p.Reproduction.Efficiency <= 0 || p.Reproduction.Efficiency > 2 {
		return fmt.Errorf("ReproductionEfficiency (%v) must be in (0, 2]", p.Reproduction.Efficiency)
	}

	type rangePair struct {
		min, max float64
		name     string
	}

	ranges := []rangePair{
		{float64(p.Creature.MinJuvenilePeriod), float64(p.Creature.MaxJuvenilePeriod), "JuvenilePeriod"},
		{p.Creature.MinVisionRadius, p.Creature.MaxVisionRadius, "VisionRadius"},
		{p.Creature.MinFieldOfView, p.Creature.MaxFieldOfView, "FieldOfView"},
		{float64(p.Neurology.MinCognitiveBreadth), float64(p.Neurology.MaxCognitiveBreadth), "CognitiveBreadth"},
		{float64(p.Neurology.MinSynapticDensity), float64(p.Neurology.MaxSynapticDensity), "SynapticDensity"},
	}

	for _, pair := range ranges {
		if pair.min > pair.max {
			return fmt.Errorf("Min%s (%v) > Max%s (%v)", pair.name, pair.min, pair.name, pair.max)
		}
	}

	return nil
}
