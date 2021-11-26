package simulation

import (
	"fmt"
	"gopop/v2/grid"
	"gopop/v2/utils"
	"math"
	"math/rand"
)

type Simulation struct {
	Grid             *grid.Grid
	Population       *Population
	Tick             int
	Generation       int // Might be useless?
	GeneticDiversity float32
	Challenge        ChallengeType
}

func New() *Simulation {
	sim := Simulation{
		Challenge: Params.Challenge,
	}
	sim.InitializeGrid()
	sim.InitializeFirstGeneration()
	return &sim
}

func (s *Simulation) InitializeGrid() {
	s.Grid = grid.NewGrid(Params.GridWidth, Params.GridHeight, 0)
}

func (s *Simulation) InitializeFirstGeneration() {
	pop := NewPopulation()
	for i := grid.RESERVED_CELL_TYPES; i < Params.StartingPopulation+grid.RESERVED_CELL_TYPES; i++ {
		loc := s.Grid.FindEmptyLocation()
		pop.Creatures[i-grid.RESERVED_CELL_TYPES] = NewCreature(i, loc, MakeRandomGenome())
		s.Grid.Set(loc, i)
	}
	s.Population = pop
}

func (s *Simulation) Update() {
	if s.Tick < Params.MaxAge {
		s.Step()
	} else {
		s.InitializeNewGeneration()
	}
	if s.Generation >= Params.MaxGenerations {
		panic("Simulation ended")
	}
}

func (s *Simulation) InitializeNewGeneration() {
	// s.GeneticDiversity = s.Population.GeneticDiversity()
	s.Generation += 1
	s.Tick = 0
	childrenGenomes := []*Genome{}
	for _, creature := range s.Population.Creatures {
		if PassedSurvivalCriteria(creature, s) {
			newGenome := AsexualReproduction(creature.Genome)
			childrenGenomes = append(childrenGenomes, newGenome)
		}
	}

	if len(childrenGenomes) == 0 {
		panic("The creatures have gone extinct.")
	}
	survivalPercentage := float64(len(childrenGenomes)) / float64(len(s.Population.Creatures)) * 100
	fmt.Printf("Generation: %d\t%.2f%% Survived\n", s.Generation, survivalPercentage)

	children := []*Creature{}
	for i := grid.RESERVED_CELL_TYPES; i < Params.MaxPopulation+grid.RESERVED_CELL_TYPES; i++ {
		loc := s.Grid.FindEmptyLocation()
		child := NewCreature(i-grid.RESERVED_CELL_TYPES, loc, childrenGenomes[(i-grid.RESERVED_CELL_TYPES)%len(childrenGenomes)])
		children = append(children, child)
		s.Grid.Set(loc, i)
	}

	s.Population = &Population{
		Creatures:  children,
		DeathQueue: []DeathInstruction{},
		MoveQueue:  []MoveInstruction{},
	}
	s.Grid.ZeroFill()
	s.Grid.CreateWall()
}

func (s *Simulation) Step() {
	for _, creature := range s.Population.Creatures {
		if creature.Alive {
			s.StepCreature(creature)
		}
	}
	s.Population.ProcessMoveQueue(s.Grid)
	// TODO()
	// s.Population.ProcessReproductionQueue(s.Grid)
	// s.Population.ProcessDeathQueue()
	s.Tick++
}

func (s *Simulation) StepCreature(c *Creature) {
	c.Age++
	actionLevels := c.FeedForward(s.Grid, s.Population, s.Tick)
	s.ExecuteActions(c, actionLevels)
}

func (s *Simulation) Print() {
	s.Grid.Print()
	fmt.Printf("Population Size: %d", len(s.Population.Creatures))
}

func (s *Simulation) ExecuteActions(c *Creature, actionLevels []float32) {

	if IsActionEnabled(SET_RESPONSIVENESS) {
		responsivenessLevel := actionLevels[SET_RESPONSIVENESS]
		responsivenessLevel = (float32(math.Tanh(float64(responsivenessLevel/float32(utils.ClampByteAsFloat32(0, 1, c.Genome.Responsiveness))))) + 1) / 2
		c.Responsiveness = responsivenessLevel
	}

	// Adjust action levels based on responsiveness
	responseAdjust := responseCurve(c.Responsiveness)

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
		moveX = actionLevels[MOVE_Y]
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
		} else if level < 0 {
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

	moveX = float32(math.Tanh(float64(moveX)))
	moveY = float32(math.Tanh(float64(moveY)))
	moveX *= responseAdjust
	moveY *= responseAdjust
	moveXSign := 1
	moveYSign := 1
	if moveX < 0 {
		moveXSign = -1
	} else {
		moveXSign = 1
	}
	if moveY < 0 {
		moveYSign = -1
	} else {
		moveYSign = 1
	}

	moveXBool := prob2Bool(math.Abs(float64(moveX)))
	moveYBool := prob2Bool(math.Abs(float64(moveY)))
	movementOffset := grid.Dir{X: moveXBool * moveXSign, Y: moveYBool * moveYSign}
	// Move if it's a valid location
	newCoord := c.GetNextLoc(movementOffset)
	if s.Grid.IsInBounds(newCoord) && s.Grid.IsEmptyAt(newCoord) {
		s.Population.QueueForMove(c, newCoord)
	}
}

// Range in 0...1
func prob2Bool(val float64) int {
	if rand.Float64() < val {
		return 1
	} else {
		return 0
	}
}

func responseCurve(resp float32) float32 {
	k := float64(Params.ResponseCurveKFactor)
	return float32(math.Pow(float64(resp)-2.0, -2*k)) - float32(math.Pow(2.0, -2.0*k))*(1-resp)
}
