package components

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
)

type Label struct {
	Text  string
	Font  *textv2.GoXFace
	Color color.Color
}

func (l *Label) Draw(screen *ebiten.Image, x, y float32) (float32, float32) {
	op := &textv2.DrawOptions{}
	op.GeoM.Translate(float64(int(x)), float64(int(y)))
	op.ColorScale.ScaleWithColor(l.Color)
	textv2.Draw(screen, l.Text, l.Font, op)
	return 0, 18
}

// Size returns the preferred dimensions of a label line.
func (l *Label) Size() (float32, float32) {
	return 0, 18
}
