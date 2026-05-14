package simulation

import (
	"biogo/v2/world"
	"math"
)

// processCollisions applies soft repulsive corrections between overlapping creatures
// and resolves tunneling for fast-moving creatures using swept-sphere detection.
func (s *Simulation) processCollisions() {
	if s.Params.CollisionRepulsion <= 0 {
		return
	}

	ids := s.Population.AliveIDs()
	if len(ids) < 2 {
		return
	}

	maxRadius := math.Sqrt(float64(s.Params.MaxMass) * math.Pi)
	repulsion := s.Params.CollisionRepulsion

	// Expand the search radius to cover the swept path of both creatures this tick.
	// Each creature moves at most MaxSpeedPerStep per tick, so the worst case
	// (two creatures tunneling toward each other) requires 2× that expansion.
	sweptExpansion := 2 * s.Params.MaxSpeedPerStep

	var buf []int
	for _, id := range ids {
		c := s.Population.Creatures[id]
		if !c.Alive {
			continue
		}

		buf = s.World.GetCreaturesInRadius(c.Loc, c.Radius+maxRadius+sweptExpansion, buf)

		for _, otherID := range buf {
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
			minDistSq := minDist * minDist

			if distSq < minDistSq {
				// Current overlap: apply soft repulsion.
				dist := math.Sqrt(distSq)
				var nx, ny float64
				if dist < 1e-9 {
					nx, ny = 1, 0
				} else {
					nx = dx / dist
					ny = dy / dist
				}

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
				continue
			}

			// Swept-sphere check: did either creature tunnel through the other?
			// Use relative motion so both moving creatures are handled correctly.
			relStartX := c.LastLoc.X - other.LastLoc.X
			relStartY := c.LastLoc.Y - other.LastLoc.Y
			relEndX := c.Loc.X - other.Loc.X
			relEndY := c.Loc.Y - other.Loc.Y

			t, hit := sweptSphereContactT(relStartX, relStartY, relEndX, relEndY, minDist)
			if !hit {
				continue
			}

			// Interpolate each creature's position at the moment of first contact.
			cContactX := c.LastLoc.X + t*(c.Loc.X-c.LastLoc.X)
			cContactY := c.LastLoc.Y + t*(c.Loc.Y-c.LastLoc.Y)
			oContactX := other.LastLoc.X + t*(other.Loc.X-other.LastLoc.X)
			oContactY := other.LastLoc.Y + t*(other.Loc.Y-other.LastLoc.Y)

			// Collision normal from other to c at contact.
			ncx := cContactX - oContactX
			ncy := cContactY - oContactY
			nlen := math.Sqrt(ncx*ncx + ncy*ncy)
			if nlen < 1e-9 {
				ncx, ncy = 1, 0
			} else {
				ncx /= nlen
				ncy /= nlen
			}

			// Place creatures separated by minDist along the collision normal,
			// mass-weighted so heavier creatures are displaced less.
			totalMass := float64(c.Mass + other.Mass)
			pushC := minDist * float64(other.Mass) / totalMass
			pushOther := minDist * float64(c.Mass) / totalMass

			newCLoc := s.World.ClampToBounds(world.Position{
				X: oContactX + ncx*pushC,
				Y: oContactY + ncy*pushC,
			})
			newOtherLoc := s.World.ClampToBounds(world.Position{
				X: oContactX - ncx*pushOther,
				Y: oContactY - ncy*pushOther,
			})

			c.Loc = newCLoc
			other.Loc = newOtherLoc
			s.World.MoveCreature(id, newCLoc)
			s.World.MoveCreature(otherID, newOtherLoc)

			// Kill the velocity that caused the tunnel so the creatures don't
			// immediately pass through each other again next tick.
			c.Velocity = 0
			other.Velocity = 0
		}
	}
}

// sweptSphereContactT returns the fraction t in (0, 1] of the tick at which two
// spheres first touch, given their relative displacement from relStart to relEnd
// and their combined radius minDist. Returns (t, true) on contact, (0, false) otherwise.
func sweptSphereContactT(relStartX, relStartY, relEndX, relEndY, minDist float64) (float64, bool) {
	ddx := relEndX - relStartX
	ddy := relEndY - relStartY

	a := ddx*ddx + ddy*ddy
	if a < 1e-12 {
		return 0, false // no relative motion
	}

	b := 2 * (relStartX*ddx + relStartY*ddy)
	c := relStartX*relStartX + relStartY*relStartY - minDist*minDist

	disc := b*b - 4*a*c
	if disc < 0 {
		return 0, false
	}

	t := (-b - math.Sqrt(disc)) / (2 * a)
	if t <= 0 || t > 1 {
		return 0, false
	}
	return t, true
}
