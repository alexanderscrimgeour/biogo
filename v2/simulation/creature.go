package simulation

import (
	"biogo/v2/utils"
	"biogo/v2/world"
	"fmt"
	"image/color"
	"math"
	"math/rand"
)

// simCacheCapacity must be a power of two; index = otherID & (simCacheCapacity-1).
const simCacheCapacity = 64

// simCacheEntry is one slot in the direct-mapped similarity cache.
// key==0 with peerUID==0 is the zero value; it never matches a real creature
// because all genome UIDs are assigned by atomic.AddUint64 starting at 1.
type simCacheEntry struct {
	peerUID uint64
	key     int32
	sim     float32
}

type Creature struct {
	Id                   int
	Generation           float32
	Energy               float32
	LastTickEnergy       float32
	Responsiveness       float32
	Age                  int
	Alive                bool
	Clock                int
	BaseOscTick          float64
	Nnet                 NeuralNet
	Loc                  world.Position
	BirthLoc             world.Position
	Heading              float32 // radians; 0 = east, π/2 = south (screen-down)
	Speed                float32 // current speed along heading; updated each tick via ACCELERATE action
	Genome               *Genome
	VisionRadius         float32
	Mass                 float32 // tracked body mass; grows toward Genome.Mass each tick via GrowMass
	SurvivalMass         float32 // Smallest mass required for survival
	Dopamine             float32
	Stomach              float32 // current food mass in stomach; digested into energy each tick
	LastActionMask       uint16
	LastDopamine         float32
	LastStomach          float32
	LastLoc              world.Position
	IsResting            bool
	ReproductionCooldown int
	Color                color.RGBA
	Radius               float32
	Tier                 byte

	// Genome-derived constants cached at construction to avoid recomputing each tick.
	halfFOVCos           float32 // math.Cos(FieldOfView/2 in radians)
	cachedMetabolicGene  float32 // 0.7 + 0.6*(MetabolicRate/255)
	cachedJuvenilePeriod int     // MinJuvenilePeriod + genome fraction * range
	Sensors              SensorContext
	// Buffers to avoid heap allocation
	SightFoliageBuffer     []int
	SightMeatBuffer        []int
	SightFungiBuffer       []int
	SightCreatureBuffer    []int
	SightCreatureSimBuffer []float32 // parallel to SightCreatureBuffer; genome similarity to self
	LocalCreatureBuffer    []int
	LocalCreatureSimBuffer []float32 // parallel to LocalCreatureBuffer; genome similarity to self

	// simCache is a 256-slot direct-mapped cache of genome similarity results.
	// Lookup: slot = otherID & (simCacheCapacity-1); hit when key matches and peerUID matches.
	simCache [simCacheCapacity]simCacheEntry
}

// NewCreature creates a creature from a genome at the given mass.
// it initialises the creature's traits and constructs the neurology.
func NewCreature(id int, loc world.Position, g *Genome, mass float32, p *Parameters) *Creature {
	g.recomputeBytes()
	c := Creature{
		Id:             id,
		Generation:     1.0,
		Age:            0,
		Alive:          true,
		Clock:          int(g.OscPeriod),
		Nnet:           NeuralNet{},
		Loc:            loc,
		BirthLoc:       loc,
		Responsiveness: utils.LerpByteAsFloat32(0, 1, g.Responsiveness),
		Heading:        float32(rand.Float64()*2*math.Pi - math.Pi),
		Genome:         g,
	}

	c.BaseOscTick = 2.0 * math.Pow(5000.0/2.0, float64(c.Genome.OscPeriod)/255.0)
	c.Mass = mass
	c.SurvivalMass = float32(MapGeneToRange(c.Genome.SurvivalMass, p.Creature.MinSurvivalMass, p.Creature.MaxSurvivalMass))
	// Set the radius before vision
	c.UpdateRadius(p)
	c.VisionRadius = float32(MapGeneToRange(c.Genome.VisionRadius, p.Creature.MinVisionRadius, p.Creature.MaxVisionRadius))
	c.initCachedFields(g, p)
	c.Energy = c.MaxEnergy(p)
	c.CreateNeuralNet()
	c.Color = c.CalculateColor(p)
	c.Tier = GetTierFromGeneration(c.Generation, p)
	return &c
}

