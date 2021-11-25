package simulation

import "gopop/v2/grid"

type Simulation struct {
	Grid       *grid.Grid
	Population *Population
	step       int
	Generation int // Might be useless?
	Challenge  ChallengeType
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
	s.Grid = grid.NewGrid(Params.GridWidth, Params.GridHeight)
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

func (s *Simulation) Step() {
	for _, creature := range s.Population.Creatures {
		if creature.Alive {
			go s.StepCreature(creature)
		}
	}
	s.Population.ProcessMoveQueue(s.Grid)
	// TODO()
	// s.Population.ProcessReproductionQueue(s.Grid)
	// s.Population.ProcessDeathQueue()
	s.step++
}

func (s *Simulation) StepCreature(c *Creature) {
	c.Age += 1
	actionLevels := c.FeedForward(s.Grid, s.Population, s.step)
	s.ExecuteActions(c, actionLevels)
}
