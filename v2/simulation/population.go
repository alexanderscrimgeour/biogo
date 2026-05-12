package simulation

import (
	"biogo/v2/utils"
	"biogo/v2/world"
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
	AttackQueue       []AttackInstruction
	ReproductionQueue []ReproductionInstruction
}

type DeathInstruction struct {
	Creature *Creature
}

type ReproductionInstruction struct {
	Creature *Creature
	Partner  *Creature // nil = asexual; non-nil = sexual (crossover)
}

type MoveInstruction struct {
	Creature   *Creature
	Loc        world.Position
	MoveAmount float64
}

type AttackInstruction struct {
	Creature *Creature
}

// pendingInstructions accumulates instructions produced by a single goroutine's
// creature batch before they are merged into the shared Population queues.
type pendingInstructions struct {
	death        []DeathInstruction
	move         []MoveInstruction
	attack       []AttackInstruction
	reproduction []ReproductionInstruction
	mate         []*Creature // creatures firing the MATE action this tick
}

func NewPopulation(p *Parameters) *Population {
	return &Population{
		Creatures:         make(map[int]*Creature, p.StartingPopulation),
		aliveIDs:          make([]int, 0, p.StartingPopulation),
		DeathQueue:        []DeathInstruction{},
		MoveQueue:         []MoveInstruction{},
		AttackQueue:       []AttackInstruction{},
		ReproductionQueue: []ReproductionInstruction{},
	}
}

func (p *Population) QueueForMove(creature *Creature, newLoc world.Position, moveAmount float64) {
	p.MoveQueue = append(p.MoveQueue, MoveInstruction{creature, newLoc, moveAmount})
}

func (p *Population) QueueForAttack(creature *Creature) {
	p.AttackQueue = append(p.AttackQueue, AttackInstruction{creature})
}

func (p *Population) QueueForDeath(creature *Creature) {
	p.DeathQueue = append(p.DeathQueue, DeathInstruction{creature})
}

func (p *Population) QueueForReproduction(creature *Creature) {
	p.ReproductionQueue = append(p.ReproductionQueue, ReproductionInstruction{Creature: creature})
}

