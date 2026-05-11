package simulation

import (
	"biogo/v2/grid"
	"biogo/v2/utils"
	"fmt"
	"image/color"
	"math"
	"math/rand"
)

type Creature struct {
	Id             int
	Energy         float32
	LastTickEnergy float32
	Responsiveness float32
	Age            int
	Alive          bool
	Clock          int
	Nnet           NeuralNet
	Loc            grid.Position
	BirthLoc       grid.Position
	Heading        float64 // radians; 0 = east, π/2 = south (screen-down)
	LastAction     string
	Genome         *Genome
	Mass           float32 // tracked body mass; grows toward Genome.Mass each tick via GrowMass
	Dopamine       float32
	LastDopamine   float32
	Stomach        float32 // current food mass in stomach; digested into energy each tick
	LastStomach    float32
	IsResting      bool
	Color          color.RGBA

	// Genome-derived constants cached at construction to avoid recomputing each tick.
	halfFOVCos           float64 // math.Cos(FieldOfView/2 in radians)
	cachedMetabolicGene  float32 // 0.7 + 0.6*(MetabolicRate/255)
	cachedJuvenilePeriod int     // MinJuvenilePeriod + genome fraction * range
	Sensors              SensorContext
	// Buffers to avoid heap allocation
	SightFoodBuffer     []int
	SightCreatureBuffer []int
	LocalCreatureBuffer []int
}

func NewCreature(id int, loc grid.Position, g *Genome, p *Parameters) *Creature {
	c := Creature{
		Id:             id,
		Age:            0,
		Alive:          true,
		Clock:          int(g.OscPeriod),
		Nnet:           NeuralNet{},
		Loc:            loc,
		BirthLoc:       loc,
		Responsiveness: float32(utils.ClampByteAsFloat32(0, 1, g.Responsiveness)) / 2,
		Heading:        rand.Float64()*2*math.Pi - math.Pi,
		Genome:         g,
		Mass:           float32(g.MinMass),
	}
	c.initCachedFields(g, p)
	c.Energy = c.MaxEnergy(p)
	c.CreateNeuralNet()
	c.Color = c.CalculateColor(p)
	return &c
}

func NewAdultCreature(id int, loc grid.Position, g *Genome, p *Parameters) *Creature {
	c := Creature{
		Id:             id,
		Age:            0,
		Alive:          true,
		Clock:          int(g.OscPeriod),
		Nnet:           NeuralNet{},
		Loc:            loc,
		BirthLoc:       loc,
		Responsiveness: float32(utils.ClampByteAsFloat32(0, 1, g.Responsiveness)) / 2,
		Heading:        rand.Float64()*2*math.Pi - math.Pi,
		Genome:         g,
		Mass:           float32(g.Mass),
	}
	c.initCachedFields(g, p)
	c.Energy = c.MaxEnergy(p)
	c.CreateNeuralNet()
	c.Age = c.cachedJuvenilePeriod
	c.Color = c.CalculateColor(p)
	return &c
}

