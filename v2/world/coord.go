package world

import "math"

// Position is a continuous 2D coordinate in world-space.
type Position struct {
	X, Y float64
}

func (p Position) DistanceTo(other Position) float64 {
	dx := p.X - other.X
	dy := p.Y - other.Y
	return math.Sqrt(dx*dx + dy*dy)
}

func (p Position) Add(dx, dy float64) Position {
	return Position{p.X + dx, p.Y + dy}
}
