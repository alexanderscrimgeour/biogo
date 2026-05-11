package simulation

import (
	"biogo/v2/grid"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
)

type Simulation struct {
	World        *grid.World
	Population   *Population
	Tick         int
	Params       *Parameters
	TargetEnergy float64 // total liquid energy to maintain (set at initialisation)
	displayCache []CreatureView
	foodCache    []FoodView
	corpseCache  []CorpseView
	cacheMu      sync.RWMutex
}

func New(params *Parameters) *Simulation {
	sim := &Simulation{
		Params: params,
	}
	sim.initializeWorld()
	sim.initializePopulation()
	sim.TargetEnergy = sim.TotalEnergy()
	return sim
}

func (s *Simulation) initializeWorld() {
	s.World = grid.NewWorld(s.Params.GridWidth, s.Params.GridHeight, 1)
	s.World.SpawnRandom(s.Params.MaxFood, s.Params.FoodMass)
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
			genome = MakeRandomGenome(s.Params)
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
	s.TargetEnergy = s.TotalEnergy()
}

func (s *Simulation) Update() {
	s.step()
	s.cacheMu.Lock()
	s.updatePopulationCaches() // Rename your existing updateDisplayCache to this
	s.updateFoodCache()
	s.cacheMu.Unlock()
}

func (s *Simulation) step() {
	s.World.StepFountains(s.Params.FountainDriftSpeed)

	if s.Tick%s.Params.FoodSpawnInterval == 0 {
		deficit := s.TargetEnergy - s.TotalEnergy()
		toSpawn := int(deficit / float64(s.Params.FoodMass))
		if toSpawn > 0 {
			if available := s.Params.MaxFood - s.World.FoodCount(); toSpawn > available {
				toSpawn = available
			}
			s.World.SpawnFood(toSpawn, s.Params.FountainRadius, s.Params.FoodMass)
		}
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
	}
	s.pairMates(wantMate)

	s.Population.ProcessMoveQueue(s.World, s.Params)
	s.Population.ProcessAttackQueue(s.World, s.Params)
	s.Population.ProcessDeathQueue(s.World, s.Params)
	s.Population.ProcessCorpseDecay(s.World, s.Params)
	s.Population.ProcessReproductionQueue(s.World, s.Params)

	// Reward decay — iterate only alive creatures via the maintained index.
	for _, id := range s.Population.aliveIDs {
		c := s.Population.Creatures[id]
		c.Dopamine *= 0.9
		if c.Dopamine > -0.01 && c.Dopamine < 0.01 {
			c.Dopamine = 0
		}
	}

	spawnParams := *s.Params

	aliveCount := s.Population.AliveCount()
	for aliveCount < s.Params.MinPopulation {
		loc, ok := s.World.FindEmptyLocation()
		if !ok {
			break
		}
		var genome *Genome
		if source := s.Population.OldestGenome(); source != nil {
			genome = ArtificialReproduction(source, &spawnParams)
		} else {
			genome = MakeRandomGenome(&spawnParams)
		}
		id := s.World.AddCreature(loc)
		c := NewAdultCreature(id, loc, genome, s.Params)
		s.Population.Creatures[id] = c
		s.Population.AddAlive(id)
		aliveCount++
	}

	s.Tick++
}

func (s *Simulation) stepCreatureLocal(c *Creature, pending *pendingInstructions) {
	c.Age++
	c.GrowMass(s.Params)
	c.LastAction = ""
	temp := s.World.TemperatureAt(c.Loc.Y)
	c.DrainEnergy(c.MetabolicRate(s.Params, temp))
	if c.Loc.X < s.Params.RadiationZoneWidth*s.Params.GridWidth {
		massNorm := c.Mass / float32(s.Params.MaxMass)
		c.DrainEnergy(s.Params.RadiationDamagePerTick * float32(math.Pow(float64(massNorm), 0.75)))
	}
	c.Digest(s.Params)
	if c.Energy <= 0 || c.Age > c.MaxAge(s.Params) {
		pending.death = append(pending.death, DeathInstruction{c})
		return
	}

	actionLevels := c.FeedForward(s.World, s.Population, s.Tick, s.Params)
	s.executeActionsLocal(c, actionLevels, pending, temp)

	c.LastTickEnergy = c.Energy
	c.LastStomach = c.Stomach
	c.LastLoc = c.Loc
}

func (s *Simulation) Print() {
	fmt.Printf("Population Size: %d", len(s.Population.Creatures))
}

