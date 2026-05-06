package simulation

import (
	"biogo/v2/grid"
	"biogo/v2/utils"
	"math/rand"
)

type Population struct {
	Creatures         map[int]*Creature
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

func NewPopulation(p *Parameters) *Population {
	return &Population{
		Creatures:         make(map[int]*Creature, p.StartingPopulation),
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

// GeneticDiversity samples the population and returns average pairwise genome dissimilarity.
func (p *Population) GeneticDiversity() float32 {
	if len(p.Creatures) < 2 {
		return 0
	}

	keys := make([]int, 0, len(p.Creatures))
	for k := range p.Creatures {
		keys = append(keys, k)
	}

	sampleSize := utils.Min(200, len(keys))
	total := float32(0)
	for i := 0; i < sampleSize; i++ {
		i1 := rand.Intn(len(keys))
		i2 := rand.Intn(len(keys))
		for i2 == i1 {
			i2 = rand.Intn(len(keys))
		}
		c1 := p.Creatures[keys[i1]]
		c2 := p.Creatures[keys[i2]]
		total += 1 - GenomeSimilarity(*c1.Genome, *c2.Genome)
	}
	return total / float32(sampleSize)
}
