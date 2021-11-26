package simulation

import (
	"fmt"
	"gopop/v2/jaro"
	"gopop/v2/utils"
	"math"
	"math/rand"
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
	// NEUROLOGY - not counted
	GENOME_STRUCTURE_COUNT
)

// All data must be expressed via a byte
type Gene struct {
	SourceID   byte
	SourceType byte
	SinkID     byte
	SinkType   byte
	Weight     byte
}

// All data must be expressed via a byte
type Genome struct {
	OscPeriod        byte
	MaxEnergy        byte
	SightDistance    byte
	Responsiveness   byte
	MutationRate     byte
	ReproductionType byte
	NeuronCount      byte // Neurons count as the middle layer in the nnet
	NeurologyLength  byte // NeurologyLength determines the number of connections
	Neurology        []*Gene
	// TODO
	// ReproductionRate <- Determines how many children
}

func (g Gene) String() string {
	return fmt.Sprintf("%b%08b%b%08b%08b", g.SourceID, g.SourceType, g.SinkID, g.SinkType, g.Weight)
}

func (g Genome) String() string {
	str := fmt.Sprintf("%08b%08b%08b%08b%b%08b", g.OscPeriod, g.SightDistance, g.Responsiveness, g.MutationRate, g.ReproductionType, g.NeurologyLength)
	for _, gene := range g.Neurology {
		str += gene.String()
	}
	return str
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
	arr = append(arr, g.NeurologyLength)
	for _, n := range g.Neurology {
		arr = append(arr, n.SourceType)
		arr = append(arr, n.SourceID)
		arr = append(arr, n.SinkType)
		arr = append(arr, n.SinkID)
	}
	return arr
}

func (g Gene) PrettyString() string {
	return fmt.Sprintf("|%b|%d|%b|%d|%08b", g.SourceType, g.SourceID, g.SinkType, g.SinkID, g.Weight)
}

func (g Genome) PrettyString() string {
	str := fmt.Sprintf("|%08b|%08d|%08b|%08b|%08b|%b|%08b", g.OscPeriod, g.MaxEnergy, g.SightDistance, g.Responsiveness, g.MutationRate, g.ReproductionType, g.NeurologyLength)
	for _, gene := range g.Neurology {
		str += gene.PrettyString()
	}
	return str
}

// WeightAsFloat converts from a byte to a float64 range 0...1
func byteAsFloat(val byte) float32 {
	return 2*(float32(val)/math.MaxUint8) - 1
}

func (g Gene) WeightAsFloat32() float32 {
	return byteAsFloat(g.Weight)
}

// makeRandomByte creates a random byte
func makeRandomBool() byte {
	return byte(rand.Uint32() >> 31)
}

// MakeRandomGene creates a random gene
func MakeRandomGene() *Gene {
	return &Gene{
		SourceType: utils.MakeRandomByte() & 1,
		SourceID:   utils.MakeRandomByte(),
		SinkType:   utils.MakeRandomByte() & 1,
		SinkID:     utils.MakeRandomByte(),
		Weight:     utils.MakeRandomByte(),
	}
}

func MakeRandomGenome() *Genome {
	g := Genome{
		OscPeriod:        utils.ClampByte(1, math.MaxUint8, utils.MakeRandomByte()), // Must be clamped above zero
		MaxEnergy:        utils.ClampByte(Params.MinEnergy, Params.MaxEnergy, utils.MakeRandomByte()),
		SightDistance:    utils.ClampByte(Params.MinSightDistance, Params.MaxSightDistance, utils.MakeRandomByte()),
		Responsiveness:   utils.MakeRandomByte(),
		MutationRate:     utils.MakeRandomByte(),
		ReproductionType: makeRandomBool(),
		NeuronCount:      utils.ClampByte(Params.MinHiddenLayerCount, Params.MaxHiddenLayerCount, utils.MakeRandomByte()),
		NeurologyLength:  utils.ClampByte(Params.MinNeuronCount, Params.MaxNeuronCount, utils.MakeRandomByte()),
	}
	for i := byte(0); i < g.NeurologyLength; i++ {
		gene := MakeRandomGene()
		g.Neurology = append(g.Neurology, gene)
	}

	return &g
}

