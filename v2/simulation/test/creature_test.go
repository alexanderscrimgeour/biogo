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

	c := simulation.NewCreature(1, grid.Position{}, genome)
	got := c.CurrentSize(params)
	want := float32(params.MinSize)
	if got != want {
		t.Errorf("CurrentSize at age 0: got %f, want %f", got, want)
	}
}

func TestCurrentSizeAtAdulthood(t *testing.T) {
	params := defaultParams()
	genome := simulation.MakeRandomGenome(params)
	genome.Size = 200
	genome.JuvenilePeriod = 0

	c := simulation.NewCreature(1, grid.Position{}, genome)
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
	params.MaxJuvenilePeriod = 100

	genome := simulation.MakeRandomGenome(params)
	genome.Size = 100
	genome.JuvenilePeriod = 0

	c := simulation.NewCreature(1, grid.Position{}, genome)
	c.Age = 50

	got := c.CurrentSize(params)
	want := float32(params.MinSize) + (float32(genome.Size)-float32(params.MinSize))*0.5
	if got != want {
		t.Errorf("CurrentSize at mid-juvenile: got %f, want %f", got, want)
	}
}

func TestCurrentSizeNeverExceedsGenomeSize(t *testing.T) {
	params := defaultParams()
	genome := simulation.MakeRandomGenome(params)

	c := simulation.NewCreature(1, grid.Position{}, genome)
	for age := 0; age <= params.MaxJuvenilePeriod+10; age++ {
		c.Age = age
		s := c.CurrentSize(params)
		if s > float32(genome.Size) {
			t.Errorf("CurrentSize %f exceeds genome.Size %d at age %d", s, genome.Size, age)
		}
		if s < float32(params.MinSize) {
			t.Errorf("CurrentSize %f below MinSize %d at age %d", s, params.MinSize, age)
		}
	}
}
