package test

import (
	"biogo/v2/grid"
	"testing"
)

func TestRotate90CW(t *testing.T) {
	cases := []struct {
		in, want grid.Dir
	}{
		{grid.N, grid.E},
		{grid.E, grid.S},
		{grid.S, grid.W},
		{grid.W, grid.N},
	}
	for _, c := range cases {
		got := c.in.Rotate90CW()
		if got != c.want {
			t.Errorf("Rotate90CW(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestRotate90CCW(t *testing.T) {
	cases := []struct {
		in, want grid.Dir
	}{
		{grid.N, grid.W},
		{grid.W, grid.S},
		{grid.S, grid.E},
		{grid.E, grid.N},
	}
	for _, c := range cases {
		got := c.in.Rotate90CCW()
		if got != c.want {
			t.Errorf("Rotate90CCW(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestGetDirection(t *testing.T) {
	cases := []struct {
		from, to grid.Coord
		wantX    int
		wantY    int
	}{
		{grid.Coord{X: 0, Y: 0}, grid.Coord{X: 3, Y: 0}, 1, -1},
		{grid.Coord{X: 3, Y: 0}, grid.Coord{X: 0, Y: 0}, -1, -1},
		{grid.Coord{X: 0, Y: 0}, grid.Coord{X: 0, Y: 3}, -1, 1},
		{grid.Coord{X: 0, Y: 0}, grid.Coord{X: 0, Y: 0}, 0, 0},
	}
	for _, c := range cases {
		got := grid.GetDirection(c.from, c.to)
		if c.from == c.to {
			if got.X != 0 || got.Y != 0 {
				t.Errorf("GetDirection same point: want {0,0}, got %v", got)
			}
			continue
		}
		if c.wantX != 0 && got.X != c.wantX {
			t.Errorf("GetDirection(%v→%v).X = %d, want %d", c.from, c.to, got.X, c.wantX)
		}
		if c.wantY != 0 && got.Y != c.wantY {
			t.Errorf("GetDirection(%v→%v).Y = %d, want %d", c.from, c.to, got.Y, c.wantY)
		}
	}
}

func TestRaySameness(t *testing.T) {
	// Parallel same direction → 1
	same := grid.RaySameness(grid.N, grid.N)
	if same < 0.99 {
		t.Errorf("RaySameness(N,N) = %f, want ~1.0", same)
	}

	// Opposite directions → -1
	opposite := grid.RaySameness(grid.N, grid.S)
	if opposite > -0.99 {
		t.Errorf("RaySameness(N,S) = %f, want ~-1.0", opposite)
	}

	// Zero vector → 1
	zero := grid.RaySameness(grid.CENTER, grid.N)
	if zero != 1 {
		t.Errorf("RaySameness(CENTER,N) = %f, want 1.0", zero)
	}
}

func TestRandomDir(t *testing.T) {
	for i := 0; i < 100; i++ {
		d := grid.RandomDir()
		if d.X < -1 || d.X > 1 || d.Y < -1 || d.Y > 1 {
			t.Errorf("RandomDir out of range: %v", d)
		}
	}
}
