package test

import (
	"biogo/v2/grid"
	"testing"
)

func TestNewGrid(t *testing.T) {
	g := grid.NewGrid(10, 20, 0)
	if g.SizeX() != 10 {
		t.Errorf("expected SizeX 10, got %d", g.SizeX())
	}
	if g.SizeY() != 20 {
		t.Errorf("expected SizeY 20, got %d", g.SizeY())
	}
}

func TestIsInBounds(t *testing.T) {
	g := grid.NewGrid(10, 10, 0)
	cases := []struct {
		loc      grid.Coord
		expected bool
	}{
		{grid.Coord{X: 0, Y: 0}, true},
		{grid.Coord{X: 9, Y: 9}, true},
		{grid.Coord{X: 5, Y: 5}, true},
		{grid.Coord{X: -1, Y: 0}, false},
		{grid.Coord{X: 0, Y: -1}, false},
		{grid.Coord{X: 10, Y: 0}, false},
		{grid.Coord{X: 0, Y: 10}, false},
	}
	for _, c := range cases {
		if got := g.IsInBounds(c.loc); got != c.expected {
			t.Errorf("IsInBounds(%v) = %v, want %v", c.loc, got, c.expected)
		}
	}
}

func TestSetAndAt(t *testing.T) {
	g := grid.NewGrid(10, 10, 0)
	g.ZeroFill() // clear the auto-generated wall so cells are predictably empty
	loc := grid.Coord{X: 1, Y: 1}
	g.Set(loc, 42)
	if got := g.At(loc); got != 42 {
		t.Errorf("At after Set: got %d, want 42", got)
	}
}

func TestIsEmptyAt(t *testing.T) {
	g := grid.NewGrid(10, 10, 0)
	g.ZeroFill()
	loc := grid.Coord{X: 1, Y: 1}
	if !g.IsEmptyAt(loc) {
		t.Error("expected cell to be empty before Set")
	}
	g.Set(loc, grid.WALL)
	if g.IsEmptyAt(loc) {
		t.Error("expected cell to be non-empty after Set(WALL)")
	}
}

func TestIsOccupiedAt(t *testing.T) {
	g := grid.NewGrid(10, 10, 0)
	g.ZeroFill()
	loc := grid.Coord{X: 1, Y: 1}
	if g.IsOccupiedAt(loc) {
		t.Error("empty cell should not be occupied")
	}
	g.Set(loc, grid.WALL)
	if g.IsOccupiedAt(loc) {
		t.Error("WALL cell should not count as occupied creature")
	}
	g.Set(loc, grid.RESERVED_CELL_TYPES+1)
	if !g.IsOccupiedAt(loc) {
		t.Error("creature cell should be occupied")
	}
}

func TestIsBorder(t *testing.T) {
	g := grid.NewGrid(10, 10, 0)
	borders := []grid.Coord{
		{X: 0, Y: 5}, {X: 9, Y: 5},
		{X: 5, Y: 0}, {X: 5, Y: 9},
	}
	for _, b := range borders {
		if !g.IsBorder(b) {
			t.Errorf("expected %v to be a border", b)
		}
	}
	inner := grid.Coord{X: 5, Y: 5}
	if g.IsBorder(inner) {
		t.Errorf("expected %v to not be a border", inner)
	}
}

func TestZeroFill(t *testing.T) {
	g := grid.NewGrid(10, 10, 0)
	g.Set(grid.Coord{X: 3, Y: 3}, 99)
	g.ZeroFill()
	if g.At(grid.Coord{X: 3, Y: 3}) != 0 {
		t.Error("ZeroFill should reset all cells to 0")
	}
}

func TestDrawBox(t *testing.T) {
	g := grid.NewGrid(20, 20, 0)
	g.ZeroFill()
	g.DrawBox(2, 2, 5, 5)
	for x := 2; x < 5; x++ {
		for y := 2; y < 5; y++ {
			if g.At(grid.Coord{X: x, Y: y}) != grid.WALL {
				t.Errorf("expected WALL at (%d,%d)", x, y)
			}
		}
	}
}

