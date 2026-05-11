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
	World          *grid.World
	Population     *Population
	Tick           int
	nextCreatureID int
	Params         *Parameters
	TargetEnergy   float64 // total liquid energy to maintain (set at initialisation)
}

func New(params *Parameters) *Simulation {
	sim := &Simulation{
		Params:         params,
		nextCreatureID: grid.StartingCreatureID,
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

	for i := 0; i < s.Params.StartingPopulation; i++ {
		loc, ok := s.World.FindEmptyLocation()
		if !ok {
			break
		}

		id := s.allocateID()
		var genome *Genome

		// If we have saved genomes and haven't hit the seeding limit,
		// cycle through savedGenomes to ensure equal distribution.
		if numSaved > 0 && i < maxSeeded {
			genome = savedGenomes[i%numSaved]
		} else {
			genome = MakeRandomGenome(s.Params)
		}

		c := NewAdultCreature(id, loc, genome, s.Params)
		pop.Creatures[id] = c
		s.World.AddCreature(id, loc)
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
	s.nextCreatureID = grid.StartingCreatureID
	s.initializeWorld()
	s.initializePopulation()
	s.TargetEnergy = s.TotalEnergy()
}

func (s *Simulation) allocateID() int {
	id := s.nextCreatureID
	s.nextCreatureID++
	return id
}

func (s *Simulation) Update() {
	s.step()
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

	// Collect alive IDs before spawning goroutines so the map is not modified
	// while goroutines are reading it.
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
	for i := range results {
		s.Population.DeathQueue = append(s.Population.DeathQueue, results[i].death...)
		s.Population.MoveQueue = append(s.Population.MoveQueue, results[i].move...)
		s.Population.ReproductionQueue = append(s.Population.ReproductionQueue, results[i].reproduction...)
	}

	s.Population.ProcessMoveQueue(s.World, s.Params)
	s.Population.ProcessDeathQueue(s.World, s.Params)
	s.Population.ProcessCorpseDecay(s.World, s.Params)
	s.Population.ProcessReproductionQueue(s.World, s.Params, s.allocateID)

	// Reward decay
	for _, c := range s.Population.Creatures {
		if !c.Alive {
			continue
		}

		// Decay the dopamine signal
		c.Dopamine *= 0.6

		if c.Dopamine < 0.01 {
			c.Dopamine = 0
		}
		c.LastTickEnergy = c.Energy
		c.LastStomach = c.Stomach
	}

	spawnParams := *s.Params

	for s.Population.AliveCount() < s.Params.MinPopulation {
		loc, ok := s.World.FindEmptyLocation()
		if !ok {
			break
		}
		id := s.allocateID()
		var genome *Genome
		if source := s.Population.OldestGenome(); source != nil {
			genome = ArtificialReproduction(source, &spawnParams)
		} else {
			genome = MakeRandomGenome(&spawnParams)
		}
		c := NewAdultCreature(id, loc, genome, s.Params)
		s.Population.Creatures[id] = c
		s.World.AddCreature(id, loc)
	}

	s.Tick++
}

func (s *Simulation) stepCreatureLocal(c *Creature, pending *pendingInstructions) {
	c.Age++
	c.GrowMass(s.Params)
	c.LastAction = ""
	c.DrainEnergy(c.MetabolicRate(s.Params))
	c.Digest(s.Params)
	if c.Energy <= 0 || c.Age > c.MaxAge(s.Params) {
		pending.death = append(pending.death, DeathInstruction{c})
		return
	}

	juvenilePeriod := c.cachedJuvenilePeriod
	reproThreshold := s.Params.ReproductionEnergyThreshold * c.MaxEnergy(s.Params)
	if c.Energy >= reproThreshold && c.Age >= juvenilePeriod && c.Mass >= float32(c.Genome.Mass) {
		pending.reproduction = append(pending.reproduction, ReproductionInstruction{c})
		c.LastAction = appendActionString(c.LastAction, "Reproducing")
	}

	actionLevels := c.FeedForward(s.World, s.Population, s.Tick, s.Params)
	s.executeActionsLocal(c, actionLevels, pending)
}

func (s *Simulation) Print() {
	fmt.Printf("Population Size: %d", len(s.Population.Creatures))
}

func (s *Simulation) executeActionsLocal(c *Creature, actionLevels []float32, pending *pendingInstructions) {
	if IsActionEnabled(REST) {
		level := actionLevels[REST]
		if math.Abs(float64(level)) > 0.75 {
			// Resting pays fraction of the basal metabolic rate: refund the base drain
			// already charged this tick, then re-charge the lower resting rate.
			rate := c.MetabolicRate(s.Params)

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

	// Rotation: positive level turns CCW (left), negative turns CW (right).
	rotateAmount := float64(0)
	if IsActionEnabled(ROTATE) {
		rotateAmount = math.Tanh(float64(actionLevels[ROTATE])) * float64(responseAdjust) * s.Params.MaxRotationPerStep
	}
	if rotateAmount != 0 {
		massNorm := c.CurrentMass(s.Params) / float32(s.Params.MaxMass)
		massCostMult := 0.5 + massNorm
		c.DrainEnergy(s.Params.MoveCost * float32(math.Abs(rotateAmount)) * 0.5 * massCostMult)
		c.LastAction = appendActionString(c.LastAction, "Rotating")
	}
	c.Heading = grid.NormalizeAngle(c.Heading + rotateAmount)

	// Forward/backward movement: positive = forward, negative = backward.
	moveAmount := float64(0)
	if IsActionEnabled(MOVE) {
		moveAmount = math.Tanh(float64(actionLevels[MOVE])) * float64(responseAdjust)
	}
	massFactor := 1.0 + float64(c.CurrentMass(s.Params))/255.0
	moveAmount *= s.Params.MaxSpeedPerStep / massFactor

	if math.Abs(moveAmount) < 0.001 {
		return
	}

	dx := math.Cos(c.Heading) * moveAmount
	dy := math.Sin(c.Heading) * moveAmount
	newPos := s.World.ClampToBounds(grid.Position{X: c.Loc.X + dx, Y: c.Loc.Y + dy})

	if !s.World.IsWall(newPos) {
		massNorm := c.CurrentMass(s.Params) / float32(s.Params.MaxMass)
		massCostMult := 0.5 + massNorm // heavier creatures pay more energy per unit distance
		c.DrainEnergy(s.Params.MoveCost * float32(math.Abs(moveAmount)) * massCostMult)
		c.LastAction = appendActionString(c.LastAction, "Moving")
		pending.move = append(pending.move, MoveInstruction{c, newPos, moveAmount})
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

	id := s.allocateID()
	genome := MakeRandomGenome(&spawnParams)
	c := NewAdultCreature(id, pos, genome, s.Params)
	s.Population.Creatures[id] = c
	s.World.AddCreature(id, pos)

	return true
}

// SpawnGenome places a new adult creature with the given genome at a random empty location.
func (s *Simulation) SpawnGenome(g *Genome) bool {
	loc, ok := s.World.FindEmptyLocation()
	if !ok {
		return false
	}
	id := s.allocateID()
	c := NewAdultCreature(id, loc, g, s.Params)
	s.Population.Creatures[id] = c
	s.World.AddCreature(id, loc)
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
// Body mass is treated as structural and excluded; corpse mass is excluded too since
// it is not reliably converted back to energy.
func (s *Simulation) TotalEnergy() float64 {
	energy := s.World.TotalFoodMass()
	for _, c := range s.Population.Creatures {
		if c.Alive {
			energy += float64(c.Energy) + float64(c.Stomach)
		}
	}
	return energy
}

func (s *Simulation) AverageAge() float64 {
	total := 0
	count := 0
	for _, c := range s.Population.Creatures {
		if c.Alive {
			total += c.Age
			count++
		}
	}
	if count == 0 {
		return 0
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
	} else {
		return fmt.Sprintf("%s | %s", base, new)
	}
}
