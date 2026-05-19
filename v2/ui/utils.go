package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
)

var simRates = []int{1, 5, 25, 50, 100}
var UnitSize int = 2

func drawText(dst *ebiten.Image, str string, face *textv2.GoXFace, x, y int, clr color.Color) {
	op := &textv2.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(clr)
	textv2.Draw(dst, str, face, op)
}

func nextSimRate(current, dir int) int {
	for i, r := range simRates {
		if r == current {
			next := i + dir
			if next < 0 {
				next = 0
			} else if next >= len(simRates) {
				next = len(simRates) - 1
			}
			return simRates[next]
		}
	}
	return simRates[0]
}

func clamp(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
