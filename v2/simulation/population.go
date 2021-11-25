package simulation

import "gopop/v2/grid"

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
}
