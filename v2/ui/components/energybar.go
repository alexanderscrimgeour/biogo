package components

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type EnergyBar struct {
	Value, Max float32
	Width      float32
}

func (e *EnergyBar) Draw(screen *ebiten.Image, x, y float32) (float32, float32) {
	frac := e.Value / e.Max
	if frac < 0 {
		frac = 0
	} else if frac > 1 {
		frac = 1
	}

	vector.DrawFilledRect(screen, x, y, e.Width, 6, color.RGBA{35, 35, 35, 255}, false)
	vector.DrawFilledRect(screen, x, y, e.Width*frac, 6, energyBarColor(frac), false)
	return e.Width, 10
}

// TODO: Move these color definitions to the Bar itself
func energyBarColor(frac float32) color.RGBA {
	if frac > 0.65 {
		return color.RGBA{55, 185, 55, 255}
	}
	if frac > 0.33 {
		return color.RGBA{190, 175, 45, 255}
	}
	return color.RGBA{190, 55, 55, 255}
}
