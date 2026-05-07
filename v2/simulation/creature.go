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
	Genome         *Genome
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
	}
	c.CreateNeuralNet()
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

// CurrentSize returns the creature's effective size, scaling linearly from
// params.MinSize at birth to genome.Size at the end of the juvenile period.
func (c Creature) CurrentSize(params *Parameters) float32 {
	juvenilePeriod := params.MinJuvenilePeriod + int(float32(c.Genome.JuvenilePeriod)/255.0*float32(params.MaxJuvenilePeriod-params.MinJuvenilePeriod))
	if juvenilePeriod == 0 || c.Age >= juvenilePeriod {
		return float32(c.Genome.Size)
	}
	t := float32(c.Age) / float32(juvenilePeriod)
	return float32(params.MinSize) + (float32(c.Genome.Size)-float32(params.MinSize))*t
}
