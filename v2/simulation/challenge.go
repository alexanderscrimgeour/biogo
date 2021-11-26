package simulation

import (
	"gopop/v2/grid"
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
)

func PassedSurvivalCriteria(c *Creature, s *Simulation) bool {

	switch s.Challenge {
	case LeftSurvive:
		if c.Loc.X < int(Params.GridWidth/2) {
			return true
		}

	case FarLeftSurvive:
		if c.Loc.X < int(Params.GridWidth/10) {
			return true
		}

	case RightSurvive:
		if c.Loc.X > int(Params.GridWidth/2) {
			return true
		}
	case Groups:
		minNeighbours := 2
		radius := float32(1.5)

		if s.Grid.IsBorder(c.Loc) {
			return false
		}
		n := 0
		neighbours := s.Grid.GetNeighbours(c.Loc, radius)
		for _, coord := range neighbours {
			if s.Grid.IsOccupiedAt(coord) {
				n++
			}
		}
		if n >= minNeighbours {
			return true
		}
	case Center:
		center := grid.Coord{X: int(s.Grid.SizeX() / 2), Y: int(s.Grid.SizeY() / 2)}
		radius := 40
		offset := grid.Coord{
			X: int(math.Abs(float64(c.Loc.X - center.X))),
			Y: int(math.Abs(float64(c.Loc.Y - center.Y))),
		}
		dist := math.Sqrt(float64(offset.X*offset.X) + float64(offset.Y*offset.Y))
		// fmt.Printf("dist: %f, center: %v, loc: %v\n", dist, center, c.Loc)
		return int(dist) <= radius
	case AllSurvive:
		fallthrough
	default:
		return true
	}
	return false
}