func NewAdultCreature(id int, loc world.Position, g *Genome, p *Parameters) *Creature {
	g.recomputeBytes()
	c := Creature{
		Id:             id,
		Generation:     1.0,
		Age:            0,
		Alive:          true,
		Clock:          int(g.OscPeriod),
		Nnet:           NeuralNet{},
		Loc:            loc,
		BirthLoc:       loc,
		Responsiveness: utils.LerpByteAsFloat32(0, 1, g.Responsiveness),
		Heading:        float32(rand.Float64()*2*math.Pi - math.Pi),
		Genome:         g,
	}

	c.SurvivalMass = float32(MapGeneToRange(c.Genome.SurvivalMass, p.Creature.MinSurvivalMass, p.Creature.MaxSurvivalMass))
	c.Mass = c.SurvivalMass
	c.UpdateRadius(p)
	c.VisionRadius = float32(MapGeneToRange(c.Genome.VisionRadius, p.Creature.MinVisionRadius, p.Creature.MaxVisionRadius))
	c.initCachedFields(g, p)
	c.Energy = c.MaxEnergy(p)
	c.CreateNeuralNet()
	c.Age = c.cachedJuvenilePeriod
	c.Color = c.CalculateColor(p)
	c.Tier = GetTierFromGeneration(c.Generation, p)
	return &c
}

func (c *Creature) initCachedFields(g *Genome, p *Parameters) {
	fovDegrees := MapGeneToRange(g.FieldOfView, p.Creature.MinFieldOfView, p.Creature.MaxFieldOfView)
	halfAngleRad := (float64(fovDegrees) / 2.0) * (math.Pi / 180.0)
	c.halfFOVCos = float32(math.Cos(halfAngleRad))
	c.cachedMetabolicGene = 0.7 + 0.6*(float32(g.MetabolicRate)/255.0)
	c.cachedJuvenilePeriod = p.Creature.MinJuvenilePeriod + int(float32(g.JuvenilePeriod)/255.0*float32(p.Creature.MaxJuvenilePeriod-p.Creature.MinJuvenilePeriod))
}

// cachedSimilarity returns GenomeSimilarity(c, other) via a 256-slot
// direct-mapped cache. On collision the old entry is simply overwritten.
func (c *Creature) cachedSimilarity(otherID int, other *Creature) float32 {
	slot := &c.simCache[otherID&(simCacheCapacity-1)]
	if slot.key == int32(otherID) && slot.peerUID == other.Genome.uid {
		return slot.sim
	}
	sim := GenomeSimilarity(c.Genome, other.Genome)
	slot.key = int32(otherID)
	slot.peerUID = other.Genome.uid
	slot.sim = sim
	return sim
}

func (c *Creature) CreateNeuralNet() {
	c.Nnet = *CreateNeuralNetworkFromGenome(c.Genome.Brain, c.Genome.CognitiveBreadth)
}

func (c Creature) String() string {
	return fmt.Sprintf("\nCREATURE| \nID: %d,\nEnergy: %f,\nResponsiveness: %f,\nAge: %d,\nAlive: %t,\nClock: %d,\nNnet: \n%s,\nLoc: %v,\nBirthLoc: %v,\nHeading: %f",
		c.Id, c.Energy, c.Responsiveness, c.Age, c.Alive, c.Clock,
		c.Nnet.String(), c.Loc, c.BirthLoc, c.Heading)
}

// MaxEnergy returns the creature's energy storage capacity, derived from current mass.
// Energy capacity scales linearly with body size (larger creatures can store more energy).
func (c Creature) MaxEnergy(params *Parameters) float32 {
	return c.Mass * params.Metabolism.EnergyCapacityPerMass
}

