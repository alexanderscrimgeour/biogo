package simulation

import (
	"biogo/v2/world"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
)

// sensorUpdatePeriod controls how often each creature refreshes its spatial
// sensor context (GetAllInRadius). A value of 4 staggers updates across ticks
// by creature ID, reducing spatial queries by ~75% at the cost of sensor data
// being at most (sensorUpdatePeriod-1) ticks stale.
const sensorUpdatePeriod = 4

type Simulation struct {
	World        *world.World
	Population   *Population
	Tick         int
	Params       *Parameters
	Energy       float64 // total liquid energy to maintain (set at initialisation)
	displayCache []CreatureView
	foodCache    []FoodView // combined Foliages and meat
	cacheMu      sync.RWMutex
	cacheDirty   bool
}

func New(params *Parameters) *Simulation {
	InitResponseCurve(params)
	sim := &Simulation{
		Params: params,
	}
	sim.initialiseWorld()
	sim.initialisePopulation()
	sim.Energy = sim.TotalEnergy()
	return sim
}

func (s *Simulation) initialiseWorld() {
	s.World = world.NewWorld(s.Params.World.Width, s.Params.World.Height, 1)
	s.World.TempMin = s.Params.Environment.TempMin
	s.World.TempMax = s.Params.Environment.TempMax
	s.World.InitFountains(
		s.Params.Food.Foliage.Count, s.Params.Food.Fungi.Count, s.Params.Food.Meat.Count,
		s.Params.Food.Foliage.StationaryCount, s.Params.Food.Fungi.StationaryCount, s.Params.Food.Meat.StationaryCount,
	)
	s.spawnInitialFood()
}

func (s *Simulation) initialisePopulation() {
	pop := NewPopulation(s.Params)
	savedGenomes, _ := LoadAllCreatureGenomes()
	maxSeeded := int(float64(s.Params.Population.Initial) * s.Params.Spawn.SavedGenomeProportion)
	numSaved := len(savedGenomes)
	bubbleSize := 5
	spawnRadius := 15.0

	for i := 0; i < s.Params.Population.Initial; i += bubbleSize {
		centerLoc, ok := s.World.FindEmptyLocation()
		if !ok {
			break
		}

		var genome *Genome
		if numSaved > 0 && i < maxSeeded {
			genome = savedGenomes[(i/bubbleSize)%numSaved]
		} else {
			// Initialise genomes at tier zero
			genome = MakeRandomGenome(s.Params, 0)
		}

		for j := 0; j < bubbleSize; j++ {
			if i+j >= s.Params.Population.Initial {
				break
			}

			loc := s.World.FindEmptyLocationNear(centerLoc, spawnRadius)

			id := s.World.AddCreature(loc)
			c := NewAdultCreature(id, loc, genome, s.Params)
			pop.SetCreature(id, c)
			pop.AddAlive(id)
		}
	}
	s.Population = pop
}

// SaveCreature saves the genome and generation of the creature with the given id.
// name is used as the filename (empty falls back to timestamp-based naming).
func (s *Simulation) SaveCreature(id int, name string) error {
	c, ok := s.Population.Get(id)
	if !ok || !c.Alive {
		return nil
	}
	return SaveCreatureToFileNamed(c.Genome, c.Generation, name)
}

// Reset reinitialises the simulation from scratch.
func (s *Simulation) Reset() {
	s.Tick = 0
	s.initialiseWorld()
	s.initialisePopulation()
	s.Energy = s.TotalEnergy()
}

func (s *Simulation) Update() {
	s.step()
	s.cacheMu.Lock()
	s.cacheDirty = true
	s.cacheMu.Unlock()
}