// AddAlive registers a newly spawned creature in the alive-ID index.
func (p *Population) AddAlive(id int) {
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
func (p *Population) ProcessMoveQueue(w *world.World, params *Parameters) {
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
				foodIDs := w.GetFoodInCone(newPos, c.Heading, halfFOVCos, interactionRadius, c.SightFoodBuffer)
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

			// Eat the nearest corpse within interaction radius.
			creatureIDs := w.GetCreaturesInCone(newPos, c.Heading, halfFOVCos, interactionRadius, c.SightCreatureBuffer)
			if len(creatureIDs) > 0 {
				closestCorpseID := -1
				closestCorpseDistSq := math.MaxFloat64
				for _, cid := range creatureIDs {
					if cid == c.Id {
						continue
					}
					cr, ok := p.Creatures[cid]
					if !ok || cr.Alive {
						continue
					}
					cpos, ok := w.GetCreaturePos(cid)
					if !ok {
						continue
					}
					dx := cpos.X - newPos.X
					dy := cpos.Y - newPos.Y
					d2 := dx*dx + dy*dy
					if d2 < closestCorpseDistSq {
						closestCorpseDistSq = d2
						closestCorpseID = cid
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
			}
		}

		w.MoveCreature(c.Id, newPos)
		c.Loc = newPos
	}
	p.MoveQueue = p.MoveQueue[:0]
}

// ProcessAttackQueue resolves ATTACK actions: each attacker bites the nearest live
// creature within its FOV cone. Damage scales with the attacker/prey mass ratio.
// Unlike passive predation, there is no minimum mass requirement — smaller creatures
// can attack larger ones but deal proportionally less damage.
func (p *Population) ProcessAttackQueue(w *world.World, params *Parameters) {
	for _, instruction := range p.AttackQueue {
		c := instruction.Creature
		if !c.Alive {
			continue
		}

		stomachSpace := c.StomachCapacity(params) - c.Stomach
		if stomachSpace <= 0 {
			continue
		}

		bite := c.BiteSize(params)
		massRatio := float64(c.Mass / float32(params.MaxMass))
		if massRatio > 1.0 {
			massRatio = 1.0
		}
		interactionRadius := params.FoodInteractionRadius * (1.0 + massRatio)

		creatureIDs := w.GetCreaturesInCone(c.Loc, c.Heading, c.halfFOVCos, interactionRadius, c.SightCreatureBuffer)

		closestPreyID := -1
		closestPreyDistSq := math.MaxFloat64
		for _, cid := range creatureIDs {
			if cid == c.Id {
				continue
			}
			cr, ok := p.Creatures[cid]
			if !ok || !cr.Alive {
				continue
			}
			cpos, ok := w.GetCreaturePos(cid)
			if !ok {
				continue
			}
			dx := cpos.X - c.Loc.X
			dy := cpos.Y - c.Loc.Y
			d2 := dx*dx + dy*dy
			if d2 < closestPreyDistSq {
				closestPreyDistSq = d2
				closestPreyID = cid
			}
		}

		if closestPreyID == -1 {
			continue
		}

		target, ok := p.Creatures[closestPreyID]
		if !ok {
			continue
		}

		sizeRatio := c.Mass / target.Mass

		defenseFactor := 0.5 + (0.5 * utils.MinFloat32(1.0, sizeRatio))
		effectiveBite := bite * defenseFactor

		eaten := effectiveBite
		if eaten > target.Mass {
			eaten = target.Mass
		}
		if eaten > stomachSpace {
			eaten = stomachSpace
		}

		c.Stomach += eaten
		target.Mass -= eaten

		target.DrainEnergy(eaten * params.EnergyPerMassUnit)
		struggleCost := params.AttackEnergyCost
		if sizeRatio < 1.0 {
			// Gradually increases cost as target gets larger, maxing at 1.5x
			struggleCost *= (1.0 + (1.0-sizeRatio)*0.5)
		}
		c.DrainEnergy(struggleCost)

		if target.Mass <= 0.01 {
			target.Alive = false
			target.Energy = 0
			p.removeAlive(closestPreyID)
		}
	}
	p.AttackQueue = p.AttackQueue[:0]
}

// ProcessDeathQueue marks queued creatures as dead. Corpses remain in the world
// and decay over time, preserving their mass as a food source.
func (p *Population) ProcessDeathQueue(w *world.World, params *Parameters) {
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
func (p *Population) ProcessCorpseDecay(w *world.World, params *Parameters) {
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
func (p *Population) ProcessReproductionQueue(w *world.World, params *Parameters) {
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
		if parent.Mass < float32(parent.Genome.Mass)*0.9 {
			continue
		}
		if float32(parent.Genome.MinMass)*2 >= float32(parent.Genome.Mass) {
			continue
		}

		// Sexual: partner must still be alive and eligible.
		if ri.Partner != nil {
			partner := ri.Partner
			if !partner.Alive {
				continue
			}
			if partner.Energy < params.ReproductionEnergyThreshold*partner.MaxEnergy(params) {
				continue
			}
			if partner.Mass < float32(partner.Genome.Mass)*0.9 {
				continue
			}

			offspringLoc, ok := findOffspringLocation(w, parent)
			if !ok {
				continue
			}
			ratioA := 0.1 + (float32(parent.Genome.MassSplitRatio)/255.0)*0.4
			ratioB := 0.1 + (float32(partner.Genome.MassSplitRatio)/255.0)*0.4

			massFromParent := parent.Mass * ratioA
			massFromPartner := partner.Mass * ratioB
			childMass := massFromParent + massFromPartner
			energyFromParent := parent.Energy * ratioA
			energyFromPartner := partner.Energy * ratioB
			energyToSplit := energyFromParent + energyFromPartner
			energyTransferred := energyToSplit * params.ReproductionEfficiency

			parent.Mass -= massFromParent
			parent.DrainEnergy(energyFromParent)

			partner.Mass -= massFromPartner
			partner.DrainEnergy(energyFromPartner)

			radMult := radiationMult(parent.Loc.X, params)
			childGenome := Crossover(parent.Genome, partner.Genome, params, radMult)
			id := w.AddCreature(offspringLoc)
			child := NewCreature(id, offspringLoc, childGenome, params)
			child.Mass = childMass
			child.Energy = energyTransferred
			p.Creatures[id] = child
			p.AddAlive(id)
			aliveCount++
			continue
		}

		// Asexual path.
		offspringLoc, ok := findOffspringLocation(w, parent)
		if !ok {
			continue
		}
		splitRatio := 0.1 + (float32(parent.Genome.MassSplitRatio)/255.0)*0.4
		childMass := parent.Mass * splitRatio
		energyToSplit := parent.Energy * splitRatio
		energyTransferred := energyToSplit * params.ReproductionEfficiency

		parent.Mass -= childMass
		parent.DrainEnergy(energyToSplit)

		radMult := radiationMult(parent.Loc.X, params)
		childGenome := AsexualReproduction(parent.Genome, params, radMult)
		id := w.AddCreature(offspringLoc)
		child := NewCreature(id, offspringLoc, childGenome, params)
		child.Mass = childMass
		child.Energy = energyTransferred
		p.Creatures[id] = child
		p.AddAlive(id)
		aliveCount++
	}
	p.ReproductionQueue = p.ReproductionQueue[:0]
}

// radiationMult returns the mutation multiplier for a creature at world x-coordinate x.
// Returns RadiationMutationMultiplier when inside the radiation zone, 1.0 otherwise.
func radiationMult(x float64, params *Parameters) float32 {
	if x < params.RadiationZoneWidth*params.WorldWidth {
		return params.RadiationMutationMultiplier
	}
	return 1.0
}

// findOffspringLocation returns a free position for an offspring, preferring a
// spot 5 units behind the parent and falling back to random nearby positions.
func findOffspringLocation(w *world.World, parent *Creature) (world.Position, bool) {
	backX := -math.Cos(parent.Heading) * 5.0
	backY := -math.Sin(parent.Heading) * 5.0
	behind := world.Position{X: parent.Loc.X + backX, Y: parent.Loc.Y + backY}
	if w.IsInBounds(behind) && !w.IsWall(behind) {
		return behind, true
	}
	for i := 0; i < 20; i++ {
		angle := rand.Float64() * 2 * math.Pi
		dist := rand.Float64()*8.0 + 2.0
		pos := world.Position{
			X: parent.Loc.X + math.Cos(angle)*dist,
			Y: parent.Loc.Y + math.Sin(angle)*dist,
		}
		if w.IsInBounds(pos) && !w.IsWall(pos) {
			return pos, true
		}
	}
	return world.Position{}, false
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
		total += 1 - GenomeSimilarity(c1.Genome, c2.Genome)
	}
	return total / float32(sampleSize)
}
