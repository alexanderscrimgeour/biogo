package simulation

import (
	"biogo/v2/grid"
	"biogo/v2/utils"
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
}

func New(params *Parameters) *Simulation {
	sim := &Simulation{
		Params:         params,
		nextCreatureID: grid.StartingCreatureID,
	}
	sim.initializeWorld()
	sim.initializePopulation()
	return sim
}

func (s *Simulation) initializeWorld() {
	s.World = grid.NewWorld(float64(s.Params.GridWidth), float64(s.Params.GridHeight), 1)
	s.World.SpawnFood(s.Params.MaxFood, s.Params.FoodPatchRadius, s.Params.FoodPatchSize)
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
	if s.Tick%s.Params.FoodSpawnInterval == 0 {
		toSpawn := s.Params.FoodPerSpawn
		if available := s.Params.MaxFood - s.World.FoodCount(); available < toSpawn {
			toSpawn = available
		}
		if toSpawn > 0 {
			s.World.SpawnFood(toSpawn, s.Params.FoodPatchRadius, s.Params.FoodPatchSize)
		}
	}

	// Collect alive IDs before spawning goroutines so the map is not modified
	// while goroutines are reading it.
	ids := s.Population.AliveIDs()
	n := runtime.GOMAXPROCS(0)
	batches := partitionIDs(ids, n)

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
		s.Population.EatQueue = append(s.Population.EatQueue, results[i].eat...)
		s.Population.ReproductionQueue = append(s.Population.ReproductionQueue, results[i].reproduction...)
	}

	s.Population.ProcessMoveQueue(s.World, s.Params)
	s.Population.ProcessEatQueue(s.World, s.Params)
	s.Population.ProcessDeathQueue(s.World, s.Params)
	s.Population.ProcessCorpseDecay(s.World, s.Params)
	s.Population.ProcessReproductionQueue(s.World, s.Params, s.allocateID)

	spawnParams := *s.Params
	if s.Params.SpawnMutationRate > s.Params.MinMutationRate {
		spawnParams.MinMutationRate = s.Params.SpawnMutationRate
	}
	for s.Population.AliveCount() < s.Params.MinPopulation {
		loc, ok := s.World.FindEmptyLocation()
		if !ok {
			break
		}
		id := s.allocateID()
		var genome *Genome
		if source := s.Population.OldestGenome(); source != nil {
			genome = AsexualReproduction(source, &spawnParams)
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
	c.Energy -= c.MetabolicRate(s.Params)
	if c.Energy <= 0 || c.Age > c.MaxAge(s.Params) {
		pending.death = append(pending.death, DeathInstruction{c})
		return
	}

	juvenilePeriod := s.Params.MinJuvenilePeriod + int(float32(c.Genome.JuvenilePeriod)/255.0*float32(s.Params.MaxJuvenilePeriod-s.Params.MinJuvenilePeriod))
	reproThreshold := s.Params.ReproductionEnergyThreshold * float32(c.Genome.MaxEnergy)
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
	if IsActionEnabled(DO_NOTHING) {
		level := actionLevels[DO_NOTHING]
		if level > 0 && prob2Bool(float64(level)) == 1 {
			c.Energy += c.MetabolicRate(s.Params)
			c.LastAction = appendActionString(c.LastAction, "Resting")
			return
		}
	}

	if IsActionEnabled(SET_RESPONSIVENESS) {
		resp := actionLevels[SET_RESPONSIVENESS]
		resp = (float32(math.Tanh(float64(resp/float32(utils.ClampByteAsFloat32(0, 1, c.Genome.Responsiveness))))) + 1) / 2
		c.Responsiveness = resp
	}

	if IsActionEnabled(EAT) {
		level := actionLevels[EAT]
		if level > 0 && prob2Bool(float64(level)) == 1 {
			if targetID := findNearestInFOV(c, s.World, s.Params.PredationRadius); targetID != -1 {
				pending.eat = append(pending.eat, EatInstruction{c, targetID})
				c.LastAction = appendActionString(c.LastAction, "Eating")
			}
		}
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

	// Rotation: ROTATE_LEFT turns CCW, ROTATE_RIGHT turns CW.
	rotateLeft := float64(0)
	rotateRight := float64(0)
	if IsActionEnabled(ROTATE_LEFT) {
		rotateLeft = float64(actionLevels[ROTATE_LEFT])
	}
	if IsActionEnabled(ROTATE_RIGHT) {
		rotateRight = float64(actionLevels[ROTATE_RIGHT])
	}
	rotateAmount := math.Tanh(rotateLeft-rotateRight) * float64(responseAdjust) * s.Params.MaxRotationPerStep
	if rotateAmount != 0 {
		rotationCostFactor := 0.5
		c.Energy -= s.Params.MoveCost * float32(math.Abs(rotateAmount)) * float32(rotationCostFactor)
		c.LastAction = appendActionString(c.LastAction, "Rotating")
	}
	c.Heading = grid.NormalizeAngle(c.Heading + rotateAmount)

	// Forward/backward movement.
	fwd := float64(0)
	bwd := float64(0)
	if IsActionEnabled(MOVE_FORWARD) {
		fwd = float64(actionLevels[MOVE_FORWARD])
	}
	if IsActionEnabled(MOVE_BACKWARD) {
		bwd = float64(actionLevels[MOVE_BACKWARD])
	}
	if IsActionEnabled(MOVE_RANDOM) {
		level := float64(actionLevels[MOVE_RANDOM])
		c.Heading = grid.NormalizeAngle(c.Heading + (rand.Float64()*2-1)*s.Params.MaxRotationPerStep)
		fwd += level
	}

	moveAmount := math.Tanh(fwd-bwd) * float64(responseAdjust)
	massFactor := 1.0 + float64(c.CurrentMass(s.Params))/255.0
	moveAmount *= s.Params.MaxSpeedPerStep / massFactor

	if math.Abs(moveAmount) < 0.001 {
		return
	}

	dx := math.Cos(c.Heading) * moveAmount
	dy := math.Sin(c.Heading) * moveAmount
	newPos := s.World.ClampToBounds(grid.Position{X: c.Loc.X + dx, Y: c.Loc.Y + dy})

	if !s.World.IsWall(newPos) {
		c.Energy -= s.Params.MoveCost * float32(moveAmount)
		c.LastAction = appendActionString(c.LastAction, "Moving")
		pending.move = append(pending.move, MoveInstruction{c, newPos})
	}
}

// SetSpawnMutationRate sets the minimum mutation rate applied when artificially
// spawning creatures to maintain the minimum population.
func (s *Simulation) SetSpawnMutationRate(rate float32) {
	s.Params.SpawnMutationRate = rate
}

func (s *Simulation) GridWidth() int  { return s.Params.GridWidth }
func (s *Simulation) GridHeight() int { return s.Params.GridHeight }

func (s *Simulation) PopulationCount() int { return s.Population.AliveCount() }

func (s *Simulation) FoodCount() int { return s.World.FoodCount() }

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

// findNearestInFOV returns the ID of the closest creature within the FOV cone
// and the given radius, or -1 if none is found.
func findNearestInFOV(c *Creature, w *grid.World, radius float64) int {
	halfFOVCos := math.Cos(float64(c.Genome.FieldOfView) / 2.0 * math.Pi / 180.0)
	fwdX, fwdY := grid.HeadingToVec(c.Heading)

	bestID := -1
	bestDist := math.MaxFloat64

	for _, id := range w.GetCreaturesInRadius(c.Loc, radius) {
		if id == c.Id {
			continue
		}
		pos, _ := w.GetCreaturePos(id)
		dx := pos.X - c.Loc.X
		dy := pos.Y - c.Loc.Y
		d := math.Sqrt(dx*dx + dy*dy)
		if d == 0 || d >= radius {
			continue
		}
		if grid.CosSimilarity(fwdX, fwdY, dx, dy) >= halfFOVCos && d < bestDist {
			bestDist = d
			bestID = id
		}
	}
	return bestID
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
