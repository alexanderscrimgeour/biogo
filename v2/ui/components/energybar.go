package components

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type EnergyBar struct {
	Value, Max float32
	MinColor   color.Color
	MaxColor   color.Color
	Width      float32
	Centered   bool
}

func (e *EnergyBar) Draw(screen *ebiten.Image, x, y float32) (float32, float32) {
	vector.DrawFilledRect(screen, x, y, e.Width, 6, color.RGBA{35, 35, 35, 255}, false)

	if e.Centered {
		mid := x + e.Width/2
		half := e.Width / 2
		frac := e.Value / e.Max
		if frac > 1 {
			frac = 1
		} else if frac < -1 {
			frac = -1
		}
		if frac >= 0 {
			vector.DrawFilledRect(screen, mid, y, half*frac, 6, e.energyBarColor(1), false)
		} else {
			w := half * (-frac)
			vector.DrawFilledRect(screen, mid-w, y, w, 6, e.energyBarColor(0), false)
		}
	} else {
		frac := e.Value / e.Max
		if frac < 0 {
			frac = 0
		} else if frac > 1 {
			frac = 1
		}
		vector.DrawFilledRect(screen, x, y, e.Width*frac, 6, e.energyBarColor(frac), false)
	}

	return e.Width, 10
}

func (e *EnergyBar) energyBarColor(frac float32) color.RGBA {
	return lerpColor(e.MinColor, e.MaxColor, frac)
}

func lerpColor(c1, c2 color.Color, t float32) color.RGBA {
	// Extract values as uint32 (0-65535)
	r1, g1, b1, a1 := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()

	return color.RGBA{
		R: uint8(float32(r1>>8) + t*(float32(r2>>8)-float32(r1>>8))),
		G: uint8(float32(g1>>8) + t*(float32(g2>>8)-float32(g1>>8))),
		B: uint8(float32(b1>>8) + t*(float32(b2>>8)-float32(b1>>8))),
		A: uint8(float32(a1>>8) + t*(float32(a2>>8)-float32(a1>>8))),
	}
}
