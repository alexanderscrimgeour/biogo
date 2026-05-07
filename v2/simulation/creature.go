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
}

func NewCreature(id int, loc grid.Position, g *Genome) *Creature {
	c := Creature{
		Id:             id,
		Energy:         float32(g.MaxEnergy),
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
	c.CreateNeuralNet()
	return &c
}

func NewAdultCreature(id int, loc grid.Position, g *Genome, p *Parameters) *Creature {
	c := Creature{
		Id:             id,
		Energy:         float32(g.MaxEnergy),
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

// GrowMass advances the creature's mass toward Genome.Mass by one tick's worth of growth.
// Growth rate is linear: MinMass → Mass over JuvenilePeriod ticks.
func (c *Creature) GrowMass(params *Parameters) {
	maxMass := float32(c.Genome.Mass)
	if c.Mass >= maxMass {
		c.Mass = maxMass
		return
	}
	jp := c.JuvenilePeriod(params)
	if jp <= 0 {
		c.Mass = maxMass
		return
	}
	growthPerTick := (maxMass - float32(c.Genome.MinMass)) / float32(jp)
	c.Mass = utils.MinFloat32(maxMass, c.Mass+growthPerTick)
}

// MetabolicRate returns the energy drained per tick. The genome byte scales into
// [MinMetabolicRate, MaxMetabolicRate], then an inverse-size modifier is applied so
// smaller creatures burn energy faster (modifier = 2.0 at zero mass, 1.0 at MaxMass).
func (c Creature) MetabolicRate(params *Parameters) float32 {
	base := params.MinMetabolicRate + float32(c.Genome.MetabolicRate)/255.0*(params.MaxMetabolicRate-params.MinMetabolicRate)
	massNorm := c.Mass / float32(params.MaxMass)
	return base * (2.0 - massNorm)
}

func (c Creature) MaxAge(params *Parameters) int {
	baseLife := float32(params.BaseMaxAge)
	sizeMult := 0.5 + (float32(c.Genome.Mass) / 255.0)
	metabolicPenalty := 1.0 + (float32(c.Genome.Responsiveness) / 255.0)
	return int((baseLife * sizeMult) / metabolicPenalty)
}
