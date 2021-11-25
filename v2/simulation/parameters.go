package simulation

var Params = &Parameters{
	StartingPopulation:              1000,
	PopulationSensorRadius:          5,
	GridWidth:                       100,
	GridHeight:                      100,
	MaxAge:                          300, // Equivalent to "Steps per generation"
	MinEnergy:                       2,   // Byte representation of the max energy a creature can have
	MaxEnergy:                       255, // Byte representation of the max energy a creature can have
	MinNeuronCount:                  5,   // > 1 | Note: This doesn't necessarily reflect the true NNet as useless neurons are culled.
	MaxNeuronCount:                  10,  // < whatever the comuter accepts
	MinHiddenLayerCount:             4,   // > 0 | Note: This doesn't necessarily reflect the true NNet as useless neurons are culled.
	MaxHiddenLayerCount:             4,   // < MaxNeuronCount
	MinSightDistance:                5,
	MaxSightDistance:                10,
	BaseMutationRate:                0.001,
	BaseGenomeMutationRate:          0.001,
	SexualReproductionSimilarityMin: 0.9,
	SexualReproductionSimilarityMax: 0.98,
	Challenge:                       LeftSurvive,
}

type Parameters struct {
	StartingPopulation              int
	GridWidth                       int
	GridHeight                      int
	PopulationSensorRadius          int // TODO: MOVE TO GENOME
	MaxAge                          int
	MinEnergy                       byte
	MaxEnergy                       byte
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
	Challenge                       ChallengeType
}
