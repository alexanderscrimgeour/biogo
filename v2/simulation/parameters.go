package simulation

var Params = &Parameters{
	MinEnergy:              2,   // Byte representation of the max energy a creature can have
	MaxEnergy:              255, // Byte representation of the max energy a creature can have
	MinNeuronCount:         5,   // > 1 | Note: This doesn't necessarily reflect the true NNet as useless neurons are culled.
	MaxNeuronCount:         10,  // < whatever the comuter accepts
	MinHiddenLayerCount:    4,   // > 0 | Note: This doesn't necessarily reflect the true NNet as useless neurons are culled.
	MaxHiddenLayerCount:    4,   // < MaxNeuronCount
	BaseMutationRate:       0.01,
	BaseGenomeMutationRate: 0.001,
	Challenge:              LeftSurvive,
}

type Parameters struct {
	MinEnergy              byte
	MaxEnergy              byte
	MinNeuronCount         byte // The minimum number of neurons (connections) in the Nnet, pre removal of useless neurons
	MaxNeuronCount         byte // The maximum number of neurons (connections) in the Nnet
	MinHiddenLayerCount    byte
	MaxHiddenLayerCount    byte
	BaseMutationRate       float32
	BaseGenomeMutationRate float32
	Challenge              ChallengeType
}
