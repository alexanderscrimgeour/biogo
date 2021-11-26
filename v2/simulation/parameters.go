package simulation

var Params = &Parameters{
	MaxGenerations:                  10000, // For testing purposes
	MaxPopulation:                   1000,
	StartingPopulation:              1000,
	PopulationSensorRadius:          6,
	GridWidth:                       600,
	GridHeight:                      400,
	MaxAge:                          1000, // Equivalent to "Steps per generation"
	MinEnergy:                       2,    // Byte representation of the max energy a creature can have
	MaxEnergy:                       255,  // Byte representation of the max energy a creature can have
	MinStartNeuronCount:             2,
	MaxStartNeuronCount:             20,
	MinNeuronCount:                  1,  // > 1 | Note: This doesn't necessarily reflect the true NNet as useless neurons are culled.
	MaxNeuronCount:                  20, // < whatever the comuter accepts
	MinHiddenLayerCount:             2,  // > 0 | Note: This doesn't necessarily reflect the true NNet as useless neurons are culled.
	MaxHiddenLayerCount:             8,  // < MaxNeuronCount
	MinSightDistance:                2,
	MaxSightDistance:                10,
	BaseMutationRate:                0.0001, // Mutation rate is very small
	BaseGenomeMutationRate:          0.001,  // Not used, set in the
	SexualReproductionSimilarityMin: 0.9,
	SexualReproductionSimilarityMax: 0.98,
	ResponseCurveKFactor:            2,
	Challenge:                       FarLeftSurvive,
}

type Parameters struct {
	MaxGenerations                  int
	MaxPopulation                   int
	StartingPopulation              int
	GridWidth                       int
	GridHeight                      int
	PopulationSensorRadius          int // TODO: MOVE TO GENOME
	MaxAge                          int
	MinEnergy                       byte
	MaxEnergy                       byte
	MinStartNeuronCount             byte
	MaxStartNeuronCount             byte
	MinNeuronCount                  byte // The minimum number of neurons (connections) in the Nnet, pre removal of useless neurons
	MaxNeuronCount                  byte // The maximum number of neurons (connections) in the Nnet
	MinHiddenLayerCount             byte
	MaxHiddenLayerCount             byte
	MinSightDistance                byte
	MaxSightDistance                byte
	BaseMutationRate                float32
	BaseGenomeMutationRate          float32
	SexualReproductionSimilarityMin float32 // The minimum genome similarity required for sexual reproduction (i.e. species boundary)
	SexualReproductionSimilarityMax float32 // The maximum genome similarity required for sexual reproduction (i.e. prevent incest?)
	ResponseCurveKFactor            float32
	Challenge                       ChallengeType
}
