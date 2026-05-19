package simulation

import (
	"fmt"
	"math"
)

type WorldParameters struct {
	Width               float64
	Height              float64
	CollisionRepulsion  float64
	MaxVelocityFallback float32
	FluidDensity        float32 // ρ in drag equation: F_drag = 0.5 * ρ * v² * Cd * A
	DragCoefficient     float32 // Cd in drag equation; 0.47 is a sphere, lower for streamlined bodies
}

type PopulationParameters struct {
	Max     int
	Min     int
	Initial int
}

type SpawnParameters struct {
	SavedGenomeProportion float64
	ClusterEnabled        bool
	ClusterInterval       int
	ClusterSize           int
}

type FountainParameters struct {
	Count                int
	DriftSpeed           float64
	Radius               float64
	RandomFraction       float64
	StationaryCount int
}

type FoodParameters struct {
	// InitialEnergy is the total food energy seeded at world start. The same
	// value is used as the ongoing TargetEnergy unless the user overrides it.
	InitialEnergy float64

	// Proportions control how the energy deficit is split among food types.
	// They are automatically normalised to sum to 1 by the simulation.
	FoliageProportion float64
	FungiProportion   float64
	MeatProportion    float64

	FoliageMass          float32
	FungiMass            float32
	MeatMass             float32
	MeatDecayRate        float32
	SpawnInterval        int
	Foliage              FountainParameters
	Fungi                FountainParameters
	Meat                 FountainParameters
	FoliageEnergyDensity float32 // energy yielded per unit of foliage mass digested
	FungiEnergyDensity   float32 // energy yielded per unit of fungi mass digested
	MeatEnergyDensity    float32 // energy yielded per unit of meat mass digested; higher than plants
}

type CreatureParameters struct {
	BaseMaxAge        float64
	BaseBiteSize      float64
	BaseMaxForce      float32
	MinJuvenilePeriod int
	MaxJuvenilePeriod int
	MinSurvivalMass   float64
	MaxSurvivalMass   float64
	MinBirthMass      float64
	MaxBirthMass      float64
	MinVisionRadius   float64
	MaxVisionRadius   float64
	AttackRadius      float32
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
	MetabolicReferenceMass float32 // mass at which BaseBMR applies exactly; Kleiber scaling is normalised to this
	refMassEffect          float32 // cached: MetabolicReferenceMass^0.75
	EnergyCapacityPerMass  float32 // MaxEnergy = Mass * this; tunes how much energy a body can store
	EnergyPerFoodMass      float32 // energy gained per unit of digested food mass; tunes food richness
	BiosynthesisEfficiency float32 // fraction of EnergyPerFoodMass that becomes structural mass; EnergyCostToBuildMass = EnergyPerFoodMass / this
	MoveCostMultiplier     float32
	GrowthEnergyCostFactor float32
}

type ReproductionParameters struct {
	EnergyThreshold       float32
	Efficiency            float32
	MatingRadius          float64
	MinSimilarity         float32
	GestationTicksPerMass float32 // cooldown ticks per unit of child starting mass after reproduction
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
	TempMin                 float32
	TempMax                 float32
	WarmMetabolicMultiplier float32
	ColdSpeedMultiplier     float32
	Radiation               RadiationParameters
}

