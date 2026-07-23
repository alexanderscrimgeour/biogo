package snapshot

// FormatVersion is bumped when the DTO layout changes incompatibly.
const FormatVersion = 1

// SimulationDTO is the gob-encoded root of a saved game.
// All fields are flat value types or slices of value types: no pointers, no maps.
type SimulationDTO struct {
	Version    int
	Tick       int
	Energy     float64
	ParamsGob  []byte // gob-encoded simulation.Parameters (all value types, no registration needed)
	World      WorldDTO
	Population PopulationDTO
}

// WorldDTO stores mutable world state. Fountains are flattened into parallel
// X/Y slices; FountainCounts[0..2] hold foliage/fungi/meat counts respectively
// so the slice can be split on restore without a separator type field.
type WorldDTO struct {
	FountainCounts [3]int // [foliage, fungi, meat]
	FountainX      []float32
	FountainY      []float32
	// Active food only, repacked (no gaps).
	FoodX    []float32
	FoodY    []float32
	FoodMass []float32
	FoodType []byte // world.FoodTypeFoliage / FoodTypeFungi / FoodTypeMeat
}

// PopulationDTO stores only the alive creatures, repacked with no nil gaps.
type PopulationDTO struct {
	Creatures []CreatureDTO
}

// CreatureDTO stores the full mutable state of one living creature.
//
// Fields NOT stored (derived or transient):
//   - halfFOVCos, cachedMetabolicGene, cachedJuvenilePeriod → rebuilt via initCachedFields
//   - Sensors / sight buffers → rebuilt on next tick
//   - simCache → rebuilt on demand
//   - LastTickEnergy, LastDopamine, LastStomach → single-tick deltas, not critical
//   - LastActionMask → single-tick bitmask
type CreatureDTO struct {
	Generation                     float32
	Tier                           byte
	Energy                         float32
	Mass                           float32
	SurvivalMass                   float32
	Radius                         float32
	VisionRadius                   float32
	Stomach                        float32
	Dopamine                       float32
	LocX, LocY                     float32
	LastLocX, LastLocY             float32
	BirthLocX, BirthLocY           float32
	Heading                        float32
	Speed                          float32
	Responsiveness                 float32
	Clock                          int
	BaseOscTick                    float64
	IsResting                      bool
	Age                            int
	ReproductionCooldown           int
	ColorR, ColorG, ColorB, ColorA uint8
	// Genome fields (inlined; all exported bytes — gob ignores the private cache fields).
	// Brain genes are split into five parallel byte slices for flat, gzip-friendly layout.
	GenomeOscPeriod                  byte
	GenomeVisionRadius               byte
	GenomeFieldOfView                byte
	GenomeResponsiveness             byte
	GenomeMutationRate               byte
	GenomeBodyMass                   byte
	GenomeSurvivalMass               byte
	GenomeReproductionType           byte
	GenomeCognitiveBreadth           byte
	GenomeSynapticDensity            byte
	GenomeJuvenilePeriod             byte
	GenomeMetabolicRate              byte
	GenomeStomachSize                byte
	GenomeNeuroplasticity            byte
	GenomeLearningThreshold          byte
	GenomeMassSplitRatio             byte
	GenomeFoliageDigestionEfficiency byte
	GenomeFungiDigestionEfficiency   byte
	GenomeMeatDigestionEfficiency    byte
	// Brain: one byte slice per gene field (flat = gzip-friendly).
	BrainSourceID   []byte
	BrainSourceType []byte
	BrainSinkID     []byte
	BrainSinkType   []byte
	BrainWeight     []byte
	// Learned NNet state. The edge topology is reconstructed deterministically
	// from the genome on restore; only the mutable learned values are stored.
	NNetWeights    []float32 // one entry per edge, same order as reconstructed NNet
	NNetNeuronOut  []float32 // HiddenNeurons[i].Output
	NNetNeuronAvg  []float32 // HiddenNeurons[i].AverageOutput
	NNetNeuronSens []float32 // HiddenNeurons[i].Sensitivity
}
