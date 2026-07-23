package simulation

import (
	"biogo/v2/world"
	"math"
)

// processCollisions applies soft repulsive corrections between overlapping creatures
// and resolves tunneling for fast-moving creatures using swept-sphere detection.
func (s *Simulation) processCollisions() {
	if s.params.World.CollisionRepulsion <= 0 {
		return
	}

	ids := s.Population.AliveIDs()
	if len(ids) < 2 {
		return
	}

	repulsion := float32(s.params.World.CollisionRepulsion)

	// Pre-compute the maximum alive radius so small creatures search far enough
	// to find large creatures that overlap them (pairs are processed from the
	// smaller-ID creature, whose own radius may be much less than the other's).
	var maxRadius float32
	for _, id := range ids {
		if c, ok := s.Population.Get(id); ok && c.Alive && c.Radius > maxRadius {
			maxRadius = c.Radius
		}
	}

	var buf []int
	for _, id := range ids {
		c, ok := s.Population.Get(id)
		if !ok || !c.Alive {
			continue
		}
		speed := c.Speed
		if speed < 0 {
			speed = -speed
		}
		localSweptExpansion := speed * 2.0
		searchRadius := c.Radius + maxRadius + localSweptExpansion + s.params.World.MaxVelocityFallback
		buf = s.World.CreaturesInRadius(c.Loc, searchRadius, buf)

		for _, otherID := range buf {
			if otherID <= id {
				continue
			}
			other, ok := s.Population.Get(otherID)
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
				dist := float32(math.Sqrt(float64(distSq)))
				var nx, ny float32
				if dist < 1e-9 {
					nx, ny = 1, 0
				} else {
					nx = dx / dist
					ny = dy / dist
				}

				totalMass := c.Mass + other.Mass
				correction := (minDist - dist) * repulsion
				pushC := correction * other.Mass / totalMass
				pushOther := correction * c.Mass / totalMass

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
			relStartX := float64(c.LastLoc.X - other.LastLoc.X)
			relStartY := float64(c.LastLoc.Y - other.LastLoc.Y)
			relEndX := float64(c.Loc.X - other.Loc.X)
			relEndY := float64(c.Loc.Y - other.Loc.Y)

			t, hit := sweptSphereContactT(relStartX, relStartY, relEndX, relEndY, float64(minDist))
			if !hit {
				continue
			}

			// Interpolate each creature's position at the moment of first contact.
			tf := float32(t)
			cContactX := c.LastLoc.X + tf*(c.Loc.X-c.LastLoc.X)
			cContactY := c.LastLoc.Y + tf*(c.Loc.Y-c.LastLoc.Y)
			oContactX := other.LastLoc.X + tf*(other.Loc.X-other.LastLoc.X)
			oContactY := other.LastLoc.Y + tf*(other.Loc.Y-other.LastLoc.Y)

			// Collision normal from other to c at contact.
			ncx := cContactX - oContactX
			ncy := cContactY - oContactY
			nlen := float32(math.Sqrt(float64(ncx*ncx + ncy*ncy)))
			if nlen < 1e-9 {
				ncx, ncy = 1, 0
			} else {
				ncx /= nlen
				ncy /= nlen
			}

			// Place creatures separated by minDist along the collision normal,
			// mass-weighted so heavier creatures are displaced less.
			totalMass := c.Mass + other.Mass
			pushC := minDist * other.Mass / totalMass
			pushOther := minDist * c.Mass / totalMass

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

			// Kill the speed that caused the tunnel so the creatures don't
			// immediately pass through each other again next tick.
			c.Speed = 0
			other.Speed = 0
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
