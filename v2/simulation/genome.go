package simulation

import (
	"biogo/v2/utils"
	"fmt"
	"math"
	"math/bits"
	"math/rand"
	"sync/atomic"
)

// genomeUIDSeq is a monotonically increasing counter; each call to
// recomputeBytes() claims one value, giving every genome a unique identity
// that changes when the genome mutates. Used by the per-creature sim cache.
var genomeUIDSeq uint64

const (
	OSC_PERIOD = iota
	VISION_RADIUS
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
	MASS_SPLIT_RATIO
	DIGESTION_TYPE
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
	VisionRadius      byte
	FieldOfView       byte
	Responsiveness    byte
	MutationRate      byte
	Mass              byte
	MinMass           byte // birth mass; scales linearly to Mass over the juvenile period
	ReproductionType  byte
	CognitiveBreadth  byte
	SynapticDensity   byte
	JuvenilePeriod    byte
	MetabolicRate     byte
	StomachSize       byte
	Neuroplasticity   byte // base learning rate; maps to [MinNeuroplasticity, MaxNeuroplasticity]
	LearningThreshold byte // minimum learning signal to update a weight; maps to [MinLearningThreshold, MaxLearningThreshold]
	MassSplitRatio    byte // fraction of mass (and energy) transferred to daughter on asexual reproduction; maps to [0, 0.5]
	DigestionType     byte // diet specialisation: 0 = pure herbivore, 255 = pure carnivore
	Brain             []Gene

	// flat byte cache for GenomeSimilarity; recomputed after any mutation or brain change.
	// Layout: 15 header bytes + 5 bytes per gene (SourceID, SourceType, SinkID, SinkType, Weight).
	bytes []byte
	// uid is assigned by recomputeBytes() from a global counter; it lets the
	// per-creature sim cache detect stale entries when an ID is recycled.
	uid uint64
}

// recomputeBytes refreshes the flat byte cache used by GenomeSimilarity and
// stamps a new uid so per-creature similarity caches can detect stale entries.
// Call this after any field change or Brain modification.
func (g *Genome) recomputeBytes() {
	need := 17 + len(g.Brain)*5
	if cap(g.bytes) >= need {
		g.bytes = g.bytes[:need]
	} else {
		g.bytes = make([]byte, need)
	}
	b := g.bytes
	b[0] = g.OscPeriod
	b[1] = g.VisionRadius
	b[2] = g.FieldOfView
	b[3] = g.Responsiveness
	b[4] = g.MutationRate
	b[5] = g.Mass
	b[6] = g.MinMass
	b[7] = g.ReproductionType
	b[8] = g.CognitiveBreadth
	b[9] = g.SynapticDensity
	b[10] = g.JuvenilePeriod
	b[11] = g.MetabolicRate
	b[12] = g.StomachSize
	b[13] = g.Neuroplasticity
	b[14] = g.LearningThreshold
	b[15] = g.MassSplitRatio
	b[16] = g.DigestionType
	for i, gn := range g.Brain {
		off := 17 + i*5
		b[off] = gn.SourceID
		b[off+1] = gn.SourceType
		b[off+2] = gn.SinkID
		b[off+3] = gn.SinkType
		b[off+4] = gn.Weight
	}
	g.uid = atomic.AddUint64(&genomeUIDSeq, 1)
}

func (g Gene) String() string {
	return fmt.Sprintf("%b%08b%b%08b%08b", g.SourceType, g.SourceID, g.SinkType, g.SinkID, g.Weight)
}

func (g Genome) String() string {
	str := fmt.Sprintf("%08b%08b%08b%08b%08b%08b%08b%b%08b%08b%08b%08b%08b%08b%08b%08b", g.OscPeriod, g.VisionRadius, g.FieldOfView, g.Responsiveness, g.MutationRate, g.Mass, g.MinMass, g.ReproductionType, g.SynapticDensity, g.JuvenilePeriod, g.MetabolicRate, g.StomachSize, g.Neuroplasticity, g.LearningThreshold, g.MassSplitRatio, g.DigestionType)
	for _, gene := range g.Brain {
		str += gene.String()
	}
	return str
}

func (g Gene) BinaryString() string {
	return fmt.Sprintf("|%b|%08b|%b|%08b|%08b", g.SourceType, g.SourceID, g.SinkType, g.SinkID, g.Weight)
}

