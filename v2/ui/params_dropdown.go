package ui

import (
	"biogo/v2/ui/colors"
	"biogo/v2/ui/components"
	"fmt"

	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
)

const paramsPanelW = float32(300)

func newParamsDropdown(font *textv2.GoXFace, trigger *components.Button, sim SimulationState) *components.Dropdown {
	p := sim.Params()
	sw := paramsPanelW - ddPad*2
	trackOff := float32(155)
	trackW := sw - trackOff
	sliderH := float32(24)

	d := components.NewDropdown(font, trigger, "Parameters", colors.ColorLabelInfo, paramsPanelW)

	d.AddSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: colors.ColorLabelTargetE,
		Min: 0, Max: 30_000_000,
		Value: sim.TargetEnergy(),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Tgt E: %.0fk", v/1000)
		},
		OnChange: func(v float64) { sim.SetTargetEnergy(v) },
	})

	d.AddSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: colors.ColorLabelSecondary,
		Min: 0.001, Max: 0.09,
		Value: float64(p.Neurology.BaseMutationRate),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Mutation: %.3f", v)
		},
		OnChange: func(v float64) { sim.SetBaseMutationRate(float32(v)) },
	})

	return d
}
