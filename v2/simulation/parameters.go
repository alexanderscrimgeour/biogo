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
<<<<<<< Updated upstream
	MinSize                         byte
	MaxSize                         byte
=======
	MaxMass                         byte
>>>>>>> Stashed changes
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
	// Energy costs (absolute units, relative to creature MaxEnergy range)
	MinMetabolicRate float32 // energy drained per tick when genome MetabolicRate byte = 0
	MaxMetabolicRate float32 // energy drained per tick when genome MetabolicRate byte = 255
	MoveCost         float32 // energy drained per move attempt
	// Reproduction
	ReproductionEnergyThreshold float32 // fraction of MaxEnergy required to reproduce
	ReproductionEnergyCost      float32 // fraction of MaxEnergy deducted from parent on reproduction
	// Predation and corpses
	PreyEnergyFraction float32 // fraction of prey's energy the predator gains
	CorpseDecayRate    float32 // energy drained per tick from a dead creature's corpse
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
<<<<<<< Updated upstream
		MinSize:                         1,
		MaxSize:                         200,
=======
		MaxMass:                         255,
>>>>>>> Stashed changes
		MinStartNeuronCount:             2,
		MaxStartNeuronCount:             20,
		MinNeuronCount:                  1,
		MaxNeuronCount:                  20,
		MinHiddenLayerCount:             2,
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
<<<<<<< Updated upstream
		MinJuvenilePeriod:               100,
		MaxJuvenilePeriod:               500,
		MaxFood:                         15000,
		FoodSpawnInterval:               100,
		FoodPerSpawn:                    100,
		FoodEnergyFraction:              0.1,
		MetabolicRate:                   0.02,
		MoveCost:                        0.005,
=======
		BaseMaxAge:                      30000,
		MinJuvenilePeriod:               500,
		MaxJuvenilePeriod:               3000,
		MaxFood:                         15000,
		FoodSpawnInterval:               100,
		FoodPerSpawn:                    100,
		FoodEnergyAmount:                25.0,
		MinMetabolicRate:                0.01,
		MaxMetabolicRate:                0.1,
		MoveCost:                        0.05,
>>>>>>> Stashed changes
		ReproductionEnergyThreshold:     0.85,
		ReproductionEnergyCost:          0.2,
		PreyEnergyFraction:              0.2,
		CorpseDecayRate:                 0.05,
<<<<<<< Updated upstream
=======
		SavedGenomeProportion:           0.3,
>>>>>>> Stashed changes
	}
}
