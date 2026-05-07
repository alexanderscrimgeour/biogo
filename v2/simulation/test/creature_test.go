package test

import (
	"biogo/v2/grid"
	"biogo/v2/simulation"
	"testing"
)

func TestCurrentSizeAtBirth(t *testing.T) {
	params := defaultParams()
	genome := simulation.MakeRandomGenome(params)
	genome.Size = 200
	genome.MinSize = 10

	c := simulation.NewCreature(grid.RESERVED_CELL_TYPES, grid.Coord{}, genome)
	got := c.CurrentSize(params)
	want := float32(genome.MinSize)
	if got != want {
		t.Errorf("CurrentSize at age 0: got %f, want %f", got, want)
	}
}

func TestCurrentSizeAtAdulthood(t *testing.T) {
	params := defaultParams()
	genome := simulation.MakeRandomGenome(params)
	genome.Size = 200
	genome.JuvenilePeriod = 0 // maps to MinJuvenilePeriod

	c := simulation.NewCreature(grid.RESERVED_CELL_TYPES, grid.Coord{}, genome)
	c.Age = params.MaxJuvenilePeriod + 1

	got := c.CurrentSize(params)
	want := float32(genome.Size)
	if got != want {
		t.Errorf("CurrentSize at adulthood: got %f, want %f", got, want)
	}
}

func TestCurrentSizeMidJuvenile(t *testing.T) {
	params := defaultParams()
	params.MinJuvenilePeriod = 100
	params.MaxJuvenilePeriod = 100 // fix juvenile period to exactly 100 ticks

	genome := simulation.MakeRandomGenome(params)
	genome.Size = 100
	genome.MinSize = 10
	genome.JuvenilePeriod = 0 // maps to MinJuvenilePeriod (100)

	c := simulation.NewCreature(grid.RESERVED_CELL_TYPES, grid.Coord{}, genome)
	c.Age = 50 // halfway through juvenile period

	got := c.CurrentSize(params)
	want := float32(genome.MinSize) + (float32(genome.Size)-float32(genome.MinSize))*0.5
	if got != want {
		t.Errorf("CurrentSize at mid-juvenile: got %f, want %f", got, want)
	}
}

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

func TestCurrentSizeNeverExceedsGenomeSize(t *testing.T) {
	params := defaultParams()
	genome := simulation.MakeRandomGenome(params)

	c := simulation.NewCreature(grid.RESERVED_CELL_TYPES, grid.Coord{}, genome)
	for age := 0; age <= params.MaxJuvenilePeriod+10; age++ {
		c.Age = age
		s := c.CurrentSize(params)
		if s > float32(genome.Size) {
			t.Errorf("CurrentSize %f exceeds genome.Size %d at age %d", s, genome.Size, age)
		}
		if s < float32(genome.MinSize) {
			t.Errorf("CurrentSize %f below genome.MinSize %d at age %d", s, genome.MinSize, age)
		}
	}
}
