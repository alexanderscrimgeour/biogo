package simulation

import "math"

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

// PassedSurvivalCriteria reports whether a creature satisfies the given challenge.
// Retained for analysis; the main simulation does not call this.
func PassedSurvivalCriteria(c *Creature, s *Simulation, challenge ChallengeType) bool {
	switch challenge {
	case LeftSurvive:
		return c.Loc.X < s.Params.GridWidth/2

	case FarLeftSurvive:
		return c.Loc.X < s.Params.GridWidth/10

	case RightSurvive:
		return c.Loc.X > s.Params.GridWidth/2

	case Groups:
		minNeighbours := 4
		radius := 4.0
		isBorder := c.Loc.X < 1 || c.Loc.X >= s.Params.GridWidth-1 ||
			c.Loc.Y < 1 || c.Loc.Y >= s.Params.GridHeight-1
		if isBorder {
			return false
		}
		nearEdge := c.Loc.X < 5 || c.Loc.X > s.Params.GridWidth-6 ||
			c.Loc.Y < 5 || c.Loc.Y > s.Params.GridHeight-6
		if nearEdge {
			return false
		}
		neighbours := s.World.GetCreaturesInRadius(c.Loc, radius, c.LocalCreatureBuffer)
		return len(neighbours)-1 >= minNeighbours // -1 to exclude self

	case Center:
		cx := s.Params.GridWidth / 2
		cy := s.Params.GridHeight / 2
		radius := 50.0
		dx := c.Loc.X - cx
		dy := c.Loc.Y - cy
		return math.Sqrt(dx*dx+dy*dy) <= radius

	case AllSurvive:
		fallthrough
	default:
		return true
	}
}
