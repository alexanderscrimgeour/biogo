package world

import "math"

// Position is a continuous 2D coordinate in world-space.
type Position struct {
	X, Y float32
}

func (p Position) DistanceTo(other Position) float32 {
	dx := p.X - other.X
	dy := p.Y - other.Y
	return float32(math.Sqrt(float64(dx*dx + dy*dy)))
}

func (p Position) Add(dx, dy float32) Position {
	return Position{p.X + dx, p.Y + dy}
}
