package simulation

import (
	"biogo/v2/world"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
)

type Simulation struct {
	World        *world.World
	Population   *Population
	Tick         int
	Params       *Parameters
	Energy       float64 // total liquid energy to maintain (set at initialisation)
	displayCache []CreatureView
	foodCache    []FoodView // combined plants and meat
	cacheMu      sync.RWMutex
}

func New(params *Parameters) *Simulation {
	InitResponseCurve(params)
	sim := &Simulation{
		Params: params,
	}
	sim.initializeWorld()
	sim.initializePopulation()
	sim.Energy = sim.TotalEnergy()
	return sim
}

func (s *Simulation) initializeWorld() {
	s.World = world.NewWorld(s.Params.WorldWidth, s.Params.WorldHeight, 1)
	s.World.SpawnRandom(s.Params.MaxFood/2, s.Params.FoodMass)
	s.World.InitFountains(s.Params.FountainCount)
}

func (s *Simulation) initializePopulation() {
	pop := NewPopulation(s.Params)
	savedGenomes, _ := LoadAllCreatureGenomes()
	maxSeeded := int(float64(s.Params.StartingPopulation) * s.Params.SavedGenomeProportion)
	numSaved := len(savedGenomes)
	bubbleSize := 5
	spawnRadius := 15.0

	for i := 0; i < s.Params.StartingPopulation; i += bubbleSize {
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
			if i+j >= s.Params.StartingPopulation {
				break
			}

			loc := s.World.FindEmptyLocationNear(centerLoc, spawnRadius)

			id := s.World.AddCreature(loc)
			c := NewAdultCreature(id, loc, genome, s.Params)
			pop.Creatures[id] = c
			pop.AddAlive(id)
		}
	}
	s.Population = pop
}

// SaveCreature saves the genome of the creature with the given id to a unique file in data/creatures/.
func (s *Simulation) SaveCreature(id int) error {
	c, ok := s.Population.Creatures[id]
	if !ok || !c.Alive {
		return nil
	}
	return SaveCreatureToFile(c.Genome)
}

// Reset reinitialises the simulation from scratch.
func (s *Simulation) Reset() {
	s.Tick = 0
	s.initializeWorld()
	s.initializePopulation()
	s.Energy = s.TotalEnergy()
}

func (s *Simulation) Update() {
	s.step()
	s.cacheMu.Lock()
	s.updatePopulationCaches()
	s.updateFoodCache()
	s.cacheMu.Unlock()
}

