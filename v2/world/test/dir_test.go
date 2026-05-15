package test

import (
	"biogo/v2/world"
	"math"
	"testing"
)

func TestHeadingToVec(t *testing.T) {
	cases := []struct {
		heading float32
		wantDX  float32
		wantDY  float32
	}{
		{0, 1, 0},                         // east
		{math.Pi / 2, 0, 1},               // south (screen-down)
		{math.Pi, -1, 0},                  // west
		{-math.Pi / 2, 0, -1},             // north (screen-up)
	}
	for _, c := range cases {
		dx, dy := world.HeadingToVec(c.heading)
		if math.Abs(float64(dx-c.wantDX)) > 1e-6 || math.Abs(float64(dy-c.wantDY)) > 1e-6 {
			t.Errorf("HeadingToVec(%.4f) = (%.4f,%.4f), want (%.4f,%.4f)",
				c.heading, dx, dy, c.wantDX, c.wantDY)
		}
	}
}

func TestNormalizeAngle(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{0, 0},
		{math.Pi, math.Pi},
		{-math.Pi, -math.Pi},
		{3 * math.Pi, math.Pi},
		{-3 * math.Pi, -math.Pi},
	}
	for _, c := range cases {
		got := world.NormalizeAngle(c.in)
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("NormalizeAngle(%.4f) = %.4f, want %.4f", c.in, got, c.want)
		}
	}
}

func TestCosSimilarity(t *testing.T) {
	// Same direction → 1
	if v := world.CosSimilarity(1, 0, 1, 0); math.Abs(float64(v)-1) > 1e-6 {
		t.Errorf("same direction should be 1, got %f", v)
	}
	// Opposite directions → -1
	if v := world.CosSimilarity(1, 0, -1, 0); math.Abs(float64(v)+1) > 1e-6 {
		t.Errorf("opposite direction should be -1, got %f", v)
	}
	// Perpendicular → 0
	if v := world.CosSimilarity(1, 0, 0, 1); math.Abs(float64(v)) > 1e-6 {
		t.Errorf("perpendicular should be 0, got %f", v)
	}
	// Zero vector → 1
	if v := world.CosSimilarity(0, 0, 1, 0); v != 1 {
		t.Errorf("zero vector should return 1, got %f", v)
	}
}

func TestAngleBetween(t *testing.T) {
	from := world.Position{X: 0, Y: 0}
	east := world.Position{X: 1, Y: 0}
	south := world.Position{X: 0, Y: 1}

	if a := world.AngleBetween(from, east); math.Abs(a) > 1e-9 {
		t.Errorf("angle to east should be 0, got %f", a)
	}
	if a := world.AngleBetween(from, south); math.Abs(a-math.Pi/2) > 1e-9 {
		t.Errorf("angle to south should be π/2, got %f", a)
	}
}
