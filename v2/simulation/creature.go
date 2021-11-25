package simulation

import (
	"fmt"
	"gopop/v2/grid"
	"math"
)

type Creature struct {
	Id          int
	Energy      float32
	Age         int
	Alive       bool
	Nnet        NeuralNet
	Loc         grid.Coord
	BirthLoc    grid.Coord
	LastMoveDir grid.Dir
	Genome      *Genome
}

func NewCreature(id int, loc grid.Coord, g *Genome) *Creature {
	c := Creature{
		Id:       id,
		Energy:   float32(g.MaxEnergy / math.MaxUint8),
		Age:      0,
		Alive:    true,
		Nnet:     NeuralNet{},
		Loc:      loc,
		BirthLoc: loc,
		// LastMoveDir: ,
		Genome: g,
	}
	c.CreateNeuralNet()
	return &c
}

// Takes a creature's genome and uses it to build a NeuralNetwork
func (c *Creature) CreateNeuralNet() {
	c.Nnet = *CreateNeuralNetworkFromGenome(c.Genome.Neurology, c.Genome.NeuronCount)
}

func (c Creature) String() string {
	return fmt.Sprintf("\nCREATURE| \nID: %d,\nEnergy: %f,\nAge: %d,\nAlive: %t,\nNnet: \n%s,\nLoc: %v,\nBirthLoc: %v,\nLastMoveDir%v",
		c.Id,
		c.Energy,
		c.Age,
		c.Alive,
		c.Nnet.String(),
		c.Loc,
		c.BirthLoc,
		c.LastMoveDir)
}
