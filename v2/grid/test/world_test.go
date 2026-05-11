package test

import (
	"biogo/v2/grid"
	"testing"
)

func TestNewWorld(t *testing.T) {
	w := grid.NewWorld(100, 80, 0)
	if w.SizeX() != 100 {
		t.Errorf("expected SizeX 100, got %d", w.SizeX())
	}
	if w.SizeY() != 80 {
		t.Errorf("expected SizeY 80, got %d", w.SizeY())
	}
}

func TestIsInBounds(t *testing.T) {
	w := grid.NewWorld(100, 100, 0)
	cases := []struct {
		pos      grid.Position
		expected bool
	}{
		{grid.Position{X: 0, Y: 0}, true},
		{grid.Position{X: 99, Y: 99}, true},
		{grid.Position{X: 50, Y: 50}, true},
		{grid.Position{X: -1, Y: 0}, false},
		{grid.Position{X: 0, Y: -1}, false},
		{grid.Position{X: 100, Y: 0}, false},
		{grid.Position{X: 0, Y: 100}, false},
	}
	for _, c := range cases {
		if got := w.IsInBounds(c.pos); got != c.expected {
			t.Errorf("IsInBounds(%v) = %v, want %v", c.pos, got, c.expected)
		}
	}
}

func TestAddAndGetCreature(t *testing.T) {
	w := grid.NewWorld(100, 100, 0)
	pos := grid.Position{X: 50, Y: 50}
	w.AddCreature(1, pos)

	got, ok := w.GetCreaturePos(1)
	if !ok {
		t.Fatal("GetCreaturePos should find creature after AddCreature")
	}
	if got != pos {
		t.Errorf("GetCreaturePos returned %v, want %v", got, pos)
	}
}

func TestMoveCreature(t *testing.T) {
	w := grid.NewWorld(100, 100, 0)
	w.AddCreature(1, grid.Position{X: 10, Y: 10})
	newPos := grid.Position{X: 20, Y: 30}
	w.MoveCreature(1, newPos)

	got, ok := w.GetCreaturePos(1)
	if !ok || got != newPos {
		t.Errorf("after MoveCreature, got %v, want %v", got, newPos)
	}
}

func TestRemoveCreature(t *testing.T) {
	w := grid.NewWorld(100, 100, 0)
	w.AddCreature(1, grid.Position{X: 50, Y: 50})
	w.RemoveCreature(1)
	if _, ok := w.GetCreaturePos(1); ok {
		t.Error("creature should not be found after RemoveCreature")
	}
}

func TestGetCreaturesInRadius(t *testing.T) {
	w := grid.NewWorld(200, 200, 0)
	w.AddCreature(1, grid.Position{X: 100, Y: 100})
	w.AddCreature(2, grid.Position{X: 102, Y: 100}) // within radius 5
	w.AddCreature(3, grid.Position{X: 200, Y: 200}) // far away

	ids := w.GetCreaturesInRadius(grid.Position{X: 100, Y: 100}, 5)
	found := map[int]bool{}
	for _, id := range ids {
		found[id] = true
	}
	if !found[1] || !found[2] {
		t.Error("expected creatures 1 and 2 within radius 5")
	}
	if found[3] {
		t.Error("creature 3 should not be within radius 5")
	}
}

func TestAddAndRemoveFood(t *testing.T) {
	w := grid.NewWorld(100, 100, 0)
	id := w.AddFood(grid.Position{X: 50, Y: 50}, 10)
	if w.FoodCount() != 1 {
		t.Errorf("expected 1 food, got %d", w.FoodCount())
	}
	w.RemoveFood(id)
	if w.FoodCount() != 0 {
		t.Errorf("expected 0 food after removal, got %d", w.FoodCount())
	}
}

func TestGetFoodInRadius(t *testing.T) {
	w := grid.NewWorld(200, 200, 0)
	id1 := w.AddFood(grid.Position{X: 100, Y: 100}, 10)
	id2 := w.AddFood(grid.Position{X: 102, Y: 100}, 10) // within radius 5
	_ = w.AddFood(grid.Position{X: 150, Y: 150}, 10) // far away

	ids := w.GetFoodInRadius(grid.Position{X: 100, Y: 100}, 5)
	found := map[int]bool{}
	for _, id := range ids {
		found[id] = true
	}
	if !found[id1] || !found[id2] {
		t.Error("expected food items within radius 5")
	}
}

func TestSpawnFood(t *testing.T) {
	w := grid.NewWorld(200, 200, 0)
	w.InitFountains(2)
	w.SpawnFood(10, 30.0, 10)
	if w.FoodCount() != 10 {
		t.Errorf("SpawnFood(10) should place exactly 10 items, got %d", w.FoodCount())
	}
}

func TestSpawnFoodGaussianFallback(t *testing.T) {
	// Without fountains initialised, SpawnFood should fall back to random placement.
	w := grid.NewWorld(500, 500, 0)
	w.SpawnFood(50, 30.0, 10)
	if w.FoodCount() != 50 {
		t.Errorf("SpawnFood(50) without fountains should place exactly 50 items, got %d", w.FoodCount())
	}
}

func TestFindEmptyLocation(t *testing.T) {
	w := grid.NewWorld(200, 200, 0)
	pos, ok := w.FindEmptyLocation()
	if !ok {
		t.Fatal("FindEmptyLocation should succeed on empty world")
	}
	if !w.IsInBounds(pos) {
		t.Errorf("FindEmptyLocation returned out-of-bounds position %v", pos)
	}
	if w.IsWall(pos) {
		t.Errorf("FindEmptyLocation returned wall position %v", pos)
	}
}

func TestIsWall_CrossWall(t *testing.T) {
	w := grid.NewWorld(200, 200, 1)
	// Center of the world should be inside a wall segment.
	center := grid.Position{X: 100, Y: 100}
	if !w.IsWall(center) {
		t.Error("center of cross-wall world should be a wall")
	}
	// Corner should be free.
	corner := grid.Position{X: 10, Y: 10}
	if w.IsWall(corner) {
		t.Error("corner should not be a wall")
	}
}

func TestClampToBounds(t *testing.T) {
	w := grid.NewWorld(100, 100, 0)
	cases := []struct {
		in   grid.Position
		wantInBounds bool
	}{
		{grid.Position{X: -5, Y: 50}, true},
		{grid.Position{X: 50, Y: -5}, true},
		{grid.Position{X: 200, Y: 50}, true},
		{grid.Position{X: 50, Y: 200}, true},
		{grid.Position{X: 50, Y: 50}, true},
	}
	for _, c := range cases {
		clamped := w.ClampToBounds(c.in)
		if !w.IsInBounds(clamped) {
			t.Errorf("ClampToBounds(%v) = %v is out of bounds", c.in, clamped)
		}
	}
}
