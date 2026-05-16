package components

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// TextInputField is a single-line text input. Text and Focused are set each frame
// before the panel is built; lastX/lastY track the drawn position for hit testing.
type TextInputField struct {
	W, H         float32
	Text         string
	Focused      bool
	Placeholder  string
	Font         *textv2.GoXFace
	lastX, lastY float32
}

// Draw renders the input box at (x, y) and returns (W, H).
func (t *TextInputField) Draw(screen *ebiten.Image, x, y float32) (float32, float32) {
	t.lastX, t.lastY = x, y
	borderClr := color.RGBA{55, 55, 90, 200}
	if t.Focused {
		borderClr = color.RGBA{100, 110, 210, 255}
	}
	vector.FillRect(screen, x, y, t.W, t.H, color.RGBA{14, 14, 32, 230}, false)
	vector.StrokeRect(screen, x, y, t.W, t.H, 1, borderClr, false)

	display := t.Text
	var textClr color.Color = color.White
	if t.Focused {
		display += "|"
	} else if t.Text == "" {
		display = t.Placeholder
		textClr = color.RGBA{80, 80, 110, 200}
	}

	if t.Font != nil && display != "" {
		metrics := t.Font.Metrics()
		textHeight := float32(metrics.HLineGap + metrics.HAscent + metrics.HDescent)

		tx := x + 6
		ty := y + (t.H-textHeight)/2

		op := &textv2.DrawOptions{}
		op.GeoM.Translate(float64(int(tx)), float64(int(ty)))
		op.ColorScale.ScaleWithColor(textClr)
		textv2.Draw(screen, display, t.Font, op)
	}
	return t.W, t.H
}

// Size returns the preferred dimensions of the input field.
func (t *TextInputField) Size() (float32, float32) { return t.W, t.H }

// Contains reports whether (mx, my) falls inside the last drawn bounds.
func (t *TextInputField) Contains(mx, my float32) bool {
	return mx >= t.lastX && mx < t.lastX+t.W && my >= t.lastY && my < t.lastY+t.H
}