// StomachCapacity returns the maximum food mass this creature's stomach can hold.
func (c Creature) StomachCapacity(params *Parameters) float32 {
	baseFloor := float64(params.Creature.MinSurvivalMass) * 0.5
	upperBound := float64(c.Mass) * 0.5
	if upperBound < baseFloor {
		upperBound = baseFloor
	}
	lowerBound := baseFloor * 0.1
	// 3. Map the gene safely within this expanding window
	return float32(MapGeneToRange(c.Genome.StomachSize, lowerBound, upperBound))
}

// BiteSize returns the maximum mass this creature consumes in a single eating interaction.
// Scales linearly with body mass so smaller creatures take smaller bites.
func (c *Creature) BiteSize(params *Parameters) float32 {
	minSkeletalRadius := float32(math.Sqrt(float64(params.Creature.MinSurvivalMass) * math.Pi))
	radiusScaleFactor := c.Radius / minSkeletalRadius
	return float32(params.Creature.BaseBiteSize) * radiusScaleFactor
}

// GetFoodEfficiency returns the normalised digestion efficiency for the given food
// type. Each of the three raw gene values (foliage, fungi, meat) is divided by
// their sum, so a creature with all genes at maximum still gets 33% per food type.
// Returns 0 if all three genes are zero (edge case: creatures cannot digest anything).
func (c Creature) GetFoodEfficiency(foodType uint8) float32 {
	total := float32(c.Genome.FoliageDigestionEfficiency) +
		float32(c.Genome.FungiDigestionEfficiency) +
		float32(c.Genome.MeatDigestionEfficiency)
	if total == 0 {
		return 0
	}
	switch foodType {
	case world.FoodTypeFoliage:
		return float32(c.Genome.FoliageDigestionEfficiency) / total
	case world.FoodTypeFungi:
		return float32(c.Genome.FungiDigestionEfficiency) / total
	case world.FoodTypeMeat:
		return float32(c.Genome.MeatDigestionEfficiency) / total
	}
	return 0
}

func (c *Creature) Digest(params *Parameters) {
	if c.Stomach <= 0 {
		return
	}

	maxEng := c.MaxEnergy(params)
	energySpace := maxEng - c.Energy
	if energySpace <= 0 {
		return
	}
	absoluteMassScale := float64(c.Mass) / float64(params.Creature.MinSurvivalMass)
	// Efficient M^0.75
	massEffect := math.Sqrt(absoluteMassScale * math.Sqrt(absoluteMassScale))

	currentCap := c.StomachCapacity(params)
	var sizeFactor float64
	if currentCap > 0 {
		standardCap := float64(params.Creature.MinSurvivalMass) * 0.2
		ratio := float64(currentCap) / standardCap
		sizeFactor = math.Sqrt(ratio * math.Sqrt(ratio))
	} else {
		sizeFactor = 1.0
	}

	digestionRate := params.Metabolism.DigestionRate * massEffect * sizeFactor

	// Small creatures at 75% baseline, large creatures 95% efficient
	efficiency := float32(0.75 + (absoluteMassScale * 0.20))
	if efficiency > 0.95 {
		efficiency = 0.95
	}
	if c.IsResting {
		// Resting remains a massive booster, especially rewarding for giant temp-regulation
		restingBoost := 1.5 + (1.5 * absoluteMassScale)
		digestionRate *= restingBoost
	}

	potentialEnergyGain := digestionRate * float64(params.Metabolism.EnergyPerFoodMass*efficiency)
	digested := float32(digestionRate)

	if potentialEnergyGain > float64(energySpace) {
		digested = energySpace / (params.Metabolism.EnergyPerFoodMass * efficiency)
	}

	if digested > c.Stomach {
		digested = c.Stomach
	}

	c.Stomach -= digested
	actualGain := float64(digested) * float64(params.Metabolism.EnergyPerFoodMass*efficiency)
	c.GainEnergy(float32(actualGain), params)
}