func (s *Simulation) step() {
	s.World.StepFountains(s.Params.Food.Foliage.DriftSpeed, s.Params.Food.Fungi.DriftSpeed, s.Params.Food.Meat.DriftSpeed)
	if s.Tick%s.Params.Food.SpawnInterval == 0 {
		deficit := s.Energy - s.TotalEnergy()
		if deficit <= 0 {
			deficit = 0
		}

		s.spawnDeficit(deficit)
	}

	// AliveIDs() now returns the live backing slice — no allocation, O(1).
	ids := s.Population.AliveIDs()
	n := runtime.GOMAXPROCS(0)
	batches := partitionIDs(ids, n)

	// Pre-size shared queues so Process* functions can reuse the backing array.
	popSize := len(ids)
	if cap(s.Population.DeathQueue) < popSize {
		s.Population.DeathQueue = make([]DeathInstruction, 0, popSize)
	}
	if cap(s.Population.MoveQueue) < popSize {
		s.Population.MoveQueue = make([]MoveInstruction, 0, popSize)
	}
	if cap(s.Population.AttackQueue) < popSize {
		s.Population.AttackQueue = make([]AttackInstruction, 0, popSize)
	}
	if cap(s.Population.ReproductionQueue) < popSize {
		s.Population.ReproductionQueue = make([]ReproductionInstruction, 0, popSize)
	}
	if cap(s.Population.FeedQueue) < popSize {
		s.Population.FeedQueue = make([]FeedInstruction, 0, popSize)
	}

	// Each goroutine writes to its own pendingInstructions; no shared mutation.
	results := make([]pendingInstructions, n)
	var wg sync.WaitGroup
	for i, batch := range batches {
		wg.Add(1)
		go func(idx int, b []int) {
			defer wg.Done()
			for _, id := range b {
				s.stepCreatureLocal(s.Population.Creatures[id], &results[idx])
			}
		}(i, batch)
	}
	wg.Wait()

	// Merge per-goroutine command buffers into the shared queues.
	var wantMate []*Creature
	for i := range results {
		s.Population.DeathQueue = append(s.Population.DeathQueue, results[i].death...)
		s.Population.MoveQueue = append(s.Population.MoveQueue, results[i].move...)
		s.Population.AttackQueue = append(s.Population.AttackQueue, results[i].attack...)
		s.Population.ReproductionQueue = append(s.Population.ReproductionQueue, results[i].reproduction...)
		wantMate = append(wantMate, results[i].mate...)
		s.Population.FeedQueue = append(s.Population.FeedQueue, results[i].feed...)
	}
	s.pairMates(wantMate)

	s.Population.ProcessMoveQueue(s.World)
	s.Population.ProcessEatingParallel(s.World, s.Params, batches)
	s.processCollisions()
	s.Population.ProcessAttackQueue(s.World, s.Params)
	s.Population.ProcessFeedQueue(s.Params)
	s.Population.ProcessDeathQueue(s.World, s.Params)
	s.Population.ProcessReproductionQueue(s.World, s.Params)
	s.World.DecayMeat(s.Params.Food.MeatDecayRate)

	aliveCount := s.Population.AliveCount()
	const toSpawn = 5
	for aliveCount < s.Params.Population.Min {
		loc, ok := s.fountainLocation()
		if !ok {
			break
		}
		s.SpawnClusterAt(float64(loc.X), float64(loc.Y), toSpawn)
		aliveCount += toSpawn
	}

	if s.Params.Spawn.ClusterEnabled && s.Tick > 0 && s.Tick%s.Params.Spawn.ClusterInterval == 0 &&
		aliveCount < s.Params.Population.Max {
		loc, ok := s.fountainLocation()
		if ok {
			s.SpawnClusterAt(float64(loc.X), float64(loc.Y), s.Params.Spawn.ClusterSize)
		}
	}

	s.Tick++
}

func (s *Simulation) stepCreatureLocal(c *Creature, pending *pendingInstructions) {
	c.Dopamine *= 0.9
	if c.Dopamine > -0.01 && c.Dopamine < 0.01 {
		c.Dopamine = 0
	}
	c.Age++
	if c.ReproductionCooldown > 0 {
		c.ReproductionCooldown--
	}
	c.LastActionMask = 0
	temp := s.World.TemperatureAt(c.Loc.Y)
	c.DrainEnergy(c.MetabolicRate(s.Params, temp))
	if float64(c.Loc.X) < s.Params.Environment.Radiation.ZoneWidth*s.Params.World.Width {
		absoluteMassScale := float64(c.Mass) / float64(s.Params.Creature.MinSurvivalMass)
		// Kleiber's Law (M^0.75)
		massEffect := float32(math.Sqrt(absoluteMassScale * math.Sqrt(absoluteMassScale)))
		c.DrainEnergy(s.Params.Environment.Radiation.DamagePerTick * massEffect)
	}
	c.Digest(s.Params)
	c.GrowMass(s.Params, temp)

	// Starvation catabolism: when energy reserves run critically low, the body
	// breaks down structural mass to keep itself alive — at lower efficiency than
	// digestion. This burns through mass faster than food would imply, making
	// prolonged starvation fatal even in a creature with mass to spare.
	const catabolismThreshold float32 = 0.15
	if c.Energy < c.MaxEnergy(s.Params)*catabolismThreshold {
		const catabolismEfficiency float32 = 0.35
		bmr := c.MetabolicRate(s.Params, temp)
		massConsumed := bmr / (s.Params.Metabolism.EnergyPerFoodMass * catabolismEfficiency)
		c.Mass -= massConsumed
		c.UpdateRadius(s.Params)
		c.GainEnergy(massConsumed*s.Params.Metabolism.EnergyPerFoodMass*catabolismEfficiency, s.Params)
	}

	if c.Energy <= 0 || c.Mass < c.SurvivalMass || c.Age > c.MaxAge(s.Params) {
		pending.death = append(pending.death, DeathInstruction{c})
		return
	}

	if s.Tick%sensorUpdatePeriod == c.Id%sensorUpdatePeriod {
		c.UpdateSensorContext(s.World, s.Population, s.Params)
	}
	c.FeedForward(s.World, s.Population, s.Tick, s.Params)
	s.executeActionsLocal(c, c.Nnet.LastActionValues[:], pending, temp)

	c.LastTickEnergy = c.Energy
	c.LastStomach = c.Stomach
	c.LastLoc = c.Loc
}

func (s *Simulation) Print() {
	fmt.Printf("Population Size: %d", len(s.Population.Creatures))
}

