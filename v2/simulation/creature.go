package simulation

import (
	"biogo/v2/grid"
	"biogo/v2/utils"
	"fmt"
)

type Creature struct {
	Id             int
	Energy         float32
	Responsiveness float32
	Age            int
	Alive          bool
	Clock          int
	Nnet           NeuralNet
	Loc            grid.Coord
	BirthLoc       grid.Coord
	LastMoveDir    grid.Dir
	Genome         *Genome
}

func NewCreature(id int, loc grid.Coord, g *Genome) *Creature {
	c := Creature{
		Id:             id,
		Energy:         float32(g.MaxEnergy),
		Age:            0,
		Alive:          true,
		Clock:          int(g.OscPeriod), // TODO() Maybe fix this?
		Nnet:           NeuralNet{},
		Loc:            loc,
		BirthLoc:       loc,
		Responsiveness: float32(utils.ClampByteAsFloat32(0, 1, g.Responsiveness)) / 2,
		// LastMoveDir: ,
		Genome: g,
	}
	c.CreateNeuralNet()
	return &c
}

// Takes a creature's genome and uses it to build a NeuralNetwork
func (c *Creature) CreateNeuralNet() {
	c.Nnet = *CreateNeuralNetworkFromGenome(c.Genome.Brain, c.Genome.NeuronCount)
}

func (c Creature) String() string {
	return fmt.Sprintf("\nCREATURE| \nID: %d,\nEnergy: %f,\nResponsiveness: %f,\nAge: %d,\nAlive: %t,\nClock: %d,\nNnet: \n%s,\nLoc: %v,\nBirthLoc: %v,\nLastMoveDir%v",
		c.Id,
		c.Energy,
		c.Responsiveness,
		c.Age,
		c.Alive,
		c.Clock,
		c.Nnet.String(),
		c.Loc,
		c.BirthLoc,
		c.LastMoveDir)
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

// CurrentSize returns the creature's effective size, scaling linearly from genome.MinSize
// at birth to genome.Size at the end of the juvenile period.
func (c Creature) CurrentSize(params *Parameters) float32 {
	jp := c.JuvenilePeriod(params)
	if jp == 0 || c.Age >= jp {
		return float32(c.Genome.Size)
	}
	t := float32(c.Age) / float32(jp)
	return float32(c.Genome.MinSize) + (float32(c.Genome.Size)-float32(c.Genome.MinSize))*t
}

func (c Creature) GetNextLoc(d grid.Dir) grid.Coord {
	x := c.Loc.X + d.X
	y := c.Loc.Y + d.Y
	return grid.Coord{
		X: x,
		Y: y,
	}
}