func (c *Creature) UpdateRadius(p *Parameters) {
	if c.Mass <= 0 {
		c.Radius = 0
		return
	}
	c.Radius = float32(math.Sqrt(float64(c.Mass) * math.Pi))
}

// JuvenilePeriod returns the number of ticks before this creature is considered an adult.
func (c Creature) JuvenilePeriod() int {
	return c.cachedJuvenilePeriod
}

// IsJuvenile reports whether the creature has not yet completed its juvenile phase.
func (c Creature) IsJuvenile() bool {
	jp := c.JuvenilePeriod()
	return jp > 0 && c.Age < jp
}

// CurrentMass returns the creature's actual tracked body mass.
func (c Creature) CurrentMass() float64 {
	return float64(c.Mass)
}

// GrowMass advances the creature's mass toward Genome.Mass using a von Bertalanffy
// growth curve: slowest at birth, fastest at ~1/3 of adult mass, tapering to zero at adult.
func (c *Creature) GrowMass(params *Parameters, temp float32) {
	bmr := c.MetabolicRate(params, temp)

	survivalBuffer := c.MaxEnergy(params) * 0.15
	// if the energy less than survival buffer + bmr, then don't grown
	if c.Energy <= survivalBuffer+bmr {
		return
	}
	// Physical surplus available
	energySurplus := c.Energy - survivalBuffer - bmr
	// Von Bertalanffy, what portion of our energy do we allocate to growth
	// If BMR (which scales with mass) reaches maximum energy, then less
	// capacity to grow
	somaticScale := c.Mass / c.SurvivalMass
	// Scale Max size based on BodyMass gene
	geneRatio := float32(c.Genome.BodyMass) / 255.0
	progressiveScale := geneRatio * geneRatio
	const minMultiplier float32 = 2.0
	const maxPossibleMultiplier float32 = 200.0
	dynamicMaxSomaticMultiplier := minMultiplier + (progressiveScale * (maxPossibleMultiplier - minMultiplier))
	allocationFactor := float32(1.0) - (somaticScale / dynamicMaxSomaticMultiplier)
	if allocationFactor < 0 {
		allocationFactor = 0
	}
	// Portion of surplus is spent on growth
	const baseGrowthInvestment = 0.05
	energyToInvest := energySurplus * baseGrowthInvestment * allocationFactor
	// Energy is converted into mass for growth; cost = food energy density / synthesis efficiency
	energyCostToBuildMass := params.Metabolism.EnergyPerFoodMass / params.Metabolism.BiosynthesisEfficiency
	actualGrowth := energyToInvest / energyCostToBuildMass

	if actualGrowth > 0.001 {
		c.Mass += float32(actualGrowth)
		c.UpdateRadius(params)
		efficiency := float32(0.75 + (float64(somaticScale) * 0.20))
		if efficiency > 0.95 {
			efficiency = 0.95
		}
		actualEnergyCost := energyToInvest / efficiency
		c.DrainEnergy(float32(actualEnergyCost))
	}
}

func (c *Creature) DrainEnergy(amount float32) {
	c.Energy -= amount
	if c.Energy < 0 {
		c.Energy = 0
	}
}

func (c *Creature) GainEnergy(amount float32, params *Parameters) {
	maxE := c.MaxEnergy(params)
	c.Energy = utils.MinFloat32(maxE, c.Energy+amount)
}

func (c *Creature) GainDopamine(ratio float32) {
	gain := ratio * 2.0

	current := c.Dopamine + gain
	c.Dopamine = 1.2 * (current / (1.0 + float32(math.Abs(float64(current)))))
}

func (c *Creature) LoseDopamine(ratio float32) {
	drop := ratio * 2.0

	c.Dopamine -= drop

	if c.Dopamine < -1.2 {
		c.Dopamine = -1.2
	}
}