func (s *Simulation) executeActionsLocal(c *Creature, actionLevels []float32, pending *pendingInstructions, temp float32) {
	c.IsResting = false
	if IsActionEnabled(REST) {
		level := actionLevels[REST]
		absLevel := level
		if absLevel < 0 {
			absLevel = -absLevel
		}
		if absLevel > 0.75 {
			// Resting pays fraction of the basal metabolic rate: refund the base drain
			// already charged this tick, then re-charge the lower resting rate.
			const restingCostFactor float32 = 0.20
			rate := c.MetabolicRate(s.Params, temp)

			// Refund energy as resting
			c.GainEnergy(rate, s.Params)
			c.DrainEnergy(rate * restingCostFactor)
			c.IsResting = true

			c.LastActionMask |= ActionResting
		}
	}

	if c.Nnet.ActiveActions[SET_RESPONSIVENESS] {
		c.Responsiveness = tanhf(actionLevels[SET_RESPONSIVENESS])
	}

	responseAdjust := GetResponseCurve(c.Responsiveness)

	if IsActionEnabled(SET_OSCILLATOR_PERIOD) {
		actionVal := float64(actionLevels[SET_OSCILLATOR_PERIOD]) // [-1, 1]

		// Fast approximation of 2^x for x in [-1, 1]
		// 2^x ≈ 1 + 0.6931x + 0.2402x^2
		multiplier := 1.0 + (0.693147 * actionVal) + (0.240226 * actionVal * actionVal)

		finalTicks := c.BaseOscTick / multiplier

		if finalTicks < 2 {
			finalTicks = 2
		}
		c.Clock = int(finalTicks)
	}

	if IsActionEnabled(ATTACK) {
		level := math.Abs(float64(actionLevels[ATTACK]))
		if level > 0.7 && !c.IsResting {
			pending.attack = append(pending.attack, AttackInstruction{c, level})
			c.LastActionMask |= ActionAttacking
		}
	}

	if IsActionEnabled(REWARD) {
		level := float64(actionLevels[REWARD])
		if level > 0 {
			// Fast Softsign: level / (1 + level)
			c.GainDopamine(float32(level / (1.0 + level)))
			c.LastActionMask |= ActionRewarding
		}
	}

	if IsActionEnabled(PUNISH) {
		level := float64(actionLevels[PUNISH])
		if level > 0 {
			// Fast Softsign: level / (1 + level)
			c.LoseDopamine(float32(level / (1.0 + level)))
			c.LastActionMask |= ActionPunishing
		}
	}

	if IsActionEnabled(REPRODUCE) {
		currentTier := c.Tier
		baseThreshold := s.Params.Reproduction.EnergyThreshold * c.MaxEnergy(s.Params)

		// Mass the parent must donate: their split ratio × their current mass
		requiredMassContribution := c.Mass * c.Genome.CalculateSplitRatio()
		// Creature is of age and has the required mass to contribute to child
		milestonesMet := c.Age >= c.cachedJuvenilePeriod &&
			c.CanAffordMassInvestment(requiredMassContribution) &&
			c.ReproductionCooldown <= 0
		if currentTier >= 2 {
			level := actionLevels[REPRODUCE]
			brainWantsToReproduce := math.Abs(float64(level)) > 0.5

			// If the brain fires the action, give them a 10% discount on the energy required to encourage using it
			var dynamicEnergyThreshold float32
			if brainWantsToReproduce {
				dynamicEnergyThreshold = baseThreshold * 0.90
			} else {
				dynamicEnergyThreshold = baseThreshold // Standard high requirement
			}

			isPhysicallyReady := c.Energy >= dynamicEnergyThreshold && milestonesMet
			if isPhysicallyReady {
				if brainWantsToReproduce {
					if c.Genome.ReproductionType == 0 {
						pending.reproduction = append(pending.reproduction, ReproductionInstruction{Creature: c, IsFallback: false})
						c.LastActionMask |= ActionReproducing
					} else {
						pending.mate = append(pending.mate, c)
						c.LastActionMask |= ActionSeekingMate
					}
				} else if currentTier >= 2 && c.Energy >= baseThreshold {
					// If the brain fails to press the button at tier 2,
					// they still auto-split at maximum energy so they don't go extinct.
					pending.reproduction = append(pending.reproduction, ReproductionInstruction{Creature: c, IsFallback: true})
					c.LastActionMask |= ActionAutoSplitting
				}
			}
		} else {
			// Tier 0/1: Purely automatic physiological reproduction — requires full energy capacity
			isPhysicallyReady := c.Energy >= baseThreshold && milestonesMet
			if isPhysicallyReady {
				pending.reproduction = append(pending.reproduction, ReproductionInstruction{Creature: c, IsFallback: true})
				c.LastActionMask |= ActionAutoSplitting
			}
		}
	}

	if IsActionEnabled(FEED) {
		level := actionLevels[FEED]
		if (level > 0.5 && c.Stomach > 0) || (level < -0.5) {
			fwdX, fwdY := world.HeadingToVec(c.Heading)
			var bestDistSq float32 = math.MaxFloat32
			var recipient *Creature
			for _, id := range c.Sensors.SightCreatureIDs {
				if id == c.Id {
					continue
				}
				other, ok := s.Population.Get(id)
				if !ok || !other.Alive {
					continue
				}
				dx, dy := other.Loc.X-c.Loc.X, other.Loc.Y-c.Loc.Y
				d2 := dx*dx + dy*dy
				if d2 == 0 || d2 >= bestDistSq {
					continue
				}
				dist := float32(math.Sqrt(float64(d2)))
				dot := (fwdX*dx + fwdY*dy) / dist
				if dot < c.halfFOVCos {
					continue
				}
				if dist <= c.Radius+other.Radius {
					bestDistSq = d2
					recipient = other
				}
			}
			if recipient != nil {
				pending.feed = append(pending.feed, FeedInstruction{c, recipient, level})
				c.LastActionMask |= ActionFeeding
			}
		}
	}

	if !c.IsResting {
		// Inertia
		momentOfInertia := 0.5 * c.Mass * c.Radius * c.Radius

		rotateAmount := float32(0)
		if IsActionEnabled(ROTATE) {
			act := actionLevels[ROTATE]
			rawTorque := tanhf(act) * responseAdjust * s.Params.Creature.BaseMaxForce * c.Radius
			angularAccel := rawTorque / momentOfInertia
			rotateAmount = angularAccel * float32(1.0)
		}

		if rotateAmount != 0 {
			// Kinetic Energy Tax: E_k = 0.5 * I * omega^2
			rotationalWork := float32(0.5) * momentOfInertia * (rotateAmount * rotateAmount)
			energyTax := s.Params.Metabolism.MoveCostMultiplier * rotationalWork

			c.DrainEnergy(energyTax)
			c.LastActionMask |= ActionRotating
		}
		c.Heading = float32(world.NormalizeAngle(float64(c.Heading + rotateAmount)))

		accelAmount := float32(0)
		if IsActionEnabled(ACCELERATE) {
			act := actionLevels[ACCELERATE]
			accelAmount = act * responseAdjust * s.Params.Creature.BaseMaxForce * c.Radius
		}

		optTemp := (s.Params.Environment.TempMin + s.Params.Environment.TempMax) / 2
		speedMult := float32(1.0)
		if temp < optTemp && optTemp > s.Params.Environment.TempMin {
			coldNorm := (optTemp - temp) / (optTemp - s.Params.Environment.TempMin)
			if coldNorm > 1 {
				coldNorm = 1
			}
			speedMult = s.Params.Environment.ColdSpeedMultiplier + (1.0-s.Params.Environment.ColdSpeedMultiplier)*(1.0-coldNorm)
		}
		accelAmount *= speedMult
		// A = F / M
		c.Speed += accelAmount / c.Mass

		fluidDensity := s.Params.World.FluidDensity
		frontalArea := c.Radius * 2.0
		dragCoefficient := s.Params.World.DragCoefficient

		currentSpeed := c.Speed
		var dragForce float32
		if currentSpeed >= 0 {
			dragForce = 0.5 * fluidDensity * (currentSpeed * currentSpeed) * dragCoefficient * frontalArea
		} else {
			dragForce = -0.5 * fluidDensity * (currentSpeed * currentSpeed) * dragCoefficient * frontalArea
		}
		dragDecel := dragForce / c.Mass

		absDragDecel := dragDecel
		if absDragDecel < 0 {
			absDragDecel = -absDragDecel
		}
		absSpeed := currentSpeed
		if absSpeed < 0 {
			absSpeed = -absSpeed
		}

		if absDragDecel > absSpeed {
			c.Speed = 0
		} else {
			c.Speed -= dragDecel
		}

		if math.Abs(float64(c.Speed)) >= 0.001 {
			dx := float32(math.Cos(float64(c.Heading))) * c.Speed
			dy := float32(math.Sin(float64(c.Heading))) * c.Speed
			newPos := s.World.ClampToBounds(world.Position{X: c.Loc.X + dx, Y: c.Loc.Y + dy})

			if !s.World.IsWall(newPos) {
				absAccel := accelAmount
				if absAccel < 0 {
					absAccel = -absAccel
				}
				if absAccel > 0.001 {
					energyTax := s.Params.Metabolism.MoveCostMultiplier * absAccel
					c.DrainEnergy(energyTax)
				}
				c.LastActionMask |= ActionMoving
				pending.move = append(pending.move, MoveInstruction{c, newPos, c.Speed})
			}
		}
	}
}

