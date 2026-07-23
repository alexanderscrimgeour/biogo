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
	Creature   *Creature
	Partner    *Creature // nil = asexual; non-nil = sexual (crossover)
	IsFallback bool      // If true, creature has reproduced via fallback mechanism
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

type eatInstruction struct {
	creature *Creature
	foodID   int
	maxBite  float32
	eff      float32
	foodType uint8
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

// ProcessEatingParallel gathers eat decisions across batches in parallel (read-only
// spatial queries) then applies them serially (world mutations). Batches must be
// the same partition used for the creature step so each creature appears in exactly
// one batch and buffer reuse is safe.
func (p *Population) ProcessEatingParallel(w *world.World, params *Parameters, batches [][]int) {
	results := make([][]eatInstruction, len(batches))
	var wg sync.WaitGroup
	for i, batch := range batches {
		wg.Add(1)
		go func(idx int, b []int) {
			defer wg.Done()
			results[idx] = gatherEatBatch(b, p, w, params, results[idx])
		}(i, batch)
	}
	wg.Wait()
	for _, instrs := range results {
		applyEatInstructions(w, params, instrs)
	}
}

// gatherEatBatch runs in a goroutine. It performs only read-only world queries to
// find the nearest food item of each type for each creature, recording the desired
// bite without modifying any shared state.
func gatherEatBatch(ids []int, p *Population, w *world.World, params *Parameters, out []eatInstruction) []eatInstruction {
	for _, id := range ids {
		c, ok := p.Get(id)
		if !ok || !c.Alive {
			continue
		}
		if c.StomachCapacity(params)-c.Stomach <= 0 {
			continue
		}
		bite := c.BiteSize(params)
		foodIDs, meatIDs, fungiIDs := w.GetFoodAndMeatInRadius(c.Loc, c.Radius, c.SightFoliageBuffer, c.SightMeatBuffer, c.SightFungiBuffer)

		appendNearest := func(fids []int, foodType uint8) {
			eff := c.GetFoodEfficiency(foodType)
			if len(fids) == 0 || eff <= 0 {
				return
			}
			closestID := fids[0]
			var closestDistSq float32 = math.MaxFloat32
			for _, fid := range fids {
				fpos := w.GetFoodPos(fid)
				dx := fpos.X - c.Loc.X
				dy := fpos.Y - c.Loc.Y
				d2 := dx*dx + dy*dy
				if d2 < closestDistSq {
					closestDistSq = d2
					closestID = fid
				}
			}
			out = append(out, eatInstruction{c, closestID, bite, eff, foodType})
		}

		appendNearest(foodIDs, world.FoodTypeFoliage)
		appendNearest(fungiIDs, world.FoodTypeFungi)
		appendNearest(meatIDs, world.FoodTypeMeat)
	}
	return out
}

// applyEatInstructions runs serially. It re-checks stomach space and actual food
// mass at apply time so that two creatures targeting the same item each get only
// what remains.
func applyEatInstructions(w *world.World, params *Parameters, instrs []eatInstruction) {
	for _, inst := range instrs {
		c := inst.creature
		if !c.Alive {
			continue
		}
		stomachSpace := c.StomachCapacity(params) - c.Stomach
		if stomachSpace <= 0 {
			continue
		}
		foodMass := w.GetFoodMass(inst.foodID)
		if foodMass <= 0 {
			continue
		}
		eaten := inst.maxBite
		if eaten > foodMass {
			eaten = foodMass
		}
		var densityRatio float32
		switch inst.foodType {
		case world.FoodTypeFoliage:
			densityRatio = params.Food.FoliageEnergyDensity / params.Metabolism.EnergyPerFoodMass
		case world.FoodTypeFungi:
			densityRatio = params.Food.FungiEnergyDensity / params.Metabolism.EnergyPerFoodMass
		case world.FoodTypeMeat:
			densityRatio = params.Food.MeatEnergyDensity / params.Metabolism.EnergyPerFoodMass
		default:
			densityRatio = 1.0
		}
		stomachGain := eaten * inst.eff * densityRatio
		if stomachGain > stomachSpace {
			stomachGain = stomachSpace
			if inst.eff*densityRatio > 0 {
				eaten = stomachSpace / (inst.eff * densityRatio)
			}
		}
		c.Stomach += stomachGain
		c.GainDopamine(0.01)
		w.ReduceFoodMass(inst.foodID, eaten)
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
		creatureIDs := w.GetCreaturesInCone(c.Loc, c.Heading, c.halfFOVCos, c.Radius+params.Creature.AttackRadius, c.SightCreatureBuffer)

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

		damage := effectiveBite
		if damage > target.Mass {
			damage = target.Mass
		}

		meatEff := c.GetFoodEfficiency(world.FoodTypeMeat)
		meatDensityRatio := params.Food.MeatEnergyDensity / params.Metabolism.EnergyPerFoodMass

		var eaten float32

		if meatEff > 0 && (meatEff*meatDensityRatio) > 0 {
			maxDigestible := stomachSpace / (meatEff * meatDensityRatio)
			eaten = damage
			if eaten > maxDigestible {
				eaten = maxDigestible
			}
		}

		stomachGain := eaten * meatEff * meatDensityRatio
		c.Stomach += stomachGain

		energyToDrain := damage * params.Metabolism.EnergyPerFoodMass
		if energyToDrain > target.Energy {
			energyToDrain = target.Energy
		}

		target.Mass -= damage
		target.UpdateRadius(params)
		target.DrainEnergy(energyToDrain)

		droppedMeatMass := damage - eaten
		if droppedMeatMass > 0.01 {
			w.AddMeat(target.Loc, droppedMeatMass)
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
	chunkMass := params.Food.MeatMass
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
		offspringLoc, ok := findOffspringLocation(w, parent)
		if !ok {
			continue
		}

		// Sexual: partner must still be alive and eligible.
		if ri.Partner != nil {
			if sexualReproduction(w, params, p, ri, offspringLoc) {
				aliveCount++
			}
			continue
		}
		// Asexual reproduction
		if asexualReproduction(w, params, p, ri, offspringLoc) {
			aliveCount++
		}
	}
	// Clear queue
	p.ReproductionQueue = p.ReproductionQueue[:0]
}

func sexualReproduction(
	world *world.World,
	params *Parameters,
	population *Population,
	ri ReproductionInstruction,
	offspringLoc world.Position,
) bool {
	if !ri.Partner.Alive {
		return false
	}
	if ri.Partner.Energy < params.Reproduction.EnergyThreshold*ri.Partner.MaxEnergy(params) {
		return false
	}

	// 1. Find the baseline mid-point of the parental lineages
	baseGen := (ri.Creature.Generation + ri.Partner.Generation) * 0.5
	bonusA := ri.Creature.CalculateGenerationBonus(params)
	bonusB := ri.Partner.CalculateGenerationBonus(params)
	childGen := baseGen + (bonusA+bonusB)*0.5

	mutMult := radiationMult(ri.Creature.Loc.X, params)
	// Using the fallback in tiers 1+ is punished
	if ri.IsFallback && ri.Creature.Tier >= 1 {
		mutMult *= 1.25
	}
	childGenome := Crossover(ri.Creature.Genome, ri.Partner.Genome, params, mutMult, childGen)

	ratioA := ri.Creature.Genome.CalculateSplitRatio()
	ratioB := ri.Partner.Genome.CalculateSplitRatio()

	massFromParent := ri.Creature.Mass * ratioA
	massFromPartner := ri.Partner.Mass * ratioB
	childStartingMass := massFromParent + massFromPartner

	if childStartingMass < float32(params.Creature.MinBirthMass) {
		return false
	}

	energyFromParent := ri.Creature.Energy * ratioA
	energyFromPartner := ri.Partner.Energy * ratioB
	energyToSplit := energyFromParent + energyFromPartner
	energyTransferred := energyToSplit * params.Reproduction.Efficiency

	// Ensure this investment doesn't physically drop either parent below their survival skeleton
	if (ri.Creature.Mass-massFromParent) < ri.Creature.SurvivalMass || (ri.Partner.Mass-massFromPartner) < ri.Partner.SurvivalMass {
		return false
	}
	ri.Creature.ApplyReproductionCost(massFromParent, energyFromParent, ri.IsFallback, params)
	ri.Partner.ApplyReproductionCost(massFromPartner, energyFromPartner, ri.IsFallback, params)
	ri.Creature.ReproductionCooldown = int(massFromParent * params.Reproduction.GestationTicksPerMass)
	ri.Partner.ReproductionCooldown = int(massFromPartner * params.Reproduction.GestationTicksPerMass)

	id := world.AddCreature(offspringLoc)
	child := NewCreature(id, offspringLoc, childGenome, childStartingMass, params)
	child.Generation = childGen
	child.Tier = GetTierFromGeneration(childGen, params)
	child.GainEnergy(energyTransferred, params)
	population.SetCreature(id, child)
	population.AddAlive(id)
	return true
}

func asexualReproduction(
	world *world.World,
	params *Parameters,
	population *Population,
	ri ReproductionInstruction,
	offspringLoc world.Position,
) bool {
	childGen := ri.Creature.Generation + ri.Creature.CalculateGenerationBonus(params)
	mutMult := radiationMult(ri.Creature.Loc.X, params)
	// Using the fallback in tiers 1+ is punished
	if ri.IsFallback && ri.Creature.Tier >= 1 {
		mutMult *= 1.25
	}
	childGenome := AsexualReproduction(ri.Creature.Genome, params, mutMult, childGen)
	splitRatio := ri.Creature.Genome.CalculateSplitRatio()
	childStartingMass := ri.Creature.Mass * splitRatio
	if childStartingMass < float32(params.Creature.MinBirthMass) {
		return false
	}
	if (ri.Creature.Mass - childStartingMass) < ri.Creature.SurvivalMass {
		return false
	}
	energyToSplit := ri.Creature.Energy * splitRatio
	energyTransferred := energyToSplit * params.Reproduction.Efficiency
	ri.Creature.ApplyReproductionCost(childStartingMass, energyToSplit, ri.IsFallback, params)
	ri.Creature.ReproductionCooldown = int(childStartingMass * params.Reproduction.GestationTicksPerMass)

	id := world.AddCreature(offspringLoc)
	child := NewCreature(id, offspringLoc, childGenome, childStartingMass, params)
	child.Generation = childGen
	child.Tier = GetTierFromGeneration(childGen, params)
	child.Energy = energyTransferred
	child.UpdateRadius(params)
	population.SetCreature(id, child)
	population.AddAlive(id)
	return true
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
