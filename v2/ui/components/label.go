package components

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
)

type Label struct {
	Text  string
	Font  font.Face
	Color color.Color
}

func (l *Label) Draw(screen *ebiten.Image, x, y float32) (float32, float32) {
	text.Draw(screen, l.Text, l.Font, int(x), int(y), l.Color)
	return 0, 18 // Return the height of a standard line
}