// pairMates pairs candidates that fired MATE this tick. For each unpaired candidate it
// finds the best candidate within MatingRadius whose genome similarity meets
// MinMatingSimilarity, then appends a sexual ReproductionInstruction for both.
// Each creature can only participate in one pairing per tick.
func (s *Simulation) pairMates(candidates []*Creature) {
	if len(candidates) < 2 {
		return
	}

	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	paired := make(map[int]bool, len(candidates))

	for i, c := range candidates {
		// guard against dead or auto-splitting creatures
		if paired[c.Id] || !c.Alive || (c.LastActionMask&ActionAutoSplitting) != 0 {
			continue
		}

		bestIdx := -1
		var bestSimilarity float32 = -1.0
		var bestDistSq float32 = math.MaxFloat32

		for j, other := range candidates {
			if i == j || paired[other.Id] || !other.Alive {
				continue
			}

			dx := other.Loc.X - c.Loc.X
			dy := other.Loc.Y - c.Loc.Y
			d2 := dx*dx + dy*dy
			mr := float32(s.Params.Reproduction.MatingRadius)
			matingRadiusSq := (c.VisionRadius + c.Radius + mr) * (c.VisionRadius + c.Radius + mr)

			if d2 <= matingRadiusSq {
				similarity := c.cachedSimilarity(other.Id, other)
				if similarity >= s.Params.Reproduction.MinSimilarity {
					if similarity > bestSimilarity {
						bestSimilarity = similarity
						bestDistSq = d2
						bestIdx = j
					} else if similarity == bestSimilarity && d2 < bestDistSq {
						bestDistSq = d2
						bestIdx = j
					}
				}
			}
		}

		if bestIdx != -1 {
			partner := candidates[bestIdx]
			paired[c.Id] = true
			paired[partner.Id] = true
			s.Population.ReproductionQueue = append(s.Population.ReproductionQueue,
				ReproductionInstruction{Creature: c, Partner: partner, IsFallback: false})
		}
	}
}

