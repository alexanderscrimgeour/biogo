package world

import "math"

// HeadingToVec converts a heading angle (radians) to a unit vector (dx, dy).
// 0 = east (+X), π/2 = south (+Y, screen-down), π = west, -π/2 = north.
func HeadingToVec(heading float64) (float64, float64) {
	return math.Cos(heading), math.Sin(heading)
}

// AngleBetween returns the heading angle (radians) from position `from` to `to`.
func AngleBetween(from, to Position) float64 {
	return math.Atan2(to.Y-from.Y, to.X-from.X)
}

// NormalizeAngle wraps an angle into [-π, π].
func NormalizeAngle(a float64) float64 {
	for a > math.Pi {
		a -= 2 * math.Pi
	}
	for a < -math.Pi {
		a += 2 * math.Pi
	}
	return a
}

// CosSimilarity returns the cosine of the angle between two direction vectors.
// Returns 1.0 when either vector is zero-length.
func CosSimilarity(dx1, dy1, dx2, dy2 float64) float64 {
	mag1 := math.Sqrt(dx1*dx1 + dy1*dy1)
	mag2 := math.Sqrt(dx2*dx2 + dy2*dy2)
	if mag1 == 0 || mag2 == 0 {
		return 1
	}
	cos := (dx1*dx2 + dy1*dy2) / (mag1 * mag2)
	if cos < -1 {
		return -1
	}
	if cos > 1 {
		return 1
	}
	return cos
}