// Copy copies a gene
func (g *Gene) Copy() *Gene {
	new := *g
	return &new
}

// Copy deep copies a genome
func (g *Genome) Copy() *Genome {
	new := *g
	temp := []*Gene{}
	for _, n := range g.Neurology {
		temp = append(temp, n.Copy())
	}
	new.Neurology = temp
	return &new
}

// Mutate takes a genome and randomly flips bits in it at the rate of Params.BaseMutationRate * g.MutationRate
func Mutate(g *Genome) {
	mutationRate := Params.BaseMutationRate * float32(g.MutationRate)

	// Super hacky fix, will need improving
	for i := 0; i < GENOME_STRUCTURE_COUNT; i++ {
		r := rand.Float32()
		if r < mutationRate {
			switch i {
			case OSC_PERIOD:
				g.OscPeriod ^= byte(1 << (rand.Uint32() >> 29))
			case MAX_ENERGY:
				new := g.MaxEnergy
				new ^= byte(1 << (rand.Uint32() >> 29))
				g.MaxEnergy = utils.ClampByte(Params.MinEnergy, Params.MaxEnergy, new)
			case SIGHT_DISTANCE:
				new := g.SightDistance
				new ^= byte(1 << (rand.Uint32() >> 29))
				g.SightDistance ^= utils.ClampByte(Params.MinSightDistance, Params.MaxSightDistance, new)
			case RESPONSIVENESS:
				g.Responsiveness ^= byte(1 << (rand.Uint32() >> 29))
			case MUTATION_RATE:
				g.MutationRate ^= byte(1 << (rand.Uint32() >> 29))
			case REPRODUCTION_TYPE:
				g.ReproductionType ^= 1
			case NEURON_COUNT:
				new := g.NeuronCount
				new ^= byte(1 << (rand.Uint32() >> 29))
				g.NeurologyLength = utils.ClampByte(Params.MinHiddenLayerCount, Params.MaxHiddenLayerCount, new)
			case NEUROLOGY_LENGTH:
				new := g.NeurologyLength
				new ^= byte(1 << (rand.Uint32() >> 29))
				g.NeurologyLength = utils.ClampByte(Params.MinNeuronCount, Params.MaxNeuronCount, new)
			}
		}
	}
	for j := 0; j < len(g.Neurology); j++ {
		r := rand.Float32()
		if r < mutationRate {
			chance := rand.Float32()
			switch {
			case chance < 0.2:
				g.Neurology[j].SourceType ^= 1
			case chance < 0.4:
				g.Neurology[j].SinkType ^= 1
			case chance < 0.6:
				g.Neurology[j].SourceID ^= byte(1 << (rand.Uint32() >> 29))
			case chance < 0.8:
				g.Neurology[j].SinkID ^= byte(1 << (rand.Uint32() >> 29))
			default:
				g.Neurology[j].Weight ^= byte(1 << (rand.Uint32() >> 29))
			}
		}
	}
	diff := int(g.NeurologyLength) - len(g.Neurology)
	if diff > 0 {
		for i := 0; i < diff; i++ {
			g.Neurology = append(g.Neurology, MakeRandomGene())
		}
	} else if diff < 0 {
		for i := 0; i > diff; i-- {
			index := rand.Intn(len(g.Neurology))
			g.Neurology = append(g.Neurology[:index], g.Neurology[index+1:]...)
		}
	}

}

// Creates a deep copy of the parent genome, then mutates it.
func AsexualReproduction(parent *Genome) *Genome {
	child := parent.Copy()
	Mutate(child)
	return child
}

// GenomeSimilarity compares two genomes using the Jaro Winkler Similiarty
func GenomeSimilarity(g1, g2 Genome) float32 {
	return jaro.JaroWinklerSimilarity(g1.String(), g2.String())
}
