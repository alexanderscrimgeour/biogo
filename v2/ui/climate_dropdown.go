package ui

import (
	"biogo/v2/ui/colors"
	"biogo/v2/ui/components"
	"fmt"

	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	climatePanelW = float32(300)

	ddPad       = float32(8)
	ddTitleH    = float32(20)
	ddSliderGap = float32(6)
)

func newClimateDropdown(font *textv2.GoXFace, trigger *components.Button, sim SimulationState) *components.Dropdown {
	p := sim.GetParams()
	sw := climatePanelW - ddPad*2
	trackOff := float32(150)
	trackW := sw - trackOff
	sliderH := float32(24)

	d := components.NewDropdown(font, trigger, "Climate", colors.ColorClimateCool, climatePanelW)

	d.AddSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: colors.ColorClimateCool,
		Min: 0, Max: 45,
		Value: float64(p.Environment.TempMin),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Min Temp: %.0f°C", v)
		},
		OnChange: func(v float64) { sim.SetTempMin(float32(v)) },
	})

	d.AddSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: colors.ColorClimateHot,
		Min: 5, Max: 100,
		Value: float64(p.Environment.TempMax),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Max Temp: %.0f°C", v)
		},
		OnChange: func(v float64) { sim.SetTempMax(float32(v)) },
	})

	d.AddSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: colors.ColorClimateColdMp,
		Min: 0.05, Max: 1.0,
		Value: float64(p.Environment.ColdSpeedMultiplier),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Cold Speed: %.2fx", v)
		},
		OnChange: func(v float64) { sim.SetColdSpeedMultiplier(float32(v)) },
	})

	d.AddSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: colors.ColorClimateWarmBMR,
		Min: 1.0, Max: 10.0,
		Value: float64(p.Environment.WarmMetabolicMultiplier),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Warm BMR: %.1fx", v)
		},
		OnChange: func(v float64) { sim.SetWarmMetabolicMultiplier(float32(v)) },
	})

	return d
}
