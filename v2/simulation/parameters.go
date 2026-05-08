package simulation

import "math"

type Parameters struct {
	MaxPopulation                   int
	MinPopulation                   int
	StartingPopulation              int
	GridWidth                       int
	GridHeight                      int
	PopulationSensorRadius          int
	MinEnergy                       byte
	MaxEnergy                       byte
	MaxMass                         byte
	MinStartNeuronCount             byte
	MaxStartNeuronCount             byte
	MinNeuronCount                  byte
	MaxNeuronCount                  byte
	MinHiddenLayerCount             byte
	MaxHiddenLayerCount             byte
	MinSightDistance                byte
	MaxSightDistance                byte
	MinFieldOfView                  byte
	MaxFieldOfView                  byte
	MinMutationRate                 float32
	BaseGenomeMutationRate          float32
	SpawnMutationRate               float32
	SexualReproductionSimilarityMin float32
	SexualReproductionSimilarityMax float32
	ResponseCurveKFactor            float32
	// Age sensor reference point — creatures older than this saturate the AGE sensor at 1.0
	MaxExpectedAge int
	BaseMaxAge     int
	// Juvenile phase length in ticks. Genome JuvenilePeriod byte scales linearly into [MinJuvenilePeriod, MaxJuvenilePeriod].
	MinJuvenilePeriod int
	MaxJuvenilePeriod int
	// Food system
	MaxFood            int     // maximum food items on the grid at any time
	FoodSpawnInterval  int     // ticks between food spawns
	FoodPerSpawn       int     // food items placed per spawn event
	FoodEnergyFraction float32 // food restores this fraction of MaxEnergy
	FoodPatchRadius    float64 // world units radius of each food cluster
	FoodPatchSize      int     // maximum food items placed per cluster
	// Energy costs (absolute units, relative to creature MaxEnergy range)
	MinMetabolicRate float32 // energy drained per tick when genome MetabolicRate byte = 0
	MaxMetabolicRate float32 // energy drained per tick when genome MetabolicRate byte = 255
	MoveCost         float32 // energy drained per move attempt
	// Reproduction
	ReproductionEnergyThreshold float32
	ReproductionEnergyCost      float32
	GrowthEnergyCostFactor      float32
	// Predation and corpses
	CorpseDecayRate float32
	// Continuous-space movement
	MaxSpeedPerStep    float64 // world units per tick (max)
	MaxRotationPerStep float64 // radians per tick (max)
	// Interaction radii (world units)
	FoodInteractionRadius float64
	PredationRadius       float64
	SavedGenomeProportion float64
}

func DefaultParams() *Parameters {
	return &Parameters{
		MaxPopulation:                   20000,
		MinPopulation:                   400,
		StartingPopulation:              500,
		PopulationSensorRadius:          6,
		GridWidth:                       1000,
		GridHeight:                      600,
		MinEnergy:                       2,
		MaxEnergy:                       255,
		MaxMass:                         255,
		MinStartNeuronCount:             10,
		MaxStartNeuronCount:             26,
		MinNeuronCount:                  1,
		MaxNeuronCount:                  26,
		MinHiddenLayerCount:             3,
		MaxHiddenLayerCount:             8,
		MinSightDistance:                2,
		MaxSightDistance:                50,
		MinFieldOfView:                  10,
		MaxFieldOfView:                  180,
		MinMutationRate:                 0.0001,
		BaseGenomeMutationRate:          0.001,
		SpawnMutationRate:               0.01,
		SexualReproductionSimilarityMin: 0.9,
		SexualReproductionSimilarityMax: 0.98,
		ResponseCurveKFactor:            2,
		MaxExpectedAge:                  50000,
		BaseMaxAge:                      5000,
		MinJuvenilePeriod:               300,
		MaxJuvenilePeriod:               1000,
		MaxFood:                         15000,
		FoodSpawnInterval:               20,
		GrowthEnergyCostFactor:          0.2,
		FoodPerSpawn:                    1000,
		FoodEnergyFraction:              0.1,
		FoodPatchRadius:                 20.0,
		FoodPatchSize:                   200,
		MinMetabolicRate:                0.01,
		MaxMetabolicRate:                0.1,
		MoveCost:                        0.05,
		ReproductionEnergyThreshold:     0.85,
		ReproductionEnergyCost:          0.4,
		CorpseDecayRate:                 0.05,
		MaxSpeedPerStep:                 2.0,
		MaxRotationPerStep:              math.Pi / 4,
		FoodInteractionRadius:           3.0,
		PredationRadius:                 2.0,
		SavedGenomeProportion:           0.3,
	}
}
