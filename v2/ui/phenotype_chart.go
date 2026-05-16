package ui

import (
	"biogo/v2/simulation"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	phenoChartSize      = float32(64)
	phenoBarWidth       = float32(10)
	phenoBarSpacing     = float32(14)
	phenoBarLabelOffset = float32(14)
	phenoTitleH         = float32(40) // space for title above chart
	phenoLabelH         = float32(18) // space for bar labels below chart
)

// PhenotypeChart renders the genetic profile visualisation (gradient square + bar meters).
// It implements components.Component and can be embedded in a Panel.
type PhenotypeChart struct {
	Font *textv2.GoXFace
	Data simulation.CreatureDetailView
}

// Draw renders the phenotype chart at (x, y) and returns (width, height).
func (c *PhenotypeChart) Draw(screen *ebiten.Image, x, y float32) (float32, float32) {
	w, h := c.Size()

	textHeight := phenoTitleH
	if c.Font != nil {
		metrics := c.Font.Metrics()
		textHeight = float32(metrics.HAscent + metrics.HDescent)
		drawText(screen, "GENETIC PROFILE", c.Font, int(x), int(y+(14-textHeight/2)), color.RGBA{120, 120, 180, 255})
	}
	chartY := y + textHeight
	step := float32(2)

	for gy := float32(0); gy < phenoChartSize; gy += step {
		gPercInv := 1.0 - (gy / phenoChartSize)
		gVal := uint8(gPercInv*185) + 70

		for rx := float32(0); rx < phenoChartSize; rx += step {
			rVal := uint8((rx/phenoChartSize)*185) + 70
			vector.FillRect(screen, x+rx, chartY+gy, step, step, color.RGBA{rVal, gVal, c.Data.B, 255}, false)
		}
	}

	rPerc := clamp((float32(c.Data.R) - 70) / 185)
	gPerc := clamp((float32(c.Data.G) - 70) / 185)

	cx := x + (rPerc * phenoChartSize)
	cy := chartY + (phenoChartSize - (gPerc * phenoChartSize))

	vector.StrokeLine(screen, cx, chartY, cx, chartY+phenoChartSize, 1, color.White, false)
	vector.StrokeLine(screen, x, cy, x+phenoChartSize, cy, 1, color.White, false)

	mx := x + phenoChartSize + 15
	meters := []struct {
		val float32
		clr color.RGBA
		lbl string
	}{
		{gPerc, color.RGBA{100, 255, 100, 255}, "I"},
		{rPerc, color.RGBA{255, 100, 100, 255}, "P"},
		{clamp((float32(c.Data.B) - 70) / 185), color.RGBA{100, 150, 255, 255}, "S"},
		{clamp(c.Data.MutationPct / 1), color.RGBA{255, 100, 255, 255}, "M"},
	}

	for i, m := range meters {
		bx := mx + float32(i)*phenoBarSpacing
		vector.FillRect(screen, bx, chartY, phenoBarWidth, phenoChartSize, color.RGBA{20, 20, 25, 255}, false)
		vector.FillRect(screen, bx, chartY+(phenoChartSize-(m.val*phenoChartSize)), phenoBarWidth, m.val*phenoChartSize, m.clr, false)

		if c.Font != nil {
			metrics := c.Font.Metrics()
			textHeight := float32(metrics.HAscent + metrics.HDescent)

			tw, _ := textv2.Measure(m.lbl, c.Font, 0)
			lblW := float32(tw)

			tx := bx + (phenoBarWidth-lblW)/2
			ty := chartY + phenoChartSize + float32(phenoBarLabelOffset) - textHeight/2

			drawText(screen, m.lbl, c.Font, int(tx), int(ty), m.clr)
		}
	}

	return w, h
}

// Size returns the preferred dimensions of the chart component.
func (c *PhenotypeChart) Size() (float32, float32) {
	textHeight := phenoTitleH
	if c.Font != nil {
		metrics := c.Font.Metrics()
		textHeight = float32(metrics.HAscent + metrics.HDescent)
	}
	w := phenoChartSize + 15 + phenoBarWidth + phenoBarSpacing*3
	h := textHeight + phenoChartSize + phenoLabelH
	return w, h
}
