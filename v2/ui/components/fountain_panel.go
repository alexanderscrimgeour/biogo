package components

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
)

const fpTitleH = float32(18)
const fpGap = float32(2)

// FountainPanel renders a labelled block of per-fountain control sliders.
// Construct with NewFountainPanel; pass the pre-built sliders in display order.
type FountainPanel struct {
	Title      string
	TitleColor color.Color
	Font       *textv2.GoXFace
	H          float32
	sliders    []*Slider
	lastX      float32
	lastY      float32
}

// NewFountainPanel builds a FountainPanel from an ordered list of sliders.
func NewFountainPanel(title string, titleColor color.Color, font *textv2.GoXFace, sliders ...*Slider) *FountainPanel {
	h := fpTitleH + fpGap
	for i, s := range sliders {
		if i > 0 {
			h += fpGap
		}
		h += s.H
	}
	return &FountainPanel{
		Title: title, TitleColor: titleColor, Font: font,
		sliders: sliders,
		H:       h,
	}
}

func (fp *FountainPanel) Draw(screen *ebiten.Image, x, y float32) (float32, float32) {
	fp.lastX, fp.lastY = x, y
	if fp.Font != nil && fp.Title != "" {
		op := &textv2.DrawOptions{}
		op.GeoM.Translate(float64(x+5), float64(y+2))
		clr := fp.TitleColor
		if clr == nil {
			clr = color.White
		}
		op.ColorScale.ScaleWithColor(clr)
		textv2.Draw(screen, fp.Title, fp.Font, op)
	}
	cy := y + fpTitleH + fpGap
	for i, s := range fp.sliders {
		if i > 0 {
			cy += fpGap
		}
		s.Draw(screen, x, cy)
		cy += s.H
	}
	w := float32(0)
	if len(fp.sliders) > 0 {
		w = fp.sliders[0].W
	}
	return w, fp.H
}

func (fp *FountainPanel) Size() (float32, float32) {
	w := float32(0)
	if len(fp.sliders) > 0 {
		w = fp.sliders[0].W
	}
	return w, fp.H
}

func (fp *FountainPanel) HandleDown(mx, my int) {
	for _, s := range fp.sliders {
		if s.InBounds(mx, my) {
			s.Dragging = true
			s.UpdateValue(mx)
		}
	}
}

func (fp *FountainPanel) HandleDrag(mx int) {
	for _, s := range fp.sliders {
		if s.Dragging {
			s.UpdateValue(mx)
		}
	}
}

func (fp *FountainPanel) Release() {
	for _, s := range fp.sliders {
		s.Dragging = false
	}
}

func (fp *FountainPanel) IsDragging() bool {
	for _, s := range fp.sliders {
		if s.Dragging {
			return true
		}
	}
	return false
}
