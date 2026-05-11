package simulation

import (
	"biogo/v2/grid"
	"biogo/v2/utils"
	"math"
	"math/rand"
	"runtime"
	"sync"
)

type Population struct {
	Creatures         map[int]*Creature
	aliveIDs          []int // incrementally maintained; avoids full-map scan each step
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
		aliveIDs:          make([]int, 0, p.StartingPopulation),
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

// addAlive registers a newly spawned creature in the alive-ID index.
func (p *Population) addAlive(id int) {
	p.aliveIDs = append(p.aliveIDs, id)
}

// removeAlive removes id from the alive-ID index via swap-and-truncate.
func (p *Population) removeAlive(id int) {
	for i, v := range p.aliveIDs {
		if v == id {
			p.aliveIDs[i] = p.aliveIDs[len(p.aliveIDs)-1]
			p.aliveIDs = p.aliveIDs[:len(p.aliveIDs)-1]
			return
		}
	}
}

// AliveIDs returns the slice of currently alive creature IDs.
// The returned slice is the live backing store — callers must not modify it.
func (p *Population) AliveIDs() []int {
	return p.aliveIDs
}

func (p *Population) AliveCount() int {
	return len(p.aliveIDs)
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
			halfFOVCos := c.halfFOVCos

			bite := c.BiteSize(params)
			stomachSpace := c.StomachCapacity(params) - c.Stomach
			massRatio := float64(c.Mass / float32(params.MaxMass))
			if massRatio > 1.0 {
				massRatio = 1.0
			}
			interactionRadius := params.FoodInteractionRadius * (1.0 + massRatio)

			// Consume the nearest food item within interaction radius.
			// If the item has more mass than the bite or the remaining stomach space,
			// only the appropriate portion is taken and the item stays in the world
			// with its remaining mass.
			if stomachSpace > 0 {
				foodIDs := w.GetFoodInCone(newPos, c.Heading, halfFOVCos, interactionRadius)
				if len(foodIDs) > 0 {
					closestID := foodIDs[0]
					closestDistSq := math.MaxFloat64
					for _, fid := range foodIDs {
						fpos := w.GetFoodPos(fid)
						dx := fpos.X - newPos.X
						dy := fpos.Y - newPos.Y
						d2 := dx*dx + dy*dy
						if d2 < closestDistSq {
							closestDistSq = d2
							closestID = fid
						}
					}
					foodMass := w.GetFoodMass(closestID)
					eaten := bite
					if eaten > foodMass {
						eaten = foodMass
					}
					if eaten > stomachSpace {
						eaten = stomachSpace
					}
					c.Stomach += eaten
					stomachSpace -= eaten
					w.ReduceFoodMass(closestID, eaten)
				}
			}

			// Eat the nearest corpse and/or the nearest live creature within interaction radius.
			creatureIDs := w.GetCreaturesInCone(newPos, c.Heading, halfFOVCos, interactionRadius)
			if len(creatureIDs) > 0 {
				closestCorpseID := -1
				closestPreyID := -1
				closestCorpseDistSq := math.MaxFloat64
				closestPreyDistSq := math.MaxFloat64
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
					d2 := dx*dx + dy*dy
					cr, ok := p.Creatures[cid]
					if !ok {
						continue
					}
					if cr.Alive {
						if d2 < closestPreyDistSq {
							closestPreyDistSq = d2
							closestPreyID = cid
						}
					} else {
						if d2 < closestCorpseDistSq {
							closestCorpseDistSq = d2
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
							// Damage scales with attacker/prey mass ratio.
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
								p.removeAlive(closestPreyID)
							}
						}
					}
				}
			}
		}

		w.MoveCreature(c.Id, newPos)
		c.Loc = newPos
	}
	p.MoveQueue = p.MoveQueue[:0]
}

// ProcessDeathQueue marks queued creatures as dead. Corpses remain in the world
// and decay over time, preserving their mass as a food source.
func (p *Population) ProcessDeathQueue(w *grid.World, params *Parameters) {
	if len(p.DeathQueue) == 0 {
		return
	}
	n := runtime.GOMAXPROCS(0)
	batchSize := (len(p.DeathQueue) + n - 1) / n
	var wg sync.WaitGroup
	for i := 0; i < len(p.DeathQueue); i += batchSize {
		end := i + batchSize
		if end > len(p.DeathQueue) {
			end = len(p.DeathQueue)
		}
		wg.Add(1)
		go func(batch []DeathInstruction) {
			defer wg.Done()
			for _, di := range batch {
				di.Creature.Alive = false
				di.Creature.Mass = di.Creature.CurrentMass(params)
				di.Creature.Energy = 0
			}
		}(p.DeathQueue[i:end])
	}
	wg.Wait()
	// Serial: remove newly dead creatures from the alive-ID index.
	for _, di := range p.DeathQueue {
		p.removeAlive(di.Creature.Id)
	}
	p.DeathQueue = p.DeathQueue[:0]
}

// ProcessCorpseDecay drains mass from every dead creature. Fully decayed
// corpses are removed from both the world and the population map.
func (p *Population) ProcessCorpseDecay(w *grid.World, params *Parameters) {
	corpseIDs := make([]int, 0, len(p.Creatures))
	for id, c := range p.Creatures {
		if !c.Alive {
			corpseIDs = append(corpseIDs, id)
		}
	}
	if len(corpseIDs) == 0 {
		return
	}

	n := runtime.GOMAXPROCS(0)
	batches := partitionIDs(corpseIDs, n)
	var wg sync.WaitGroup
	for _, batch := range batches {
		if len(batch) == 0 {
			continue
		}
		wg.Add(1)
		go func(b []int) {
			defer wg.Done()
			for _, id := range b {
				p.Creatures[id].Mass -= params.CorpseDecayRate
			}
		}(batch)
	}
	wg.Wait()

	// Map writes must be serial — remove fully decayed corpses after all goroutines finish.
	for _, id := range corpseIDs {
		if p.Creatures[id].Mass <= 0 {
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
	aliveCount := p.AliveCount()
	for _, ri := range p.ReproductionQueue {
		if aliveCount >= params.MaxPopulation {
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
		energyTransferred := halfMass * params.ReproductionEfficiency
		metabolicWaste := energyTransferred * (1 - params.ReproductionEfficiency)
		parent.Mass = halfMass
		parent.DrainEnergy(energyTransferred + metabolicWaste)
		parent.GainDopamine(energyTransferred / utils.MaxFloat32(parent.MaxEnergy(params), 1))

		parent.Mass = halfMass

		childGenome := AsexualReproduction(parent.Genome, params)
		id := nextID()
		child := NewCreature(id, offspringLoc, childGenome, params)
		child.Mass = halfMass
		child.Energy = energyTransferred
		p.Creatures[id] = child
		p.addAlive(id)
		w.AddCreature(id, offspringLoc)
		aliveCount++
	}
	p.ReproductionQueue = p.ReproductionQueue[:0]
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
	for _, id := range p.aliveIDs {
		c := p.Creatures[id]
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
	n := len(p.aliveIDs)
	if n < 2 {
		return 0
	}
	sampleSize := utils.Min(200, n)
	total := float32(0)
	for i := 0; i < sampleSize; i++ {
		i1 := rand.Intn(n)
		i2 := rand.Intn(n)
		for i2 == i1 {
			i2 = rand.Intn(n)
		}
		c1 := p.Creatures[p.aliveIDs[i1]]
		c2 := p.Creatures[p.aliveIDs[i2]]
		total += 1 - GenomeSimilarity(*c1.Genome, *c2.Genome)
	}
	return total / float32(sampleSize)
}
