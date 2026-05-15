package simulation

import "math"

// tanhf approximates tanh using a Pade(6,6) rational polynomial.
// Error < 0.01% for |x| ≤ 4.97; clamped beyond ±4.97.
func tanhf(x float32) float32 {
	if x > 4.97 {
		return 1
	}
	if x < -4.97 {
		return -1
	}
	x2 := x * x
	return x * (135135 + x2*(17325+x2*(378+x2))) / (135135 + x2*(62370+x2*(3150+x2*28)))
}

func softsign(x float32) float32 {
	if x >= 0 {
		return x / (1 + x)
	}
	return x / (1 - x)
}

// absf32 clears the IEEE-754 sign bit without a branch.
func absf32(x float32) float32 {
	return math.Float32frombits(math.Float32bits(x) &^ (1 << 31))
}
