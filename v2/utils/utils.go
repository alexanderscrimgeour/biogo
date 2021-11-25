package utils

import (
	"math"
)

func MinByte(a byte, b byte) byte {
	if b < a {
		return b
	}
	return a
}

// Max return the largest integer among the two in parameters
func MaxByte(a byte, b byte) byte {
	if b > a {
		return b
	}
	return a
}

func RestrictByte(min, max, val byte) byte {
	return MinByte(MaxByte(val, min), max)
}

func ClampByte(min, max, val byte) byte {
	return byte(((float32(val) / math.MaxUint8) * float32(max-min)) + float32(min))
}

func Min(a int, b int) int {
	if b < a {
		return b
	}
	return a
}

// Max return the largest integer among the two in parameters
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

// Max return the largest integer among the two in parameters
func MaxFloat32(a, b float32) float32 {
	if b > a {
		return b
	}
	return a
}

// Equal compare two rune arrays and return if they are equals or not
func Equal(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
