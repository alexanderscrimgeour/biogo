package test

import (
	"biogo/v2/grid"
	"biogo/v2/simulation"
	"testing"
)

// makeCreatureAt creates a minimal creature at loc facing east (1,0).
func makeCreatureAt(loc grid.Coord, sightDist, fov byte) *simulation.Creature {
	p := defaultParams()
	g := simulation.MakeRandomGenome(p)
	g.SightDistance = sightDist
	g.FieldOfView = fov
	c := simulation.NewCreature(grid.RESERVED_CELL_TYPES, loc, g)
	c.LastMoveDir = grid.Dir{X: 1, Y: 0} // facing east
	return c
}

func makeGrid(w, h int) *grid.Grid {
	// gridMap 0 = MIDDLE_WALL; use a large grid so the wall is far away
	return grid.NewGrid(w, h, 0)
}

func TestSightFoodForward_DetectsFood(t *testing.T) {
	g := makeGrid(200, 200)
	loc := grid.Coord{X: 100, Y: 100}
	c := makeCreatureAt(loc, 5, 90) // 90-degree FOV, facing east

	// Place food directly ahead (east)
	foodLoc := grid.Coord{X: 103, Y: 100}
	g.Set(foodLoc, grid.FOOD)
	g.FoodLocations = append(g.FoodLocations, foodLoc)

	params := defaultParams()
	val := c.GetSensor(simulation.SIGHT_FOOD_FORWARD, g, nil, 0, params)
	if val <= 0 {
		t.Errorf("expected food to be detected ahead, got %f", val)
	}
}

func TestSightFoodForward_NoFoodBehind(t *testing.T) {
	g := makeGrid(200, 200)
	loc := grid.Coord{X: 100, Y: 100}
	c := makeCreatureAt(loc, 5, 90) // facing east, 90-degree FOV

	// Place food directly behind (west) – outside any forward FOV
	foodLoc := grid.Coord{X: 97, Y: 100}
	g.Set(foodLoc, grid.FOOD)
	g.FoodLocations = append(g.FoodLocations, foodLoc)

	params := defaultParams()
	val := c.GetSensor(simulation.SIGHT_FOOD_FORWARD, g, nil, 0, params)
	if val != 0 {
		t.Errorf("expected 0 for food behind creature, got %f", val)
	}
}

func TestSightFoodForward_WiderFOVSeesMoreFood(t *testing.T) {
	// Food placed diagonally ahead-north (NE). A narrow FOV (10°) should miss it;
	// a wide FOV (180°) should see it.
	loc := grid.Coord{X: 100, Y: 100}
	foodLoc := grid.Coord{X: 103, Y: 103} // NE of creature

	gNarrow := makeGrid(200, 200)
	gNarrow.Set(foodLoc, grid.FOOD)
	gNarrow.FoodLocations = append(gNarrow.FoodLocations, foodLoc)
	cNarrow := makeCreatureAt(loc, 6, 10) // very narrow FOV facing east

	gWide := makeGrid(200, 200)
	gWide.Set(foodLoc, grid.FOOD)
	gWide.FoodLocations = append(gWide.FoodLocations, foodLoc)
	cWide := makeCreatureAt(loc, 6, 180) // full-hemisphere FOV facing east

	params := defaultParams()
	narrow := cNarrow.GetSensor(simulation.SIGHT_FOOD_FORWARD, gNarrow, nil, 0, params)
	wide := cWide.GetSensor(simulation.SIGHT_FOOD_FORWARD, gWide, nil, 0, params)

	if narrow != 0 {
		t.Errorf("narrow FOV should not see NE food, got %f", narrow)
	}
	if wide <= 0 {
		t.Errorf("wide FOV should see NE food, got %f", wide)
	}
}

func TestSightPopForward_EmptyGridReturnsOne(t *testing.T) {
	g := makeGrid(200, 200)
	loc := grid.Coord{X: 100, Y: 100}
	c := makeCreatureAt(loc, 4, 90)

	params := defaultParams()
	val := c.GetSensor(simulation.SIGHT_POPULATION_FORWARD, g, nil, 0, params)
	// All cells in the forward cone of an empty grid (away from walls) should be empty
	if val <= 0 {
		t.Errorf("expected positive empty fraction on empty grid, got %f", val)
	}
}

func TestSightFoodForward_ScalesWithDistance(t *testing.T) {
	// Close food should score higher than distant food.
	loc := grid.Coord{X: 100, Y: 100}

	gClose := makeGrid(200, 200)
	close := grid.Coord{X: 102, Y: 100}
	gClose.Set(close, grid.FOOD)
	gClose.FoodLocations = append(gClose.FoodLocations, close)
	cClose := makeCreatureAt(loc, 8, 90)

	gFar := makeGrid(200, 200)
	far := grid.Coord{X: 107, Y: 100}
	gFar.Set(far, grid.FOOD)
	gFar.FoodLocations = append(gFar.FoodLocations, far)
	cFar := makeCreatureAt(loc, 8, 90)

	params := defaultParams()
	scoreClose := cClose.GetSensor(simulation.SIGHT_FOOD_FORWARD, gClose, nil, 0, params)
	scoreFar := cFar.GetSensor(simulation.SIGHT_FOOD_FORWARD, gFar, nil, 0, params)

	if scoreClose <= scoreFar {
		t.Errorf("close food (%f) should score higher than far food (%f)", scoreClose, scoreFar)
	}
}
