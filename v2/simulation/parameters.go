package simulation

type Parameters struct {
	MaxGenerations                  int
	MaxPopulation                   int
	StartingPopulation              int
	GridWidth                       int
	GridHeight                      int
	PopulationSensorRadius          int
	MaxAge                          int
	MinEnergy                       byte
	MaxEnergy                       byte
	MinStartNeuronCount             byte
	MaxStartNeuronCount             byte
	MinNeuronCount                  byte
	MaxNeuronCount                  byte
	MinHiddenLayerCount             byte
	MaxHiddenLayerCount             byte
	MinSightDistance                byte
	MaxSightDistance                byte
	BaseMutationRate                float32
	BaseGenomeMutationRate          float32
	SexualReproductionSimilarityMin float32
	SexualReproductionSimilarityMax float32
	ResponseCurveKFactor            float32
	Challenge                       ChallengeType
}

func DefaultParams() *Parameters {
	return &Parameters{
		MaxGenerations:                  10000,
		MaxPopulation:                   1000,
		StartingPopulation:              1000,
		PopulationSensorRadius:          6,
		GridWidth:                       600,
		GridHeight:                      400,
		MaxAge:                          1000,
		MinEnergy:                       2,
		MaxEnergy:                       255,
		MinStartNeuronCount:             2,
		MaxStartNeuronCount:             20,
		MinNeuronCount:                  1,
		MaxNeuronCount:                  20,
		MinHiddenLayerCount:             2,
		MaxHiddenLayerCount:             8,
		MinSightDistance:                2,
		MaxSightDistance:                10,
		BaseMutationRate:                0.0001,
		BaseGenomeMutationRate:          0.001,
		SexualReproductionSimilarityMin: 0.9,
		SexualReproductionSimilarityMax: 0.98,
		ResponseCurveKFactor:            2,
		Challenge:                       FarLeftSurvive,
	}
}
