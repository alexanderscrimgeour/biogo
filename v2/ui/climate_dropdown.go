package ui

import (
	"biogo/v2/ui/components"
	"fmt"
	"image/color"

	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
)

const climatePanelW = float32(300)

func newClimateDropdown(font *textv2.GoXFace, trigger *components.Button, sim SimulationState) *Dropdown {
	p := sim.GetParams()
	sw := climatePanelW - ddPad*2
	trackOff := float32(130)
	trackW := sw - trackOff
	sliderH := float32(28)

	d := newDropdown(font, trigger, "Climate", color.RGBA{120, 200, 255, 255}, climatePanelW)

	d.addRangeSlider(&components.RangeSlider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: color.White,
		Min: 0.1, Max: 8.0,
		Lo: float64(p.Environment.ColdMetabolicMultiplier),
		Hi: float64(p.Environment.WarmMetabolicMultiplier),
		FormatFunc: func(lo, hi float64) string {
			return fmt.Sprintf("Cold:%.2f Warm:%.2f", lo, hi)
		},
		OnChange: func(lo, hi float64) {
			sim.SetColdMetabolicMultiplier(float32(lo))
			sim.SetWarmMetabolicMultiplier(float32(hi))
		},
	})

	return d
}