func (g Genome) BinaryString() string {
	str := fmt.Sprintf("%08b|%08b|%08b|%08b|%08b|%08b|%08b|%b|%08b|%08b|%08b|%08b|%08b|%08b|%08b|%08b", g.OscPeriod, g.VisionRadius, g.FieldOfView, g.Responsiveness, g.MutationRate, g.Mass, g.MinMass, g.ReproductionType, g.SynapticDensity, g.JuvenilePeriod, g.MetabolicRate, g.StomachSize, g.Neuroplasticity, g.LearningThreshold, g.MassSplitRatio, g.DigestionType)
	for _, gene := range g.Brain {
		str += gene.BinaryString()
	}
	return str
}

func (g Genome) ToByteArray() []byte {
	arr := make([]byte, 0, 17+len(g.Brain)*4)
	arr = append(arr, g.OscPeriod)
	arr = append(arr, g.VisionRadius)
	arr = append(arr, g.FieldOfView)
	arr = append(arr, g.Responsiveness)
	arr = append(arr, g.MutationRate)
	arr = append(arr, g.Mass)
	arr = append(arr, g.MinMass)
	arr = append(arr, g.ReproductionType)
	arr = append(arr, g.CognitiveBreadth)
	arr = append(arr, g.SynapticDensity)
	arr = append(arr, g.JuvenilePeriod)
	arr = append(arr, g.MetabolicRate)
	arr = append(arr, g.StomachSize)
	arr = append(arr, g.Neuroplasticity)
	arr = append(arr, g.LearningThreshold)
	arr = append(arr, g.MassSplitRatio)
	arr = append(arr, g.DigestionType)
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
	str := fmt.Sprintf("|%08b|%08b|%08b|%08b|%08b|%08b|%08b|%b|%08b|%08b|%08b|%08b|%08b|%08b|%08b|%08b", g.OscPeriod, g.VisionRadius, g.FieldOfView, g.Responsiveness, g.MutationRate, g.Mass, g.MinMass, g.ReproductionType, g.SynapticDensity, g.JuvenilePeriod, g.MetabolicRate, g.StomachSize, g.Neuroplasticity, g.LearningThreshold, g.MassSplitRatio, g.DigestionType)
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

// generateTierExpansionGene creates a new connection that targets the newly unlocked space
// between the parent's constraints and the child's expanded actions/sensors/ hidden neurons.
func (g *Genome) generateTierExpansionGene(pBreadth, cBreadth, pSensors, cSensors, pActions, cActions byte) Gene {
	var gene Gene
	gene.Weight = utils.MakeRandomByte()

	// Source
	if rand.Float32() < 0.5 && cBreadth > pBreadth {
		gene.SourceType = 1

		// Start at the parent's boundary, then add a random value
		// bounded strictly by the size of the new expansion tier.
		delta := cBreadth - pBreadth
		gene.SourceID = pBreadth + (utils.MakeRandomByte() % delta)
	} else {
		// Target a Sensor
		gene.SourceType = 0

		if cSensors > pSensors {
			// A new sensor was unlocked! Force the connection to read from it.
			delta := cSensors - pSensors
			gene.SourceID = pSensors + (utils.MakeRandomByte() % delta)
		} else {
			// No new sensors unlocked; fall back to sampling the existing ones uniformly.
			gene.SourceID = utils.MakeRandomByte() % cSensors
		}
	}

	if rand.Float32() < 0.5 && cBreadth > pBreadth {
		gene.SinkType = 1

		// Offset the index into the new hidden neuron real estate.
		delta := cBreadth - pBreadth
		gene.SinkID = pBreadth + (utils.MakeRandomByte() % delta)
	} else {
		gene.SinkType = 2

		if cActions > pActions {
			// A new action was unlocked! Force the connection to drive it.
			delta := cActions - pActions
			gene.SinkID = pActions + (utils.MakeRandomByte() % delta)
		} else {
			// No new actions unlocked; fall back to sampling the existing ones uniformly.
			gene.SinkID = utils.MakeRandomByte() % cActions
		}
	}

	return gene
}

// MakeRandomGene creates a random gene
func MakeRandomGene(allowedSensors, allowedActions, cognitiveBreadth byte) Gene {
	sourceType := byte(SENSOR)
	var sourceID byte

	// Single random byte for both structural type flags
	typeFlags := utils.MakeRandomByte()

	if cognitiveBreadth > 0 && (typeFlags&1) == 1 {
		sourceType = NEURON
		sourceID = utils.MakeRandomByte() % cognitiveBreadth
	} else {
		sourceID = utils.MakeRandomByte() % allowedSensors
	}

	sinkType := byte(ACTION)
	var sinkID byte
	if cognitiveBreadth > 0 && ((typeFlags>>1)&1) == 1 {
		sinkType = NEURON
		sinkID = utils.MakeRandomByte() % cognitiveBreadth
	} else {
		sinkID = utils.MakeRandomByte() % allowedActions
	}
	return Gene{
		SourceType: sourceType,
		SourceID:   sourceID,
		SinkType:   sinkType,
		SinkID:     sinkID,
		Weight:     utils.MakeRandomByte(),
	}
}

func MakeRandomGenome(p *Parameters, tier byte) *Genome {
	// Mass must be >= 3 to guarantee a valid MinMass (MinMass < Mass/2 requires Mass > 2).
	massByte := utils.MakeRandomByte()
	if massByte < 3 {
		massByte = 3
	}

	// Generation
	// 1. Calculate Tier-Constrained Cognitive Breadth (Split 0-255 into 4 parts)
	// Tier 0: 0-63, Tier 1: 64-127, Tier 2: 128-191, Tier 3: 192-255
	minBreadth := tier * 64
	maxBreadth := minBreadth + 63
	cogBreadth := utils.LerpByte(minBreadth, maxBreadth, utils.MakeRandomByte())
	// Scale SynapticDensity alongside tear
	minDensity := utils.LerpByte(p.Neurology.MinSynapticDensity, p.Neurology.MaxSynapticDensity, tier*64)
	maxDensity := utils.LerpByte(p.Neurology.MinSynapticDensity, p.Neurology.MaxSynapticDensity, (tier+1)*64-1)
	synDensity := utils.LerpByte(minDensity, maxDensity, utils.MakeRandomByte())

	g := Genome{
		OscPeriod:         utils.LerpByte(1, math.MaxUint8, utils.MakeRandomByte()),
		VisionRadius:      utils.MakeRandomByte(),
		FieldOfView:       utils.MakeRandomByte(),
		Responsiveness:    utils.MakeRandomByte(),
		MutationRate:      utils.LerpByte(1, math.MaxUint8, utils.MakeRandomByte()),
		Mass:              massByte,
		ReproductionType:  makeRandomBool(),
		CognitiveBreadth:  cogBreadth,
		SynapticDensity:   synDensity,
		JuvenilePeriod:    utils.MakeRandomByte(),
		MetabolicRate:     utils.MakeRandomByte(),
		StomachSize:       utils.MakeRandomByte(),
		Neuroplasticity:   utils.MakeRandomByte(),
		LearningThreshold: utils.MakeRandomByte(),
		MassSplitRatio:    utils.MakeRandomByte(),
		DigestionType:     utils.MakeRandomByte(),
	}
	maxMinMass := (g.Mass - 1) / 2
	if maxMinMass < 1 {
		maxMinMass = 1
	}
	g.MinMass = utils.LerpByte(1, maxMinMass, utils.MakeRandomByte())

	// Allocate
	g.Brain = make([]Gene, 0, g.SynapticDensity)

	allowedSensors := getAllowedSensorCount(g.CognitiveBreadth)
	allowedActions := getAllowedActionCount(g.CognitiveBreadth)

	for i := byte(0); i < g.SynapticDensity; i++ {
		gene := MakeRandomGene(allowedSensors, allowedActions, g.CognitiveBreadth)
		g.Brain = append(g.Brain, gene)
	}
	g.recomputeBytes()
	return &g
}

// Copy returns a value copy of the gene.
func (g Gene) Copy() Gene { return g }

// Copy deep copies a genome.
func (g *Genome) Copy() *Genome {
	new := *g
	new.Brain = make([]Gene, len(g.Brain))
	copy(new.Brain, g.Brain)
	if len(g.bytes) > 0 {
		new.bytes = make([]byte, len(g.bytes))
		copy(new.bytes, g.bytes)
	}
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

// Mutate randomly mutates genes in the genome at a rate of p.BaseMutationRate * g.MutationRate * mutationMult.
// Pass mutationMult=1.0 for normal reproduction; pass params.Environment.Radiation.MutationMultiplier for irradiated parents.
func Mutate(g *Genome, p *Parameters, isArtificial bool, mutationMult float32, childGeneration float32) {
	rateMultiplier := float32(g.MutationRate) / 128.0
	mutationRate := p.Neurology.BaseMutationRate * rateMultiplier * mutationMult

	// Track parent dimentional limits so that we don't lose
	// the pre-existing constructed brain on tier upgrades
	parentBreadth := g.CognitiveBreadth
	parentSensors := getAllowedSensorCount(parentBreadth)
	parentActions := getAllowedActionCount(parentBreadth)

	// Helper function to "mutate" - i.e. nudge byte in a direction
	mutateTarget := func(val *byte, min, max byte, strength int) {
		if rand.Float32() < mutationRate {
			*val = utils.LerpByte(min, max, nudgeByte(*val, strength))
		}
	}

	// Physical attribute mutation
	mutateTarget(&g.OscPeriod, 0, 255, 15)
	mutateTarget(&g.VisionRadius, 0, 255, 10)
	mutateTarget(&g.FieldOfView, 0, 255, 10)
	mutateTarget(&g.Responsiveness, 0, 255, 20)
	mutateTarget(&g.MutationRate, 1, 255, 5)
	mutateTarget(&g.Mass, 3, math.MaxUint8, 12)
	mutateTarget(&g.JuvenilePeriod, 0, 255, 15)
	mutateTarget(&g.MetabolicRate, 0, 255, 15)
	mutateTarget(&g.StomachSize, 0, 255, 15)
	mutateTarget(&g.Neuroplasticity, 0, 255, 10)
	mutateTarget(&g.LearningThreshold, 0, 255, 10)
	mutateTarget(&g.MassSplitRatio, 0, 255, 15)
	mutateTarget(&g.DigestionType, 0, 255, 15)

	maxMinMass := (g.Mass - 1) / 2
	if maxMinMass < 1 {
		maxMinMass = 1
	}
	if g.MinMass > maxMinMass {
		g.MinMass = maxMinMass
	}
	mutateTarget(&g.MinMass, 1, maxMinMass, 8)

	// Chance to flip reproduction type
	// TODO(): Need to consider this, because becoming the only sexual
	// creature in your species is probably a sad and frustrating existence.
	if rand.Float32() < mutationRate {
		g.ReproductionType ^= 1
	}

	// New tier constraints and cognitive breadth
	minTierBreadth, maxTierBreadth := getTierBoundaries(childGeneration, p)
	mutateTarget(&g.CognitiveBreadth, minTierBreadth, maxTierBreadth, 5)

	// Force SynapticDensity bounds to slide up with the tier scaling
	minDensity := utils.LerpByte(p.Neurology.MinSynapticDensity, p.Neurology.MaxSynapticDensity, minTierBreadth)
	maxDensity := utils.LerpByte(p.Neurology.MinSynapticDensity, p.Neurology.MaxSynapticDensity, maxTierBreadth)
	mutateTarget(&g.SynapticDensity, minDensity, maxDensity, 5)

	allowedSensors := getAllowedSensorCount(g.CognitiveBreadth)
	allowedActions := getAllowedActionCount(g.CognitiveBreadth)

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
				// Protected mapping using parent boundaries
				if g.Brain[j].SourceType == 1 && parentBreadth > 0 {
					g.Brain[j].SourceID = utils.MakeRandomByte() % parentBreadth
				} else {
					g.Brain[j].SourceID = utils.MakeRandomByte() % parentSensors
				}
			case chance < 0.20:
				if g.Brain[j].SinkType == 1 && parentBreadth > 0 {
					g.Brain[j].SinkID = utils.MakeRandomByte() % parentBreadth
				} else {
					g.Brain[j].SinkID = utils.MakeRandomByte() % parentActions
				}
			default:
				g.Brain[j].Weight = nudgeByte(g.Brain[j].Weight, 25)
			}
		}
	}

	// Brain expansion padding loop: fills vacant structural space when SynapticDensity scales upwards.
	diff := int(g.SynapticDensity) - len(g.Brain)
	if diff > 0 {
		for i := 0; i < diff; i++ {
			var newGene Gene

			// Allocate 75% of new connections directly to newly discovered tier capabilities
			if (g.CognitiveBreadth > parentBreadth || allowedSensors > parentSensors || allowedActions > parentActions) && rand.Float32() < 0.75 {
				newGene = g.generateTierExpansionGene(parentBreadth, g.CognitiveBreadth, parentSensors, allowedSensors, parentActions, allowedActions)
			} else {
				// 25% is random connection
				newGene = MakeRandomGene(allowedSensors, allowedActions, g.CognitiveBreadth)
			}

			g.Brain = append(g.Brain, newGene)
		}
	}
	g.recomputeBytes()
}

