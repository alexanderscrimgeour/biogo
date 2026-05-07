package simulation

import (
	"biogo/v2/jaro"
	"biogo/v2/utils"
	"fmt"
	"math"
	"math/rand"
)

const (
	OSC_PERIOD = iota
	MAX_ENERGY
	SIGHT_DISTANCE
	FIELD_OF_VIEW
	RESPONSIVENESS
	MUTATION_RATE
<<<<<<< Updated upstream
	SIZE
=======
	MASS
	MIN_MASS
>>>>>>> Stashed changes
	REPRODUCTION_TYPE
	NEURON_COUNT
	NEUROLOGY_LENGTH
	JUVENILE_PERIOD
	METABOLIC_RATE
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
	FieldOfView      byte // total FOV angle in degrees (0–180)
	Responsiveness   byte
	MutationRate     byte
<<<<<<< Updated upstream
	Size             byte
=======
	Mass             byte
	MinMass          byte // birth mass; scales linearly to Mass over the juvenile period
>>>>>>> Stashed changes
	ReproductionType byte
	NeuronCount      byte
	BrainLength      byte
	JuvenilePeriod   byte
	MetabolicRate    byte
	Brain            []*Gene
}

func (g Gene) String() string {
	return fmt.Sprintf("%b%08b%b%08b%08b", g.SourceType, g.SourceID, g.SinkType, g.SinkID, g.Weight)
}

func (g Genome) String() string {
<<<<<<< Updated upstream
	str := fmt.Sprintf("%08b%08b%08b%08b%08b%08b%08b%b%08b%08b", g.OscPeriod, g.MaxEnergy, g.SightDistance, g.FieldOfView, g.Responsiveness, g.MutationRate, g.Size, g.ReproductionType, g.BrainLength, g.JuvenilePeriod)
=======
	str := fmt.Sprintf("%08b%08b%08b%08b%08b%08b%08b%08b%b%08b%08b%08b", g.OscPeriod, g.MaxEnergy, g.SightDistance, g.FieldOfView, g.Responsiveness, g.MutationRate, g.Mass, g.MinMass, g.ReproductionType, g.BrainLength, g.JuvenilePeriod, g.MetabolicRate)
>>>>>>> Stashed changes
	for _, gene := range g.Brain {
		str += gene.String()
	}
	return str
}

func (g Gene) BinaryString() string {
	return fmt.Sprintf("|%b|%08b|%b|%08b|%08b", g.SourceType, g.SourceID, g.SinkType, g.SinkID, g.Weight)
}

func (g Genome) BinaryString() string {
<<<<<<< Updated upstream
	str := fmt.Sprintf("%08b|%08b|%08b|%08b|%08b|%08b|%08b|%b|%08b|%08b", g.OscPeriod, g.MaxEnergy, g.SightDistance, g.FieldOfView, g.Responsiveness, g.MutationRate, g.Size, g.ReproductionType, g.BrainLength, g.JuvenilePeriod)
=======
	str := fmt.Sprintf("%08b|%08b|%08b|%08b|%08b|%08b|%08b|%08b|%b|%08b|%08b|%08b", g.OscPeriod, g.MaxEnergy, g.SightDistance, g.FieldOfView, g.Responsiveness, g.MutationRate, g.Mass, g.MinMass, g.ReproductionType, g.BrainLength, g.JuvenilePeriod, g.MetabolicRate)
>>>>>>> Stashed changes
	for _, gene := range g.Brain {
		str += gene.BinaryString()
	}
	return str
}