func (c Creature) GetVisionRadius() float32 {
	return c.VisionRadius
}

func (c *Creature) CanAffordMassInvestment(massCost float32) bool {
	return (c.Mass - massCost) >= c.SurvivalMass
}

// MetabolicRate returns the basal energy cost per tick.
// Follows Kleiber's Law: absolute BMR scales as Mass^0.75. The genome gene shifts
// efficiency in [0.7, 1.3]. Above the optimal temperature the cost scales up to
// WarmMetabolicMultiplier; at or below optimal there is no metabolic penalty.
func (c Creature) MetabolicRate(params *Parameters, temp float32) float32 {
	// Kleiber's law: BMR ∝ M^0.75, normalised to MetabolicReferenceMass so that
	// BaseBMR is the exact cost at reference body size rather than an unscaled coefficient.
	massEffect := float32(math.Pow(float64(c.Mass), 0.75))
	refEffect := float32(math.Pow(float64(params.Metabolism.MetabolicReferenceMass), 0.75))
	base := params.Metabolism.BaseBMR * (massEffect / refEffect) * c.cachedMetabolicGene

	optTemp := (params.Environment.TempMin + params.Environment.TempMax) / 2
	var tempMult float32 = 1.0
	if temp > optTemp && params.Environment.TempMax > optTemp {
		warmNorm := (temp - optTemp) / (params.Environment.TempMax - optTemp)
		if warmNorm > 1 {
			warmNorm = 1
		}
		tempMult = 1.0 + (params.Environment.WarmMetabolicMultiplier-1.0)*(warmNorm*warmNorm)
	}

	// Gut complexity tax: more investment in digestion genes costs more energy.
	// Max totalGut = 765 (3 × 255); gutTax max ≈ 1.38 (38% overhead).
	totalGut := float32(c.Genome.FoliageDigestionEfficiency) +
		float32(c.Genome.FungiDigestionEfficiency) +
		float32(c.Genome.MeatDigestionEfficiency)
	gutTax := 1.0 + totalGut*0.0005

	return base * tempMult * gutTax
}

// MaxAge returns the creature's maximum lifespan in ticks.
// Lifespan scales with sqrt of actual somatic growth (mass / survival floor), capped at 3x.
// Higher metabolic gene shortens life (rate-of-living theory).
func (c Creature) MaxAge(params *Parameters) int {
	baseLife := float32(params.Creature.BaseMaxAge)
	somaticScale := c.Mass / c.SurvivalMass
	if somaticScale < 1 {
		somaticScale = 1
	}
	sizeMult := float32(math.Sqrt(float64(somaticScale)))
	const maxSizeBoost float32 = 3.0
	if sizeMult > maxSizeBoost {
		sizeMult = maxSizeBoost
	}
	metabolicGeneNorm := float32(c.Genome.MetabolicRate) / 255.0
	metabolicPenalty := 0.75 + metabolicGeneNorm // [0.75, 1.75]
	return int((baseLife * sizeMult) / metabolicPenalty)
}

