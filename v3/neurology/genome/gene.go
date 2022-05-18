package genome

import (
	"math"
)

const (
	OSC_PERIOD = iota
	MAX_ENERGY
	SIGHT_DISTANCE
	RESPONSIVENESS
	MUTATION_RATE
	REPRODUCTION_TYPE
	NEURON_COUNT
	NEUROLOGY_LENGTH
	GENOME_STRUCTURE_COUNT
)

// Neuron Gene represented as struct of bytes
// [00000000|00000000|00000000|00000000|00000000|00000000]
type Gene struct {
	SourceID byte
	SourceType byte
	SinkID byte
	SinkType byte
	Weight byte
}

// Genome contains the genes that build the neural network
// As well as defining the parameters that the brain operates with
// This allows mutations in parameters
type Genome struct {
	OscPeriod byte
	MaxEnergy byte
	SightDistance byte
	Responsiveness byte
	MutationRate byte
	ReproductionRate byte
	NeuronCount byte
	BrainLength byte
	Brain []*Gene
}

func (g Genome) ToByteArray() []byte {
	arr := []byte{}
	arr = append(arr, g.OscPeriod)
	arr = append(arr, g.MaxEnergy)
	arr = append(arr, g.SightDistance)
	arr = append(arr, g.Responsiveness)
	arr = append(arr, g.MutationRate)
	arr = append(arr, g.ReproductionType)
	arr = append(arr, g.NeuronCount)
	arr = append(arr, g.BrainLength)
	for _, n := range g.Brain {
		arr = append(arr, n.SourceType)
		arr = append(arr, n.SourceID)
		arr = append(arr, n.SinkType)
		arr = append(arr, n.SinkID)
	}
	return arr
}

// byteAsFloat converts from a byte to a float32 range 0...1
func byteAsFloat(val byte) float32 {
	return 2*(float32(val)/math.MaxUint8) - 1
}

func (g Gene) WeightAsFloat32() float32 {
	return byteAsFloat(g.Weight)
}