func (g Genome) ToByteArray() []byte {
	arr := []byte{}
	arr = append(arr, g.OscPeriod)
	arr = append(arr, g.MaxEnergy)
	arr = append(arr, g.SightDistance)
	arr = append(arr, g.FieldOfView)
	arr = append(arr, g.Responsiveness)
	arr = append(arr, g.MutationRate)
<<<<<<< Updated upstream
	arr = append(arr, g.Size)
=======
	arr = append(arr, g.Mass)
	arr = append(arr, g.MinMass)
>>>>>>> Stashed changes
	arr = append(arr, g.ReproductionType)
	arr = append(arr, g.NeuronCount)
	arr = append(arr, g.BrainLength)
	arr = append(arr, g.JuvenilePeriod)
	arr = append(arr, g.MetabolicRate)
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
<<<<<<< Updated upstream
	str := fmt.Sprintf("|%08b|%08d|%08b|%08b|%08b|%08b|%08b|%b|%08b|%08b", g.OscPeriod, g.MaxEnergy, g.SightDistance, g.FieldOfView, g.Responsiveness, g.MutationRate, g.Size, g.ReproductionType, g.BrainLength, g.JuvenilePeriod)
=======
	str := fmt.Sprintf("|%08b|%08d|%08b|%08b|%08b|%08b|%08b|%08b|%b|%08b|%08b|%08b", g.OscPeriod, g.MaxEnergy, g.SightDistance, g.FieldOfView, g.Responsiveness, g.MutationRate, g.Mass, g.MinMass, g.ReproductionType, g.BrainLength, g.JuvenilePeriod, g.MetabolicRate)
>>>>>>> Stashed changes
	for _, gene := range g.Brain {
		str += gene.PrettyString()
	}
	return str
}

// genomeColor derives display RGB values from gene structure bytes.
// The color encodes genetic diversity visually without involving rendering concerns in Genome.
func genomeColor(g *Genome) (uint8, uint8, uint8, uint8) {
	first := g.Brain[0]
	mid := g.Brain[len(g.Brain)/2]
	last := g.Brain[len(g.Brain)-1]
	firstAsByte := (first.SourceID&3)<<6 | (first.SourceType&3)<<4 | (first.SinkID&3)<<2 | (first.SinkType & 3)
	midAsByte := (mid.SourceID&3)<<6 | (mid.SourceType&3)<<4 | (mid.SinkID&3)<<2 | (mid.SinkType & 3)
	lastAsByte := (last.SourceID&3)<<6 | (last.SourceType&3)<<4 | (last.SinkID&3)<<2 | (last.SinkType & 3)
	return firstAsByte, midAsByte, lastAsByte, 255
}

// WeightAsFloat converts from a byte to a float64 range 0...1
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
		SinkType:   utils.MakeRandomByte() & 1,
		SinkID:     utils.MakeRandomByte(),
		Weight:     utils.MakeRandomByte(),
	}
}

