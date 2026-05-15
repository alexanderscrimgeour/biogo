package components

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Panel struct {
	X, Y, W, H float32
	BaseColor  color.RGBA
	Border     color.RGBA
}

func (p *Panel) Draw(screen *ebiten.Image, isPressed bool) {
	drawColor := p.BaseColor
	if isPressed {
		drawColor = lighten(p.BaseColor, 0.10)
	}

	// Background and Border
	vector.FillRect(screen, p.X, p.Y, p.W, p.H, drawColor, false)
	vector.StrokeRect(screen, p.X, p.Y, p.W, p.H, 1, p.Border, false)
}

func (p *Panel) Contains(x, y float64) bool {
	return x >= float64(p.X) && x <= float64(p.X+p.W) &&
		y >= float64(p.Y) && y <= float64(p.Y+p.H)
}
