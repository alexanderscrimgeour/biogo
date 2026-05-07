package simulation

import (
	"biogo/v2/grid"
	"biogo/v2/utils"
	"math"
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
	Loc      grid.Position
}

func NewPopulation(p *Parameters) *Population {
	return &Population{
		Creatures:         make(map[int]*Creature, p.StartingPopulation),
		DeathQueue:        []DeathInstruction{},
		MoveQueue:         []MoveInstruction{},
		ReproductionQueue: []ReproductionInstruction{},
	}
}

func (p *Population) QueueForMove(creature *Creature, newLoc grid.Position) {
	p.MoveQueue = append(p.MoveQueue, MoveInstruction{creature, newLoc})
}

func (p *Population) QueueForDeath(creature *Creature) {
	p.DeathQueue = append(p.DeathQueue, DeathInstruction{creature})
}

func (p *Population) QueueForReproduction(creature *Creature) {
	p.ReproductionQueue = append(p.ReproductionQueue, ReproductionInstruction{creature})
}

func (p *Population) AliveCount() int {
	count := 0
	for _, c := range p.Creatures {
		if c.Alive {
			count++
		}
	}
	return count
}

// ProcessMoveQueue moves each queued creature to its target position, then checks
// for food and creatures in proximity to trigger eating and predation.
func (p *Population) ProcessMoveQueue(w *grid.World, params *Parameters) {
	for _, instruction := range p.MoveQueue {
		c := instruction.Creature
		if !c.Alive {
			continue
		}
		newPos := instruction.Loc

		// Eat the closest food within interaction radius.
		foodIDs := w.GetFoodInRadius(newPos, params.FoodInteractionRadius)
		if len(foodIDs) > 0 {
			closestID := foodIDs[0]
			closestDist := math.MaxFloat64
			for _, fid := range foodIDs {
				fpos := w.GetFoodPos(fid)
				dx := fpos.X - newPos.X
				dy := fpos.Y - newPos.Y
				d := math.Sqrt(dx*dx + dy*dy)
				if d < closestDist {
					closestDist = d
					closestID = fid
				}
			}
			maxE := float32(c.Genome.MaxEnergy)
			c.Energy = utils.MinFloat32(maxE, c.Energy+params.FoodEnergyFraction*maxE)
			w.RemoveFood(closestID)
		}

		// Predate or scavenge the closest creature within predation radius.
		nearbyIDs := w.GetCreaturesInRadius(newPos, params.PredationRadius)
		for _, targetID := range nearbyIDs {
			if targetID == c.Id {
				continue
			}
			target, ok := p.Creatures[targetID]
			if !ok {
				continue
			}
			maxE := float32(c.Genome.MaxEnergy)
			targetSize := target.CurrentSize(params)
			gain := params.PreyEnergyFraction * targetSize
			c.Energy = utils.MinFloat32(maxE, c.Energy+gain)

			if target.Alive {
				// Predation: kill prey in place; corpse stays in world.
				target.Alive = false
				target.Energy = targetSize
			} else {
				// Scavenging: consume and remove the corpse.
				w.RemoveCreature(target.Id)
				delete(p.Creatures, target.Id)
			}
			break
		}

		// Move the creature.
		w.MoveCreature(c.Id, newPos)
		c.Loc = newPos
	}
	p.MoveQueue = []MoveInstruction{}
}

// ProcessDeathQueue marks queued creatures as dead and sets their energy to their
// size-based food value. Corpses remain in the world and decay over time.
func (p *Population) ProcessDeathQueue(w *grid.World, params *Parameters) {
	for _, di := range p.DeathQueue {
		di.Creature.Alive = false
		di.Creature.Energy = di.Creature.CurrentSize(params)
	}
	p.DeathQueue = []DeathInstruction{}
}

// ProcessCorpseDecay drains energy from every dead creature. Fully decayed
// corpses are removed from both the world and the population map.
func (p *Population) ProcessCorpseDecay(w *grid.World, params *Parameters) {
	for id, c := range p.Creatures {
		if c.Alive {
			continue
		}
		c.Energy -= params.CorpseDecayRate
		if c.Energy <= 0 {
			w.RemoveCreature(id)
			delete(p.Creatures, id)
		}
	}
}

// ProcessReproductionQueue spawns offspring near queued parents.
func (p *Population) ProcessReproductionQueue(w *grid.World, params *Parameters, nextID func() int) {
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

		offspringLoc, ok := findOffspringLocation(w, parent)
		if !ok {
			continue
		}

		parent.Energy -= cost
		childGenome := AsexualReproduction(parent.Genome, params)
		id := nextID()
		child := NewCreature(id, offspringLoc, childGenome)
		child.Energy = cost / 2
		p.Creatures[id] = child
		w.AddCreature(id, offspringLoc)
	}
	p.ReproductionQueue = []ReproductionInstruction{}
}

// findOffspringLocation returns a free position for an offspring, preferring a
// spot 5 units behind the parent and falling back to random nearby positions.
func findOffspringLocation(w *grid.World, parent *Creature) (grid.Position, bool) {
	backX := -math.Cos(parent.Heading) * 5.0
	backY := -math.Sin(parent.Heading) * 5.0
	behind := grid.Position{X: parent.Loc.X + backX, Y: parent.Loc.Y + backY}
	if w.IsInBounds(behind) && !w.IsWall(behind) {
		return behind, true
	}
	for i := 0; i < 20; i++ {
		angle := rand.Float64() * 2 * math.Pi
		dist := rand.Float64()*8.0 + 2.0
		pos := grid.Position{
			X: parent.Loc.X + math.Cos(angle)*dist,
			Y: parent.Loc.Y + math.Sin(angle)*dist,
		}
		if w.IsInBounds(pos) && !w.IsWall(pos) {
			return pos, true
		}
	}
	return grid.Position{}, false
}

// OldestGenome returns the genome of the oldest living creature, or nil if none.
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

// GeneticDiversity samples the population and returns average pairwise dissimilarity.
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
