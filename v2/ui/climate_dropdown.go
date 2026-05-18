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
	trackOff := float32(150)
	trackW := sw - trackOff
	sliderH := float32(24)

	d := newDropdown(font, trigger, "Climate", color.RGBA{120, 200, 255, 255}, climatePanelW)

	d.addSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: color.RGBA{120, 200, 255, 255},
		Min: 0, Max: 45,
		Value: float64(p.Environment.TempMin),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Min Temp: %.0f°C", v)
		},
		OnChange: func(v float64) { sim.SetTempMin(float32(v)) },
	})

	d.addSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: color.RGBA{255, 120, 60, 255},
		Min: 5, Max: 100,
		Value: float64(p.Environment.TempMax),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Max Temp: %.0f°C", v)
		},
		OnChange: func(v float64) { sim.SetTempMax(float32(v)) },
	})

	d.addSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: color.RGBA{100, 180, 255, 255},
		Min: 0.05, Max: 1.0,
		Value: float64(p.Environment.ColdSpeedMultiplier),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Cold Speed: %.2fx", v)
		},
		OnChange: func(v float64) { sim.SetColdSpeedMultiplier(float32(v)) },
	})

	d.addSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: color.RGBA{255, 160, 60, 255},
		Min: 1.0, Max: 10.0,
		Value: float64(p.Environment.WarmMetabolicMultiplier),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Warm BMR: %.1fx", v)
		},
		OnChange: func(v float64) { sim.SetWarmMetabolicMultiplier(float32(v)) },
	})

	return d
}