func pickByte(a, b byte) byte {
	if rand.Intn(2) == 0 {
		return a
	}
	return b
}

func pickGene(a, b Gene) Gene {
	if rand.Intn(2) == 0 {
		return a
	}
	return b
}

// Crossover produces a child genome by uniform crossover of two parents followed
// by mutation. Each header byte is drawn independently from either parent. Brain
// genes in the overlapping range are drawn from either parent; excess genes from
// the longer parent survive with 50% probability.
// mutationMult amplifies the mutation rate; pass 1.0 for normal conditions.
func Crossover(g1, g2 *Genome, p *Parameters, mutationMult float32, childGeneration float32) *Genome {
	child := &Genome{
		OscPeriod:         pickByte(g1.OscPeriod, g2.OscPeriod),
		VisionRadius:      pickByte(g1.VisionRadius, g2.VisionRadius),
		FieldOfView:       pickByte(g1.FieldOfView, g2.FieldOfView),
		Responsiveness:    pickByte(g1.Responsiveness, g2.Responsiveness),
		MutationRate:      pickByte(g1.MutationRate, g2.MutationRate),
		ReproductionType:  pickByte(g1.ReproductionType, g2.ReproductionType),
		CognitiveBreadth:  pickByte(g1.CognitiveBreadth, g2.CognitiveBreadth),
		JuvenilePeriod:    pickByte(g1.JuvenilePeriod, g2.JuvenilePeriod),
		MetabolicRate:     pickByte(g1.MetabolicRate, g2.MetabolicRate),
		StomachSize:       pickByte(g1.StomachSize, g2.StomachSize),
		Neuroplasticity:   pickByte(g1.Neuroplasticity, g2.Neuroplasticity),
		LearningThreshold: pickByte(g1.LearningThreshold, g2.LearningThreshold),
		MassSplitRatio:    pickByte(g1.MassSplitRatio, g2.MassSplitRatio),
		DigestionType:     pickByte(g1.DigestionType, g2.DigestionType),
	}

	// If parents from different tiers cross over, force child into its generation tier bounds
	minTierBreadth, maxTierBreadth := getTierBoundaries(childGeneration, p)
	if child.CognitiveBreadth < minTierBreadth {
		child.CognitiveBreadth = minTierBreadth
	} else if child.CognitiveBreadth > maxTierBreadth {
		child.CognitiveBreadth = maxTierBreadth
	}

	// --- GROUPED TRAITS ---
	// Inherit Body Scale as a package to maintain physical proportions
	if rand.Intn(2) == 0 {
		child.Mass = g1.Mass
		child.MinMass = g1.MinMass
	} else {
		child.Mass = g2.Mass
		child.MinMass = g2.MinMass
	}

	// --- INVARIANT PROTECTION ---
	if child.Mass < 3 {
		child.Mass = 3
	}
	maxMinMass := (child.Mass - 1) / 2
	if maxMinMass < 1 {
		maxMinMass = 1
	}
	if child.MinMass < 1 {
		child.MinMass = 1
	}
	if child.MinMass > maxMinMass {
		child.MinMass = maxMinMass
	}

	// --- BRAIN INHERITANCE ---
	len1 := len(g1.Brain)
	len2 := len(g2.Brain)
	targetLen := len1
	if rand.Intn(2) == 0 {
		targetLen = len2
	}

	child.Brain = make([]Gene, targetLen)
	for i := 0; i < targetLen; i++ {
		if i < len1 && i < len2 {
			child.Brain[i] = pickGene(g1.Brain[i], g2.Brain[i])
		} else if i < len1 {
			child.Brain[i] = g1.Brain[i]
		} else {
			child.Brain[i] = g2.Brain[i]
		}
	}
	// Align SynapticDensity with the actual brain length so Mutate does not
	// immediately re-grow the brain beyond what crossover intended.
	child.SynapticDensity = byte(targetLen)
	mutationChance := 0.01 * mutationMult
	if rand.Float32() < mutationChance {
		Mutate(child, p, false, mutationMult, childGeneration)
	}
	return child
}

