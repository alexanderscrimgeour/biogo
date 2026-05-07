package test

import (
	"biogo/v2/grid"
	"biogo/v2/simulation"
	"math"
	"testing"
)

// makeCreatureAt creates a minimal creature at pos facing east (heading = 0).
func makeCreatureAt(pos grid.Position, sightDist, fov byte) *simulation.Creature {
	p := defaultParams()
	g := simulation.MakeRandomGenome(p)
	g.SightDistance = sightDist
	g.FieldOfView = fov
	c := simulation.NewCreature(1, pos, g)
	c.Heading = 0 // facing east
	return c
}

func makeWorld(w, h float64) *grid.World {
	return grid.NewWorld(w, h, 0)
}

func TestSightFoodForward_DetectsFood(t *testing.T) {
	w := makeWorld(200, 200)
	loc := grid.Position{X: 100, Y: 100}
	c := makeCreatureAt(loc, 5, 90)
	w.AddCreature(c.Id, loc)

	// Place food directly ahead (east)
	w.AddFood(grid.Position{X: 103, Y: 100})

	params := defaultParams()
	val := c.GetSensor(simulation.SIGHT_FOOD_FORWARD, w, nil, 0, params)
	if val <= 0 {
		t.Errorf("expected food to be detected ahead, got %f", val)
	}
}

func TestSightFoodForward_NoFoodBehind(t *testing.T) {
	w := makeWorld(200, 200)
	loc := grid.Position{X: 100, Y: 100}
	c := makeCreatureAt(loc, 5, 90)
	w.AddCreature(c.Id, loc)

	// Place food directly behind (west)
	w.AddFood(grid.Position{X: 97, Y: 100})

	params := defaultParams()
	val := c.GetSensor(simulation.SIGHT_FOOD_FORWARD, w, nil, 0, params)
	if val != 0 {
		t.Errorf("expected 0 for food behind creature, got %f", val)
	}
}

func TestSightFoodForward_WiderFOVSeesMoreFood(t *testing.T) {
	// Food placed diagonally NE (ahead-right). Narrow FOV (10°) misses; wide (180°) sees.
	loc := grid.Position{X: 100, Y: 100}
	foodPos := grid.Position{X: 103, Y: 103}

	wNarrow := makeWorld(200, 200)
	wNarrow.AddFood(foodPos)
	cNarrow := makeCreatureAt(loc, 6, 10)
	wNarrow.AddCreature(cNarrow.Id, loc)

	wWide := makeWorld(200, 200)
	wWide.AddFood(foodPos)
	cWide := makeCreatureAt(loc, 6, 180)
	wWide.AddCreature(cWide.Id, loc)

	params := defaultParams()
	narrow := cNarrow.GetSensor(simulation.SIGHT_FOOD_FORWARD, wNarrow, nil, 0, params)
	wide := cWide.GetSensor(simulation.SIGHT_FOOD_FORWARD, wWide, nil, 0, params)

	if narrow != 0 {
		t.Errorf("narrow FOV should not see NE food, got %f", narrow)
	}
	if wide <= 0 {
		t.Errorf("wide FOV should see NE food, got %f", wide)
	}
}

func TestSightPopForward_EmptyWorldReturnsOne(t *testing.T) {
	w := makeWorld(200, 200)
	loc := grid.Position{X: 100, Y: 100}
	c := makeCreatureAt(loc, 4, 90)
	w.AddCreature(c.Id, loc)

	params := defaultParams()
	val := c.GetSensor(simulation.SIGHT_POPULATION_FORWARD, w, nil, 0, params)
	if val <= 0 {
		t.Errorf("expected positive value for empty world ahead, got %f", val)
	}
}

func TestSightFoodForward_ScalesWithDistance(t *testing.T) {
	loc := grid.Position{X: 100, Y: 100}

	wClose := makeWorld(200, 200)
	wClose.AddFood(grid.Position{X: 102, Y: 100})
	cClose := makeCreatureAt(loc, 8, 90)
	wClose.AddCreature(cClose.Id, loc)

	wFar := makeWorld(200, 200)
	wFar.AddFood(grid.Position{X: 107, Y: 100})
	cFar := makeCreatureAt(loc, 8, 90)
	wFar.AddCreature(cFar.Id, loc)

	params := defaultParams()
	scoreClose := cClose.GetSensor(simulation.SIGHT_FOOD_FORWARD, wClose, nil, 0, params)
	scoreFar := cFar.GetSensor(simulation.SIGHT_FOOD_FORWARD, wFar, nil, 0, params)

	if scoreClose <= scoreFar {
		t.Errorf("close food (%f) should score higher than far food (%f)", scoreClose, scoreFar)
	}
}

func TestPopulationFOV_DetectsCreatureInRange(t *testing.T) {
	w := makeWorld(200, 200)
	loc := grid.Position{X: 100, Y: 100}
	c := makeCreatureAt(loc, 10, 90)
	c.Heading = 0 // facing east
	w.AddCreature(c.Id, loc)

	// Place another creature just ahead and within radius 2.
	other := makeCreatureAt(grid.Position{X: 101.5, Y: 100}, 10, 90)
	other.Id = 2
	w.AddCreature(other.Id, other.Loc)

	params := defaultParams()
	val := c.GetSensor(simulation.POPULATION_FOV, w, nil, 0, params)
	if val <= 0 {
		t.Errorf("expected POPULATION_FOV > 0 for creature ahead in range, got %f", val)
	}
}

