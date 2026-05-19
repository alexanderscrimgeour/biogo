package components

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Button is a clickable UI element. It stores its last drawn position
// so IsClicked works without requiring absolute coordinates at construction time.
type Button struct {
	W, H       float32
	Label      string
	Color      color.RGBA
	LabelColor color.Color
	Font       *textv2.GoXFace
	OnClick    func()
	Active     bool // when true the button renders with a highlighted (pressed) appearance
	lastX      float32
	lastY      float32
}

// Draw renders the button at (x, y) and returns (W, H).
func (b *Button) Draw(screen *ebiten.Image, x, y float32) (float32, float32) {
	b.lastX, b.lastY = x, y
	displayColor := b.Color
	if b.Active {
		displayColor = lighten(b.Color, 0.25)
	} else {
		mx, my := ebiten.CursorPosition()
		isHovered := float32(mx) >= x && float32(mx) <= x+b.W && float32(my) >= y && float32(my) <= y+b.H
		if isHovered && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			displayColor = lighten(b.Color, 0.1)
		}
	}

	vector.FillRect(screen, x, y, b.W, b.H, displayColor, false)

	if b.Font != nil && b.Label != "" {
		metrics := b.Font.Metrics()
		textHeight := metrics.HLineGap + metrics.HAscent + metrics.HDescent

		op := &textv2.DrawOptions{}
		op.GeoM.Translate(float64(x)+5, float64(y)+(float64(b.H)-textHeight)/2)

		lc := b.LabelColor
		if lc == nil {
			lc = color.White
		}
		op.ColorScale.ScaleWithColor(lc)
		textv2.Draw(screen, b.Label, b.Font, op)
	}

	return b.W, b.H
}

// Size returns the button's preferred dimensions for layout.
func (b *Button) Size() (float32, float32) {
	return b.W, b.H
}

// IsClicked returns true if (mx, my) falls within the last drawn bounds.
func (b *Button) IsClicked(mx, my int) bool {
	return float32(mx) >= b.lastX && float32(mx) < b.lastX+b.W &&
		float32(my) >= b.lastY && float32(my) < b.lastY+b.H
}

// Bounds returns the last drawn x, y, width, height.
func (b *Button) Bounds() (float32, float32, float32, float32) {
	return b.lastX, b.lastY, b.W, b.H
}