func (s *Simulation) step() {
	s.World.StepFountains(s.Params.FountainDriftSpeed)

	if s.Tick%s.Params.FoodSpawnInterval == 0 {
		// Spawning food by energy temporarily disabled. Not enough creatures
		// eat meat and so it energy ends up stagnating in meat.+
		deficit := s.Energy - s.TotalEnergy()
		if deficit < 0 {
			deficit = 0
		}
		energyPerPiece := float64(s.Params.FoodMass) * float64(s.Params.EnergyPerMassUnit)
		// number of food items to spawn
		n := int(deficit / energyPerPiece)
		// n := s.Params.MaxFood - s.World.PlantCount()
		s.World.SpawnPlant(n, s.Params.FountainRadius, s.Params.FoodMass)
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
	s.Population.ProcessEating(s.World, s.Params)
	s.processCollisions()
	s.Population.ProcessAttackQueue(s.World, s.Params)
	s.Population.ProcessFeedQueue(s.Params)
	s.Population.ProcessDeathQueue(s.World, s.Params)
	s.Population.ProcessReproductionQueue(s.World, s.Params)

	aliveCount := s.Population.AliveCount()
	const toSpawn = 5
	for aliveCount < s.Params.MinPopulation {
		loc, ok := s.World.FindEmptyLocation()
		if !ok {
			break
		}
		s.SpawnClusterAt(float64(loc.X), float64(loc.Y), toSpawn)
		aliveCount += toSpawn
	}

	s.Tick++
}

func (s *Simulation) stepCreatureLocal(c *Creature, pending *pendingInstructions) {
	c.Dopamine *= 0.9
	if c.Dopamine > -0.01 && c.Dopamine < 0.01 {
		c.Dopamine = 0
	}
	c.Age++
	c.LastAction = ""
	temp := s.World.TemperatureAt(c.Loc.Y)
	c.DrainEnergy(c.MetabolicRate(s.Params, temp))
	if float64(c.Loc.X) < s.Params.RadiationZoneWidth*s.Params.WorldWidth {
		massNorm := float64(c.Mass) / s.Params.MaxMass
		massEffect := float32(math.Sqrt(massNorm * math.Sqrt(massNorm)))
		c.DrainEnergy(s.Params.RadiationDamagePerTick * massEffect)
	}
	c.Digest(s.Params)
	c.GrowMass(s.Params)

	if c.Energy <= 0 || c.Age > c.MaxAge(s.Params) {
		pending.death = append(pending.death, DeathInstruction{c})
		return
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
		if math.Abs(float64(level)) > 0.75 {
			// Resting pays fraction of the basal metabolic rate: refund the base drain
			// already charged this tick, then re-charge the lower resting rate.
			massRatio := float64(c.Mass) / s.Params.MaxMass
			restingCostFactor := 0.1 - (massRatio * 0.08)
			rate := c.MetabolicRate(s.Params, temp)

			c.GainEnergy(rate, s.Params)
			c.DrainEnergy(rate * float32(restingCostFactor))
			c.IsResting = true

			c.LastAction = appendActionString(c.LastAction, "Resting")
		}
	}

	if IsActionEnabled(SET_RESPONSIVENESS) {
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
		if level > 0.7 {
			pending.attack = append(pending.attack, AttackInstruction{c, level})
			c.LastAction = appendActionString(c.LastAction, "Attacking")
		}
	}

	if IsActionEnabled(REWARD) {
		level := float64(actionLevels[REWARD])
		if level > 0 {
			// Fast Softsign: level / (1 + level)
			c.GainDopamine(float32(level / (1.0 + level)))
			c.LastAction = appendActionString(c.LastAction, "Rewarding")
		}
	}

	if IsActionEnabled(PUNISH) {
		level := float64(actionLevels[PUNISH])
		if level > 0 {
			// Fast Softsign: level / (1 + level)
			c.LoseDopamine(float32(level / (1.0 + level)))
			c.LastAction = appendActionString(c.LastAction, "Punishing")
		}
	}

	if IsActionEnabled(REPRODUCE) {
		currentTier := c.Tier
		reproThreshold := s.Params.ReproductionEnergyThreshold * c.MaxEnergy(s.Params)
		isPhysicallyReady := c.Energy >= reproThreshold &&
			c.Age >= c.cachedJuvenilePeriod &&
			float64(c.Mass) >= float64(c.Genome.Mass)*0.9 &&
			float32(c.Genome.MinMass)*2 < float32(c.Genome.Mass)
		// Reproduction is not introduced until tier 3
		if currentTier >= 2 {
			level := actionLevels[REPRODUCE]
			if math.Abs(float64(level)) > 0.5 && isPhysicallyReady {
				if c.Genome.ReproductionType == 0 {
					pending.reproduction = append(pending.reproduction, ReproductionInstruction{Creature: c})
					c.LastAction = appendActionString(c.LastAction, "Reproducing (Intent)")
				} else {
					pending.mate = append(pending.mate, c)
					c.LastAction = appendActionString(c.LastAction, "Seeking mate")
				}
			}
		} else {
			// Automatic physiological reproduction
			if isPhysicallyReady {
				// Force asexual division for lower tiers to scale population early on
				pending.reproduction = append(pending.reproduction, ReproductionInstruction{Creature: c})
				c.LastAction = appendActionString(c.LastAction, "Auto-Splitting")
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
				other, ok := s.Population.Creatures[id]
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
				c.LastAction = appendActionString(c.LastAction, "Feeding")
			}
		}
	}

	if !c.IsResting {
		// Rotation: positive level turns CCW (left), negative turns CW (right).
		rotateAmount := float64(0)
		massNorm := float64(c.Mass) / s.Params.MaxMass
		if IsActionEnabled(ROTATE) {
			act := float64(actionLevels[ROTATE])
			turnInertia := 1.0 / (1.0 + (massNorm * 4.0))
			rotateAmount = float64(tanhf(float32(act))) * float64(responseAdjust) * s.Params.MaxRotationPerStep * turnInertia
		}
		if rotateAmount != 0 {
			massCostMult := 0.5 + (massNorm * massNorm * 2.0)
			c.DrainEnergy(s.Params.MoveCost * float32(math.Abs(rotateAmount)) * 0.5 * float32(massCostMult))
			c.LastAction = appendActionString(c.LastAction, "Rotating")
		}
		c.Heading = float32(world.NormalizeAngle(float64(c.Heading) + rotateAmount))

		massFactor := 1.0 + (massNorm * massNorm * 5.0)
		maxAccel := s.Params.MaxSpeedPerStep / massFactor
		accelAmount := float64(0)
		// ACCELERATE: positive = accelerate forward, negative = decelerate/reverse.
		if IsActionEnabled(ACCELERATE) {
			act := float64(actionLevels[ACCELERATE])
			accelAmount = float64(act) * float64(responseAdjust) * maxAccel
		}

		// Colder temperatures reduce acceleration capability (ectotherm-like muscle penalty).
		tempNorm := float64((temp - 10.0) / 30.0)
		if tempNorm < 0 {
			tempNorm = 0
		} else if tempNorm > 1 {
			tempNorm = 1
		}
		speedMult := float64(s.Params.ColdSpeedMultiplier) + (1.0-float64(s.Params.ColdSpeedMultiplier))*tempNorm
		accelAmount *= speedMult

		// Integrate acceleration into velocity, apply drag, clamp to mass-adjusted max speed.
		c.Velocity += float32(accelAmount)
		c.Velocity = float32(float64(c.Velocity) * s.Params.VelocityDamping)
		maxSpeed := float32(s.Params.MaxSpeedPerStep / massFactor)
		if c.Velocity > maxSpeed {
			c.Velocity = maxSpeed
		} else if c.Velocity < -maxSpeed {
			c.Velocity = -maxSpeed
		}

		if math.Abs(float64(c.Velocity)) >= 0.001 {
			dx := float32(math.Cos(float64(c.Heading))) * c.Velocity
			dy := float32(math.Sin(float64(c.Heading))) * c.Velocity
			newPos := s.World.ClampToBounds(world.Position{X: c.Loc.X + dx, Y: c.Loc.Y + dy})

			if !s.World.IsWall(newPos) {
				if math.Abs(accelAmount) > 0.001 {
					// Energy cost scales linearly with mass: heavier creatures need more force to accelerate.
					massCostMult := 0.5 + massNorm*2.0
					c.DrainEnergy(s.Params.MoveCost * float32(math.Abs(accelAmount)) * float32(massCostMult))
				}
				c.LastAction = appendActionString(c.LastAction, "Moving")
				pending.move = append(pending.move, MoveInstruction{c, newPos, c.Velocity})
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
		if paired[c.Id] || !c.Alive {
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
			mr := float32(s.Params.MatingRadius)
			matingRadiusSq := (c.SightDistance + mr) * (c.SightDistance + mr)

			if d2 <= matingRadiusSq {
				similarity := c.cachedSimilarity(other.Id, other)
				if similarity >= s.Params.MinMatingSimilarity {
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
				ReproductionInstruction{Creature: c, Partner: partner})
		}
	}
}

// SetSpawnMutationRate sets the minimum mutation rate applied when artificially
// spawning creatures to maintain the minimum population.
func (s *Simulation) SetSpawnMutationRate(rate float32) {
	s.Params.SpawnMutationRate = rate
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
	s.Population.Creatures[id] = c
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
		s.Population.Creatures[id] = c
		s.Population.AddAlive(id)
		spawned++
	}
	return spawned > 0
}

// SpawnGenome places a new adult creature with the given genome at a random empty location.
func (s *Simulation) SpawnGenome(g *Genome) bool {
	loc, ok := s.World.FindEmptyLocation()
	if !ok {
		return false
	}
	id := s.World.AddCreature(loc)
	c := NewAdultCreature(id, loc, g, s.Params)
	s.Population.Creatures[id] = c
	s.Population.AddAlive(id)
	return true
}

// CreatureGenomeCopy returns a deep copy of a living creature's genome by ID.
func (s *Simulation) CreatureGenomeCopy(id int) (*Genome, bool) {
	c, ok := s.Population.Creatures[id]
	if !ok || !c.Alive {
		return nil, false
	}
	return c.Genome.Copy(), true
}

// GetParams exposes the simulation parameters to the UI layer.
func (s *Simulation) GetParams() *Parameters { return s.Params }

func (s *Simulation) WorldWidth() float64  { return s.Params.WorldWidth }
func (s *Simulation) WorldHeight() float64 { return s.Params.WorldHeight }

func (s *Simulation) PopulationCount() int { return s.Population.AliveCount() }

func (s *Simulation) PlantCount() int { return s.World.PlantCount() }

// TotalEnergy returns the total liquid energy in the system: food, meat, and the
// immediate metabolic stores (energy + stomach contents) of all living creatures.
func (s *Simulation) TotalEnergy() float64 {
	epu := float64(s.Params.EnergyPerMassUnit)
	energy := s.World.TotalPlantMass() * epu
	energy += s.World.TotalMeatMass() * epu
	for _, c := range s.Population.Creatures {
		energy += float64(c.Energy) + (float64(c.Mass)+float64(c.Stomach))*epu
	}
	return energy
}

func (s *Simulation) TargetEnergy() float64 {
	return s.Energy
}

func (s *Simulation) AverageAge() float64 {
	count := len(s.Population.aliveIDs)
	if count == 0 {
		return 0
	}
	total := 0
	for _, id := range s.Population.aliveIDs {
		total += s.Population.Creatures[id].Age
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
		total += s.Population.Creatures[id].Generation
	}
	return float64(total) / float64(count)
}

// partitionIDs splits ids into n roughly equal batches using round-robin assignment.
func partitionIDs(ids []int, n int) [][]int {
	if n <= 0 {
		n = 1
	}
	batches := make([][]int, n)
	for i, id := range ids {
		batches[i%n] = append(batches[i%n], id)
	}
	return batches
}

func appendActionString(base, new string) string {
	if base == "" {
		return new
	}
	return base + " | " + new
}

func (s *Simulation) updatePopulationCaches() {
	s.displayCache = s.displayCache[:0]

	for id, c := range s.Population.Creatures {
		r, g, b, a := c.Color.RGBA()
		s.displayCache = append(s.displayCache, CreatureView{
			ID: id, X: float64(c.Loc.X), Y: float64(c.Loc.Y), Heading: float64(c.Heading),
			R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8),
			CurrentMass:      float64(c.Mass),
			SightDistance:    float64(c.GetSightDistance()),
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
// Food contains both plants (Type==FoodTypePlant) and meat (Type==FoodTypeMeat).
type StateSnapshot struct {
	Creatures []CreatureView
	Food      []FoodView
}

func (s *Simulation) GetSnapshot() StateSnapshot {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	return StateSnapshot{
		Creatures: append([]CreatureView(nil), s.displayCache...),
		Food:      append([]FoodView(nil), s.foodCache...),
	}
}

// Pre-cached response curve to save on math.Pow calls
var ResponseCurveLUT [256]float32

// Initialize at startup
func InitResponseCurve(params *Parameters) {
	for i := 0; i < 256; i++ {
		// Map index 0 -> 255 directly to a 0.0 -> 1.0 float range
		resp := float32(i) / 255.0
		ResponseCurveLUT[i] = calculateResponseCurve(resp, params.ResponseCurveKFactor)
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
