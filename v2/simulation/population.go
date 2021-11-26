package simulation

import (
	"biogo/v2/grid"
	"biogo/v2/utils"
	"math/rand"
)

type Population struct {
	Creatures         []*Creature
	DeathQueue        []DeathInstruction
	MoveQueue         []MoveInstruction
	ReproductionQueue []ReproductionInstruction
}

type DeathInstruction struct {
	Creature *Creature
}

type ReproductionInstruction struct {
	Creature *Creature
}

type MoveInstruction struct {
	Creature *Creature
	Loc      grid.Coord
}

func NewPopulation() *Population {
	creatures := make([]*Creature, Params.StartingPopulation)
	return &Population{
		Creatures:         creatures,
		DeathQueue:        []DeathInstruction{},
		MoveQueue:         []MoveInstruction{},
		ReproductionQueue: []ReproductionInstruction{},
	}
}

func (p *Population) QueueForMove(creature *Creature, newLoc grid.Coord) {
	instruction := MoveInstruction{creature, newLoc}
	p.MoveQueue = append(p.MoveQueue, instruction)
}

func (p *Population) ProcessMoveQueue(g *grid.Grid) {
	for _, instruction := range p.MoveQueue {
		if g.IsEmptyAt(instruction.Loc) {
			g.Set(instruction.Creature.Loc, 0)
			g.Set(instruction.Loc, instruction.Creature.Id)
			instruction.Creature.LastMoveDir = grid.GetDirection(instruction.Creature.Loc, instruction.Loc)
			instruction.Creature.Loc = instruction.Loc
		}
	}
	p.MoveQueue = []MoveInstruction{}
}

// Random sample of population and compare genetics
func (p *Population) GeneticDiversity() float32 {
	if len(p.Creatures) < 2 {
		return 0
	}

	sampleSize := utils.Min(200, len(p.Creatures))
	count := sampleSize
	genomeSimilarityTotal := float32(0)
	for count > 0 {
		i1 := rand.Intn(len(p.Creatures))
		i2 := rand.Intn(len(p.Creatures))
		for i2 == i1 {
			i2 = rand.Intn(len(p.Creatures))
		}
		c1 := p.Creatures[i1]
		c2 := p.Creatures[i2]
		genomeSimilarityTotal += 1 - GenomeSimilarity(*c1.Genome, *c2.Genome)
		sampleSize--
	}
	return genomeSimilarityTotal / float32(sampleSize)
}
