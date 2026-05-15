package test

import (
	grid "biogo/v2/world"
	"biogo/v2/simulation"
	"math"
	"testing"
)

// makeCreatureAt creates a minimal creature registered in w at pos facing east (heading = 0).
func makeCreatureAt(w *grid.World, pos grid.Position, sightDist, fov byte) *simulation.Creature {
	p := defaultParams()
	g := simulation.MakeRandomGenome(p)
	g.SightDistance = sightDist
	g.FieldOfView = fov
	id := w.AddCreature(pos)
	c := simulation.NewCreature(id, pos, g, p)
	c.Heading = 0 // facing east
	return c
}

func makeWorld(w, h float64) *grid.World {
	return grid.NewWorld(w, h, 0)
}

func TestSightFoodForward_DetectsFood(t *testing.T) {
	w := makeWorld(200, 200)
	loc := grid.Position{X: 100, Y: 100}
	c := makeCreatureAt(w, loc, 5, 90)

	// Place food directly ahead (east)
	w.AddPlant(grid.Position{X: 103, Y: 100}, 10)

	params := defaultParams()
	c.UpdateSensorContext(w, nil, params)
	val := c.GetSensor(simulation.SIGHT_FOOD_FORWARD, w, nil, &c.Sensors, 0, params)
	if val <= -1 {
		t.Errorf("expected food to be detected ahead, got %f", val)
	}
}

func TestSightFoodForward_NoFoodBehind(t *testing.T) {
	w := makeWorld(200, 200)
	loc := grid.Position{X: 100, Y: 100}
	c := makeCreatureAt(w, loc, 5, 90)

	// Place food directly behind (west)
	w.AddPlant(grid.Position{X: 97, Y: 100}, 10)

	params := defaultParams()
	c.UpdateSensorContext(w, nil, params)
	val := c.GetSensor(simulation.SIGHT_FOOD_FORWARD, w, nil, &c.Sensors, 0, params)
	if val != -1 {
		t.Errorf("expected -1 for food behind creature (none in FOV), got %f", val)
	}
}

func TestSightFoodForward_WiderFOVSeesMoreFood(t *testing.T) {
	// Food placed diagonally NE (ahead-right). Narrow FOV (10°) misses; wide (180°) sees.
	loc := grid.Position{X: 100, Y: 100}
	foodPos := grid.Position{X: 103, Y: 103}

	wNarrow := makeWorld(200, 200)
	wNarrow.AddPlant(foodPos, 10)
	cNarrow := makeCreatureAt(wNarrow, loc, 6, 10)

	wWide := makeWorld(200, 200)
	wWide.AddPlant(foodPos, 10)
	cWide := makeCreatureAt(wWide, loc, 6, 180)

	params := defaultParams()
	cNarrow.UpdateSensorContext(wNarrow, nil, params)
	cWide.UpdateSensorContext(wWide, nil, params)
	narrow := cNarrow.GetSensor(simulation.SIGHT_FOOD_FORWARD, wNarrow, nil, &cNarrow.Sensors, 0, params)
	wide := cWide.GetSensor(simulation.SIGHT_FOOD_FORWARD, wWide, nil, &cWide.Sensors, 0, params)

	if narrow != -1 {
		t.Errorf("narrow FOV should not see NE food (expect -1), got %f", narrow)
	}
	if wide <= -1 {
		t.Errorf("wide FOV should see NE food (expect > -1), got %f", wide)
	}
}

func TestSightPopForward_EmptyWorldReturnsZero(t *testing.T) {
	w := makeWorld(200, 200)
	loc := grid.Position{X: 100, Y: 100}
	c := makeCreatureAt(w, loc, 4, 90)

	params := defaultParams()
	c.UpdateSensorContext(w, nil, params)
	val := c.GetSensor(simulation.SIGHT_POPULATION_FORWARD, w, nil, &c.Sensors, 0, params)
	if val != 0 {
		t.Errorf("expected 0 for empty world ahead, got %f", val)
	}
}

func TestSightFoodForward_ScalesWithDistance(t *testing.T) {
	loc := grid.Position{X: 100, Y: 100}

	wClose := makeWorld(200, 200)
	wClose.AddPlant(grid.Position{X: 102, Y: 100}, 10)
	cClose := makeCreatureAt(wClose, loc, 8, 90)

	wFar := makeWorld(200, 200)
	wFar.AddPlant(grid.Position{X: 107, Y: 100}, 10)
	cFar := makeCreatureAt(wFar, loc, 8, 90)

	params := defaultParams()
	cClose.UpdateSensorContext(wClose, nil, params)
	cFar.UpdateSensorContext(wFar, nil, params)
	scoreClose := cClose.GetSensor(simulation.SIGHT_FOOD_FORWARD, wClose, nil, &cClose.Sensors, 0, params)
	scoreFar := cFar.GetSensor(simulation.SIGHT_FOOD_FORWARD, wFar, nil, &cFar.Sensors, 0, params)

	if scoreClose <= scoreFar {
		t.Errorf("close food (%f) should score higher than far food (%f)", scoreClose, scoreFar)
	}
}