func (s *Simulation) executeActionsLocal(c *Creature, actionLevels []float32, pending *pendingInstructions, temp float32) {
	if IsActionEnabled(REST) {
		level := actionLevels[REST]
		if math.Abs(float64(level)) > 0.75 {
			// Resting pays fraction of the basal metabolic rate: refund the base drain
			// already charged this tick, then re-charge the lower resting rate.
			rate := c.MetabolicRate(s.Params, temp)

			c.GainEnergy(rate, s.Params)
			c.DrainEnergy(rate * 0.1)
			c.IsResting = true

			c.LastAction = appendActionString(c.LastAction, "Resting")
			return
		}
	}
	c.IsResting = false

	if IsActionEnabled(SET_RESPONSIVENESS) {
		c.Responsiveness = (float32(math.Tanh(float64(actionLevels[SET_RESPONSIVENESS]))) + 1) / 2
	}

	responseAdjust := responseCurve(c.Responsiveness, s.Params.ResponseCurveKFactor)

	if IsActionEnabled(SET_OSCILLATOR_PERIOD) {
		periodf := actionLevels[SET_OSCILLATOR_PERIOD]
		newPeriodf := float32(math.Tanh(float64(periodf)+1) / 2)
		newPeriod := 1 + int(1.5+math.Exp(7*float64(newPeriodf)))
		if newPeriod >= 2 && newPeriod <= math.MaxUint8 {
			c.Clock = newPeriod
		}
	}

	if IsActionEnabled(ATTACK) {
		level := actionLevels[ATTACK]
		if math.Abs(float64(level)) > 0.8 {
			pending.attack = append(pending.attack, AttackInstruction{c})
			c.LastAction = appendActionString(c.LastAction, "Attacking")
		}
	}

	if IsActionEnabled(REWARD) {
		level := actionLevels[REWARD]
		if level > 0 {
			c.GainDopamine(float32(math.Tanh(float64(level))))
			c.LastAction = appendActionString(c.LastAction, "Rewarding")
		}
	}

	if IsActionEnabled(PUNISH) {
		level := actionLevels[PUNISH]
		if level > 0 {
			c.LoseDopamine(float32(math.Tanh(float64(level))))
			c.LastAction = appendActionString(c.LastAction, "Punishing")
		}
	}

	if IsActionEnabled(REPRODUCE) {
		level := actionLevels[REPRODUCE]
		if math.Abs(float64(level)) > 0.5 {
			reproThreshold := s.Params.ReproductionEnergyThreshold * c.MaxEnergy(s.Params)
			if c.Energy >= reproThreshold && c.Age >= c.cachedJuvenilePeriod && c.Mass >= float32(c.Genome.Mass)*0.9 {
				if c.Genome.ReproductionType == 0 {
					pending.reproduction = append(pending.reproduction, ReproductionInstruction{Creature: c})
					c.LastAction = appendActionString(c.LastAction, "Reproducing")
				} else {
					pending.mate = append(pending.mate, c)
					c.LastAction = appendActionString(c.LastAction, "Seeking mate")
				}
			}
		}
	}

	// Rotation: positive level turns CCW (left), negative turns CW (right).
	rotateAmount := float64(0)
	massNorm := c.CurrentMass(s.Params) / float32(s.Params.MaxMass)
	if IsActionEnabled(ROTATE) {
		turnInertia := 1.0 / (1.0 + (massNorm * 4.0))
		rotateAmount = math.Tanh(float64(actionLevels[ROTATE])) *
			float64(responseAdjust) *
			s.Params.MaxRotationPerStep * float64(turnInertia)
	}
	if rotateAmount != 0 {
		massCostMult := 0.5 + (massNorm * massNorm * 2.0)
		c.DrainEnergy(s.Params.MoveCost * float32(math.Abs(rotateAmount)) * 0.5 * massCostMult)
		c.LastAction = appendActionString(c.LastAction, "Rotating")
	}
	c.Heading = grid.NormalizeAngle(c.Heading + rotateAmount)

	// Forward/backward movement: positive = forward, negative = backward.
	moveAmount := float64(0)
	if IsActionEnabled(MOVE) {
		moveAmount = math.Tanh(float64(actionLevels[MOVE])) * float64(responseAdjust)
	}
	massFactor := 1.0 + math.Pow(float64(c.CurrentMass(s.Params)/float32(s.Params.MaxMass)), 2)*5.0
	moveAmount *= s.Params.MaxSpeedPerStep / massFactor

	// Colder temperatures slow movement (ectotherm-like muscle penalty).
	tempNorm := float64((temp - 10.0) / 30.0)
	if tempNorm < 0 {
		tempNorm = 0
	} else if tempNorm > 1 {
		tempNorm = 1
	}
	speedMult := float64(s.Params.ColdSpeedMultiplier) + (1.0-float64(s.Params.ColdSpeedMultiplier))*tempNorm
	moveAmount *= speedMult

	if math.Abs(moveAmount) < 0.001 {
		return
	}

	dx := math.Cos(c.Heading) * moveAmount
	dy := math.Sin(c.Heading) * moveAmount
	newPos := s.World.ClampToBounds(grid.Position{X: c.Loc.X + dx, Y: c.Loc.Y + dy})

	if !s.World.IsWall(newPos) {
		massCostMult := 0.5 + (massNorm * massNorm * 2.0)
		c.DrainEnergy(s.Params.MoveCost * float32(math.Abs(moveAmount)) * massCostMult)
		c.LastAction = appendActionString(c.LastAction, "Moving")
		pending.move = append(pending.move, MoveInstruction{c, newPos, moveAmount})
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
	matingRadiusSq := s.Params.MatingRadius * s.Params.MatingRadius

	for i, c := range candidates {
		if paired[c.Id] || !c.Alive {
			continue
		}

		bestIdx := -1
		var bestSimilarity float32 = -1.0
		bestDistSq := math.MaxFloat64

		for j, other := range candidates {
			if i == j || paired[other.Id] || !other.Alive {
				continue
			}

			dx := other.Loc.X - c.Loc.X
			dy := other.Loc.Y - c.Loc.Y
			d2 := dx*dx + dy*dy

			if d2 <= matingRadiusSq {
				similarity := GenomeSimilarity(c.Genome, other.Genome)
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
	pos := s.World.ClampToBounds(grid.Position{X: x, Y: y})
	if s.World.IsWall(pos) {
		return false
	}
	spawnParams := *s.Params

	id := s.World.AddCreature(pos)
	genome := MakeRandomGenome(&spawnParams)
	c := NewAdultCreature(id, pos, genome, s.Params)
	s.Population.Creatures[id] = c
	s.Population.AddAlive(id)

	return true
}

// SpawnClusterAt creates count identical creatures near the given world-space position.
// All creatures share the same randomly generated genome. Positions that are walls are skipped.
func (s *Simulation) SpawnClusterAt(x, y float64, count int) bool {
	spawnParams := *s.Params
	genome := MakeRandomGenome(&spawnParams)

	offsets := [][2]float64{{0, 0}, {4, 0}, {-4, 0}, {0, 4}, {0, -4}}
	spawned := 0
	for i := 0; spawned < count && i < len(offsets); i++ {
		pos := s.World.ClampToBounds(grid.Position{X: x + offsets[i][0], Y: y + offsets[i][1]})
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

func (s *Simulation) GridWidth() float64  { return s.Params.GridWidth }
func (s *Simulation) GridHeight() float64 { return s.Params.GridHeight }

func (s *Simulation) PopulationCount() int { return s.Population.AliveCount() }

func (s *Simulation) FoodCount() int { return s.World.FoodCount() }

// TotalEnergy returns the total liquid energy in the system: food energy plus the
// immediate metabolic stores (energy + stomach contents) of all living creatures.
func (s *Simulation) TotalEnergy() float64 {
	energy := s.World.TotalFoodMass()
	for _, id := range s.Population.aliveIDs {
		c := s.Population.Creatures[id]
		energy += float64(c.Energy) + float64(c.Stomach)
	}
	return energy
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

func prob2Bool(val float64) int {
	if rand.Float64() < val {
		return 1
	}
	return 0
}

func responseCurve(resp float32, kFactor float32) float32 {
	k := float64(kFactor)
	return float32(math.Pow(float64(resp)-2.0, -2*k)) - float32(math.Pow(2.0, -2.0*k))*(1-resp)
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
	s.corpseCache = s.corpseCache[:0]

	for id, c := range s.Population.Creatures {
		if c.Alive {
			// Living Creature Logic
			r, g, b, a := c.Color.RGBA()
			s.displayCache = append(s.displayCache, CreatureView{
				ID: id, X: c.Loc.X, Y: c.Loc.Y, Heading: c.Heading,
				R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8),
				CurrentMass:      float64(c.Mass),
				SightDistance:    c.Genome.SightDistance,
				FieldOfView:      c.Genome.FieldOfView,
				ReproductionType: c.Genome.ReproductionType,
			})
		} else {
			// Corpse Logic
			energyFraction := float32(0.0)
			if max := c.MaxEnergy(s.Params); max > 0 {
				energyFraction = c.Energy / max
			}

			s.corpseCache = append(s.corpseCache, CorpseView{
				ID:             id,
				X:              c.Loc.X,
				Y:              c.Loc.Y,
				EnergyFraction: energyFraction,
				Mass:           c.Mass,
			})
		}
	}
}

func (s *Simulation) updateFoodCache() {
	// Reuse the existing slice capacity
	s.foodCache = s.foodCache[:0]

	// Use our new iterator to pull only active food
	s.World.ForEachActiveFood(func(id int, x, y float64, mass float32) {
		s.foodCache = append(s.foodCache, FoodView{
			ID: id,
			X:  x,
			Y:  y,
			// Include mass if your FoodView supports it, otherwise remove
			Mass: mass,
		})
	})
}

type StateSnapshot struct {
	Creatures []CreatureView
	Food      []FoodView
	Corpses   []CorpseView
}

func (s *Simulation) GetSnapshot() StateSnapshot {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	// We return a copy of the slices so the UI can iterate safely
	return StateSnapshot{
		Creatures: append([]CreatureView(nil), s.displayCache...),
		Food:      append([]FoodView(nil), s.foodCache...),
		Corpses:   append([]CorpseView(nil), s.corpseCache...),
	}
}
