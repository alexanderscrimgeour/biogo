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
		return float64(c.Loc.X) < s.Params.World.Width/2

	case FarLeftSurvive:
		return float64(c.Loc.X) < s.Params.World.Width/10

	case RightSurvive:
		return float64(c.Loc.X) > s.Params.World.Width/2

	case Groups:
		minNeighbours := 4
		radius := float32(4.0)
		isBorder := float64(c.Loc.X) < 1 || float64(c.Loc.X) >= s.Params.World.Width-1 ||
			float64(c.Loc.Y) < 1 || float64(c.Loc.Y) >= s.Params.World.Height-1
		if isBorder {
			return false
		}
		nearEdge := float64(c.Loc.X) < 5 || float64(c.Loc.X) > s.Params.World.Width-6 ||
			float64(c.Loc.Y) < 5 || float64(c.Loc.Y) > s.Params.World.Height-6
		if nearEdge {
			return false
		}
		neighbours := s.World.GetCreaturesInRadius(c.Loc, radius, c.LocalCreatureBuffer)
		return len(neighbours)-1 >= minNeighbours // -1 to exclude self

	case Center:
		cx := float32(s.Params.World.Width / 2)
		cy := float32(s.Params.World.Height / 2)
		radius := float32(50.0)
		dx := c.Loc.X - cx
		dy := c.Loc.Y - cy
		return math.Sqrt(float64(dx*dx+dy*dy)) <= float64(radius)

	case AllSurvive:
		fallthrough
	default:
		return true
	}
}
