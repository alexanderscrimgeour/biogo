package test

import (
	"biogo/v2/grid"
	"biogo/v2/simulation"
	"testing"
)

func TestCreateNeuralNetworkFromGenome(t *testing.T) {
	p := defaultParams()
	genome := simulation.MakeRandomGenome(p)
	nnet := simulation.CreateNeuralNetworkFromGenome(genome.Brain, genome.NeuronCount)
	if nnet == nil {
		t.Fatal("CreateNeuralNetworkFromGenome returned nil")
	}
}

func TestNeuralNetworkEdgesAreValid(t *testing.T) {
	p := defaultParams()
	genome := simulation.MakeRandomGenome(p)
	nnet := simulation.CreateNeuralNetworkFromGenome(genome.Brain, genome.NeuronCount)

	for _, edge := range nnet.Edges {
		if edge.SinkType == simulation.ACTION && edge.SinkID >= simulation.ACTION_COUNT {
			t.Errorf("edge sinks to invalid action ID %d", edge.SinkID)
		}
		if edge.SourceType == simulation.SENSOR && edge.SourceID >= simulation.SENSOR_COUNT {
			t.Errorf("edge sources from invalid sensor ID %d", edge.SourceID)
		}
	}
}

func TestCreateInitialNeuronOutput(t *testing.T) {
	out := simulation.CreateInitialNeuronOutput()
	if out != 0.5 {
		t.Errorf("initial neuron output should be 0.5, got %f", out)
	}
}

func TestNewCreature(t *testing.T) {
	p := defaultParams()
	genome := simulation.MakeRandomGenome(p)
	loc := grid.Coord{X: 5, Y: 5}
	c := simulation.NewCreature(grid.RESERVED_CELL_TYPES, loc, genome)
	if c == nil {
		t.Fatal("NewCreature returned nil")
	}
	if !c.Alive {
		t.Error("new creature should be alive")
	}
	if c.Age != 0 {
		t.Errorf("new creature age should be 0, got %d", c.Age)
	}
	if c.Loc != loc {
		t.Errorf("creature Loc = %v, want %v", c.Loc, loc)
	}
}

func TestGetNextLoc(t *testing.T) {
	p := defaultParams()
	genome := simulation.MakeRandomGenome(p)
	loc := grid.Coord{X: 5, Y: 5}
	c := simulation.NewCreature(grid.RESERVED_CELL_TYPES, loc, genome)

	next := c.GetNextLoc(grid.E)
	if next.X != 6 || next.Y != 5 {
		t.Errorf("GetNextLoc(E) = %v, want {6,5}", next)
	}

	next = c.GetNextLoc(grid.N)
	if next.X != 5 || next.Y != 6 {
		t.Errorf("GetNextLoc(N) = %v, want {5,6}", next)
	}
}
