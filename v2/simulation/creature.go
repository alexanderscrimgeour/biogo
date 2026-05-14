package simulation

import (
	"biogo/v2/utils"
	"biogo/v2/world"
	"fmt"
	"image/color"
	"math"
	"math/rand"
)

// simCacheEntry records a cached genome similarity result paired with the
// peer's genome uid at the time of computation. A uid mismatch means the
// peer's genome has changed (or its ID was recycled), forcing a recompute.
type simCacheEntry struct {
	peerUID uint64
	sim     float32
}

type Creature struct {
	Id             int
	Energy         float32
	LastTickEnergy float32
	Responsiveness float32
	Age            int
	Alive          bool
	Clock          int
	Nnet           NeuralNet
	Loc            world.Position
	BirthLoc       world.Position
	Heading        float64 // radians; 0 = east, π/2 = south (screen-down)
	Velocity       float64 // current speed along heading; updated each tick via ACCELERATE action
	Genome         *Genome
	SightDistance  float64
	Mass           float64 // tracked body mass; grows toward Genome.Mass each tick via GrowMass
	MaxMass        float64
	Dopamine       float32
	Stomach        float64 // current food mass in stomach; digested into energy each tick
	LastAction     string
	LastDopamine   float32
	LastStomach    float64
	LastLoc        world.Position
	IsResting      bool
	Color          color.RGBA
	Radius         float64

	// Genome-derived constants cached at construction to avoid recomputing each tick.
	halfFOVCos           float64 // math.Cos(FieldOfView/2 in radians)
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

	// simCache memoises GenomeSimilarity results keyed by peer creature ID.
	// Entries are validated against the peer's genome uid to handle ID recycling.
	// Cleared when the cache exceeds 512 entries to bound memory use.
	simCache map[int]simCacheEntry
}

func NewCreature(id int, loc world.Position, g *Genome, p *Parameters) *Creature {
	g.recomputeBytes()
	c := Creature{
		Id:             id,
		Age:            0,
		Alive:          true,
		Clock:          int(g.OscPeriod),
		Nnet:           NeuralNet{},
		Loc:            loc,
		BirthLoc:       loc,
		Responsiveness: float32(utils.LerpByteAsFloat32(0, 1, g.Responsiveness)) / 2,
		Heading:        rand.Float64()*2*math.Pi - math.Pi,
		Genome:         g,
		simCache:       make(map[int]simCacheEntry, 32),
	}

	c.Mass = MapGeneToRange(c.Genome.MinMass, float64(3), p.MaxMass)
	c.MaxMass = MapGeneToRange(c.Genome.Mass, float64(3), p.MaxMass)
	c.SightDistance = MapGeneToRange(c.Genome.SightDistance, p.MinSightDistance+c.Radius, p.MaxSightDistance)
	c.initCachedFields(g, p)
	c.Energy = c.MaxEnergy(p)
	c.CreateNeuralNet()
	c.Color = c.CalculateColor(p)
	c.UpdateSize(p)
	return &c
}

func NewAdultCreature(id int, loc world.Position, g *Genome, p *Parameters) *Creature {
	g.recomputeBytes()
	c := Creature{
		Id:             id,
		Age:            0,
		Alive:          true,
		Clock:          int(g.OscPeriod),
		Nnet:           NeuralNet{},
		Loc:            loc,
		BirthLoc:       loc,
		Responsiveness: float32(utils.LerpByteAsFloat32(0, 1, g.Responsiveness)) / 2,
		Heading:        rand.Float64()*2*math.Pi - math.Pi,
		Genome:         g,
		simCache:       make(map[int]simCacheEntry, 32),
	}

	c.Mass = MapGeneToRange(c.Genome.Mass, float64(3), p.MaxMass)
	c.MaxMass = MapGeneToRange(c.Genome.Mass, float64(3), p.MaxMass)
	c.SightDistance = MapGeneToRange(c.Genome.SightDistance, p.MinSightDistance+c.Radius, p.MaxSightDistance)
	c.initCachedFields(g, p)
	c.Energy = c.MaxEnergy(p)
	c.CreateNeuralNet()
	c.Age = c.cachedJuvenilePeriod
	c.Color = c.CalculateColor(p)
	c.UpdateSize(p)
	return &c
}

func (c *Creature) initCachedFields(g *Genome, p *Parameters) {
	fovDegrees := MapGeneToRange(g.FieldOfView, p.MinFieldOfView, p.MaxFieldOfView)
	halfAngleRad := (float64(fovDegrees) / 2.0) * (math.Pi / 180.0)
	c.halfFOVCos = math.Cos(halfAngleRad)
	c.cachedMetabolicGene = 0.7 + 0.6*(float32(g.MetabolicRate)/255.0)
	c.cachedJuvenilePeriod = p.MinJuvenilePeriod + int(float32(g.JuvenilePeriod)/255.0*float32(p.MaxJuvenilePeriod-p.MinJuvenilePeriod))
}

// cachedSimilarity returns GenomeSimilarity(c, other), using c.simCache to
// avoid recomputation when other's genome has not changed since last lookup.
// The cache is bounded to 512 entries to prevent unbounded memory growth.
func (c *Creature) cachedSimilarity(otherID int, other *Creature) float32 {
	if entry, ok := c.simCache[otherID]; ok && entry.peerUID == other.Genome.uid {
		return entry.sim
	}
	sim := GenomeSimilarity(c.Genome, other.Genome)
	if len(c.simCache) >= 512 {
		clear(c.simCache)
	}
	c.simCache[otherID] = simCacheEntry{peerUID: other.Genome.uid, sim: sim}
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
	return float32(c.Mass) * params.EnergyPerMassUnit
}