// fountainLocation picks a random fountain (foliage or fungi) and returns a
// non-wall position within FountainRadius of it. Falls back to FindEmptyLocation
// if no fountains have been initialised.
func (s *Simulation) fountainLocation() (world.Position, bool) {
	foliage := s.World.FoliageFountains
	fungi := s.World.FungiFountains
	total := len(foliage) + len(fungi)
	if total == 0 {
		return s.World.FindEmptyLocation()
	}
	idx := rand.Intn(total)
	var center world.Position
	var radius float64
	if idx < len(foliage) {
		center = foliage[idx].Pos
		radius = s.Params.Food.Foliage.Radius
	} else {
		center = fungi[idx-len(foliage)].Pos
		radius = s.Params.Food.Fungi.Radius
	}
	pos := s.World.FindEmptyLocationNear(center, radius)
	return pos, true
}

// spawnInitialFood seeds the world with food equal to Food.InitialEnergy,
// split by the configured proportions and placed around fountains.
func (s *Simulation) spawnInitialFood() {
	p := s.Params.Food
	fp, funp, mp := normProportions(p.FoliageProportion, p.FungiProportion, p.MeatProportion, len(s.World.MeatFountains) > 0)
	epu := p.InitialEnergy
	epf := float64(p.FoliageMass) * float64(p.FoliageEnergyDensity)
	epu2 := float64(p.FungiMass) * float64(p.FungiEnergyDensity)
	epm := float64(p.MeatMass) * float64(p.MeatEnergyDensity)
	if nFoliage := int(epu*fp/epf + 0.5); nFoliage > 0 {
		s.World.SpawnFoliage(nFoliage, p.Foliage.Radius, p.FoliageMass, p.Foliage.RandomFraction, s.Tick)
	}
	if nFungi := int(epu*funp/epu2 + 0.5); nFungi > 0 {
		s.World.SpawnFungi(nFungi, p.Fungi.Radius, p.FungiMass, p.Fungi.RandomFraction, s.Tick)
	}
	if nMeat := int(epu*mp/epm + 0.5); nMeat > 0 {
		s.World.SpawnMeat(nMeat, p.Meat.Radius, p.MeatMass, p.Meat.RandomFraction, s.Tick)
	}
}

// spawnDeficit spawns food to cover an energy deficit, split by proportions.
func (s *Simulation) spawnDeficit(deficit float64) {
	if deficit <= 0 {
		return
	}
	p := s.Params.Food
	fp, funp, mp := normProportions(p.FoliageProportion, p.FungiProportion, p.MeatProportion, len(s.World.MeatFountains) > 0)
	epf := float64(p.FoliageMass) * float64(p.FoliageEnergyDensity)
	epu2 := float64(p.FungiMass) * float64(p.FungiEnergyDensity)
	epm := float64(p.MeatMass) * float64(p.MeatEnergyDensity)
	if nFoliage := int(deficit*fp/epf + 0.5); nFoliage > 0 {
		s.World.SpawnFoliage(nFoliage, p.Foliage.Radius, p.FoliageMass, p.Foliage.RandomFraction, s.Tick)
	}
	if nFungi := int(deficit*funp/epu2 + 0.5); nFungi > 0 {
		s.World.SpawnFungi(nFungi, p.Fungi.Radius, p.FungiMass, p.Fungi.RandomFraction, s.Tick)
	}
	if nMeat := int(deficit*mp/epm + 0.5); nMeat > 0 {
		s.World.SpawnMeat(nMeat, p.Meat.Radius, p.MeatMass, p.Meat.RandomFraction, s.Tick)
	}
}

