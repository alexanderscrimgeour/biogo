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
		Font: font, LabelColor: color.RGBA{255, 230, 50, 255},
		Min: 0, Max: 20000000,
		Value: sim.TargetEnergy(),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Target E: %.0fk", v/1000)
		},
		OnChange: func(v float64) { sim.SetTargetEnergy(v) },
	})

	// Ratio of energy deficit allocated to foliage vs fungi.
	// The absolute values of MaxFoliage/MaxFungi only matter as a ratio.
	foodTotal := p.Food.MaxFoliage + p.Food.MaxFungi
	var initFoliageRatio float64
	if foodTotal > 0 {
		initFoliageRatio = float64(p.Food.MaxFoliage) / float64(foodTotal)
	}

	foliageColor := color.RGBA{65, 180, 55, 255}
	fungiColor := color.RGBA{160, 80, 200, 255}
	lerpFoodColor := func(t float64) color.RGBA {
		// t=1 → foliage green, t=0 → fungi purple
		return color.RGBA{
			R: uint8(float64(fungiColor.R) + t*(float64(foliageColor.R)-float64(fungiColor.R))),
			G: uint8(float64(fungiColor.G) + t*(float64(foliageColor.G)-float64(fungiColor.G))),
			B: uint8(float64(fungiColor.B) + t*(float64(foliageColor.B)-float64(fungiColor.B))),
			A: 255,
		}
	}

	ratioSlider := &components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: lerpFoodColor(initFoliageRatio),
		Min: 0, Max: 1,
		Value: initFoliageRatio,
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("F:%.0f%% Fu:%.0f%%", v*100, (1-v)*100)
		},
	}
	ratioSlider.OnChange = func(v float64) {
		ratioSlider.LabelColor = lerpFoodColor(v)
		sim.SetMaxFoliage(int(math.Round(v * float64(foodTotal))))
		sim.SetMaxFungi(foodTotal - int(math.Round(v*float64(foodTotal))))
	}
	d.addSlider(ratioSlider)
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
