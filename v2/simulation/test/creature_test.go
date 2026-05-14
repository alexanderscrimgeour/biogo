package test

import (
	grid "biogo/v2/world"
	"biogo/v2/simulation"
	"testing"
)

func TestCurrentMassAtBirth(t *testing.T) {
	params := defaultParams()
	genome := simulation.MakeRandomGenome(params)
	genome.Mass = 200
	genome.MinMass = 10

	c := simulation.NewCreature(1, grid.Position{}, genome, params)
	got := c.CurrentMass()
	want := float32(genome.MinMass)
	if got != want {
		t.Errorf("CurrentMass at age 0: got %f, want %f", got, want)
	}
}

func TestCurrentMassAtAdulthood(t *testing.T) {
	params := defaultParams()
	genome := simulation.MakeRandomGenome(params)
	genome.Mass = 200

	c := simulation.NewAdultCreature(1, grid.Position{}, genome, params)

	got := c.CurrentMass()
	want := float32(genome.Mass)
	if got != want {
		t.Errorf("CurrentMass at adulthood: got %f, want %f", got, want)
	}
}

func TestCurrentMassGrowsWithVonBertalanffy(t *testing.T) {
	params := defaultParams()
	params.MinJuvenilePeriod = 100
	params.MaxJuvenilePeriod = 100

	genome := simulation.MakeRandomGenome(params)
	genome.Mass = 100
	genome.MinMass = 10
	genome.JuvenilePeriod = 0 // maps to MinJuvenilePeriod (100)

	c := simulation.NewCreature(1, grid.Position{}, genome, params)
	startMass := c.CurrentMass()

	for i := 0; i < 50; i++ {
		c.GrowMass(params)
	}

	midMass := c.CurrentMass()
	if midMass <= startMass {
		t.Errorf("mass should grow after 50 ticks: start=%f mid=%f", startMass, midMass)
	}
	if midMass >= float32(genome.Mass) {
		t.Errorf("should not reach adult mass in 50 ticks: mass=%f adult=%d", midMass, genome.Mass)
	}

	for i := 0; i < 5000; i++ {
		c.GrowMass(params)
	}
	finalMass := c.CurrentMass()
	if finalMass > float32(genome.Mass) {
		t.Errorf("mass should never exceed genome.Mass: got %f, max %d", finalMass, genome.Mass)
	}
}

func TestIsJuvenileBlocksBeforeAdulthood(t *testing.T) {
	params := defaultParams()
	params.MinJuvenilePeriod = 100
	params.MaxJuvenilePeriod = 100

	genome := simulation.MakeRandomGenome(params)
	genome.JuvenilePeriod = 0 // maps to MinJuvenilePeriod (100)

	c := simulation.NewCreature(1, grid.Position{}, genome, params)

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
	c := simulation.NewCreature(1, grid.Position{}, genome, params)

	c.Age = 0
	if c.IsJuvenile(params) {
		t.Errorf("creature should never be juvenile when period=0")
	}
}

func TestMetabolicRateScalesWithMass(t *testing.T) {
	// Kleiber's Law: larger creatures have higher absolute basal metabolic rate.
	params := defaultParams()
	params.BaseBMR = 1.0

	genome := simulation.MakeRandomGenome(params)
	genome.MetabolicRate = 127

	small := simulation.NewCreature(1, grid.Position{}, genome, params)
	small.Mass = 10

	large := simulation.NewCreature(2, grid.Position{}, genome, params)
	large.Mass = 200

	smallRate := small.MetabolicRate(params, grid.TempCold)
	largeRate := large.MetabolicRate(params, grid.TempCold)

	if largeRate <= smallRate {
		t.Errorf("larger creature should have higher absolute metabolic rate: small=%f large=%f", smallRate, largeRate)
	}
}

func TestCurrentMassNeverExceedsGenomeMass(t *testing.T) {
	params := defaultParams()
	genome := simulation.MakeRandomGenome(params)

	c := simulation.NewCreature(1, grid.Position{}, genome, params)
	for tick := 0; tick <= params.MaxJuvenilePeriod+10; tick++ {
		s := c.CurrentMass()
		if s > float32(genome.Mass) {
			t.Errorf("CurrentMass %f exceeds genome.Mass %d at tick %d", s, genome.Mass, tick)
		}
		if s < float32(genome.MinMass) {
			t.Errorf("CurrentMass %f below genome.MinMass %d at tick %d", s, genome.MinMass, tick)
		}
		c.GrowMass(params)
	}
}
