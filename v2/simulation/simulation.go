package simulation

import (
	"biogo/v2/grid"
	"biogo/v2/utils"
	"fmt"
	"math"
	"math/rand"
)

type Simulation struct {
	Grid           *grid.Grid
	Population     *Population
	Tick           int
	nextCreatureID int
	Params         *Parameters
}

func New(params *Parameters) *Simulation {
	sim := &Simulation{
		Params:         params,
		nextCreatureID: grid.RESERVED_CELL_TYPES,
	}
	sim.initializeGrid()
	sim.initializePopulation()
	return sim
}

func (s *Simulation) initializeGrid() {
	s.Grid = grid.NewGrid(s.Params.GridWidth, s.Params.GridHeight, 1)
	s.Grid.SpawnFood(s.Params.MaxFood)
}

func (s *Simulation) initializePopulation() {
<<<<<<< Updated upstream
=======
	savedGenomes, _ := LoadAllCreatureGenomes()
	maxSeeded := int(float32(s.Params.StartingPopulation) * s.Params.SavedGenomeProportion)
	seeded := 0

>>>>>>> Stashed changes
	pop := NewPopulation(s.Params)
	for i := 0; i < s.Params.StartingPopulation; i++ {
		loc, ok := s.Grid.FindEmptyLocation()
		if !ok {
			break
		}
		id := s.allocateID()
		pop.Creatures[id] = NewCreature(id, loc, MakeRandomGenome(s.Params))
		s.Grid.Set(loc, id)
	}
	s.Population = pop
}

<<<<<<< Updated upstream
=======
// SaveCreature saves the genome of the creature with the given id to a unique file in data/creatures/.
func (s *Simulation) SaveCreature(id int) error {
	c, ok := s.Population.Creatures[id]
	if !ok || !c.Alive {
		return nil
	}
	return SaveCreatureToFile(c.Genome)
}

// Reset reinitialises the simulation from scratch. A proportion of the starting
// population is seeded from any previously saved genomes (see SavedGenomeProportion).
func (s *Simulation) Reset() {
	s.Tick = 0
	s.nextCreatureID = grid.RESERVED_CELL_TYPES
	s.initializeGrid()
	s.initializePopulation()
}

>>>>>>> Stashed changes
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
		if available := s.Params.MaxFood - len(s.Grid.FoodLocations); available < toSpawn {
			toSpawn = available
		}
		if toSpawn > 0 {
			s.Grid.SpawnFood(toSpawn)
		}
	}

	for _, creature := range s.Population.Creatures {
		if creature.Alive {
			s.stepCreature(creature)
		}
	}

	s.Population.ProcessMoveQueue(s.Grid, s.Params)
	s.Population.ProcessDeathQueue(s.Grid, s.Params)
	s.Population.ProcessCorpseDecay(s.Grid, s.Params)
	s.Population.ProcessReproductionQueue(s.Grid, s.Params, s.allocateID)

	spawnParams := *s.Params
	if s.Params.SpawnMutationRate > s.Params.MinMutationRate {
		spawnParams.MinMutationRate = s.Params.SpawnMutationRate
	}
	for s.Population.AliveCount() < s.Params.MinPopulation {
		loc, ok := s.Grid.FindEmptyLocation()
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
		s.Grid.Set(loc, id)
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

	actionLevels := c.FeedForward(s.Grid, s.Population, s.Tick, s.Params)
	s.executeActions(c, actionLevels)
}

