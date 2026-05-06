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
	p.MoveQueue = append(p.MoveQueue, MoveInstruction{creature, newLoc})
}

func (p *Population) QueueForDeath(creature *Creature) {
	p.DeathQueue = append(p.DeathQueue, DeathInstruction{creature})
}

func (p *Population) QueueForReproduction(creature *Creature) {
	p.ReproductionQueue = append(p.ReproductionQueue, ReproductionInstruction{creature})
}

// AliveCount returns the number of living (non-corpse) creatures.
func (p *Population) AliveCount() int {
	count := 0
	for _, c := range p.Creatures {
		if c.Alive {
			count++
		}
	}
	return count
}

func (p *Population) ProcessMoveQueue(g *grid.Grid, params *Parameters) {
	for _, instruction := range p.MoveQueue {
		c := instruction.Creature
		if !c.Alive {
			continue
		}
		targetLoc := instruction.Loc
		cellVal := g.At(targetLoc)

		switch {
		case cellVal == grid.EMPTY:
			g.Set(c.Loc, grid.EMPTY)
			g.Set(targetLoc, c.Id)
			c.LastMoveDir = grid.GetDirection(c.Loc, targetLoc)
			c.Loc = targetLoc

		case cellVal == grid.FOOD:
			maxE := float32(c.Genome.MaxEnergy)
			c.Energy = utils.MinFloat32(maxE, c.Energy+params.FoodEnergyFraction*maxE)
			g.RemoveFood(targetLoc)
			g.Set(c.Loc, grid.EMPTY)
			g.Set(targetLoc, c.Id)
			c.LastMoveDir = grid.GetDirection(c.Loc, targetLoc)
			c.Loc = targetLoc

		case cellVal >= grid.RESERVED_CELL_TYPES:
			target, ok := p.Creatures[cellVal]
			if !ok || target == c {
				break
			}
			maxE := float32(c.Genome.MaxEnergy)
			gain := params.PreyEnergyFraction * float32(target.Genome.Size)
			c.Energy = utils.MinFloat32(maxE, c.Energy+gain)

			if target.Alive {
				// Predation: kill the prey in place; predator does not move.
				target.Alive = false
				target.Energy = float32(target.Genome.Size)
			} else {
				// Scavenging: consume the corpse, move into its cell.
				delete(p.Creatures, target.Id)
				g.Set(c.Loc, grid.EMPTY)
				g.Set(targetLoc, c.Id)
				c.LastMoveDir = grid.GetDirection(c.Loc, targetLoc)
				c.Loc = targetLoc
			}
		}
	}
	p.MoveQueue = []MoveInstruction{}
}

// ProcessDeathQueue marks queued creatures as dead and resets their energy to their
// size-based food value. Corpses remain on the grid and decay over time via ProcessCorpseDecay.
func (p *Population) ProcessDeathQueue(g *grid.Grid) {
	for _, di := range p.DeathQueue {
		di.Creature.Alive = false
		di.Creature.Energy = float32(di.Creature.Genome.Size)
	}
	p.DeathQueue = []DeathInstruction{}
}

// ProcessCorpseDecay drains energy from every dead creature. Corpses that reach
// zero energy are removed from the grid and population map.
func (p *Population) ProcessCorpseDecay(g *grid.Grid, params *Parameters) {
	for id, c := range p.Creatures {
		if c.Alive {
			continue
		}
		c.Energy -= params.CorpseDecayRate
		if c.Energy <= 0 {
			g.Set(c.Loc, grid.EMPTY)
			delete(p.Creatures, id)
		}
	}
}

// ProcessReproductionQueue spawns offspring from queued parents. nextID is called to
// allocate a fresh creature ID. Reproduction is skipped if the population is at capacity
// or the parent no longer has enough energy.
func (p *Population) ProcessReproductionQueue(g *grid.Grid, params *Parameters, nextID func() int) {
	for _, ri := range p.ReproductionQueue {
		if p.AliveCount() >= params.MaxPopulation {
			break
		}
		parent := ri.Creature
		if !parent.Alive {
			continue
		}
		cost := params.ReproductionEnergyCost * float32(parent.Genome.MaxEnergy)
		if parent.Energy < cost {
			continue
		}

		offspringLoc, ok := findOffspringLocation(g, parent)
		if !ok {
			continue
		}

		parent.Energy -= cost

		childGenome := AsexualReproduction(parent.Genome, params)
		id := nextID()
		child := NewCreature(id, offspringLoc, childGenome)
		child.Energy = cost / 2
		p.Creatures[id] = child
		g.Set(offspringLoc, id)
	}
	p.ReproductionQueue = []ReproductionInstruction{}
}

// findOffspringLocation returns an empty cell for the offspring. It prefers the cell
// 2 steps behind the parent (opposite of LastMoveDir), falling back to any adjacent empty cell.
func findOffspringLocation(g *grid.Grid, parent *Creature) (grid.Coord, bool) {
	d := parent.LastMoveDir
	if (d.X != 0 || d.Y != 0) {
		behind := grid.Coord{X: parent.Loc.X - 2*d.X, Y: parent.Loc.Y - 2*d.Y}
		if g.IsInBounds(behind) && g.IsEmptyAt(behind) {
			return behind, true
		}
	}

	dirs := []grid.Dir{
		{X: 1, Y: 0}, {X: -1, Y: 0}, {X: 0, Y: 1}, {X: 0, Y: -1},
		{X: 1, Y: 1}, {X: -1, Y: 1}, {X: 1, Y: -1}, {X: -1, Y: -1},
	}
	for _, dir := range dirs {
		loc := grid.Coord{X: parent.Loc.X + dir.X, Y: parent.Loc.Y + dir.Y}
		if g.IsInBounds(loc) && g.IsEmptyAt(loc) {
			return loc, true
		}
	}
	return grid.Coord{}, false
}

// OldestGenome returns the genome of the oldest living creature, or nil if there are none.
// The oldest survivor is used as a fitness proxy when replenishing a depleted population.
func (p *Population) OldestGenome() *Genome {
	var oldest *Creature
	for _, c := range p.Creatures {
		if !c.Alive {
			continue
		}
		if oldest == nil || c.Age > oldest.Age {
			oldest = c
		}
	}
	if oldest == nil {
		return nil
	}
	return oldest.Genome
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