// CalculateGenerationBonus calculates the bonus a child's generation will
// increase by when reproducing. Living a long time, or having a large
// mass relative to the creature's minimum skeletal mass, will return
// a larger bonus.
func (c *Creature) CalculateGenerationBonus(params *Parameters) float32 {
	somaticScale := c.Mass / c.SurvivalMass
	if somaticScale < 1.0 {
		somaticScale = 1.0 // Safety fallback
	}

	// Instead of clamping hard at 1.0, we use a smooth saturation curve: 1.0 - (1.0 / Scale)
	// - At baseline skeleton (Scale = 1.0): Factor is 0.0
	// - At double skeleton weight (Scale = 2.0): Factor is 0.5
	// - As a colossal titan (Scale -> Infinity): Factor gracefully approaches a hard ceiling of 1.0
	massPerformance := 1.0 - (1.0 / somaticScale)
	growthFactor := massPerformance * massPerformance

	maxAge := float32(c.MaxAge(params))
	// Longevity factor scales up if they lived a long percentage of their lifespan
	longevityFactor := float32(1.0)
	if maxAge > 0 {
		longevityFactor = float32(c.Age) / maxAge
	}
	if longevityFactor > 1.0 {
		longevityFactor = 1.0
	}

	deltaGen := (0.8 * growthFactor) * (0.2 * longevityFactor)
	if deltaGen > 2.0 {
		deltaGen = 2.0
	}
	return deltaGen
}
func calculateFunctionalIntelligence(nn *NeuralNet, g *Genome) float32 {
	if nn == nil || len(nn.Edges) == 0 {
		return 0
	}

	numHidden := float32(len(nn.HiddenNeurons))
	connectivityScore := float32(0.0)
	if numHidden > 0 {
		// Measures how densely intertwined the hidden processing layer is
		density := float32(len(nn.Edges)) / numHidden
		connectivityScore = clamp(density / 15.0)
	}

	activeSensors := 0
	for _, active := range nn.ActiveSensors {
		if active {
			activeSensors++
		}
	}
	totalNodes := float32(activeSensors + len(nn.HiddenNeurons) + len(nn.LastActionValues))
	// Normalize ~30
	nodeScore := clamp(totalNodes / 30.0)
	plasticity := float32(g.Neuroplasticity) / 255.0
	complexityBlueprint := (connectivityScore * 0.45) + (nodeScore * 0.35) + (plasticity * 0.20)

	// INVERSION: The most green (1.0) is the LEAST structurally complex brain.
	return 1.0 - complexityBlueprint
}

func clamp(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func (c *Creature) ApplyReproductionCost(massLoss, energyLoss float32, isFallback bool, params *Parameters) {
	c.Mass -= massLoss
	if isFallback {
		// Using the fallback is penalised
		energyLoss = energyLoss * 1.15
	}
	c.DrainEnergy(energyLoss)
	c.UpdateRadius(params)
}

func (c *Creature) CalculateColor(p *Parameters) color.RGBA {
	g := c.Genome
	if g == nil {
		return color.RGBA{0, 0, 0, 255}
	}

	scale := func(val, min, max float64) uint8 {
		if max == min {
			return 70
		}
		t := (val - min) / (max - min)
		if t < 0 {
			t = 0
		}
		if t > 1 {
			t = 1
		}
		return uint8(t*185 + 70)
	}

	// Red: Physicality (Mass & Metabolism)
	redMass := float64(g.BodyMass)
	redMeta := float64(g.MetabolicRate) / 255.0
	rVal := (redMass/255.0 + redMeta) / 2.0
	red := uint8(rVal*185 + 70)

	// Green: Intelligence (Neuron Count & Brain Complexity)
	iq := calculateFunctionalIntelligence(&c.Nnet, g)
	green := uint8(255 - (iq * 185))

	// Blue: Perception (Sight & FOV)
	blueSight := scale(float64(g.VisionRadius), float64(p.Creature.MinVisionRadius), float64(p.Creature.MaxVisionRadius))
	blueFOV := scale(float64(g.FieldOfView), float64(p.Creature.MinFieldOfView), float64(p.Creature.MaxFieldOfView))
	blue := uint8((uint16(blueSight) + uint16(blueFOV)) / 2)

	// Alpha: 50% to 100% based on MutationRate
	alpha := 255 - uint8((uint16(g.MutationRate)*127)/255)

	return color.RGBA{red, green, blue, alpha}
}

func (c Creature) FieldOfView() float64 {
	return getFOVFromCosine(float64(c.halfFOVCos))
}

func getFOVFromCosine(halfFOVCos float64) float64 {
	halfAngleRad := math.Acos(halfFOVCos)
	fullAngleRad := halfAngleRad * 2.0
	return fullAngleRad * (180.0 / math.Pi)
}