// normProportions returns foliage/fungi/meat proportions normalised to sum to 1.
// If hasMeatFountains is false, meat proportion is forced to 0 and foliage+fungi
// absorb the remainder so the deficit is always fully covered.
func normProportions(fo, fu, me float64, hasMeatFountains bool) (float64, float64, float64) {
	if !hasMeatFountains {
		me = 0
	}
	total := fo + fu + me
	if total <= 0 {
		return 1, 0, 0
	}
	return fo / total, fu / total, me / total
}

func (s *Simulation) SetClusterEnabled(v bool)      { s.Params.Spawn.ClusterEnabled = v }
func (s *Simulation) SetClusterInterval(v int)      { s.Params.Spawn.ClusterInterval = v }
func (s *Simulation) SetClusterSize(v int)          { s.Params.Spawn.ClusterSize = v }
func (s *Simulation) SetBaseMutationRate(v float32) { s.Params.Neurology.BaseMutationRate = v }

func (s *Simulation) SetFoliageProportion(v float64)     { s.Params.Food.FoliageProportion = v }
func (s *Simulation) SetFungiProportion(v float64)       { s.Params.Food.FungiProportion = v }
func (s *Simulation) SetMeatProportion(v float64)        { s.Params.Food.MeatProportion = v }
func (s *Simulation) SetFoliageRandomFraction(v float64) { s.Params.Food.Foliage.RandomFraction = v }
func (s *Simulation) SetFoliageDriftSpeed(v float64)     { s.Params.Food.Foliage.DriftSpeed = v }
func (s *Simulation) SetFoliageRadius(v float64)         { s.Params.Food.Foliage.Radius = v }
func (s *Simulation) SetFungiRandomFraction(v float64)   { s.Params.Food.Fungi.RandomFraction = v }
func (s *Simulation) SetFungiDriftSpeed(v float64)       { s.Params.Food.Fungi.DriftSpeed = v }
func (s *Simulation) SetFungiRadius(v float64)           { s.Params.Food.Fungi.Radius = v }
func (s *Simulation) SetMeatRandomFraction(v float64)    { s.Params.Food.Meat.RandomFraction = v }
func (s *Simulation) SetMeatDriftSpeed(v float64)        { s.Params.Food.Meat.DriftSpeed = v }
func (s *Simulation) SetMeatRadius(v float64)            { s.Params.Food.Meat.Radius = v }
func (s *Simulation) SetWarmMetabolicMultiplier(v float32) {
	s.Params.Environment.WarmMetabolicMultiplier = v
}
func (s *Simulation) SetColdSpeedMultiplier(v float32) {
	s.Params.Environment.ColdSpeedMultiplier = v
}
func (s *Simulation) SetTempMin(v float32) {
	s.Params.Environment.TempMin = v
	s.World.TempMin = v
}
func (s *Simulation) SetTempMax(v float32) {
	s.Params.Environment.TempMax = v
	s.World.TempMax = v
}

func (s *Simulation) SetFoliageFountainCount(n int) {
	if n < 0 {
		n = 0
	}
	s.Params.Food.Foliage.Count = n
	s.World.SetFoliageFountainCount(n, s.Params.Food.Foliage.StationaryCount)
}

func (s *Simulation) SetFungiFountainCount(n int) {
	if n < 0 {
		n = 0
	}
	s.Params.Food.Fungi.Count = n
	s.World.SetFungiFountainCount(n, s.Params.Food.Fungi.StationaryCount)
}

func (s *Simulation) SetMeatFountainCount(n int) {
	if n < 0 {
		n = 0
	}
	s.Params.Food.Meat.Count = n
	s.World.SetMeatFountainCount(n, s.Params.Food.Meat.StationaryCount)
}

func (s *Simulation) SetFoliageStationaryCount(v int) {
	s.Params.Food.Foliage.StationaryCount = v
	s.World.RecomputeFoliageStationary(v)
}

func (s *Simulation) SetFungiStationaryCount(v int) {
	s.Params.Food.Fungi.StationaryCount = v
	s.World.RecomputeFungiStationary(v)
}

func (s *Simulation) SetMeatStationaryCount(v int) {
	s.Params.Food.Meat.StationaryCount = v
	s.World.RecomputeMeatStationary(v)
}

// SpawnAt creates a new random creature at the given world-space position.
// Returns false if the position is inside a wall.
func (s *Simulation) SpawnAt(x, y float64) bool {
	pos := s.World.ClampToBounds(world.Position{X: float32(x), Y: float32(y)})
	if s.World.IsWall(pos) {
		return false
	}
	spawnParams := *s.Params

	id := s.World.AddCreature(pos)
	// Initialise genomes at tier zero
	genome := MakeRandomGenome(&spawnParams, 0)
	c := NewAdultCreature(id, pos, genome, s.Params)
	s.Population.SetCreature(id, c)
	s.Population.AddAlive(id)

	return true
}

