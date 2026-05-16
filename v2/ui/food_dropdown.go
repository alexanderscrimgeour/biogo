package ui

import (
	"biogo/v2/ui/components"
	"fmt"
	"image/color"
	"math"

	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
)

const foodPanelW = float32(300)

func newFoodDropdown(font *textv2.GoXFace, trigger *components.Button, sim SimulationState) *Dropdown {
	p := sim.GetParams()
	sw := foodPanelW - ddPad*2
	trackOff := float32(155)
	trackW := sw - trackOff
	sliderH := float32(24)

	d := newDropdown(font, trigger, "Food Spawning", color.RGBA{255, 220, 80, 255}, foodPanelW)

	d.addSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: color.White,
		Min: 0, Max: 500000,
		Value: float64(p.Food.Max),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Max Food: %d", int(math.Round(v)))
		},
		OnChange: func(v float64) { sim.SetMaxFood(int(math.Round(v))) },
	})
	d.addSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: color.White,
		Min: 0, Max: 1,
		Value: p.Food.RandomFraction,
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Random: %.1f%%", v*100)
		},
		OnChange: func(v float64) { sim.SetFoodRandomFraction(v) },
	})
	d.addSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: color.White,
		Min: 0, Max: 50,
		Value: float64(p.Food.FountainCount),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Fountains: %d", int(math.Round(v)))
		},
		OnChange: func(v float64) { sim.SetFountainCount(int(math.Round(v))) },
	})
	d.addSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: color.White,
		Min: 0, Max: 10,
		Value: p.Food.FountainDriftSpeed,
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Drift: %.1f", v)
		},
		OnChange: func(v float64) { sim.SetFountainDriftSpeed(v) },
	})
	d.addSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: color.White,
		Min: 50, Max: 500,
		Value: p.Food.FountainRadius,
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Radius: %.0f", v)
		},
		OnChange: func(v float64) { sim.SetFountainRadius(v) },
	})

	return d
}