func TestSatiation_FullStomach(t *testing.T) {
	params := defaultParams()
	loc := grid.Position{X: 50, Y: 50}
	w := makeWorld(200, 200)
	c := makeCreatureAt(w, loc, 1, 90)

	cap := c.StomachCapacity(params)
	c.Stomach = cap
	c.UpdateSensorContext(w, nil, params)
	val := c.GetSensor(simulation.SATIATION, w, nil, &c.Sensors, 0, params)
	if math.Abs(float64(val)-1.0) > 0.01 {
		t.Errorf("SATIATION at full stomach should be ~1.0, got %f", val)
	}
}

func TestSatiation_EmptyStomach(t *testing.T) {
	params := defaultParams()
	loc := grid.Position{X: 50, Y: 50}
	w := makeWorld(200, 200)
	c := makeCreatureAt(w, loc, 1, 90)

	c.Stomach = 0
	c.UpdateSensorContext(w, nil, params)
	val := c.GetSensor(simulation.SATIATION, w, nil, &c.Sensors, 0, params)
	if math.Abs(float64(val)) > 0.01 {
		t.Errorf("SATIATION at empty stomach should be ~0.0, got %f", val)
	}
}

func TestSatiation_HalfStomach(t *testing.T) {
	params := defaultParams()
	loc := grid.Position{X: 50, Y: 50}
	w := makeWorld(200, 200)
	c := makeCreatureAt(w, loc, 1, 90)

	cap := c.StomachCapacity(params)
	c.Stomach = cap / 2
	c.UpdateSensorContext(w, nil, params)
	val := c.GetSensor(simulation.SATIATION, w, nil, &c.Sensors, 0, params)
	if math.Abs(float64(val)-0.5) > 0.01 {
		t.Errorf("SATIATION at half stomach should be ~0.5, got %f", val)
	}
}

func TestStomachRate_MaxDigestion(t *testing.T) {
	params := defaultParams()
	w := makeWorld(200, 200)
	c := makeCreatureAt(w, grid.Position{X: 50, Y: 50}, 1, 90)

	// Stomach drained by exactly DigestionRate → output should be ~1.0.
	c.LastStomach = float32(params.DigestionRate)
	c.Stomach = 0
	c.UpdateSensorContext(w, nil, params)
	val := c.GetSensor(simulation.STOMACH_RATE, w, nil, &c.Sensors, 0, params)
	if math.Abs(float64(val)-1.0) > 0.01 {
		t.Errorf("STOMACH_RATE at max digestion should be ~1.0, got %f", val)
	}
}

func TestStomachRate_NoDigestion(t *testing.T) {
	params := defaultParams()
	w := makeWorld(200, 200)
	c := makeCreatureAt(w, grid.Position{X: 50, Y: 50}, 1, 90)

	c.LastStomach = 5
	c.Stomach = 5
	c.UpdateSensorContext(w, nil, params)
	val := c.GetSensor(simulation.STOMACH_RATE, w, nil, &c.Sensors, 0, params)
	if math.Abs(float64(val)) > 0.01 {
		t.Errorf("STOMACH_RATE with no change should be ~0.0, got %f", val)
	}
}

func TestEnergyDelta_MaxGain(t *testing.T) {
	params := defaultParams()
	w := makeWorld(200, 200)
	c := makeCreatureAt(w, grid.Position{X: 50, Y: 50}, 1, 90)

	maxE := c.MaxEnergy(params)
	c.LastTickEnergy = 0
	c.Energy = maxE
	c.UpdateSensorContext(w, nil, params)
	val := c.GetSensor(simulation.ENERGY_DELTA, w, nil, &c.Sensors, 0, params)
	if math.Abs(float64(val)-1.0) > 0.01 {
		t.Errorf("ENERGY_DELTA at max gain should be ~1.0, got %f", val)
	}
}

func TestEnergyDelta_NoChange(t *testing.T) {
	params := defaultParams()
	w := makeWorld(200, 200)
	c := makeCreatureAt(w, grid.Position{X: 50, Y: 50}, 1, 90)

	maxE := c.MaxEnergy(params)
	c.Energy = maxE / 2
	c.LastTickEnergy = maxE / 2
	c.UpdateSensorContext(w, nil, params)
	val := c.GetSensor(simulation.ENERGY_DELTA, w, nil, &c.Sensors, 0, params)
	if math.Abs(float64(val)-0.5) > 0.01 {
		t.Errorf("ENERGY_DELTA with no change should be ~0.5, got %f", val)
	}
}

func TestEnergyDelta_MaxLoss(t *testing.T) {
	params := defaultParams()
	w := makeWorld(200, 200)
	c := makeCreatureAt(w, grid.Position{X: 50, Y: 50}, 1, 90)

	maxE := c.MaxEnergy(params)
	c.LastTickEnergy = maxE
	c.Energy = 0
	c.UpdateSensorContext(w, nil, params)
	val := c.GetSensor(simulation.ENERGY_DELTA, w, nil, &c.Sensors, 0, params)
	if math.Abs(float64(val)-0.0) > 0.01 {
		t.Errorf("ENERGY_DELTA at max loss should be ~0.0, got %f", val)
	}
}
