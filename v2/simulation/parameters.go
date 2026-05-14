package simulation

import (
	"fmt"
	"math"
)

type Parameters struct {
	// World geometry
	WorldWidth  float64
	WorldHeight float64

	// Population
	MaxPopulation      int
	MinPopulation      int
	StartingPopulation int

	// Genome constraints — byte ranges the genome is clamped to at birth and mutation
	MinEnergy                byte // energy floor used in SATIATION sensor
	MaxMass                  float64
	MinSpawnCognitiveBreadth byte // brain gene count at birth
	MaxSpawnCognitiveBreadth byte
	MinCognitiveBreadth      byte // brain gene count bounds during mutation
	MaxCognitiveBreadth      byte
	MinSynapticDensity       byte
	MaxSynapticDensity       byte
	MinSightDistance         float64
	MaxSightDistance         float64
	MinFieldOfView           float64
	MaxFieldOfView           float64
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
	MinStomachSize float64 // stomach capacity at StomachSize gene = 0
	MaxStomachSize float64 // stomach capacity at StomachSize gene = 255
	DigestionRate  float64 // stomach mass converted to energy per tick

	// Energy / metabolism
	BaseBMR                float32 // basal metabolic rate at MaxMass (Kleiber-scaled down for smaller mass)
	EnergyPerMassUnit      float32 // MaxEnergy = currentMass * EnergyPerMassUnit
	MoveCost               float32
	MaxSpeedPerStep        float64
	VelocityDamping        float64 // fraction of velocity retained each tick (drag/friction coefficient)
	MaxRotationPerStep     float64
	MaxGrowthRatePerTick   float32 // peak mass units gained per tick (von Bertalanffy)
	GrowthEnergyCostFactor float32

	// Reproduction
	ReproductionEnergyThreshold float32
	ReproductionEfficiency      float32 // energy cost = offspringMass * EnergyPerMassUnit * Efficiency
	MatingRadius                float64 // max distance between two creatures for sexual mating to occur
	MinMatingSimilarity         float32 // minimum GenomeSimilarity [0,1] required for a pair to mate

	// Predation
	BaseBiteSize          float64 // mass consumed per eating interaction for a max-mass creature; scales linearly with body mass
	CorpseDecayRate       float32
	MinPredationMassRatio float32 // attacker must be >= this fraction of prey mass to initiate an attack
	AttackEnergyCost      float32 // energy drained from attacker per successful bite on a live creature

	// Learning
	MinNeuroplasticity   float32 // learning rate at Neuroplasticity gene = 0
	MaxNeuroplasticity   float32 // learning rate at Neuroplasticity gene = 255
	MinLearningThreshold float32 // minimum dopamine-correlation signal at LearningThreshold gene = 0
	MaxLearningThreshold float32 // minimum dopamine-correlation signal at LearningThreshold gene = 255

	// Temperature effects
	ColdMetabolicMultiplier float32 // metabolic cost multiplier at TempCold (10°C)
	WarmMetabolicMultiplier float32 // metabolic cost multiplier at TempWarm (40°C)
	ColdSpeedMultiplier     float32 // fraction of max move speed at TempCold (10°C); 1.0 at TempWarm

	// Radiation zone (left strip of the world)
	RadiationZoneWidth          float64 // fraction of world width that is radioactive [0, 1]
	RadiationMutationMultiplier float32 // multiplier applied to offspring mutation rate when parent is in zone
	RadiationDamagePerTick      float32 // base energy drained per tick (Kleiber-scaled) while in zone

	// Collisions
	CollisionRepulsion float64 // fraction of overlap resolved per tick [0, 1]; 0 = disabled

	// Misc
	SavedGenomeProportion  float64
	PopulationSensorRadius float64
}

func DefaultParams() *Parameters {
	p := &Parameters{
		WorldWidth:                  8000,
		WorldHeight:                 8000,
		MaxPopulation:               25000,
		MinPopulation:               500,
		StartingPopulation:          1000,
		MinEnergy:                   2,
		MaxMass:                     1000,
		MinSpawnCognitiveBreadth:    10,
		MaxSpawnCognitiveBreadth:    40,
		MinCognitiveBreadth:         5,
		MaxCognitiveBreadth:         64,
		MinSynapticDensity:          5,
		MaxSynapticDensity:          100,
		MinSightDistance:            50,
		MaxSightDistance:            250,
		MinFieldOfView:              10,
		MaxFieldOfView:              180,
		ResponseCurveKFactor:        2,
		BaseMutationRate:            0.005,
		SpawnMutationRate:           0.5,
		BaseMaxAge:                  25000,
		MinJuvenilePeriod:           300,
		MaxJuvenilePeriod:           1000,
		MaxFood:                     50000,
		FoodSpawnInterval:           100,
		FoodMass:                    5.0,
		FountainCount:               20,
		FountainDriftSpeed:          0.02,
		FountainRadius:              200.0,
		DigestionRate:               0.5,
		BaseBMR:                     0.2,
		EnergyPerMassUnit:           1.0,
		MoveCost:                    0.01,
		MaxSpeedPerStep:             10.0,
		VelocityDamping:             0.85,
		MaxRotationPerStep:          math.Pi / 4,
		MaxGrowthRatePerTick:        1.0,
		GrowthEnergyCostFactor:      0.2,
		ReproductionEnergyThreshold: 0.85,
		ReproductionEfficiency:      0.9,
		MatingRadius:                16.0,
		MinMatingSimilarity:         0.75,
		BaseBiteSize:                20.0,
		CorpseDecayRate:             0.05,
		MinPredationMassRatio:       0.25,
		AttackEnergyCost:            1.0,
		MinNeuroplasticity:          0.0005,
		MaxNeuroplasticity:          0.001,
		MinLearningThreshold:        0.05,
		MaxLearningThreshold:        0.5,
		ColdMetabolicMultiplier:     1.8,
		WarmMetabolicMultiplier:     0.8,
		ColdSpeedMultiplier:         0.4,
		RadiationZoneWidth:          0.2,
		RadiationMutationMultiplier: 25.0,
		RadiationDamagePerTick:      0.1,
		CollisionRepulsion:          0.5,
		SavedGenomeProportion:       0.5,
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

	type rangePair struct {
		min, max float64
		name     string
	}

	ranges := []rangePair{
		{float64(p.MinJuvenilePeriod), float64(p.MaxJuvenilePeriod), "JuvenilePeriod"},
		{p.MinSightDistance, p.MaxSightDistance, "SightDistance"},
		{p.MinFieldOfView, p.MaxFieldOfView, "FieldOfView"},
		{float64(p.MinSpawnCognitiveBreadth), float64(p.MaxSpawnCognitiveBreadth), "SpawnCognitiveBreadth"},
		{float64(p.MinCognitiveBreadth), float64(p.MaxCognitiveBreadth), "CognitiveBreadth"},
		{float64(p.MinSynapticDensity), float64(p.MaxSynapticDensity), "SynapticDensity"},
	}

	for _, pair := range ranges {
		if pair.min > pair.max {
			return fmt.Errorf("Min%s (%v) > Max%s (%v)", pair.name, pair.min, pair.name, pair.max)
		}
	}

	return nil
}
