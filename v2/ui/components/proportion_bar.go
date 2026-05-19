package components

import (
	"image/color"

	"biogo/v2/ui/colors"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ProportionBar draws a single bar divided into contiguous coloured segments
// proportional to the provided values. Values are normalised internally so they
// don't need to sum to 1.
type ProportionBar struct {
	Segments []ProportionSegment
	Width    float32
}

type ProportionSegment struct {
	Value float32
	Color color.RGBA
}

func (b *ProportionBar) Draw(screen *ebiten.Image, x, y float32) (float32, float32) {
	vector.FillRect(screen, x, y, b.Width, 6, colors.ColorBarBG, false)

	total := float32(0)
	for _, s := range b.Segments {
		total += s.Value
	}
	if total <= 0 {
		return b.Width, 10
	}

	cursor := x
	for _, s := range b.Segments {
		w := b.Width * (s.Value / total)
		if w > 0 {
			vector.FillRect(screen, cursor, y, w, 6, s.Color, false)
			cursor += w
		}
	}
	return b.Width, 10
}

func (b *ProportionBar) Size() (float32, float32) {
	return b.Width, 10
}
