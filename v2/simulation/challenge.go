package simulation

type ChallengeType int

const (
	LeftSurvive ChallengeType = iota
	RightSurvive
)

func PassedSurvivalCriteria(c *Creature, challenge ChallengeType) bool {

	switch challenge {
	case LeftSurvive:
		if c.Loc.X < int(Params.GridWidth/2) {
			return true
		}
	case RightSurvive:
		if c.Loc.X > int(Params.GridWidth/2) {
			return true
		}
	default:
		return true
	}
	return false
}