type EvolutionParameters struct {
	Tier1Generation int
	Tier2Generation int
	Tier3Generation int
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
			Width:               25000,
			Height:              15000,
			CollisionRepulsion:  0.5,
			MaxVelocityFallback: 30.0,
			FluidDensity:        0.01,
			DragCoefficient:     0.47,
		},
		Population: PopulationParameters{
			Max:     35000,
			Min:     500,
			Initial: 1000,
		},
		Food: FoodParameters{
			InitialEnergy:     3_000_000,
			FoliageProportion: 0.6,
			FungiProportion:   0.4,
			MeatProportion:    0.0,
			FoliageMass:       10.0,
			FungiMass:         50.0,
			MeatMass:          100.0,
			MeatDecayRate:     0.0001,
			SpawnInterval:     10,
			Foliage: FountainParameters{
				Count:          6,
				DriftSpeed:     0,
				Radius:         500.0,
				RandomFraction: 0.00,
			},
			Fungi: FountainParameters{
				Count:          9,
				DriftSpeed:     0.01,
				Radius:         200.0,
				RandomFraction: 0.01,
			},
			Meat: FountainParameters{
				Count:          0,
				DriftSpeed:     0,
				Radius:         300.0,
				RandomFraction: 0.00,
			},
			FoliageEnergyDensity: 10.0,
			FungiEnergyDensity:   25.0,
			MeatEnergyDensity:    100.0,
		},
		Creature: CreatureParameters{
			BaseMaxForce:      50,
			BaseMaxAge:        25000,
			BaseBiteSize:      100.0,
			MinJuvenilePeriod: 300,
			MaxJuvenilePeriod: 1000,
			MinSurvivalMass:   50,
			MaxSurvivalMass:   500,
			MinBirthMass:      60,
			MaxBirthMass:      1500,
			MinVisionRadius:   50,
			MaxVisionRadius:   500,
			AttackRadius:      10,
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
			BaseBMR:                0.1,
			MetabolicReferenceMass: 50.0, // equals MinSurvivalMass; BaseBMR is the cost at this exact body size
			EnergyCapacityPerMass:  4.0,
			EnergyPerFoodMass:      10.0,
			BiosynthesisEfficiency: 0.2, // EnergyPerFoodMass(10) / 0.2 = 50 energy per mass unit built
			MoveCostMultiplier:     0.0008,
			GrowthEnergyCostFactor: 0.2,
		},
		Reproduction: ReproductionParameters{
			EnergyThreshold:       0.85,
			Efficiency:            0.9,
			MatingRadius:          16.0,
			MinSimilarity:         0.75,
			GestationTicksPerMass: 10.0,
		},
		Predation: PredationParameters{
			AttackEnergyCost: 1.0,
		},
		Environment: EnvironmentParameters{
			TempMin:                 0.0,
			TempMax:                 50.0,
			WarmMetabolicMultiplier: 4.0,
			ColdSpeedMultiplier:     0.01,
			Radiation: RadiationParameters{
				ZoneWidth:          0.2,
				MutationMultiplier: 25.0,
				DamagePerTick:      0.1,
			},
		},
		Evolution: EvolutionParameters{
			Tier1Generation: 2,
			Tier2Generation: 5,
			Tier3Generation: 25,
		},
		Spawn: SpawnParameters{
			SavedGenomeProportion: 0.9,
			ClusterEnabled:        true,
			ClusterInterval:       100,
			ClusterSize:           5,
		},
	}
	if err := p.Validate(); err != nil {
		panic(err)
	}
	p.recomputeCachedFields()
	return p
}

// recomputeCachedFields rebuilds unexported computed fields from exported values.
// Gob encoding skips unexported fields, so this must be called after decoding
// saved parameters (see restoreState in save.go).
func (p *Parameters) recomputeCachedFields() {
	sqrtRef := math.Sqrt(float64(p.Metabolism.MetabolicReferenceMass))
	p.Metabolism.refMassEffect = float32(math.Sqrt(float64(p.Metabolism.MetabolicReferenceMass) * sqrtRef))
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
	if p.Metabolism.EnergyCapacityPerMass <= 0 {
		return fmt.Errorf("EnergyCapacityPerMass (%v) must be > 0", p.Metabolism.EnergyCapacityPerMass)
	}
	if p.Metabolism.EnergyPerFoodMass <= 0 {
		return fmt.Errorf("EnergyPerFoodMass (%v) must be > 0", p.Metabolism.EnergyPerFoodMass)
	}
	if p.Food.FoliageEnergyDensity <= 0 {
		return fmt.Errorf("FoliageEnergyDensity (%v) must be > 0", p.Food.FoliageEnergyDensity)
	}
	if p.Food.FungiEnergyDensity <= 0 {
		return fmt.Errorf("FungiEnergyDensity (%v) must be > 0", p.Food.FungiEnergyDensity)
	}
	if p.Food.MeatEnergyDensity <= 0 {
		return fmt.Errorf("MeatEnergyDensity (%v) must be > 0", p.Food.MeatEnergyDensity)
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