// StomachCapacity returns the maximum food mass this creature's stomach can hold.
func (c Creature) StomachCapacity(params *Parameters) float64 {
	return MapGeneToRange(c.Genome.StomachSize, 1, float64(c.Mass))
}

// BiteSize returns the maximum mass this creature consumes in a single eating interaction.
// Scales linearly with body mass so smaller creatures take smaller bites.
func (c Creature) BiteSize(params *Parameters) float64 {
	return params.BaseBiteSize * (float64(c.Mass) / params.MaxMass)
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
	standardCap := float64(params.MaxMass) * 0.2
	sizeFactor := math.Pow(standardCap/currentCap, 0.75)

	massNorm := c.Mass / params.MaxMass
	digestionRate := params.DigestionRate * math.Pow(massNorm, 0.75) * sizeFactor

	massRatio := float32(c.Mass) / float32(params.MaxMass)
	// Higher mass = lower efficiency.
	efficiency := 0.95 - (massRatio * 0.25)
	if c.IsResting {
		digestionRate *= 1.5
	}

	potentialEnergyGain := digestionRate * float64(params.EnergyPerMassUnit*efficiency)
	digested := digestionRate

	if potentialEnergyGain > float64(energySpace) {
		digested = float64(energySpace / (params.EnergyPerMassUnit * efficiency))
	}

	if digested > c.Stomach {
		digested = c.Stomach
	}

	c.Stomach -= digested
	actualGain := digested * float64(params.EnergyPerMassUnit*efficiency)
	c.GainEnergy(float32(actualGain), params)
}

func (c *Creature) UpdateSize(p *Parameters) {
	if c.Mass <= 0 {
		c.Radius = 0
		return
	}
	c.Radius = math.Sqrt(float64(c.Mass) * math.Pi)
	c.SightDistance = MapGeneToRange(c.Genome.SightDistance, p.MinSightDistance+c.Radius, p.MaxSightDistance)
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
	return c.Mass
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

	massRatio := c.Mass / maxMass
	// von Bertalanffy rate: peaks at massRatio ≈ 0.33, zero at 0 and 1.
	growthRate := float64(params.MaxGrowthRatePerTick) * math.Sqrt(massRatio) * (1.0 - massRatio)
	energyCost := growthRate * float64(params.GrowthEnergyCostFactor)

	disposableEnergy := float64(c.Energy - survivalBuffer)
	actualGrowth := growthRate
	if energyCost > disposableEnergy {
		actualGrowth = disposableEnergy / float64(params.GrowthEnergyCostFactor)
		energyCost = disposableEnergy
	}

	c.Mass = utils.MinFloat64(maxMass, c.Mass+actualGrowth)
	c.UpdateSize(params)
	c.DrainEnergy(float32(energyCost))
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

func (c Creature) GetSightDistance() float64 {
	return c.SightDistance
}

// MetabolicRate returns the basal energy cost per tick.
// Follows Kleiber's Law: absolute BMR scales as Mass^0.75 — larger creatures
// have higher absolute metabolic costs, creating genuine selective pressure against
// runaway body size. The MetabolicRate genome gene shifts efficiency in [0.7, 1.3].
// Ambient temperature temp (°C) further scales cost: cold environments are more
// expensive to survive in (ColdMetabolicMultiplier at 10°C, WarmMetabolicMultiplier at 40°C).
func (c Creature) MetabolicRate(params *Parameters, temp float32) float32 {
	massNorm := c.Mass / params.MaxMass
	base := params.BaseBMR * float32(math.Pow(massNorm, 0.75)) * c.cachedMetabolicGene
	tempNorm := (temp - world.TempCold) / (world.TempWarm - world.TempCold)
	if tempNorm < 0 {
		tempNorm = 0
	} else if tempNorm > 1 {
		tempNorm = 1
	}
	tempNorm = float32(math.Pow(float64(tempNorm), 2))
	tempMult := params.ColdMetabolicMultiplier + (params.WarmMetabolicMultiplier-params.ColdMetabolicMultiplier)*tempNorm
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

func calculateFunctionalIntelligence(nn *NeuralNet, g *Genome) float32 {
	if nn == nil || len(nn.Edges) == 0 {
		return 0
	}

	numHidden := float32(len(nn.HiddenNeuronIDs))
	if numHidden == 0 {
		numHidden = 1
	} // Avoid division by zero for reflex-only brains

	density := float32(len(nn.Edges)) / numHidden
	connectivityScore := clamp(density / 15.0)

	var weightSum float32
	for _, w := range nn.Weights {
		if w < 0 {
			weightSum -= w
		} else {
			weightSum += w
		}
	}
	avgWeight := weightSum / float32(len(nn.Weights))
	weightScore := clamp(avgWeight / 2.0)

	plasticity := float32(g.Neuroplasticity) / 255.0

	activeSensors := 0
	for _, active := range nn.ActiveSensors {
		if active {
			activeSensors++
		}
	}
	ioBreadth := clamp(float32(activeSensors+len(nn.LastActionValues)) / 20.0)

	iq := (connectivityScore * 0.45) + (plasticity * 0.25) + (weightScore * 0.20) + (ioBreadth * 0.10)

	return iq
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
	return getFOVFromCosine(c.halfFOVCos)
}

func getFOVFromCosine(halfFOVCos float64) float64 {
	halfAngleRad := math.Acos(halfFOVCos)
	fullAngleRad := halfAngleRad * 2.0
	return fullAngleRad * (180.0 / math.Pi)
}
