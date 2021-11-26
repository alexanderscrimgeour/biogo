package simulation

import (
	"fmt"
	"gopop/v2/grid"
	"gopop/v2/utils"
	"math"
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
		Energy:         float32(g.MaxEnergy / math.MaxUint8),
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

func (c Creature) GetNextLoc(d grid.Dir) grid.Coord {
	x := c.Loc.X + d.X
	y := c.Loc.Y + d.Y
	return grid.Coord{
		X: x,
		Y: y,
	}
}
