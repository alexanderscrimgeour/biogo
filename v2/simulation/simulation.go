package simulation

import (
	"biogo/v2/grid"
	"biogo/v2/utils"
	"fmt"
	"math"
	"math/rand"
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
	s.World.SpawnFood(s.Params.MaxFood)
}

func (s *Simulation) initializePopulation() {
	pop := NewPopulation(s.Params)
	for i := 0; i < s.Params.StartingPopulation; i++ {
		loc, ok := s.World.FindEmptyLocation()
		if !ok {
			break
		}
		id := s.allocateID()
		c := NewCreature(id, loc, MakeRandomGenome(s.Params))
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
			s.World.SpawnFood(toSpawn)
		}
	}

	for _, creature := range s.Population.Creatures {
		if creature.Alive {
			s.stepCreature(creature)
		}
	}

	s.Population.ProcessMoveQueue(s.World, s.Params)
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
		c := NewCreature(id, loc, genome)
		s.Population.Creatures[id] = c
		s.World.AddCreature(id, loc)
	}

	s.Tick++
}

func (s *Simulation) stepCreature(c *Creature) {
	c.Age++
	c.LastAction = "Idle"
	c.Energy -= c.MetabolicRate(s.Params)
	if c.Energy <= 0 || c.Age > c.MaxAge(s.Params) {
		s.Population.QueueForDeath(c)
		return
	}

	juvenilePeriod := s.Params.MinJuvenilePeriod + int(float32(c.Genome.JuvenilePeriod)/255.0*float32(s.Params.MaxJuvenilePeriod-s.Params.MinJuvenilePeriod))
	reproThreshold := s.Params.ReproductionEnergyThreshold * float32(c.Genome.MaxEnergy)
	if c.Energy >= reproThreshold && c.Age >= juvenilePeriod {
		s.Population.QueueForReproduction(c)
		c.LastAction = "Reproducing"
	}

	actionLevels := c.FeedForward(s.World, s.Population, s.Tick, s.Params)
	s.executeActions(c, actionLevels)
}

func (s *Simulation) Print() {
	fmt.Printf("Population Size: %d", len(s.Population.Creatures))
}

func (s *Simulation) executeActions(c *Creature, actionLevels []float32) {
	if IsActionEnabled(DO_NOTHING) {
		level := actionLevels[DO_NOTHING]
		if level > 0 && prob2Bool(float64(level)) == 1 {
			c.Energy += c.MetabolicRate(s.Params)
			c.LastAction = "Resting"
			return
		}
	}

	if IsActionEnabled(SET_RESPONSIVENESS) {
		resp := actionLevels[SET_RESPONSIVENESS]
		resp = (float32(math.Tanh(float64(resp/float32(utils.ClampByteAsFloat32(0, 1, c.Genome.Responsiveness))))) + 1) / 2
		c.Responsiveness = resp
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
		c.Energy -= s.Params.MoveCost * float32(massFactor)
		s.Population.QueueForMove(c, newPos)
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

func responseCurve(resp float32, kFactor float32) float32 {
	k := float64(kFactor)
	return float32(math.Pow(float64(resp)-2.0, -2*k)) - float32(math.Pow(2.0, -2.0*k))*(1-resp)
}