func TestFindEmptyLocation(t *testing.T) {
	g := grid.NewGrid(20, 20, 0)
	loc, ok := g.FindEmptyLocation()
	if !ok {
		t.Fatal("FindEmptyLocation returned false on an empty grid")
	}
	if !g.IsInBounds(loc) {
		t.Errorf("FindEmptyLocation returned out-of-bounds coord %v", loc)
	}
	if !g.IsEmptyAt(loc) {
		t.Errorf("FindEmptyLocation returned occupied coord %v", loc)
	}
}

func TestGetNeighbours(t *testing.T) {
	g := grid.NewGrid(20, 20, 0)
	center := grid.Coord{X: 10, Y: 10}
	neighbours := g.GetNeighbours(center, 1)
	if len(neighbours) == 0 {
		t.Error("GetNeighbours should return at least one neighbour")
	}
	for _, n := range neighbours {
		if !g.IsInBounds(n) {
			t.Errorf("neighbour %v is out of bounds", n)
		}
	}
}

func TestCountNeighbours(t *testing.T) {
	g := grid.NewGrid(20, 20, 0)
	center := grid.Coord{X: 10, Y: 10}
	g.Set(grid.Coord{X: 11, Y: 10}, grid.RESERVED_CELL_TYPES+1)
	g.Set(grid.Coord{X: 9, Y: 10}, grid.RESERVED_CELL_TYPES+2)

	count := g.CountNeighbours(center, 1, func(g grid.Grid, x, y int) int {
		if g.IsOccupiedAt(grid.Coord{X: x, Y: y}) {
			return 1
		}
		return 0
	})
	if count != 2 {
		t.Errorf("CountNeighbours: expected 2, got %d", count)
	}
}

func TestDensityNeighbours(t *testing.T) {
	g := grid.NewGrid(20, 20, 0)
	center := grid.Coord{X: 10, Y: 10}
	density := g.DensityNeighbours(center, 2, func(g grid.Grid, x, y int) int {
		return 0
	})
	if density != 0 {
		t.Errorf("empty grid density should be 0, got %f", density)
	}
}

func TestSpawnFood(t *testing.T) {
	g := grid.NewGrid(20, 20, 0)
	g.ZeroFill()
	g.SpawnFood(5)

	if len(g.FoodLocations) != 5 {
		t.Errorf("expected 5 food locations, got %d", len(g.FoodLocations))
	}
	for _, loc := range g.FoodLocations {
		if !g.IsFood(loc) {
			t.Errorf("expected food at %v", loc)
		}
	}
}

func TestIsFoodNotOccupied(t *testing.T) {
	g := grid.NewGrid(20, 20, 0)
	g.ZeroFill()
	loc := grid.Coord{X: 5, Y: 5}
	g.Set(loc, grid.FOOD)

	if !g.IsFood(loc) {
		t.Error("IsFood should return true for FOOD cell")
	}
	if g.IsOccupiedAt(loc) {
		t.Error("IsOccupiedAt should return false for FOOD cell (food is not a creature)")
	}
	if g.IsEmptyAt(loc) {
		t.Error("IsEmptyAt should return false for FOOD cell")
	}
}

func TestRemoveFood(t *testing.T) {
	g := grid.NewGrid(20, 20, 0)
	g.ZeroFill()
	g.SpawnFood(3)
	initial := len(g.FoodLocations)

	loc := g.FoodLocations[0]
	g.RemoveFood(loc)

	if len(g.FoodLocations) != initial-1 {
		t.Errorf("expected %d food locations after removal, got %d", initial-1, len(g.FoodLocations))
	}
	if g.IsFood(loc) {
		t.Error("cell should no longer be food after RemoveFood")
	}
	if !g.IsEmptyAt(loc) {
		t.Error("cell should be empty after RemoveFood")
	}
}

func TestZeroFillClearsFoodLocations(t *testing.T) {
	g := grid.NewGrid(20, 20, 0)
	g.ZeroFill()
	g.SpawnFood(5)
	g.ZeroFill()

	if len(g.FoodLocations) != 0 {
		t.Errorf("ZeroFill should clear FoodLocations, got %d", len(g.FoodLocations))
	}
}