func (s *Simulation) Print() {
	s.Grid.Print()
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
		responsivenessLevel := actionLevels[SET_RESPONSIVENESS]
		responsivenessLevel = (float32(math.Tanh(float64(responsivenessLevel/float32(utils.ClampByteAsFloat32(0, 1, c.Genome.Responsiveness))))) + 1) / 2
		c.Responsiveness = responsivenessLevel
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

	moveX := float32(0)
	moveY := float32(0)
	if IsActionEnabled(MOVE_X) {
		moveX = actionLevels[MOVE_X]
	}
	if IsActionEnabled(MOVE_Y) {
		moveY = actionLevels[MOVE_Y]
	}
	if IsActionEnabled(MOVE_EAST) {
		moveX += actionLevels[MOVE_EAST]
	}
	if IsActionEnabled(MOVE_WEST) {
		moveX -= actionLevels[MOVE_WEST]
	}
	if IsActionEnabled(MOVE_NORTH) {
		moveY += actionLevels[MOVE_NORTH]
	}
	if IsActionEnabled(MOVE_SOUTH) {
		moveY += actionLevels[MOVE_SOUTH]
	}
	if IsActionEnabled(MOVE_FWD) {
		level := actionLevels[MOVE_FWD]
		moveX += float32(c.LastMoveDir.X) * level
		moveY += float32(c.LastMoveDir.Y) * level
	}
	if IsActionEnabled(MOVE_LEFT) {
		level := actionLevels[MOVE_LEFT]
		offset := c.LastMoveDir.Rotate90CCW()
		moveX += float32(offset.X) * level
		moveY += float32(offset.Y) * level
	}
	if IsActionEnabled(MOVE_RIGHT) {
		level := actionLevels[MOVE_RIGHT]
		offset := c.LastMoveDir.Rotate90CW()
		moveX += float32(offset.X) * level
		moveY += float32(offset.Y) * level
	}
	if IsActionEnabled(MOVE_RL) {
		level := actionLevels[MOVE_RL]
		offset := grid.CENTER
		if level < 0 {
			offset = c.LastMoveDir.Rotate90CCW()
		} else if level > 0 {
			offset = c.LastMoveDir.Rotate90CW()
		}
		moveX += float32(offset.X) * level
		moveY += float32(offset.Y) * level
	}
	if IsActionEnabled(MOVE_RANDOM) {
		level := actionLevels[MOVE_RANDOM]
		offset := grid.RandomDir()
		moveX += float32(offset.X) * level
		moveY += float32(offset.Y) * level
	}

<<<<<<< Updated upstream
=======
	if IsActionEnabled(EAT) {
		level := actionLevels[EAT]
		if level > 0 && prob2Bool(float64(level)) == 1 {
			fwdLoc := grid.Coord{
				X: c.Loc.X + c.LastMoveDir.X,
				Y: c.Loc.Y + c.LastMoveDir.Y,
			}
			if s.Grid.IsInBounds(fwdLoc) {
				s.Population.QueueForEat(c, fwdLoc)
				if c.LastAction != "Reproducing" {
					c.LastAction = "Eating"
				}
			}
		}
	}

>>>>>>> Stashed changes
	moveX = float32(math.Tanh(float64(moveX)))
	moveY = float32(math.Tanh(float64(moveY)))
	moveX *= responseAdjust
	moveY *= responseAdjust

	moveXSign := 1
	if moveX < 0 {
		moveXSign = -1
	}
	moveYSign := 1
	if moveY < 0 {
		moveYSign = -1
	}

	moveXBool := prob2Bool(math.Abs(float64(moveX)))
	moveYBool := prob2Bool(math.Abs(float64(moveY)))
	movementOffset := grid.Dir{X: moveXBool * moveXSign, Y: moveYBool * moveYSign}
	newCoord := c.GetNextLoc(movementOffset)

	if s.Grid.Torodial {
		newCoord = s.Grid.WrapCoords(newCoord)
	}
	if (s.Grid.Torodial || s.Grid.IsInBounds(newCoord)) && s.Grid.At(newCoord) != grid.WALL {
		massFactor := 1.0 + c.CurrentMass(s.Params)/255.0
		c.Energy -= s.Params.MoveCost * massFactor
		s.Population.QueueForMove(c, newCoord)
	}
}

// SetSpawnMutationRate sets the minimum mutation rate applied when artificially
// spawning creatures to maintain the minimum population.
func (s *Simulation) SetSpawnMutationRate(rate float32) {
	s.Params.SpawnMutationRate = rate
}

// GridWidth returns the simulation grid width.
func (s *Simulation) GridWidth() int {
	return s.Grid.SizeX()
}

// GridHeight returns the simulation grid height.
func (s *Simulation) GridHeight() int {
	return s.Grid.SizeY()
}

// PopulationCount returns the current number of living creatures (excludes corpses).
func (s *Simulation) PopulationCount() int {
	return s.Population.AliveCount()
}

// FoodCount returns the current number of food items on the grid.
func (s *Simulation) FoodCount() int {
	return len(s.Grid.FoodLocations)
}

// AverageAge returns the mean age of all living creatures, or 0 if none.
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
