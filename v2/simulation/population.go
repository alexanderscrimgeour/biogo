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
	Creature   *Creature
	Loc        grid.Position
	MoveAmount float64
}

// pendingInstructions accumulates instructions produced by a single goroutine's
// creature batch before they are merged into the shared Population queues.
type pendingInstructions struct {
	death        []DeathInstruction
	move         []MoveInstruction
	reproduction []ReproductionInstruction
}

func NewPopulation(p *Parameters) *Population {
	return &Population{
		Creatures:         make(map[int]*Creature, p.StartingPopulation),
		DeathQueue:        []DeathInstruction{},
		MoveQueue:         []MoveInstruction{},
		ReproductionQueue: []ReproductionInstruction{},
	}
}

func (p *Population) QueueForMove(creature *Creature, newLoc grid.Position, moveAmount float64) {
	p.MoveQueue = append(p.MoveQueue, MoveInstruction{creature, newLoc, moveAmount})
}

func (p *Population) QueueForDeath(creature *Creature) {
	p.DeathQueue = append(p.DeathQueue, DeathInstruction{creature})
}

func (p *Population) QueueForReproduction(creature *Creature) {
	p.ReproductionQueue = append(p.ReproductionQueue, ReproductionInstruction{creature})
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

		if instruction.MoveAmount > 0 {
			halfFOVCos := math.Cos(float64(c.Genome.FieldOfView) / 2.0 * math.Pi / 180.0)

			bite := c.BiteSize(params)
			stomachSpace := c.StomachCapacity(params) - c.Stomach
			massRatio := float64(c.Mass / float32(params.MaxMass))
			if massRatio > 1.0 {
				massRatio = 1.0
			}
			interactionRadius := params.FoodInteractionRadius * (1.0 + massRatio)

			// Consume the nearest food item within interaction radius.
			if stomachSpace > 0 {
				foodIDs := w.GetFoodInCone(newPos, c.Heading, halfFOVCos, interactionRadius)
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
					eaten := bite
					if eaten > params.FoodMass {
						eaten = params.FoodMass
					}
					if eaten > stomachSpace {
						eaten = stomachSpace
					}
					c.Stomach += eaten
					stomachSpace -= eaten
					w.RemoveFood(closestID)
				}
			}

			// Eat the nearest corpse and/or the nearest live creature within interaction radius.
			creatureIDs := w.GetCreaturesInCone(newPos, c.Heading, halfFOVCos, interactionRadius)
			if len(creatureIDs) > 0 {
				closestCorpseID := -1
				closestPreyID := -1
				closestCorpseDist := math.MaxFloat64
				closestPreyDist := math.MaxFloat64
				for _, cid := range creatureIDs {
					if cid == c.Id {
						continue
					}
					cpos, ok := w.GetCreaturePos(cid)
					if !ok {
						continue
					}
					dx := cpos.X - newPos.X
					dy := cpos.Y - newPos.Y
					d := math.Sqrt(dx*dx + dy*dy)
					cr, ok := p.Creatures[cid]
					if !ok {
						continue
					}
					if cr.Alive {
						if d < closestPreyDist {
							closestPreyDist = d
							closestPreyID = cid
						}
					} else {
						if d < closestCorpseDist {
							closestCorpseDist = d
							closestCorpseID = cid
						}
					}
				}

				if closestCorpseID != -1 && stomachSpace > 0 {
					if target, ok := p.Creatures[closestCorpseID]; ok {
						eaten := bite
						if eaten > target.Mass {
							eaten = target.Mass
						}
						if eaten > stomachSpace {
							eaten = stomachSpace
						}
						c.Stomach += eaten
						stomachSpace -= eaten
						target.Mass -= eaten
						if target.Mass <= 0 {
							w.RemoveCreature(closestCorpseID)
							delete(p.Creatures, closestCorpseID)
						}
					}
				}

				if closestPreyID != -1 && stomachSpace > 0 {
					if target, ok := p.Creatures[closestPreyID]; ok {
						// Attacker must be at least MinPredationMassRatio of prey mass.
						if c.Mass >= target.Mass*params.MinPredationMassRatio {
							// Damage scales with attacker/prey mass ratio: larger attacker = full bite, smaller = reduced.
							massRatio := utils.MinFloat32(1.0, c.Mass/target.Mass)
							effectiveBite := bite * massRatio
							eaten := effectiveBite
							if eaten > target.Mass {
								eaten = target.Mass
							}
							if eaten > stomachSpace {
								eaten = stomachSpace
							}
							c.Stomach += eaten
							target.Mass -= eaten
							c.DrainEnergy(params.AttackEnergyCost)
							if target.Mass <= 0 {
								target.Alive = false
								target.Energy = 0
							}
						}
					}
				}
			}
		}

		w.MoveCreature(c.Id, newPos)
		c.Loc = newPos
	}
	p.MoveQueue = []MoveInstruction{}
}

// ProcessDeathQueue marks queued creatures as dead. Corpses remain in the world
// and decay over time, preserving their mass as a food source.
func (p *Population) ProcessDeathQueue(w *grid.World, params *Parameters) {
	for _, di := range p.DeathQueue {
		di.Creature.Alive = false
		di.Creature.Mass = di.Creature.CurrentMass(params)
		di.Creature.Energy = 0
	}
	p.DeathQueue = []DeathInstruction{}
}

// ProcessCorpseDecay drains mass from every dead creature. Fully decayed
// corpses are removed from both the world and the population map.
func (p *Population) ProcessCorpseDecay(w *grid.World, params *Parameters) {
	for id, c := range p.Creatures {
		if c.Alive {
			continue
		}
		c.Mass -= params.CorpseDecayRate
		if c.Mass <= 0 {
			w.RemoveCreature(id)
			delete(p.Creatures, id)
		}
	}
}

// ProcessReproductionQueue spawns offspring near queued parents.
// Requires the parent to be at full mass and above the energy threshold.
// Energy cost scales with offspring mass (tissue = stored energy).
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

		if parent.Energy < params.ReproductionEnergyThreshold*parent.MaxEnergy(params) {
			continue
		}

		if parent.Mass < float32(parent.Genome.Mass) {
			continue
		}

		if float32(parent.Genome.MinMass)*2 >= float32(parent.Genome.Mass) {
			continue
		}

		offspringLoc, ok := findOffspringLocation(w, parent)
		if !ok {
			continue
		}

		halfMass := parent.Mass / 2
		// Energy cost = mass being given to offspring × caloric value × reproduction efficiency.
		cost := halfMass * params.EnergyPerMassUnit * params.ReproductionEfficiency
		parent.DrainEnergy(cost)
		parent.GainDopamine(cost / utils.MaxFloat32(parent.MaxEnergy(params), 1))

		parent.Mass = halfMass

		childGenome := AsexualReproduction(parent.Genome, params)
		id := nextID()
		child := NewCreature(id, offspringLoc, childGenome, params)
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
