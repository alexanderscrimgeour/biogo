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
	Id             int
	Generation     float32
	Energy         float32
	LastTickEnergy float32
	Responsiveness float32
	Age            int
	Alive          bool
	Clock          int
	BaseOscTick    float64
	Nnet           NeuralNet
	Loc            world.Position
	BirthLoc       world.Position
	Heading        float32 // radians; 0 = east, π/2 = south (screen-down)
	Velocity       float32 // current speed along heading; updated each tick via ACCELERATE action
	Genome         *Genome
	SightDistance  float32
	Mass           float32 // tracked body mass; grows toward Genome.Mass each tick via GrowMass
	MaxMass        float32
	Dopamine       float32
	Stomach        float32 // current food mass in stomach; digested into energy each tick
	LastActionMask uint16
	LastDopamine   float32
	LastStomach    float32
	LastLoc        world.Position
	IsResting      bool
	Color          color.RGBA
	Radius         float32
	Tier           byte

	// Genome-derived constants cached at construction to avoid recomputing each tick.
	halfFOVCos           float32 // math.Cos(FieldOfView/2 in radians)
	cachedMetabolicGene  float32 // 0.7 + 0.6*(MetabolicRate/255)
	cachedJuvenilePeriod int     // MinJuvenilePeriod + genome fraction * range
	Sensors              SensorContext
	// Buffers to avoid heap allocation
	SightFoodBuffer        []int
	SightMeatBuffer        []int
	SightCreatureBuffer    []int
	SightCreatureSimBuffer []float32 // parallel to SightCreatureBuffer; genome similarity to self
	LocalCreatureBuffer    []int
	LocalCreatureSimBuffer []float32 // parallel to LocalCreatureBuffer; genome similarity to self

	// simCache is a 256-slot direct-mapped cache of genome similarity results.
	// Lookup: slot = otherID & (simCacheCapacity-1); hit when key matches and peerUID matches.
	simCache [simCacheCapacity]simCacheEntry
}

func NewCreature(id int, loc world.Position, g *Genome, p *Parameters) *Creature {
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
	c.Mass = float32(MapGeneToRange(c.Genome.MinMass, float64(3), p.MaxMass))
	c.MaxMass = float32(MapGeneToRange(c.Genome.Mass, float64(3), p.MaxMass))
	c.SightDistance = float32(MapGeneToRange(c.Genome.SightDistance, p.MinSightDistance+float64(c.Radius), p.MaxSightDistance))
	c.initCachedFields(g, p)
	c.Energy = c.MaxEnergy(p)
	c.CreateNeuralNet()
	c.Color = c.CalculateColor(p)
	c.UpdateSize(p)
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

	c.Mass = float32(MapGeneToRange(c.Genome.Mass, float64(3), p.MaxMass))
	c.MaxMass = float32(MapGeneToRange(c.Genome.Mass, float64(3), p.MaxMass))
	c.SightDistance = float32(MapGeneToRange(c.Genome.SightDistance, p.MinSightDistance+float64(c.Radius), p.MaxSightDistance))
	c.initCachedFields(g, p)
	c.Energy = c.MaxEnergy(p)
	c.CreateNeuralNet()
	c.Age = c.cachedJuvenilePeriod
	c.Color = c.CalculateColor(p)
	c.UpdateSize(p)
	c.Tier = GetTierFromGeneration(c.Generation, p)
	return &c
}

func (c *Creature) initCachedFields(g *Genome, p *Parameters) {
	fovDegrees := MapGeneToRange(g.FieldOfView, p.MinFieldOfView, p.MaxFieldOfView)
	halfAngleRad := (float64(fovDegrees) / 2.0) * (math.Pi / 180.0)
	c.halfFOVCos = float32(math.Cos(halfAngleRad))
	c.cachedMetabolicGene = 0.7 + 0.6*(float32(g.MetabolicRate)/255.0)
	c.cachedJuvenilePeriod = p.MinJuvenilePeriod + int(float32(g.JuvenilePeriod)/255.0*float32(p.MaxJuvenilePeriod-p.MinJuvenilePeriod))
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
	return c.Mass * params.EnergyPerMassUnit
}

