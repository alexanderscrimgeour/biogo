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
	Creatures         []*Creature // indexed by creature ID; nil = slot empty
	aliveIDs          []int       // incrementally maintained; avoids full-slice scan each step
	DeathQueue        []DeathInstruction
	MoveQueue         []MoveInstruction
	AttackQueue       []AttackInstruction
	ReproductionQueue []ReproductionInstruction
	FeedQueue         []FeedInstruction
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
	MoveAmount float32
}

type AttackInstruction struct {
	Creature *Creature
	Level    float64
}

type FeedInstruction struct {
	Creature  *Creature // donor
	Recipient *Creature
	Level     float32 // raw action level; proportion = tanh(level)
}

// pendingInstructions accumulates instructions produced by a single goroutine's
// creature batch before they are merged into the shared Population queues.
type pendingInstructions struct {
	death        []DeathInstruction
	move         []MoveInstruction
	attack       []AttackInstruction
	reproduction []ReproductionInstruction
	mate         []*Creature // creatures firing the MATE action this tick
	feed         []FeedInstruction
}

func NewPopulation(p *Parameters) *Population {
	return &Population{
		Creatures:         make([]*Creature, 0, p.Population.Initial+1),
		aliveIDs:          make([]int, 0, p.Population.Initial),
		DeathQueue:        []DeathInstruction{},
		MoveQueue:         []MoveInstruction{},
		AttackQueue:       []AttackInstruction{},
		ReproductionQueue: []ReproductionInstruction{},
		FeedQueue:         []FeedInstruction{},
	}
}

// Get returns the creature with the given ID, or (nil, false) if the slot is empty.
func (p *Population) Get(id int) (*Creature, bool) {
	if id >= 0 && id < len(p.Creatures) {
		if c := p.Creatures[id]; c != nil {
			return c, true
		}
	}
	return nil, false
}

// SetCreature stores a creature at its ID, growing the slice if needed.
func (p *Population) SetCreature(id int, c *Creature) {
	if id >= len(p.Creatures) {
		if id < cap(p.Creatures) {
			// Extend length within existing capacity: O(1) reslice, no allocation.
			p.Creatures = p.Creatures[:id+1]
		} else {
			newCap := cap(p.Creatures) * 2
			if newCap <= id {
				newCap = id + 1
			}
			grown := make([]*Creature, id+1, newCap)
			copy(grown, p.Creatures)
			p.Creatures = grown
		}
	}
	p.Creatures[id] = c
}

func (p *Population) QueueForMove(creature *Creature, newLoc world.Position, moveAmount float32) {
	p.MoveQueue = append(p.MoveQueue, MoveInstruction{creature, newLoc, moveAmount})
}

