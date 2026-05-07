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
	MinSize                         byte
	MaxSize                         byte
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
	BaseMutationRate                float32
	BaseGenomeMutationRate          float32
	SexualReproductionSimilarityMin float32
	SexualReproductionSimilarityMax float32
	ResponseCurveKFactor            float32
	MaxExpectedAge                  int
	MinJuvenilePeriod               int
	MaxJuvenilePeriod               int
	// Food system
	MaxFood           int
	FoodSpawnInterval int
	FoodPerSpawn      int
	FoodEnergyFraction float32
	// Energy costs
	MetabolicRate float32
	MoveCost      float32
	// Reproduction
	ReproductionEnergyThreshold float32
	ReproductionEnergyCost      float32
	// Predation and corpses
	PreyEnergyFraction float32
	CorpseDecayRate    float32
	// Continuous-space movement
	MaxSpeedPerStep      float64 // world units per tick (max)
	MaxRotationPerStep   float64 // radians per tick (max)
	// Interaction radii (world units)
	FoodInteractionRadius float64
	PredationRadius       float64
}

func DefaultParams() *Parameters {
	return &Parameters{
		MaxPopulation:                   5000,
		MinPopulation:                   400,
		StartingPopulation:              500,
		PopulationSensorRadius:          6,
		GridWidth:                       1000,
		GridHeight:                      600,
		MinEnergy:                       2,
		MaxEnergy:                       255,
		MinSize:                         1,
		MaxSize:                         200,
		MinStartNeuronCount:             2,
		MaxStartNeuronCount:             20,
		MinNeuronCount:                  1,
		MaxNeuronCount:                  20,
		MinHiddenLayerCount:             2,
		MaxHiddenLayerCount:             8,
		MinSightDistance:                2,
		MaxSightDistance:                20,
		MinFieldOfView:                  10,
		MaxFieldOfView:                  180,
		BaseMutationRate:                0.001,
		BaseGenomeMutationRate:          0.001,
		SexualReproductionSimilarityMin: 0.9,
		SexualReproductionSimilarityMax: 0.98,
		ResponseCurveKFactor:            2,
		MaxExpectedAge:                  50000,
		MinJuvenilePeriod:               100,
		MaxJuvenilePeriod:               500,
		MaxFood:                         15000,
		FoodSpawnInterval:               100,
		FoodPerSpawn:                    100,
		FoodEnergyFraction:              0.1,
		MetabolicRate:                   0.02,
		MoveCost:                        0.005,
		ReproductionEnergyThreshold:     0.85,
		ReproductionEnergyCost:          0.2,
		PreyEnergyFraction:              0.2,
		CorpseDecayRate:                 0.05,
		MaxSpeedPerStep:                 2.0,
		MaxRotationPerStep:              math.Pi / 4,
		FoodInteractionRadius:           3.0,
		PredationRadius:                 4.0,
	}
}
