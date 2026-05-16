package components

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	gbBarW  = float32(55) // width of the value bar / swatch area
	gbBarH  = float32(7)  // height of the bar
	gbRowH  = float32(18) // total row height
)

// GenomeBar renders a trait name + numeric value as text, plus a visual bar
// showing the value in the 0–255 range. When Binary is true the bar is replaced
// by two colour swatches indicating asexual (0) or sexual (1).
type GenomeBar struct {
	Name   string
	Val    byte
	Font   *textv2.GoXFace
	W      float32
	Binary bool
}

func (b *GenomeBar) Draw(screen *ebiten.Image, x, y float32) (float32, float32) {
	m := b.Font.Metrics()
	textH := m.HAscent + m.HDescent
	textY := float64(y) + (float64(gbRowH)-textH)/2

	label := fmt.Sprintf("%s: %d", b.Name, b.Val)
	op := &textv2.DrawOptions{}
	op.GeoM.Translate(float64(x), textY)
	op.ColorScale.ScaleWithColor(color.White)
	textv2.Draw(screen, label, b.Font, op)

	barX := x + b.W - gbBarW
	barY := y + (gbRowH-gbBarH)/2

	if b.Binary {
		half := gbBarW / 2
		asexualClr := color.RGBA{200, 80, 30, 180}
		sexualClr := color.RGBA{60, 110, 210, 180}
		dimClr := color.RGBA{25, 25, 50, 160}
		if b.Val == 0 { // asexual
			vector.FillRect(screen, barX, barY, half-1, gbBarH, asexualClr, false)
			vector.FillRect(screen, barX+half, barY, half, gbBarH, dimClr, false)
		} else { // sexual
			vector.FillRect(screen, barX, barY, half-1, gbBarH, dimClr, false)
			vector.FillRect(screen, barX+half, barY, half, gbBarH, sexualClr, false)
		}
	} else {
		vector.FillRect(screen, barX, barY, gbBarW, gbBarH, color.RGBA{20, 20, 45, 200}, false)
		fillW := float32(b.Val) / 255 * gbBarW
		if fillW > 0 {
			vector.FillRect(screen, barX, barY, fillW, gbBarH, color.RGBA{55, 160, 210, 220}, false)
		}
	}

	return b.W, gbRowH
}

func (b *GenomeBar) Size() (float32, float32) {
	return b.W, gbRowH
}
