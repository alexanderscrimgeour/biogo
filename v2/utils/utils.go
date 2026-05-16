package utils

import (
	"math"
	"math/rand"
)

func MinByte(a byte, b byte) byte {
	if b < a {
		return b
	}
	return a
}

func MaxByte(a byte, b byte) byte {
	if b > a {
		return b
	}
	return a
}

func RestrictByte(min, max, val byte) byte {
	return MinByte(MaxByte(val, min), max)
}

func LerpByte(min, max, val byte) byte {
	return byte(((float32(val) / math.MaxUint8) * float32(max-min)) + float32(min))
}

func LerpByteAsFloat32(min, max, val byte) float32 {
	fMin := float32(min)
	fMax := float32(max)
	fVal := float32(val)

	return fMin + (fVal/math.MaxUint8)*(fMax-fMin)
}
func Min(a int, b int) int {
	if b < a {
		return b
	}
	return a
}

func Max(a int, b int) int {
	if b > a {
		return b
	}
	return a
}

func MinFloat32(a, b float32) float32 {
	if b < a {
		return b
	}
	return a
}

func MinFloat64(a, b float64) float64 {
	if b < a {
		return b
	}
	return a
}

func MaxFloat32(a, b float32) float32 {
	if b > a {
		return b
	}
	return a
}

func RestrictFloat32(min, max, val float32) float32 {
	return MinFloat32(MaxFloat32(val, min), max)
}

func MakeRandomByte() byte {
	return byte(rand.Uint32() >> 24)
}