func MakeRandomGenome(p *Parameters) *Genome {
<<<<<<< Updated upstream
=======
	mass := utils.ClampByte(1, p.MaxMass, utils.MakeRandomByte())
>>>>>>> Stashed changes
	g := Genome{
		OscPeriod:        utils.ClampByte(1, math.MaxUint8, utils.MakeRandomByte()),
		MaxEnergy:        utils.ClampByte(p.MinEnergy, p.MaxEnergy, utils.MakeRandomByte()),
		SightDistance:    utils.ClampByte(p.MinSightDistance, p.MaxSightDistance, utils.MakeRandomByte()),
		FieldOfView:      utils.ClampByte(p.MinFieldOfView, p.MaxFieldOfView, utils.MakeRandomByte()),
		Responsiveness:   utils.MakeRandomByte(),
<<<<<<< Updated upstream
		MutationRate:     utils.MakeRandomByte(),
		Size:             utils.ClampByte(p.MinSize, p.MaxSize, utils.MakeRandomByte()),
=======
		MutationRate:     utils.ClampByte(1, math.MaxUint8, utils.MakeRandomByte()),
		Mass:             mass,
		MinMass:          utils.ClampByte(1, mass, utils.MakeRandomByte()),
>>>>>>> Stashed changes
		ReproductionType: makeRandomBool(),
		NeuronCount:      utils.ClampByte(p.MinHiddenLayerCount, p.MaxHiddenLayerCount, utils.MakeRandomByte()),
		BrainLength:      utils.ClampByte(p.MinStartNeuronCount, p.MaxStartNeuronCount, utils.MakeRandomByte()),
		JuvenilePeriod:   utils.MakeRandomByte(),
		MetabolicRate:    utils.MakeRandomByte(),
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

// Mutate randomly flips bits in the genome at a rate of p.MinMutationRate * g.MutationRate
func Mutate(g *Genome, p *Parameters) {
	mutationRate := p.MinMutationRate * float32(g.MutationRate)

	for i := 0; i < GENOME_STRUCTURE_COUNT; i++ {
		r := rand.Float32()
		if r < mutationRate {
			switch i {
			case OSC_PERIOD:
				g.OscPeriod ^= byte(1 << (rand.Uint32() >> 29))
			case MAX_ENERGY:
				new := g.MaxEnergy
				new ^= byte(1 << (rand.Uint32() >> 29))
				g.MaxEnergy = utils.ClampByte(p.MinEnergy, p.MaxEnergy, new)
			case SIGHT_DISTANCE:
				new := g.SightDistance
				new ^= byte(1 << (rand.Uint32() >> 29))
				g.SightDistance = utils.ClampByte(p.MinSightDistance, p.MaxSightDistance, new)
			case FIELD_OF_VIEW:
				new := g.FieldOfView
				new ^= byte(1 << (rand.Uint32() >> 29))
				g.FieldOfView = utils.ClampByte(p.MinFieldOfView, p.MaxFieldOfView, new)
			case RESPONSIVENESS:
				g.Responsiveness ^= byte(1 << (rand.Uint32() >> 29))
			case MUTATION_RATE:
				g.MutationRate ^= byte(1 << (rand.Uint32() >> 29))
				g.MutationRate = utils.ClampByte(1, math.MaxUint8, g.MutationRate)
			case MASS:
				new := g.Mass
				new ^= byte(1 << (rand.Uint32() >> 29))
<<<<<<< Updated upstream
				g.Size = utils.ClampByte(p.MinSize, p.MaxSize, new)
=======
				g.Mass = utils.ClampByte(g.MinMass, p.MaxMass, new)
			case MIN_MASS:
				new := g.MinMass
				new ^= byte(1 << (rand.Uint32() >> 29))
				g.MinMass = utils.ClampByte(1, g.Mass, new)
>>>>>>> Stashed changes
			case REPRODUCTION_TYPE:
				g.ReproductionType ^= 1
			case NEURON_COUNT:
				new := g.NeuronCount
				new ^= byte(1 << (rand.Uint32() >> 29))
				g.NeuronCount = utils.ClampByte(p.MinHiddenLayerCount, p.MaxHiddenLayerCount, new)
			case NEUROLOGY_LENGTH:
				new := g.BrainLength
				new ^= byte(1 << (rand.Uint32() >> 29))
				g.BrainLength = utils.ClampByte(p.MinNeuronCount, p.MaxNeuronCount, new)
			case JUVENILE_PERIOD:
				g.JuvenilePeriod ^= byte(1 << (rand.Uint32() >> 29))
			case METABOLIC_RATE:
				g.MetabolicRate ^= byte(1 << (rand.Uint32() >> 29))
			}
		}
	}
	for j := 0; j < len(g.Brain); j++ {
		r := rand.Float32()
		if r < mutationRate {
			chance := rand.Float32()
			switch {
			case chance < 0.2:
				g.Brain[j].SourceType ^= 1
			case chance < 0.4:
				g.Brain[j].SinkType ^= 1
			case chance < 0.6:
				g.Brain[j].SourceID ^= byte(1 << (rand.Uint32() >> 29))
			case chance < 0.8:
				g.Brain[j].SinkID ^= byte(1 << (rand.Uint32() >> 29))
			default:
				g.Brain[j].Weight ^= byte(1 << (rand.Uint32() >> 29))
			}
		}
	}
	diff := int(g.BrainLength) - len(g.Brain)
	if diff > 0 {
		for i := 0; i < diff; i++ {
			g.Brain = append(g.Brain, MakeRandomGene())
		}
	} else if diff < 0 {
		for i := 0; i > diff; i-- {
			index := rand.Intn(len(g.Brain))
			g.Brain = append(g.Brain[:index], g.Brain[index+1:]...)
		}
	}
}

// AsexualReproduction creates a deep copy of the parent genome then mutates it.
func AsexualReproduction(parent *Genome, p *Parameters) *Genome {
	child := parent.Copy()
	Mutate(child, p)
	return child
}

// GenomeSimilarity compares two genomes using Jaro-Winkler similarity.
func GenomeSimilarity(g1, g2 Genome) float32 {
	return jaro.JaroWinklerSimilarity(g1.String(), g2.String())
}
