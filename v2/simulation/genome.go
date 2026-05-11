package simulation

import (
	"biogo/v2/utils"
	"fmt"
	"math"
	"math/rand"
)

const (
	OSC_PERIOD = iota
	SIGHT_DISTANCE
	FIELD_OF_VIEW
	RESPONSIVENESS
	MUTATION_RATE
	MASS
	MIN_MASS
	REPRODUCTION_TYPE
	NEURON_COUNT
	NEUROLOGY_LENGTH
	JUVENILE_PERIOD
	METABOLIC_RATE
	STOMACH_SIZE
	LEARNING_RATE
	LEARNING_THRESHOLD
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
	OscPeriod         byte
	SightDistance     byte
	FieldOfView       byte // total FOV angle in degrees (0–180)
	Responsiveness    byte
	MutationRate      byte
	Mass              byte
	MinMass           byte // birth mass; scales linearly to Mass over the juvenile period
	ReproductionType  byte
	NeuronCount       byte
	BrainLength       byte
	JuvenilePeriod    byte
	MetabolicRate     byte
	StomachSize       byte // controls stomach capacity; maps to [MinStomachSize, MaxStomachSize]
	LearningRate      byte // base learning rate; maps to [MinLearningRate, MaxLearningRate]
	LearningThreshold byte // minimum learning signal to update a weight; maps to [MinLearningThreshold, MaxLearningThreshold]
	Brain             []*Gene
}

func (g Gene) String() string {
	return fmt.Sprintf("%b%08b%b%08b%08b", g.SourceType, g.SourceID, g.SinkType, g.SinkID, g.Weight)
}

func (g Genome) String() string {
	str := fmt.Sprintf("%08b%08b%08b%08b%08b%08b%08b%b%08b%08b%08b%08b%08b%08b", g.OscPeriod, g.SightDistance, g.FieldOfView, g.Responsiveness, g.MutationRate, g.Mass, g.MinMass, g.ReproductionType, g.BrainLength, g.JuvenilePeriod, g.MetabolicRate, g.StomachSize, g.LearningRate, g.LearningThreshold)
	for _, gene := range g.Brain {
		str += gene.String()
	}
	return str
}

func (g Gene) BinaryString() string {
	return fmt.Sprintf("|%b|%08b|%b|%08b|%08b", g.SourceType, g.SourceID, g.SinkType, g.SinkID, g.Weight)
}

func (g Genome) BinaryString() string {
	str := fmt.Sprintf("%08b|%08b|%08b|%08b|%08b|%08b|%08b|%b|%08b|%08b|%08b|%08b|%08b|%08b", g.OscPeriod, g.SightDistance, g.FieldOfView, g.Responsiveness, g.MutationRate, g.Mass, g.MinMass, g.ReproductionType, g.BrainLength, g.JuvenilePeriod, g.MetabolicRate, g.StomachSize, g.LearningRate, g.LearningThreshold)
	for _, gene := range g.Brain {
		str += gene.BinaryString()
	}
	return str
}