// SpawnClusterAt creates count identical creatures near the given world-space position.
// All creatures share the same randomly generated genome. Positions that are walls are skipped.
func (s *Simulation) SpawnClusterAt(x, y float64, count int) bool {
	spawnParams := *s.Params
	// Initialise genomes at tier zero
	genome := MakeRandomGenome(&spawnParams, 0)

	offsets := [][2]float64{{0, 0}, {4, 0}, {-4, 0}, {0, 4}, {0, -4}}
	spawned := 0
	for i := 0; spawned < count && i < len(offsets); i++ {
		pos := s.World.ClampToBounds(world.Position{X: float32(x + offsets[i][0]), Y: float32(y + offsets[i][1])})
		if s.World.IsWall(pos) {
			continue
		}
		id := s.World.AddCreature(pos)
		c := NewAdultCreature(id, pos, genome.Copy(), s.Params)
		s.Population.SetCreature(id, c)
		s.Population.AddAlive(id)
		spawned++
	}
	return spawned > 0
}

// SpawnGenome places a new adult creature with the given genome at a random empty location.
// generation sets the creature's starting generation; pass 0 or 1 for a fresh spawn.
func (s *Simulation) SpawnGenome(g *Genome, generation float32) bool {
	loc, ok := s.World.FindEmptyLocation()
	if !ok {
		return false
	}
	id := s.World.AddCreature(loc)
	c := NewAdultCreature(id, loc, g, s.Params)
	if generation > 1 {
		c.Generation = generation
		c.Tier = GetTierFromGeneration(c.Generation, s.Params)
	}
	s.Population.SetCreature(id, c)
	s.Population.AddAlive(id)
	return true
}

// CreatureGenomeCopy returns a deep copy of a living creature's genome by ID.
func (s *Simulation) CreatureGenomeCopy(id int) (*Genome, bool) {
	c, ok := s.Population.Get(id)
	if !ok || !c.Alive {
		return nil, false
	}
	return c.Genome.Copy(), true
}

// GetParams exposes the simulation parameters to the UI layer.
func (s *Simulation) GetParams() *Parameters { return s.Params }

func (s *Simulation) WorldWidth() float64  { return s.Params.World.Width }
func (s *Simulation) WorldHeight() float64 { return s.Params.World.Height }

func (s *Simulation) PopulationCount() int { return s.Population.AliveCount() }

func (s *Simulation) FoliageCount() int { return s.World.FoliageCount() }

func (s *Simulation) FungiCount() int { return s.World.FungiCount() }
func (s *Simulation) MeatCount() int  { return s.World.MeatCount() }

func (s *Simulation) FoliageEnergy() float64 {
	return s.World.TotalFoliageMass() * float64(s.Params.Food.FoliageEnergyDensity)
}

func (s *Simulation) FungiEnergy() float64 {
	return s.World.TotalFungiMass() * float64(s.Params.Food.FungiEnergyDensity)
}

func (s *Simulation) MeatEnergy() float64 {
	return s.World.TotalMeatMass() * float64(s.Params.Food.MeatEnergyDensity)
}

// TotalEnergy returns the total liquid energy in the system: food, meat, and the
// immediate metabolic stores (energy + stomach contents) of all living creatures.
// Stomach contents are valued at EnergyPerFoodMass because density is baked in at eat time.
func (s *Simulation) TotalEnergy() float64 {
	epu := float64(s.Params.Metabolism.EnergyPerFoodMass)
	energy := s.World.TotalFoliageMass() * float64(s.Params.Food.FoliageEnergyDensity)
	energy += s.World.TotalFungiMass() * float64(s.Params.Food.FungiEnergyDensity)
	energy += s.World.TotalMeatMass() * float64(s.Params.Food.MeatEnergyDensity)
	for _, c := range s.Population.Creatures {
		if c == nil {
			continue
		}
		energy += float64(c.Energy) + (float64(c.Mass)+float64(c.Stomach))*epu
	}
	return energy
}

func (s *Simulation) TargetEnergy() float64     { return s.Energy }
func (s *Simulation) SetTargetEnergy(v float64) { s.Energy = v }

func (s *Simulation) AverageAge() float64 {
	count := len(s.Population.aliveIDs)
	if count == 0 {
		return 0
	}
	total := 0
	for _, id := range s.Population.aliveIDs {
		if c, ok := s.Population.Get(id); ok {
			total += c.Age
		}
	}
	return float64(total) / float64(count)
}

func (s *Simulation) AverageGeneration() float64 {
	count := len(s.Population.aliveIDs)
	if count == 0 {
		return 0
	}
	total := float32(0.0)
	for _, id := range s.Population.aliveIDs {
		if c, ok := s.Population.Get(id); ok {
			total += c.Generation
		}
	}
	return float64(total) / float64(count)
}

// partitionIDs splits ids into n contiguous sub-slices. Contiguous layout keeps
// Creatures[id] accesses sequential in memory, improving L1 cache hit rates vs
// the previous round-robin approach. Each sub-slice shares the backing array of
// ids so no additional allocation occurs for the batch data.
func partitionIDs(ids []int, n int) [][]int {
	if n <= 0 {
		n = 1
	}
	if n > len(ids) {
		n = len(ids)
	}
	if n == 0 {
		return nil
	}
	batches := make([][]int, n)
	size := (len(ids) + n - 1) / n
	for i := range batches {
		start := i * size
		if start >= len(ids) {
			batches[i] = nil
			continue
		}
		end := start + size
		if end > len(ids) {
			end = len(ids)
		}
		batches[i] = ids[start:end]
	}
	return batches
}