func (c *Creature) initCachedFields(g *Genome, p *Parameters) {
	c.halfFOVCos = math.Cos(float64(g.FieldOfView) / 2.0 * math.Pi / 180.0)
	c.cachedMetabolicGene = 0.7 + 0.6*(float32(g.MetabolicRate)/255.0)
	c.cachedJuvenilePeriod = p.MinJuvenilePeriod + int(float32(g.JuvenilePeriod)/255.0*float32(p.MaxJuvenilePeriod-p.MinJuvenilePeriod))
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
// The gene controls capacity per unit of MaxMass, scaled by current mass, so larger
// creatures always have proportionally bigger stomachs regardless of their genome mass target.
func (c Creature) StomachCapacity(params *Parameters) float32 {
	base := params.MinStomachSize + float32(c.Genome.StomachSize)/255.0*(params.MaxStomachSize-params.MinStomachSize)
	return base * c.Mass / float32(params.MaxMass)
}

// BiteSize returns the maximum mass this creature consumes in a single eating interaction.
// Scales linearly with body mass so smaller creatures take smaller bites.
func (c Creature) BiteSize(params *Parameters) float32 {
	return params.BaseBiteSize * c.Mass / float32(c.Genome.Mass)
}

// Digest converts stomach contents into energy at DigestionRate mass units per tick.
// Digestion is gated by available energy capacity: if the creature is already at max
// energy the stomach contents are held and converted only once energy has been spent.
func (c *Creature) Digest(params *Parameters) {
	if c.Stomach <= 0 {
		return
	}

	maxEng := c.MaxEnergy(params)
	energySpace := maxEng - c.Energy
	if energySpace <= 0 {
		return
	}

	massNorm := c.Mass / float32(params.MaxMass)
	digestionRate := params.DigestionRate * float32(math.Pow(float64(massNorm), 0.75))
	efficiency := float32(0.8)
	if c.IsResting {
		digestionRate *= 1.5
		efficiency = 0.95
	}

	potentialEnergyGain := digestionRate * params.EnergyPerMassUnit * efficiency

	digested := digestionRate

	if potentialEnergyGain > energySpace {
		digested = energySpace / (params.EnergyPerMassUnit * efficiency)
	}

	if digested > c.Stomach {
		digested = c.Stomach
	}

	c.Stomach -= digested
	actualGain := digested * params.EnergyPerMassUnit * efficiency
	c.GainEnergy(actualGain, params)
}

// JuvenilePeriod returns the number of ticks before this creature is considered an adult.
func (c Creature) JuvenilePeriod(_ *Parameters) int {
	return c.cachedJuvenilePeriod
}

// IsJuvenile reports whether the creature has not yet completed its juvenile phase.
func (c Creature) IsJuvenile(params *Parameters) bool {
	jp := c.JuvenilePeriod(params)
	return jp > 0 && c.Age < jp
}

// CurrentMass returns the creature's actual tracked body mass.
func (c Creature) CurrentMass(params *Parameters) float32 {
	return c.Mass
}

// GrowMass advances the creature's mass toward Genome.Mass using a von Bertalanffy
// growth curve: slowest at birth, fastest at ~1/3 of adult mass, tapering to zero at adult.
func (c *Creature) GrowMass(params *Parameters) {
	maxMass := float32(c.Genome.Mass)
	if c.Mass >= maxMass {
		c.Mass = maxMass
		return
	}
	// Snap to full mass when within 1% to avoid asymptotic convergence blocking reproduction.
	if c.Mass >= maxMass*0.99 {
		c.Mass = maxMass
		return
	}

	survivalBuffer := c.MaxEnergy(params) * 0.10
	if c.Energy <= survivalBuffer {
		return
	}

	massRatio := c.Mass / maxMass
	// von Bertalanffy rate: peaks at massRatio ≈ 0.33, zero at 0 and 1.
	growthRate := params.MaxGrowthRatePerTick * float32(math.Sqrt(float64(massRatio))) * (1.0 - massRatio)
	energyCost := growthRate * params.GrowthEnergyCostFactor

	disposableEnergy := c.Energy - survivalBuffer
	actualGrowth := growthRate
	if energyCost > disposableEnergy {
		actualGrowth = disposableEnergy / params.GrowthEnergyCostFactor
		energyCost = disposableEnergy
	}

	c.Mass = utils.MinFloat32(maxMass, c.Mass+actualGrowth)
	c.DrainEnergy(energyCost)
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
	spike := ratio * 10
	c.Dopamine += spike
	if c.Dopamine > 1.2 {
		c.Dopamine = 1.2
	}
}

func (c *Creature) LoseDopamine(ratio float32) {
	spike := ratio * 10
	c.Dopamine -= spike
	if c.Dopamine < -1.2 {
		c.Dopamine = -1.2
	}
}

// MetabolicRate returns the basal energy cost per tick.
// Follows Kleiber's Law: absolute BMR scales as Mass^0.75 — larger creatures
// have higher absolute metabolic costs, creating genuine selective pressure against
// runaway body size. The MetabolicRate genome gene shifts efficiency in [0.7, 1.3].
func (c Creature) MetabolicRate(params *Parameters) float32 {
	massNorm := c.Mass / float32(params.MaxMass)
	return params.BaseBMR * float32(math.Pow(float64(massNorm), 0.75)) * c.cachedMetabolicGene
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