func (g Genome) ToByteArray() []byte {
	arr := []byte{}
	arr = append(arr, g.OscPeriod)
	arr = append(arr, g.SightDistance)
	arr = append(arr, g.FieldOfView)
	arr = append(arr, g.Responsiveness)
	arr = append(arr, g.MutationRate)
	arr = append(arr, g.Mass)
	arr = append(arr, g.MinMass)
	arr = append(arr, g.ReproductionType)
	arr = append(arr, g.NeuronCount)
	arr = append(arr, g.BrainLength)
	arr = append(arr, g.JuvenilePeriod)
	arr = append(arr, g.MetabolicRate)
	arr = append(arr, g.StomachSize)
	arr = append(arr, g.LearningRate)
	arr = append(arr, g.LearningThreshold)
	for _, n := range g.Brain {
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
	str := fmt.Sprintf("|%08b|%08b|%08b|%08b|%08b|%08b|%08b|%b|%08b|%08b|%08b|%08b|%08b|%08b", g.OscPeriod, g.SightDistance, g.FieldOfView, g.Responsiveness, g.MutationRate, g.Mass, g.MinMass, g.ReproductionType, g.BrainLength, g.JuvenilePeriod, g.MetabolicRate, g.StomachSize, g.LearningRate, g.LearningThreshold)
	for _, gene := range g.Brain {
		str += gene.PrettyString()
	}
	return str
}

// byteAsFloat converts from a byte to a float32 range -1...1
func byteAsFloat(val byte) float32 {
	return 2*(float32(val)/math.MaxUint8) - 1
}

func (g Gene) WeightAsFloat32() float32 {
	return byteAsFloat(g.Weight)
}

func makeRandomBool() byte {
	return byte(rand.Uint32() >> 31)
}

// MakeRandomGene creates a random gene
func MakeRandomGene() *Gene {
	return &Gene{
		SourceType: utils.MakeRandomByte() & 1,
		SourceID:   utils.MakeRandomByte(),
		SinkType:   byte(rand.Intn(2) * 2),
		SinkID:     utils.MakeRandomByte(),
		Weight:     utils.MakeRandomByte(),
	}
}

func MakeRandomGenome(p *Parameters) *Genome {
	// Mass must be >= 3 to guarantee a valid MinMass (MinMass < Mass/2 requires Mass > 2).
	mass := utils.ClampByte(3, p.MaxMass, utils.MakeRandomByte())
	maxMinMass := (mass - 1) / 2
	g := Genome{
		OscPeriod:         utils.ClampByte(1, math.MaxUint8, utils.MakeRandomByte()),
		SightDistance:     utils.ClampByte(p.MinSightDistance, p.MaxSightDistance, utils.MakeRandomByte()),
		FieldOfView:       utils.ClampByte(p.MinFieldOfView, p.MaxFieldOfView, utils.MakeRandomByte()),
		Responsiveness:    utils.MakeRandomByte(),
		MutationRate:      utils.ClampByte(1, math.MaxUint8, utils.MakeRandomByte()),
		Mass:              mass,
		MinMass:           utils.ClampByte(1, maxMinMass, utils.MakeRandomByte()),
		ReproductionType:  makeRandomBool(),
		NeuronCount:       utils.ClampByte(p.MinHiddenLayerCount, p.MaxHiddenLayerCount, utils.MakeRandomByte()),
		BrainLength:       utils.ClampByte(p.MinSpawnNeuronCount, p.MaxSpawnNeuronCount, utils.MakeRandomByte()),
		JuvenilePeriod:    utils.MakeRandomByte(),
		MetabolicRate:     utils.MakeRandomByte(),
		StomachSize:       utils.MakeRandomByte(),
		LearningRate:      utils.MakeRandomByte(),
		LearningThreshold: utils.MakeRandomByte(),
	}
	for i := byte(0); i < g.BrainLength; i++ {
		gene := MakeRandomGene()
		g.Brain = append(g.Brain, gene)
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
	for _, n := range g.Brain {
		temp = append(temp, n.Copy())
	}
	new.Brain = temp
	return &new
}

// nudgeByte moves the value a small distance
func nudgeByte(val byte, strength int) byte {
	delta := rand.Intn(strength*2+1) - strength
	newVal := int(val) + delta
	if newVal < 0 {
		return 0
	}
	if newVal > 255 {
		return 255
	}
	return byte(newVal)
}

// Mutate randomly mutates genes in the genome at a rate of p.BaseMutationRate * g.MutationRate
func Mutate(g *Genome, p *Parameters, isArtificial bool) {
	rateMultiplier := float32(g.MutationRate) / 128.0
	mutationRate := p.BaseMutationRate * rateMultiplier

	if isArtificial {
		mutationRate = p.SpawnMutationRate * float32(g.MutationRate)
	}
	mutateTarget := func(val *byte, min, max byte, strength int) {
		if rand.Float32() < mutationRate {
			*val = utils.ClampByte(min, max, nudgeByte(*val, strength))
		}
	}

	mutateTarget(&g.OscPeriod, 1, 255, 15)
	mutateTarget(&g.SightDistance, p.MinSightDistance, p.MaxSightDistance, 10)
	mutateTarget(&g.FieldOfView, p.MinFieldOfView, p.MaxFieldOfView, 10)
	mutateTarget(&g.Responsiveness, 0, 255, 20)
	mutateTarget(&g.MutationRate, 1, 255, 5)
	mutateTarget(&g.Mass, 3, p.MaxMass, 12)

	maxMinMass := (g.Mass - 1) / 2
	if maxMinMass < 1 {
		maxMinMass = 1
	}
	if g.MinMass > maxMinMass {
		g.MinMass = maxMinMass
	}
	mutateTarget(&g.MinMass, 1, maxMinMass, 8)

	if rand.Float32() < mutationRate {
		g.ReproductionType ^= 1
	}

	mutateTarget(&g.NeuronCount, p.MinHiddenLayerCount, p.MaxHiddenLayerCount, 5)
	mutateTarget(&g.BrainLength, p.MinNeuronCount, p.MaxNeuronCount, 5)
	mutateTarget(&g.JuvenilePeriod, 0, 255, 15)
	mutateTarget(&g.MetabolicRate, 0, 255, 15)
	mutateTarget(&g.StomachSize, 0, 255, 15)
	mutateTarget(&g.LearningRate, 0, 255, 10)
	mutateTarget(&g.LearningThreshold, 0, 255, 10)

	for j := 0; j < len(g.Brain); j++ {
		if rand.Float32() < mutationRate {
			chance := rand.Float32()
			switch {
			case chance < 0.05:
				g.Brain[j].SourceType ^= 1
			case chance < 0.10:
				if g.Brain[j].SinkType == 2 {
					g.Brain[j].SinkType = 0
				} else {
					g.Brain[j].SinkType = 2
				}
			case chance < 0.15:
				g.Brain[j].SourceID = utils.MakeRandomByte()
			case chance < 0.20:
				g.Brain[j].SinkID = utils.MakeRandomByte()
			default:
				g.Brain[j].Weight = nudgeByte(g.Brain[j].Weight, 25)
			}
		}
	}
	diff := int(g.BrainLength) - len(g.Brain)
	if diff > 0 {
		for i := 0; i < diff; i++ {
			if len(g.Brain) > 0 && rand.Float32() < 0.8 {
				// 1. Pick a random existing gene and then change it slightly
				sourceGene := g.Brain[rand.Intn(len(g.Brain))]
				newGene := sourceGene.Copy()

				if rand.Float32() < 0.5 {
					newGene.SourceID = utils.MakeRandomByte()
				} else {
					newGene.SinkID = utils.MakeRandomByte()
				}

				newGene.Weight = nudgeByte(newGene.Weight, 40)

				g.Brain = append(g.Brain, newGene)
			} else {
				g.Brain = append(g.Brain, MakeRandomGene())
			}
		}
	}
}

// AsexualReproduction creates a deep copy of the parent genome then mutates it.
func AsexualReproduction(parent *Genome, p *Parameters) *Genome {
	child := parent.Copy()
	Mutate(child, p, false)
	return child
}

// AsexualReproduction creates a deep copy of the parent genome then mutates it by the spawn Mutation Rate
func ArtificialReproduction(parent *Genome, p *Parameters) *Genome {
	child := parent.Copy()
	Mutate(child, p, true)
	return child
}

// GenomeSimilarity returns a value in [0, 1] based on the normalised Hamming
// distance between the two genomes' byte arrays. 1 = identical, 0 = maximally
// different. Length differences are penalised as all-bits-different bytes.
func GenomeSimilarity(g1, g2 Genome) float32 {
	b1 := g1.ToByteArray()
	b2 := g2.ToByteArray()
	maxLen := len(b1)
	if len(b2) > maxLen {
		maxLen = len(b2)
	}
	if maxLen == 0 {
		return 1.0
	}
	minLen := len(b1)
	if len(b2) < minLen {
		minLen = len(b2)
	}
	diff := 0
	for i := 0; i < minLen; i++ {
		x := b1[i] ^ b2[i]
		for x != 0 {
			diff += int(x & 1)
			x >>= 1
		}
	}
	diff += (maxLen - minLen) * 8
	return 1.0 - float32(diff)/float32(maxLen*8)
}