// LastAction bitmask constants — one bit per action verb.
const (
	ActionResting       uint16 = 1 << iota // 1
	ActionAttacking                        // 2
	ActionRewarding                        // 4
	ActionPunishing                        // 8
	ActionReproducing                      // 16
	ActionSeekingMate                      // 32
	ActionAutoSplitting                    // 64
	ActionFeeding                          // 128
	ActionRotating                         // 256
	ActionMoving                           // 512
)

func actionMaskToString(mask uint16) string {
	if mask == 0 {
		return ""
	}
	names := [...]struct {
		bit  uint16
		name string
	}{
		{ActionResting, "Resting"},
		{ActionAttacking, "Attacking"},
		{ActionRewarding, "Rewarding"},
		{ActionPunishing, "Punishing"},
		{ActionReproducing, "Reproducing (Intent)"},
		{ActionSeekingMate, "Seeking mate"},
		{ActionAutoSplitting, "Auto-Splitting"},
		{ActionFeeding, "Feeding"},
		{ActionRotating, "Rotating"},
		{ActionMoving, "Moving"},
	}
	out := ""
	for _, n := range names {
		if mask&n.bit != 0 {
			if out != "" {
				out += " | "
			}
			out += n.name
		}
	}
	return out
}

func (s *Simulation) updatePopulationCaches() {
	s.displayCache = s.displayCache[:0]

	for id, c := range s.Population.Creatures {
		if c == nil {
			continue
		}
		r, g, b, a := c.Color.RGBA()
		s.displayCache = append(s.displayCache, CreatureView{
			ID: id, X: float64(c.Loc.X), Y: float64(c.Loc.Y), Heading: float64(c.Heading),
			R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8),
			CurrentMass:      float64(c.Mass),
			VisionRadius:     float64(c.GetVisionRadius()),
			FieldOfView:      c.FieldOfView(),
			Radius:           float64(c.Radius),
			ReproductionType: c.Genome.ReproductionType,
			Tier:             c.Tier,
		})
	}
}

func (s *Simulation) updateFoodCache() {
	s.foodCache = s.foodCache[:0]
	s.World.ForEachActiveFood(func(id int, x, y float64, r float64, typ uint8) {
		s.foodCache = append(s.foodCache, FoodView{ID: id, X: x, Y: y, Radius: r, Type: typ})
	})
}

// StateSnapshot holds the display state for one rendered frame.
// Food contains both Foliages (Type==FoodTypeFoliage) and meat (Type==FoodTypeMeat).
type StateSnapshot struct {
	Creatures []CreatureView
	Food      []FoodView
}

func (s *Simulation) GetSnapshot() StateSnapshot {
	s.cacheMu.Lock()
	if s.cacheDirty {
		s.updatePopulationCaches()
		s.updateFoodCache()
		s.cacheDirty = false
	}
	snap := StateSnapshot{
		Creatures: append([]CreatureView(nil), s.displayCache...),
		Food:      append([]FoodView(nil), s.foodCache...),
	}
	s.cacheMu.Unlock()
	return snap
}

// FillSnapshot writes the current display state into dst, reusing its backing
// slices if they are large enough. After the first call the slices reach full
// capacity and subsequent calls allocate nothing.
func (s *Simulation) FillSnapshot(dst *StateSnapshot) {
	s.cacheMu.Lock()
	if s.cacheDirty {
		s.updatePopulationCaches()
		s.updateFoodCache()
		s.cacheDirty = false
	}
	dst.Creatures = append(dst.Creatures[:0], s.displayCache...)
	dst.Food = append(dst.Food[:0], s.foodCache...)
	s.cacheMu.Unlock()
}

// Pre-cached response curve to save on math.Pow calls
var ResponseCurveLUT [256]float32

// Initialise at startup
func InitResponseCurve(params *Parameters) {
	for i := 0; i < 256; i++ {
		// Map index 0 -> 255 directly to a 0.0 -> 1.0 float range
		resp := float32(i) / 255.0
		ResponseCurveLUT[i] = calculateResponseCurve(resp, params.Neurology.ResponseCurveKFactor)
	}
}

// Fast access using the float32 from the brain
func GetResponseCurve(resp float32) float32 {
	// Clamp resp to [-1, 1] then convert to 0-255 index
	if resp < -1 {
		resp = -1
	}
	if resp > 1 {
		resp = 1
	}
	index := uint8((resp + 1.0) * 127.5)
	return ResponseCurveLUT[index]
}

func calculateResponseCurve(resp float32, kFactor float32) float32 {
	// Use Absolute value so (r - 2.0) is always negative,
	// but we use the absolute distance from 2.0 to avoid NaN.
	// Bio-logic: We care about the magnitude of responsiveness.
	r := math.Abs(float64(resp))
	k2 := -2.0 * float64(kFactor)

	// Result = (dist from 2)^k2 - (2)^k2 * (1 - r)
	term1 := math.Pow(2.0-r, k2) // 2.0 - r ensures base is positive (1.0 to 3.0)
	term2 := math.Pow(2.0, k2)

	return float32(term1 - term2*(1.0-r))
}
