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
		Font: font, LabelColor: color.RGBA{65, 180, 55, 255},
		Min: 0, Max: 1000000,
		Value: float64(p.Food.MaxFoliage),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Max Foliage: %d", int(math.Round(v)))
		},
		OnChange: func(v float64) { sim.SetMaxFoliage(int(math.Round(v))) },
	})
	d.addSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: color.RGBA{160, 80, 200, 255},
		Min: 0, Max: 1000000,
		Value: float64(p.Food.MaxFungi),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Max Fungi: %d", int(math.Round(v)))
		},
		OnChange: func(v float64) { sim.SetMaxFungi(int(math.Round(v))) },
	})
	d.addSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: color.RGBA{65, 180, 55, 255},
		Min: 0, Max: 1,
		Value: p.Food.FoliageRandomFraction,
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Foliage Rnd: %.1f%%", v*100)
		},
		OnChange: func(v float64) { sim.SetFoliageRandomFraction(v) },
	})
	d.addSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: color.RGBA{160, 80, 200, 255},
		Min: 0, Max: 1,
		Value: p.Food.FungiRandomFraction,
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Fungi Rnd: %.1f%%", v*100)
		},
		OnChange: func(v float64) { sim.SetFungiRandomFraction(v) },
	})
	d.addSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: color.RGBA{65, 180, 55, 255},
		Min: 0, Max: 50,
		Value: float64(p.Food.FoliageFountainCount),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Foliage Founts: %d", int(math.Round(v)))
		},
		OnChange: func(v float64) { sim.SetFoliageFountainCount(int(math.Round(v))) },
	})
	d.addSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: color.RGBA{160, 80, 200, 255},
		Min: 0, Max: 50,
		Value: float64(p.Food.FungiFountainCount),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Fungi Founts: %d", int(math.Round(v)))
		},
		OnChange: func(v float64) { sim.SetFungiFountainCount(int(math.Round(v))) },
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
