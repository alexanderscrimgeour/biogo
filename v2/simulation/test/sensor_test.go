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
	c := simulation.NewCreature(1, pos, g, p)
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
	w.AddFood(grid.Position{X: 103, Y: 100}, 10)

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
	w.AddFood(grid.Position{X: 97, Y: 100}, 10)

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
	wNarrow.AddFood(foodPos, 10)
	cNarrow := makeCreatureAt(loc, 6, 10)
	wNarrow.AddCreature(cNarrow.Id, loc)

	wWide := makeWorld(200, 200)
	wWide.AddFood(foodPos, 10)
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

func TestSightPopForward_EmptyWorldReturnsZero(t *testing.T) {
	w := makeWorld(200, 200)
	loc := grid.Position{X: 100, Y: 100}
	c := makeCreatureAt(loc, 4, 90)
	w.AddCreature(c.Id, loc)

	params := defaultParams()
	val := c.GetSensor(simulation.SIGHT_POPULATION_FORWARD, w, nil, 0, params)
	if val != 0 {
		t.Errorf("expected 0 for empty world ahead, got %f", val)
	}
}

func TestSightFoodForward_ScalesWithDistance(t *testing.T) {
	loc := grid.Position{X: 100, Y: 100}

	wClose := makeWorld(200, 200)
	wClose.AddFood(grid.Position{X: 102, Y: 100}, 10)
	cClose := makeCreatureAt(loc, 8, 90)
	wClose.AddCreature(cClose.Id, loc)

	wFar := makeWorld(200, 200)
	wFar.AddFood(grid.Position{X: 107, Y: 100}, 10)
	cFar := makeCreatureAt(loc, 8, 90)
	wFar.AddCreature(cFar.Id, loc)

	params := defaultParams()
	scoreClose := cClose.GetSensor(simulation.SIGHT_FOOD_FORWARD, wClose, nil, 0, params)
	scoreFar := cFar.GetSensor(simulation.SIGHT_FOOD_FORWARD, wFar, nil, 0, params)

	if scoreClose <= scoreFar {
		t.Errorf("close food (%f) should score higher than far food (%f)", scoreClose, scoreFar)
	}
}

func TestSatiation_FullStomach(t *testing.T) {
	params := defaultParams()
	loc := grid.Position{X: 50, Y: 50}
	c := makeCreatureAt(loc, 1, 90)
	w := makeWorld(200, 200)
	w.AddCreature(c.Id, loc)

	cap := c.StomachCapacity(params)
	c.Stomach = cap
	val := c.GetSensor(simulation.SATIATION, w, nil, 0, params)
	if math.Abs(float64(val)-1.0) > 0.01 {
		t.Errorf("SATIATION at full stomach should be ~1.0, got %f", val)
	}
}

func TestSatiation_EmptyStomach(t *testing.T) {
	params := defaultParams()
	loc := grid.Position{X: 50, Y: 50}
	c := makeCreatureAt(loc, 1, 90)
	w := makeWorld(200, 200)
	w.AddCreature(c.Id, loc)

	c.Stomach = 0
	val := c.GetSensor(simulation.SATIATION, w, nil, 0, params)
	if math.Abs(float64(val)) > 0.01 {
		t.Errorf("SATIATION at empty stomach should be ~0.0, got %f", val)
	}
}

func TestSatiation_HalfStomach(t *testing.T) {
	params := defaultParams()
	loc := grid.Position{X: 50, Y: 50}
	c := makeCreatureAt(loc, 1, 90)
	w := makeWorld(200, 200)
	w.AddCreature(c.Id, loc)

	cap := c.StomachCapacity(params)
	c.Stomach = cap / 2
	val := c.GetSensor(simulation.SATIATION, w, nil, 0, params)
	if math.Abs(float64(val)-0.5) > 0.01 {
		t.Errorf("SATIATION at half stomach should be ~0.5, got %f", val)
	}
}

