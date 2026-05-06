package test

import (
	"biogo/v2/jaro"
	"math"
	"testing"
)

func TestJaroSimilarity(t *testing.T) {
	cases := []struct {
		a, b string
		want float32
	}{
		{"", "abc", 0},
		{"abc", "", 0},
		{"abc", "abc", 1},
		{"martha", "marhta", 0.944},
		{"dixon", "dicksonx", 0.767},
	}
	for _, c := range cases {
		got := jaro.JaroSimilarity(c.a, c.b)
		if c.want == 0 || c.want == 1 {
			if got != c.want {
				t.Errorf("JaroSimilarity(%q, %q) = %f, want %f", c.a, c.b, got, c.want)
			}
		} else {
			if math.Abs(float64(got-c.want)) > 0.01 {
				t.Errorf("JaroSimilarity(%q, %q) = %f, want ~%f", c.a, c.b, got, c.want)
			}
		}
	}
}

func TestJaroWinklerSimilarity(t *testing.T) {
	cases := []struct {
		a, b string
		want float32
	}{
		{"", "abc", 0},
		{"abc", "abc", 1},
		{"MARTHA", "MARHTA", 0.961},
		{"DWAYNE", "DUANE", 0.84},
	}
	for _, c := range cases {
		got := jaro.JaroWinklerSimilarity(c.a, c.b)
		if c.want == 0 || c.want == 1 {
			if got != c.want {
				t.Errorf("JaroWinklerSimilarity(%q, %q) = %f, want %f", c.a, c.b, got, c.want)
			}
		} else {
			if math.Abs(float64(got-c.want)) > 0.01 {
				t.Errorf("JaroWinklerSimilarity(%q, %q) = %f, want ~%f", c.a, c.b, got, c.want)
			}
		}
	}
}

func TestJaroWinklerOrderSymmetry(t *testing.T) {
	a, b := "abcdef", "abcxyz"
	sim1 := jaro.JaroWinklerSimilarity(a, b)
	sim2 := jaro.JaroWinklerSimilarity(b, a)
	if math.Abs(float64(sim1-sim2)) > 0.001 {
		t.Errorf("JaroWinkler not symmetric: %f vs %f", sim1, sim2)
	}
}

func TestJaroWinklerPrefixBoost(t *testing.T) {
	// Shared prefix should score higher than no shared prefix with same length strings
	withPrefix := jaro.JaroWinklerSimilarity("abcxxx", "abcyyy")
	withoutPrefix := jaro.JaroWinklerSimilarity("xyzabc", "uvwabc")
	if withPrefix <= withoutPrefix {
		t.Errorf("prefix boost not observed: withPrefix=%f, withoutPrefix=%f", withPrefix, withoutPrefix)
	}
}
