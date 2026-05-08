package simulation

import (
	"biogo/v2/grid"
	"biogo/v2/utils"
	"fmt"
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
	c.Energy = c.MaxEnergy(p)
	c.CreateNeuralNet()
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
	c.Energy = c.MaxEnergy(p)
	c.CreateNeuralNet()
	c.Age = c.JuvenilePeriod(p)
	return &c
}

func (c *Creature) CreateNeuralNet() {
	c.Nnet = *CreateNeuralNetworkFromGenome(c.Genome.Brain, c.Genome.NeuronCount)
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

// JuvenilePeriod returns the number of ticks before this creature is considered an adult.
func (c Creature) JuvenilePeriod(params *Parameters) int {
	return params.MinJuvenilePeriod + int(float32(c.Genome.JuvenilePeriod)/255.0*float32(params.MaxJuvenilePeriod-params.MinJuvenilePeriod))
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
	if maxE > 0 {
		c.GainDopamine(amount / maxE)
	}
}

func (c *Creature) GainDopamine(ratio float32) {
	spike := ratio * 10
	c.Dopamine += spike
	if c.Dopamine > 1.2 {
		c.Dopamine = 1.2
	}
}

// MetabolicRate returns the basal energy cost per tick.
// Follows Kleiber's Law: absolute BMR scales as Mass^0.75 — larger creatures
// have higher absolute metabolic costs, creating genuine selective pressure against
// runaway body size. The MetabolicRate genome gene shifts efficiency in [0.7, 1.3].
func (c Creature) MetabolicRate(params *Parameters) float32 {
	massNorm := c.Mass / float32(params.MaxMass)
	metabolicGene := 0.7 + 0.6*(float32(c.Genome.MetabolicRate)/255.0) // [0.7, 1.3]
	return params.BaseBMR * float32(math.Pow(float64(massNorm), 0.75)) * metabolicGene
}

// MaxAge returns the creature's maximum lifespan in ticks.
// Larger creatures live longer (rate-of-living theory); higher metabolic gene shortens life.
func (c Creature) MaxAge(params *Parameters) int {
	baseLife := float32(params.BaseMaxAge)
	sizeMult := 0.5 + float32(c.Genome.Mass)/255.0        // [0.5, 1.5]
	metabolicGeneNorm := float32(c.Genome.MetabolicRate) / 255.0
	metabolicPenalty := 0.75 + metabolicGeneNorm           // [0.75, 1.75]
	return int((baseLife * sizeMult) / metabolicPenalty)
}
