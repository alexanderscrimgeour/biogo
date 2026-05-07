package test

import (
	"biogo/v2/grid"
	"biogo/v2/simulation"
	"testing"
)

func TestCurrentMassAtBirth(t *testing.T) {
	params := defaultParams()
	genome := simulation.MakeRandomGenome(params)
<<<<<<< Updated upstream
	genome.Size = 200

	c := simulation.NewCreature(grid.RESERVED_CELL_TYPES, grid.Coord{}, genome)
	// age 0: size should be params.MinSize
	got := c.CurrentSize(params)
	want := float32(params.MinSize)
=======
	genome.Mass = 200
	genome.MinMass = 10

	c := simulation.NewCreature(grid.RESERVED_CELL_TYPES, grid.Coord{}, genome)
	got := c.CurrentMass(params)
	want := float32(genome.MinMass)
>>>>>>> Stashed changes
	if got != want {
		t.Errorf("CurrentMass at age 0: got %f, want %f", got, want)
	}
}

func TestCurrentMassAtAdulthood(t *testing.T) {
	params := defaultParams()
	genome := simulation.MakeRandomGenome(params)
	genome.Mass = 200
	genome.JuvenilePeriod = 0 // maps to MinJuvenilePeriod

	c := simulation.NewCreature(grid.RESERVED_CELL_TYPES, grid.Coord{}, genome)
	c.Age = params.MaxJuvenilePeriod + 1

	got := c.CurrentMass(params)
	want := float32(genome.Mass)
	if got != want {
		t.Errorf("CurrentMass at adulthood: got %f, want %f", got, want)
	}
}

func TestCurrentMassMidJuvenile(t *testing.T) {
	params := defaultParams()
	params.MinJuvenilePeriod = 100
	params.MaxJuvenilePeriod = 100 // fix juvenile period to exactly 100 ticks

	genome := simulation.MakeRandomGenome(params)
<<<<<<< Updated upstream
	genome.Size = 100
=======
	genome.Mass = 100
	genome.MinMass = 10
>>>>>>> Stashed changes
	genome.JuvenilePeriod = 0 // maps to MinJuvenilePeriod (100)

	c := simulation.NewCreature(grid.RESERVED_CELL_TYPES, grid.Coord{}, genome)
	c.Age = 50 // halfway through juvenile period

<<<<<<< Updated upstream
	got := c.CurrentSize(params)
	// expect midpoint between MinSize and genome.Size
	want := float32(params.MinSize) + (float32(genome.Size)-float32(params.MinSize))*0.5
=======
	got := c.CurrentMass(params)
	want := float32(genome.MinMass) + (float32(genome.Mass)-float32(genome.MinMass))*0.5
>>>>>>> Stashed changes
	if got != want {
		t.Errorf("CurrentMass at mid-juvenile: got %f, want %f", got, want)
	}
}

<<<<<<< Updated upstream
func TestCurrentSizeNeverExceedsGenomeSize(t *testing.T) {
=======
func TestIsJuvenileBlocksBeforeAdulthood(t *testing.T) {
	params := defaultParams()
	params.MinJuvenilePeriod = 100
	params.MaxJuvenilePeriod = 100

	genome := simulation.MakeRandomGenome(params)
	genome.JuvenilePeriod = 0 // maps to MinJuvenilePeriod (100)

	c := simulation.NewCreature(grid.RESERVED_CELL_TYPES, grid.Coord{}, genome)

	c.Age = 99
	if !c.IsJuvenile(params) {
		t.Errorf("creature at age 99 should still be juvenile (period=100)")
	}
	c.Age = 100
	if c.IsJuvenile(params) {
		t.Errorf("creature at age 100 should no longer be juvenile (period=100)")
	}
}

func TestIsJuvenileZeroPeriod(t *testing.T) {
	params := defaultParams()
	params.MinJuvenilePeriod = 0
	params.MaxJuvenilePeriod = 0

	genome := simulation.MakeRandomGenome(params)
	c := simulation.NewCreature(grid.RESERVED_CELL_TYPES, grid.Coord{}, genome)

	c.Age = 0
	if c.IsJuvenile(params) {
		t.Errorf("creature should never be juvenile when period=0")
	}
}

func TestCurrentMassNeverExceedsGenomeMass(t *testing.T) {
>>>>>>> Stashed changes
	params := defaultParams()
	genome := simulation.MakeRandomGenome(params)

	c := simulation.NewCreature(grid.RESERVED_CELL_TYPES, grid.Coord{}, genome)
	for age := 0; age <= params.MaxJuvenilePeriod+10; age++ {
		c.Age = age
		s := c.CurrentMass(params)
		if s > float32(genome.Mass) {
			t.Errorf("CurrentMass %f exceeds genome.Mass %d at age %d", s, genome.Mass, age)
		}
<<<<<<< Updated upstream
		if s < float32(params.MinSize) {
			t.Errorf("CurrentSize %f below MinSize %d at age %d", s, params.MinSize, age)
=======
		if s < float32(genome.MinMass) {
			t.Errorf("CurrentMass %f below genome.MinMass %d at age %d", s, genome.MinMass, age)
>>>>>>> Stashed changes
		}
	}
}