func (p *Population) QueueForAttack(creature *Creature, level float64) {
	p.AttackQueue = append(p.AttackQueue, AttackInstruction{creature, level})
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

// ProcessMoveQueue moves each queued creature to its target position.
func (p *Population) ProcessMoveQueue(w *world.World) {
	for _, instruction := range p.MoveQueue {
		c := instruction.Creature
		if !c.Alive {
			continue
		}
		if instruction.MoveAmount > 0 {
			w.MoveCreature(c.Id, instruction.Loc)
			c.Loc = instruction.Loc
		}
	}
	p.MoveQueue = p.MoveQueue[:0]
}

// ProcessEating consumes the nearest food/meat within interaction radius for every
// alive creature. Called after ProcessMoveQueue so positions are up to date.
func (p *Population) ProcessEating(w *world.World, params *Parameters) {
	for _, id := range p.aliveIDs {
		c, ok := p.Get(id)
		if !ok || !c.Alive {
			continue
		}

		bite := c.BiteSize(params)
		foodIDs, meatIDs := w.GetFoodAndMeatInRadius(c.Loc, c.Radius, c.SightFoodBuffer, c.SightMeatBuffer)
		foodEff, meatEff := c.DigestionEfficiencies()

		stomachSpace := c.StomachCapacity(params) - c.Stomach
		if stomachSpace > 0 && len(foodIDs) > 0 && foodEff > 0 {
			closestID := foodIDs[0]
			var closestDistSq float32 = math.MaxFloat32
			for _, fid := range foodIDs {
				fpos := w.GetFoodPos(fid)
				dx := fpos.X - c.Loc.X
				dy := fpos.Y - c.Loc.Y
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
			stomachGain := eaten * foodEff
			if stomachGain > stomachSpace {
				stomachGain = stomachSpace
				if foodEff > 0 {
					eaten = stomachSpace / foodEff
				}
			}
			c.Stomach += stomachGain
			w.ReduceFoodMass(closestID, eaten)
		}

		stomachSpace = c.StomachCapacity(params) - c.Stomach
		if stomachSpace > 0 && len(meatIDs) > 0 && meatEff > 0 {
			closestMeatID := meatIDs[0]
			var closestMeatDistSq float32 = math.MaxFloat32
			for _, mid := range meatIDs {
				mpos := w.GetFoodPos(mid)
				dx := mpos.X - c.Loc.X
				dy := mpos.Y - c.Loc.Y
				d2 := dx*dx + dy*dy
				if d2 < closestMeatDistSq {
					closestMeatDistSq = d2
					closestMeatID = mid
				}
			}
			meatMass := w.GetFoodMass(closestMeatID)
			eaten := bite
			if eaten > meatMass {
				eaten = meatMass
			}
			stomachGain := eaten * meatEff
			if stomachGain > stomachSpace {
				stomachGain = stomachSpace
				if meatEff > 0 {
					eaten = stomachSpace / meatEff
				}
			}
			c.Stomach += stomachGain
			w.ReduceFoodMass(closestMeatID, eaten)
		}
	}
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

		bite := c.BiteSize(params) * float32(instruction.Level)
		creatureIDs := w.GetCreaturesInCone(c.Loc, c.Heading, c.halfFOVCos, c.VisionRadius, c.SightCreatureBuffer)

		closestPreyID := -1
		var closestPreyDistSq float32 = math.MaxFloat32
		for _, cid := range creatureIDs {
			if cid == c.Id {
				continue
			}
			cr, ok := p.Get(cid)
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

		target, ok := p.Get(closestPreyID)
		if !ok {
			continue
		}

		sizeRatio := c.Mass / target.Mass

		defenseFactor := float32(0.5) + 0.5*float32(math.Min(1.0, float64(sizeRatio)))
		effectiveBite := bite * defenseFactor

		_, meatEff := c.DigestionEfficiencies()

		eaten := effectiveBite
		if eaten > target.Mass {
			eaten = target.Mass
		}

		stomachSpace = c.StomachCapacity(params) - c.Stomach
		stomachGain := eaten * meatEff
		if stomachGain > stomachSpace {
			stomachGain = stomachSpace
			if meatEff > 0 {
				eaten = stomachSpace / meatEff
			} else {
				eaten = effectiveBite
				if eaten > target.Mass {
					eaten = target.Mass
				}
			}
		}

		// waste includes compensation for the energy drained from target so TotalEnergy is conserved.
		waste := 2*eaten - stomachGain
		c.Stomach += stomachGain
		target.Mass -= eaten
		target.UpdateSize(params)
		target.DrainEnergy(eaten * params.Metabolism.EnergyPerMassUnit)
		if waste > 0.01 {
			w.AddMeat(target.Loc, waste)
		}
		struggleCost := params.Predation.AttackEnergyCost
		if sizeRatio < 1.0 {
			// Gradually increases cost as target gets larger, maxing at 1.5x
			struggleCost *= 1.0 + (1.0-sizeRatio)*0.5
		}
		c.DrainEnergy(struggleCost)

		if target.Mass <= 0.01 {
			target.Alive = false
			target.Energy = 0
			p.removeAlive(closestPreyID)
			w.RemoveCreature(closestPreyID)
			p.Creatures[closestPreyID] = nil
		}
	}
	p.AttackQueue = p.AttackQueue[:0]
}

// ProcessFeedQueue transfers stomach content from each donor to its recipient.
// The proportion donated is tanh(level), capped at the recipient's free stomach space.
func (p *Population) ProcessFeedQueue(params *Parameters) {
	for _, fi := range p.FeedQueue {
		donor := fi.Creature
		recipient := fi.Recipient

		absLevel := math.Abs(float64(fi.Level))
		if absLevel < 0.5 {
			continue
		}
		// Re-map: 0.5 -> 0.0 and 1.0 -> 1.0
		proportion := float32((absLevel - 0.5) / 0.5)
		if proportion > 1.0 {
			proportion = 1.0
		}

		if fi.Level < 0 {
			donor, recipient = recipient, donor
		}
		if !donor.Alive || !recipient.Alive {
			continue
		}
		amount := donor.Stomach * proportion
		if amount <= 0 {
			continue
		}

		space := recipient.StomachCapacity(params) - recipient.Stomach
		if space <= 0 {
			continue
		}
		if amount > space {
			amount = space
		}

		donor.Stomach -= amount
		recipient.Stomach += amount
	}
	p.FeedQueue = p.FeedQueue[:0]
}

// ProcessDeathQueue marks queued creatures as dead, spawns meat matching their
// mass at the death location, then removes them from the world and population map.
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
				di.Creature.Energy = 0
			}
		}(p.DeathQueue[i:end])
	}
	wg.Wait()
	// Serial: spawn meat, remove from world and population map.
	for _, di := range p.DeathQueue {
		c := di.Creature
		p.removeAlive(c.Id)
		spawnMeatFromCreature(w, c, params)
		w.RemoveCreature(c.Id)
		p.Creatures[c.Id] = nil
	}
	p.DeathQueue = p.DeathQueue[:0]
}

