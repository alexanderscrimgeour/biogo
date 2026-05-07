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
	EatQueue          []EatInstruction
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

type EatInstruction struct {
	Predator *Creature
	TargetID int
}

// pendingInstructions accumulates instructions produced by a single goroutine's
// creature batch before they are merged into the shared Population queues.
type pendingInstructions struct {
	death        []DeathInstruction
	move         []MoveInstruction
	eat          []EatInstruction
	reproduction []ReproductionInstruction
}

func NewPopulation(p *Parameters) *Population {
	return &Population{
		Creatures:         make(map[int]*Creature, p.StartingPopulation),
		DeathQueue:        []DeathInstruction{},
		MoveQueue:         []MoveInstruction{},
		EatQueue:          []EatInstruction{},
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

func (p *Population) QueueForEat(predator *Creature, targetID int) {
	p.EatQueue = append(p.EatQueue, EatInstruction{predator, targetID})
}

// AliveIDs returns the IDs of all currently alive creatures.
func (p *Population) AliveIDs() []int {
	ids := make([]int, 0, len(p.Creatures))
	for id, c := range p.Creatures {
		if c.Alive {
			ids = append(ids, id)
		}
	}
	return ids
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

// ProcessMoveQueue moves each queued creature to its target position and
// consumes the nearest food item within interaction radius if one is present.
func (p *Population) ProcessMoveQueue(w *grid.World, params *Parameters) {
	for _, instruction := range p.MoveQueue {
		c := instruction.Creature
		if !c.Alive {
			continue
		}
		newPos := instruction.Loc

		// Eat the closest food within interaction radius.
		halfFOVCos := math.Cos(float64(c.Genome.FieldOfView) / 2.0 * math.Pi / 180.0)
		foodIDs := w.GetFoodInCone(newPos, c.Heading, halfFOVCos, params.FoodInteractionRadius)
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

		// Eat the closest Corpse within interaction radius
		creatureIDs := w.GetCreaturesInCone(newPos, c.Heading, halfFOVCos, params.FoodInteractionRadius)
		if len(creatureIDs) > 0 {
			closestCreatureID := -1
			closestDist := math.MaxFloat64
			for _, cid := range creatureIDs {
				if c, ok := p.Creatures[cid]; ok {
					if c.Alive {
						continue
					}
				}
				cpos, ok := w.GetCreaturePos(cid)
				if !ok {
					continue
				}
				dx := cpos.X - newPos.X
				dy := cpos.Y - newPos.Y
				d := math.Sqrt(dx*dx + dy*dy)
				if d < closestDist {
					closestDist = d
					closestCreatureID = cid
				}
			}
			if closestCreatureID != -1 {
				if target, ok := p.Creatures[closestCreatureID]; ok {
					maxE := float32(c.Genome.MaxEnergy)
					c.Energy = utils.MinFloat32(maxE, c.Energy+target.Energy)
					w.RemoveCreature(closestCreatureID)
					delete(p.Creatures, closestCreatureID)
				}
			}
		}

		// Move the creature.
		w.MoveCreature(c.Id, newPos)
		c.Loc = newPos
	}
	p.MoveQueue = []MoveInstruction{}
}

// ProcessEatQueue resolves EAT-action predation queued during the current tick.
// Only living targets are consumed.
func (p *Population) ProcessEatQueue(w *grid.World, params *Parameters) {
	for _, instruction := range p.EatQueue {
		predator := instruction.Predator
		if !predator.Alive {
			continue
		}
		target, ok := p.Creatures[instruction.TargetID]
		if !ok || !target.Alive {
			continue
		}
		targetMass := target.CurrentMass(params)
		gain := targetMass * float32(target.Genome.MaxEnergy) / float32(params.MaxMass)
		maxE := float32(predator.Genome.MaxEnergy)
		predator.Energy = utils.MinFloat32(maxE, predator.Energy+gain)
		target.Alive = false
		target.Energy = targetMass
	}
	p.EatQueue = []EatInstruction{}
}

// ProcessDeathQueue marks queued creatures as dead and sets their energy to their
// mass-based food value. Corpses remain in the world and decay over time.
func (p *Population) ProcessDeathQueue(w *grid.World, params *Parameters) {
	for _, di := range p.DeathQueue {
		di.Creature.Alive = false
		di.Creature.Energy = di.Creature.CurrentMass(params)
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
// Requires the parent to be at full mass and above the energy threshold.
// On reproduction the parent loses energy and half its body mass; the child
// is created at that half-mass size and grows back to full mass over time.
func (p *Population) ProcessReproductionQueue(w *grid.World, params *Parameters, nextID func() int) {
	for _, ri := range p.ReproductionQueue {
		if p.AliveCount() >= params.MaxPopulation {
			break
		}
		parent := ri.Creature
		if !parent.Alive {
			continue
		}

		// Re-check energy threshold (creature may have spent energy since queueing).
		if parent.Energy < params.ReproductionEnergyThreshold*float32(parent.Genome.MaxEnergy) {
			continue
		}

		// Parent must be fully grown before it can split its mass.
		if parent.Mass < float32(parent.Genome.Mass) {
			continue
		}

		// MinMass must be strictly less than half of Mass to guarantee the child
		// starts above the creature's minimum viable size.
		if float32(parent.Genome.MinMass)*2 >= float32(parent.Genome.Mass) {
			continue
		}

		offspringLoc, ok := findOffspringLocation(w, parent)
		if !ok {
			continue
		}

		cost := params.ReproductionEnergyCost * float32(parent.Genome.MaxEnergy)
		parent.Energy -= cost

		// Halve parent's body mass; the parent must regrow before reproducing again.
		halfMass := parent.Mass / 2
		parent.Mass = halfMass

		childGenome := AsexualReproduction(parent.Genome, params)
		id := nextID()
		child := NewCreature(id, offspringLoc, childGenome)
		child.Mass = halfMass
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