func TestPopulationFOV_IgnoresCreatureBehind(t *testing.T) {
	w := makeWorld(200, 200)
	loc := grid.Position{X: 100, Y: 100}
	c := makeCreatureAt(loc, 10, 90)
	c.Heading = 0 // facing east
	w.AddCreature(c.Id, loc)

	// Place creature behind (west), still within radius 2.
	other := makeCreatureAt(grid.Position{X: 98.5, Y: 100}, 10, 90)
	other.Id = 2
	w.AddCreature(other.Id, other.Loc)

	params := defaultParams()
	val := c.GetSensor(simulation.POPULATION_FOV, w, nil, 0, params)
	if val != 0 {
		t.Errorf("expected POPULATION_FOV = 0 for creature behind, got %f", val)
	}
}

func TestPopulationFOV_IgnoresCreatureTooFar(t *testing.T) {
	w := makeWorld(200, 200)
	loc := grid.Position{X: 100, Y: 100}
	c := makeCreatureAt(loc, 10, 90)
	c.Heading = 0 // facing east
	w.AddCreature(c.Id, loc)

	// Place creature ahead but outside eat radius (> 2).
	other := makeCreatureAt(grid.Position{X: 105, Y: 100}, 10, 90)
	other.Id = 2
	w.AddCreature(other.Id, other.Loc)

	params := defaultParams()
	val := c.GetSensor(simulation.POPULATION_FOV, w, nil, 0, params)
	if val != 0 {
		t.Errorf("expected POPULATION_FOV = 0 for creature outside eat radius, got %f", val)
	}
}

func TestSatiation_FullEnergy(t *testing.T) {
	params := defaultParams()
	loc := grid.Position{X: 50, Y: 50}
	c := makeCreatureAt(loc, 1, 90)
	w := makeWorld(200, 200)
	w.AddCreature(c.Id, loc)

	c.Energy = float32(c.Genome.MaxEnergy)
	val := c.GetSensor(simulation.SATIATION, w, nil, 0, params)
	if math.Abs(float64(val)-1.0) > 0.01 {
		t.Errorf("SATIATION at full energy should be ~1.0, got %f", val)
	}
}

func TestSatiation_MinEnergy(t *testing.T) {
	params := defaultParams()
	loc := grid.Position{X: 50, Y: 50}
	c := makeCreatureAt(loc, 1, 90)
	w := makeWorld(200, 200)
	w.AddCreature(c.Id, loc)

	c.Energy = float32(params.MinEnergy)
	val := c.GetSensor(simulation.SATIATION, w, nil, 0, params)
	if math.Abs(float64(val)) > 0.01 {
		t.Errorf("SATIATION at min energy should be ~0.0, got %f", val)
	}
}

func TestSatiation_MidEnergy(t *testing.T) {
	params := defaultParams()
	loc := grid.Position{X: 50, Y: 50}
	c := makeCreatureAt(loc, 1, 90)
	c.Genome.MaxEnergy = 100
	w := makeWorld(200, 200)
	w.AddCreature(c.Id, loc)

	// Energy halfway between min (2) and max (100) → satiation ≈ 0.5.
	c.Energy = float32(params.MinEnergy) + (float32(c.Genome.MaxEnergy)-float32(params.MinEnergy))/2
	val := c.GetSensor(simulation.SATIATION, w, nil, 0, params)
	if math.Abs(float64(val)-0.5) > 0.01 {
		t.Errorf("SATIATION at mid energy should be ~0.5, got %f", val)
	}
}

func TestHeadingSensor(t *testing.T) {
	params := defaultParams()
	loc := grid.Position{X: 50, Y: 50}

	// Facing east: cos(0)=1 → LAST_MOVE_DIR_X should be 1.0 (normalized to 1.0)
	cEast := makeCreatureAt(loc, 1, 90)
	cEast.Heading = 0
	w := makeWorld(200, 200)
	w.AddCreature(cEast.Id, loc)

	xVal := cEast.GetSensor(simulation.LAST_MOVE_DIR_X, w, nil, 0, params)
	if math.Abs(float64(xVal)-1.0) > 0.01 {
		t.Errorf("LAST_MOVE_DIR_X facing east should be ~1.0, got %f", xVal)
	}

	// Facing west: cos(π)=-1 → normalized to 0.0
	cWest := makeCreatureAt(loc, 1, 90)
	cWest.Heading = math.Pi
	w.AddCreature(cWest.Id, loc)
	xValW := cWest.GetSensor(simulation.LAST_MOVE_DIR_X, w, nil, 0, params)
	if math.Abs(float64(xValW)-0.0) > 0.01 {
		t.Errorf("LAST_MOVE_DIR_X facing west should be ~0.0, got %f", xValW)
	}
}