// AsexualReproduction creates a deep copy of the parent genome then mutates it.
// mutationMult amplifies the mutation rate; pass 1.0 for normal conditions.
func AsexualReproduction(parent *Genome, p *Parameters, mutationMult float32, childGeneration float32) *Genome {
	child := parent.Copy()
	minTierBreadth, _ := getTierBoundaries(childGeneration, p)
	if child.CognitiveBreadth < minTierBreadth {
		child.CognitiveBreadth = minTierBreadth // Push up to the new tier minimum
	}
	Mutate(child, p, false, mutationMult, childGeneration)
	return child
}

// GenomeSimilarity returns a value in [0, 1] based on the normalised Hamming
// distance between the two genomes' byte arrays. 1 = identical, 0 = maximally
// different. Length differences are penalised as all-bits-different bytes.
//
// Uses the pre-computed flat byte cache (g.bytes) and processes 8 bytes at a
// time via OnesCount64, avoiding per-gene pointer chasing and reducing the
// call count by ~8× vs the old byte-at-a-time loop.
func GenomeSimilarity(g1, g2 *Genome) float32 {
	b1, b2 := g1.bytes, g2.bytes
	l1, l2 := len(b1), len(b2)
	minLen := l1
	if l2 < minLen {
		minLen = l2
	}

	diff := 0
	i := 0
	for ; i+8 <= minLen; i += 8 {
		v1 := uint64(b1[i]) | uint64(b1[i+1])<<8 | uint64(b1[i+2])<<16 | uint64(b1[i+3])<<24 |
			uint64(b1[i+4])<<32 | uint64(b1[i+5])<<40 | uint64(b1[i+6])<<48 | uint64(b1[i+7])<<56
		v2 := uint64(b2[i]) | uint64(b2[i+1])<<8 | uint64(b2[i+2])<<16 | uint64(b2[i+3])<<24 |
			uint64(b2[i+4])<<32 | uint64(b2[i+5])<<40 | uint64(b2[i+6])<<48 | uint64(b2[i+7])<<56
		diff += bits.OnesCount64(v1 ^ v2)
	}
	for ; i < minLen; i++ {
		diff += bits.OnesCount(uint(b1[i] ^ b2[i]))
	}

	absDiff := l1 - l2
	if absDiff < 0 {
		absDiff = -absDiff
	}
	diff += absDiff * 8

	totalBits := (minLen + absDiff) * 8
	if totalBits == 0 {
		return 1.0
	}
	return 1.0 - float32(diff)/float32(totalBits)
}

func MapGeneToRange(gene byte, minRange, maxRange float64) float64 {
	// 1. Normalize the gene (0-255) to a 0.0-1.0 percentage
	percentage := float64(gene) / 255.0

	// 2. Map that percentage to the [min, max] range
	return minRange + (percentage * (maxRange - minRange))
}

func IsSensorAllowed(sensorID byte, breadth byte) bool {
	// Tier 1 always allowed (Breadth >= 0)
	if sensorID <= MaxTier1Sensor {
		return true
	}
	// Tier 2 requires ~25% cognitive breadth
	if sensorID <= MaxTier2Sensor && breadth >= 64 {
		return true
	}
	// Tier 3 requires ~50% cognitive breadth
	if sensorID <= MaxTier3Sensor && breadth >= 128 {
		return true
	}
	// Tier 4 requires ~75% cognitive breadth
	return breadth >= 192
}

func IsActionAllowed(actionID byte, breadth byte) bool {
	if actionID <= MaxTier1Action {
		return true
	}
	if actionID <= MaxTier2Action && breadth >= 64 {
		return true
	}
	if actionID <= MaxTier3Action && breadth >= 128 {
		return true
	}
	return breadth >= 192
}