// spawnMeatFromCreature places meat items at the creature's death location
// totalling the creature's body mass, split into FoodMass-sized chunks.
func spawnMeatFromCreature(w *world.World, c *Creature, params *Parameters) {
	remaining := c.Mass
	chunkMass := params.Food.BaseMass
	for remaining > 0 {
		m := chunkMass
		if m > remaining {
			m = remaining
		}
		pos := w.FindEmptyLocationNear(c.Loc, 5.0)
		w.AddMeat(pos, m)
		remaining -= m
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
		if aliveCount >= params.Population.Max {
			break
		}
		parent := ri.Creature
		if !parent.Alive {
			continue
		}
		if parent.Energy < params.Reproduction.EnergyThreshold*parent.MaxEnergy(params) {
			continue
		}
		if parent.Mass < parent.MaxMass*0.9 {
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
			if partner.Energy < params.Reproduction.EnergyThreshold*partner.MaxEnergy(params) {
				continue
			}
			if partner.Mass < partner.MaxMass*0.9 {
				continue
			}

			offspringLoc, ok := findOffspringLocation(w, parent)
			if !ok {
				continue
			}

			// 1. Find the baseline mid-point of the parental lineages
			baseGen := (parent.Generation + partner.Generation) * 0.5
			bonusA := parent.CalculateGenerationBonus(params)
			bonusB := partner.CalculateGenerationBonus(params)
			childGen := baseGen + (bonusA+bonusB)*0.5

			radMult := radiationMult(parent.Loc.X, params)
			childGenome := Crossover(parent.Genome, partner.Genome, params, radMult, childGen)
			childStartingMass := float32(childGenome.Mass)

			ratioA := 0.1 + (float32(parent.Genome.MassSplitRatio)/255.0)*0.4
			ratioB := 0.1 + (float32(partner.Genome.MassSplitRatio)/255.0)*0.4
			totalRatio := ratioA + ratioB
			shareA := ratioA / totalRatio
			shareB := ratioB / totalRatio

			massFromParent := childStartingMass * shareA
			massFromPartner := childStartingMass * shareB

			energyFromParent := parent.Energy * ratioA
			energyFromPartner := partner.Energy * ratioB
			energyToSplit := energyFromParent + energyFromPartner
			energyTransferred := energyToSplit * params.Reproduction.Efficiency

			parent.Mass -= massFromParent
			parent.DrainEnergy(energyFromParent)
			parent.UpdateSize(params)

			partner.Mass -= massFromPartner
			partner.DrainEnergy(energyFromPartner)
			partner.UpdateSize(params)

			id := w.AddCreature(offspringLoc)
			child := NewCreature(id, offspringLoc, childGenome, params)
			child.Generation = childGen
			child.Tier = GetTierFromGeneration(childGen, params)
			child.GainEnergy(energyTransferred, params)
			p.SetCreature(id, child)
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
		energyTransferred := energyToSplit * params.Reproduction.Efficiency
		massRatio := parent.Mass / parent.MaxMass
		if massRatio > 1.0 {
			massRatio = 1.0
		}

		childGen := parent.Generation + parent.CalculateGenerationBonus(params)

		parent.Mass -= childMass
		parent.UpdateSize(params)
		parent.DrainEnergy(energyToSplit)
		radMult := radiationMult(parent.Loc.X, params)
		childGenome := AsexualReproduction(parent.Genome, params, radMult, childGen)
		id := w.AddCreature(offspringLoc)
		child := NewCreature(id, offspringLoc, childGenome, params)
		child.Generation = childGen
		child.Tier = GetTierFromGeneration(childGen, params)
		child.Mass = childMass
		child.UpdateSize(params)
		child.Energy = energyTransferred
		p.SetCreature(id, child)
		p.AddAlive(id)
		aliveCount++
	}
	p.ReproductionQueue = p.ReproductionQueue[:0]
}

// radiationMult returns the mutation multiplier for a creature at world x-coordinate x.
// Returns RadiationMutationMultiplier when inside the radiation zone, 1.0 otherwise.
func radiationMult(x float32, params *Parameters) float32 {
	if float64(x) < params.Environment.Radiation.ZoneWidth*params.World.Width {
		return params.Environment.Radiation.MutationMultiplier
	}
	return 1.0
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// findOffspringLocation returns a free position for an offspring, preferring a
// spot 5 units behind the parent and falling back to random nearby positions.
func findOffspringLocation(w *world.World, parent *Creature) (world.Position, bool) {
	backX := float32(-math.Cos(float64(parent.Heading)) * 5.0)
	backY := float32(-math.Sin(float64(parent.Heading)) * 5.0)
	behind := world.Position{X: parent.Loc.X + backX, Y: parent.Loc.Y + backY}
	if w.IsInBounds(behind) && !w.IsWall(behind) {
		return behind, true
	}
	for i := 0; i < 20; i++ {
		angle := rand.Float64() * 2 * math.Pi
		dist := rand.Float64()*8.0 + 2.0
		pos := world.Position{
			X: parent.Loc.X + float32(math.Cos(angle)*dist),
			Y: parent.Loc.Y + float32(math.Sin(angle)*dist),
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
		c, ok := p.Get(id)
		if !ok {
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
		c1, ok1 := p.Get(p.aliveIDs[i1])
		c2, ok2 := p.Get(p.aliveIDs[i2])
		if !ok1 || !ok2 {
			continue
		}
		total += 1 - GenomeSimilarity(c1.Genome, c2.Genome)
	}
	return total / float32(sampleSize)
}
