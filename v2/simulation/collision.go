package simulation

import (
	"biogo/v2/world"
	"math"
)

// processCollisions applies soft repulsive corrections between overlapping creatures.
// Each ordered pair (a, b) where a.Id < b.Id is processed exactly once per tick,
// preventing double-counting without an explicit visited set. The spatial hash
// limits neighbor queries to cells within reach of the sum of radii, so the
// cost scales with local density rather than total population.
func (s *Simulation) processCollisions() {
	if s.Params.CollisionRepulsion <= 0 {
		return
	}

	ids := s.Population.AliveIDs()
	if len(ids) < 2 {
		return
	}

	// Upper bound on any creature's radius, used to bound the neighbor search
	// without knowing the neighbour's actual radius ahead of time.
	maxRadius := math.Sqrt(float64(s.Params.MaxMass) * math.Pi)
	repulsion := s.Params.CollisionRepulsion

	var buf []int
	for _, id := range ids {
		c := s.Population.Creatures[id]
		if !c.Alive {
			continue
		}

		buf = s.World.GetCreaturesInRadius(c.Loc, c.Radius+maxRadius, buf)

		for _, otherID := range buf {
			// Only process the pair once, from the creature with the smaller ID.
			if otherID <= id {
				continue
			}
			other, ok := s.Population.Creatures[otherID]
			if !ok || !other.Alive {
				continue
			}

			dx := other.Loc.X - c.Loc.X
			dy := other.Loc.Y - c.Loc.Y
			distSq := dx*dx + dy*dy
			minDist := c.Radius + other.Radius
			if distSq >= minDist*minDist {
				continue
			}

			dist := math.Sqrt(distSq)

			var nx, ny float64
			if dist < 1e-9 {
				// Degenerate: two creatures at the same point; push along a fixed axis.
				nx, ny = 1, 0
			} else {
				nx = dx / dist
				ny = dy / dist
			}

			// Mass-weighted push: heavier creatures are displaced less.
			totalMass := float64(c.Mass + other.Mass)
			correction := (minDist - dist) * repulsion
			pushC := correction * float64(other.Mass) / totalMass
			pushOther := correction * float64(c.Mass) / totalMass

			newCLoc := s.World.ClampToBounds(world.Position{
				X: c.Loc.X - pushC*nx,
				Y: c.Loc.Y - pushC*ny,
			})
			newOtherLoc := s.World.ClampToBounds(world.Position{
				X: other.Loc.X + pushOther*nx,
				Y: other.Loc.Y + pushOther*ny,
			})

			c.Loc = newCLoc
			other.Loc = newOtherLoc
			s.World.MoveCreature(id, newCLoc)
			s.World.MoveCreature(otherID, newOtherLoc)
		}
	}
}
