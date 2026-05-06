package simulation

import (
	"biogo/v2/grid"
	"math"
)

type ChallengeType int

const (
	LeftSurvive ChallengeType = iota
	RightSurvive
	FarLeftSurvive
	Groups
	Center
	AllSurvive
	MiddleWall
)

func PassedSurvivalCriteria(c *Creature, s *Simulation) bool {
	switch s.Challenge {
	case LeftSurvive:
		return c.Loc.X < s.Params.GridWidth/2

	case FarLeftSurvive:
		return c.Loc.X < s.Params.GridWidth/10

	case RightSurvive:
		return c.Loc.X > s.Params.GridWidth/2

	case Groups:
		minNeighbours := 4
		radius := float32(4)

		if s.Grid.IsBorder(c.Loc) {
			return false
		}
		if c.Loc.X < 5 || c.Loc.X > s.Grid.SizeX()-6 || c.Loc.Y < 5 || c.Loc.Y > s.Grid.SizeY()-6 {
			return false
		}
		n := 0
		neighbours := s.Grid.GetNeighbours(c.Loc, radius)
		for _, coord := range neighbours {
			if s.Grid.IsOccupiedAt(coord) {
				n++
			}
		}
		return n >= minNeighbours

	case Center:
		center := grid.Coord{X: s.Grid.SizeX() / 2, Y: s.Grid.SizeY() / 2}
		radius := 50
		offset := grid.Coord{
			X: int(math.Abs(float64(c.Loc.X - center.X))),
			Y: int(math.Abs(float64(c.Loc.Y - center.Y))),
		}
		dist := math.Sqrt(float64(offset.X*offset.X) + float64(offset.Y*offset.Y))
		return int(dist) <= radius

	case AllSurvive:
		fallthrough
	default:
		return true
	}
}