// StomachCapacity returns the maximum food mass this creature's stomach can hold.
func (c Creature) StomachCapacity(params *Parameters) float32 {
	return float32(MapGeneToRange(c.Genome.StomachSize, 1, float64(c.Mass)))
}

// BiteSize returns the maximum mass this creature consumes in a single eating interaction.
// Scales linearly with body mass so smaller creatures take smaller bites.
func (c Creature) BiteSize(params *Parameters) float32 {
	return float32(params.BaseBiteSize) * (c.Mass / float32(params.MaxMass))
}

// DigestionEfficiencies returns the fraction of food and meat mass this creature
// can absorb into its stomach per bite.
// gene=0 → (1.0, 0.0) pure herbivore; gene=255 → (0.0, 1.0) pure carnivore;
// gene=128 → (~0.5, ~0.5) omnivore.
func (c Creature) DigestionEfficiencies() (foodEff, meatEff float32) {
	dt := float32(c.Genome.DigestionType) / 255.0
	return 1.0 - dt, dt
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

	currentCap := c.StomachCapacity(params)
	var sizeFactor float64
	if currentCap > 0 {
		standardCap := float64(params.MaxMass) * 0.2
		ratio := standardCap / float64(currentCap)
		// x^0.75 = sqrt(ratio * sqrt(ratio))
		sizeFactor = math.Sqrt(ratio * math.Sqrt(ratio))
	} else {
		sizeFactor = 1.0 // Fallback
	}

	massNorm := float64(c.Mass) / params.MaxMass
	// Efficient M^0.75
	massEffect := math.Sqrt(massNorm * math.Sqrt(massNorm))
	digestionRate := params.DigestionRate * massEffect * sizeFactor

	massRatio := c.Mass / float32(params.MaxMass)
	// Higher mass = lower efficiency.
	efficiency := 0.95 - (massRatio * 0.25)
	if c.IsResting {
		restingBoost := 1.5 + (2.0 * float64(massNorm))
		digestionRate *= restingBoost
	}

	potentialEnergyGain := digestionRate * float64(params.EnergyPerMassUnit*efficiency)
	digested := float32(digestionRate)

	if potentialEnergyGain > float64(energySpace) {
		digested = energySpace / (params.EnergyPerMassUnit * efficiency)
	}

	if digested > c.Stomach {
		digested = c.Stomach
	}

	c.Stomach -= digested
	actualGain := float64(digested) * float64(params.EnergyPerMassUnit*efficiency)
	c.GainEnergy(float32(actualGain), params)
}

