package simulation

type Parameters struct {
	MaxPopulation                   int
	MinPopulation                   int
	StartingPopulation              int
	GridWidth                       int
	GridHeight                      int
	PopulationSensorRadius          int
	MinEnergy                       byte
	MaxEnergy                       byte
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
	// Age sensor reference point — creatures older than this saturate the AGE sensor at 1.0
	MaxExpectedAge int
	// Juvenile phase length in ticks. Genome JuvenilePeriod byte scales linearly into [MinJuvenilePeriod, MaxJuvenilePeriod].
	MinJuvenilePeriod int
	MaxJuvenilePeriod int
	// Food system
	MaxFood           int     // maximum food items on the grid at any time
	FoodSpawnInterval int     // ticks between food spawns
	FoodPerSpawn      int     // food items placed per spawn event
	FoodEnergyAmount  float32 // flat energy gained from eating food; smallest creatures give 2x this amount
	// Energy costs (absolute units, relative to creature MaxEnergy range)
	MetabolicRate float32 // energy drained per tick
	MoveCost      float32 // energy drained per move attempt
	// Reproduction
	ReproductionEnergyThreshold float32 // fraction of MaxEnergy required to reproduce
	ReproductionEnergyCost      float32 // fraction of MaxEnergy deducted from parent on reproduction
	// Predation and corpses
	CorpseDecayRate float32 // energy drained per tick from a dead creature's corpse
	// Fraction of StartingPopulation seeded from saved genomes on new game (0..1)
	SavedGenomeProportion float32
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
		MaxJuvenilePeriod:               5000,
		MaxFood:                         15000,
		FoodSpawnInterval:               100,
		FoodPerSpawn:                    100,
		FoodEnergyAmount:                25.0,
		MetabolicRate:                   0.02,
		MoveCost:                        0.005,
		ReproductionEnergyThreshold:     0.85,
		ReproductionEnergyCost:          0.6,
		CorpseDecayRate:                 0.05,
		SavedGenomeProportion:           0.5,
	}
}