func (c *Creature) UpdateSize(p *Parameters) {
	if c.Mass <= 0 {
		c.Radius = 0
		return
	}
	c.Radius = float32(math.Sqrt(float64(c.Mass) * math.Pi))
	c.SightDistance = float32(MapGeneToRange(c.Genome.SightDistance, p.MinSightDistance+float64(c.Radius), p.MaxSightDistance))
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
func (c *Creature) GrowMass(params *Parameters) {
	maxMass := c.MaxMass
	if c.Mass >= maxMass {
		c.Mass = maxMass
		c.UpdateSize(params)
		return
	}
	// Snap to full mass when within 1% to avoid asymptotic convergence blocking reproduction.
	if c.Mass >= maxMass*0.99 {
		c.Mass = maxMass
		c.UpdateSize(params)
		return
	}

	survivalBuffer := c.MaxEnergy(params) * 0.10
	if c.Energy <= survivalBuffer {
		return
	}

	energyRatio := float64(c.Energy) / float64(c.MaxEnergy(params))
	var energyFactor float64
	if energyRatio > 0.2 {
		energyFactor = (energyRatio - 0.2) / 0.8
	}
	massRatio := float64(c.Mass) / float64(maxMass)

	// von Bertalanffy rate: peaks at massRatio ≈ 0.33, zero at 0 and 1.
	growthRate := float64(params.MaxGrowthRatePerTick) * math.Sqrt(massRatio) * (1.0 - massRatio)
	actualGrowth := growthRate * energyFactor
	energyCost := actualGrowth * float64(params.GrowthEnergyCostFactor)

	if actualGrowth > 0.001 {
		c.Mass = utils.MinFloat32(maxMass, c.Mass+float32(actualGrowth))
		c.UpdateSize(params)
		c.DrainEnergy(float32(energyCost))
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

func (c Creature) GetSightDistance() float32 {
	return c.SightDistance
}

// MetabolicRate returns the basal energy cost per tick.
// Follows Kleiber's Law: absolute BMR scales as Mass^0.75 — larger creatures
// have higher absolute metabolic costs, creating genuine selective pressure against
// runaway body size. The MetabolicRate genome gene shifts efficiency in [0.7, 1.3].
// Ambient temperature temp (°C) further scales cost: cold environments are more
// expensive to survive in (ColdMetabolicMultiplier at 10°C, WarmMetabolicMultiplier at 40°C).
func (c Creature) MetabolicRate(params *Parameters, temp float32) float32 {
	massNorm := float64(c.Mass) / params.MaxMass

	// x^0.75 is much faster as sqrt(x * sqrt(x))
	massEffect := float32(math.Sqrt(massNorm * math.Sqrt(massNorm)))
	base := params.BaseBMR * massEffect * c.cachedMetabolicGene

	tempNorm := (temp - world.TempCold) / (world.TempWarm - world.TempCold)
	if tempNorm < 0 {
		tempNorm = 0
	} else if tempNorm > 1 {
		tempNorm = 1
	}

	tempMult := params.ColdMetabolicMultiplier + (params.WarmMetabolicMultiplier-params.ColdMetabolicMultiplier)*(tempNorm*tempNorm)

	return base * tempMult
}

// MaxAge returns the creature's maximum lifespan in ticks.
// Larger creatures live longer (rate-of-living theory); higher metabolic gene shortens life.
func (c Creature) MaxAge(params *Parameters) int {
	baseLife := float32(params.BaseMaxAge)
	sizeMult := 0.5 + float32(c.Genome.Mass)/255.0 // [0.5, 1.5]
	metabolicGeneNorm := float32(c.Genome.MetabolicRate) / 255.0
	metabolicPenalty := 0.75 + metabolicGeneNorm // [0.75, 1.75]
	return int((baseLife * sizeMult) / metabolicPenalty)
}

func (c *Creature) CalculateGenerationBonus(params *Parameters) float32 {
	massRatio := c.Mass / c.MaxMass
	if massRatio > 1.0 {
		massRatio = 1.0
	}

	// High-performance squared growth factor
	growthFactor := massRatio * massRatio
	maxAge := float32(c.MaxAge(params))
	// Longevity factor scales up if they lived past their juvenile threshold
	longevityFactor := float32(1.0)
	if maxAge > 0 {
		longevityFactor = float32(c.Age) / maxAge
	}

	deltaGen := growthFactor * longevityFactor
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
	redMass := float64(g.Mass)
	redMeta := float64(g.MetabolicRate) / 255.0
	rVal := (redMass/255.0 + redMeta) / 2.0
	red := uint8(rVal*185 + 70)

	// Green: Intelligence (Neuron Count & Brain Complexity)
	iq := calculateFunctionalIntelligence(&c.Nnet, g)
	green := uint8(255 - (iq * 185))

	// Blue: Perception (Sight & FOV)
	blueSight := scale(float64(g.SightDistance), float64(p.MinSightDistance), float64(p.MaxSightDistance))
	blueFOV := scale(float64(g.FieldOfView), float64(p.MinFieldOfView), float64(p.MaxFieldOfView))
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
