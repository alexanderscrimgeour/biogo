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

	d := newDropdown(font, trigger, "Food Spawning", ColorLabelPrimary, foodPanelW)

	d.addSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: ColorLabelTargetE,
		Min: 0, Max: 30000000,
		Value: sim.TargetEnergy(),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Target E: %.0fk", v/1000)
		},
		OnChange: func(v float64) { sim.SetTargetEnergy(v) },
	})

	// Ratio of energy deficit allocated to foliage vs fungi.
	foodTotal := p.Food.MaxFoliage + p.Food.MaxFungi
	var initFoliageRatio float64
	if foodTotal > 0 {
		initFoliageRatio = float64(p.Food.MaxFoliage) / float64(foodTotal)
	}

	lerpFoodColor := func(t float64) color.RGBA {
		return color.RGBA{
			R: uint8(float64(ColorFungi.R) + t*(float64(ColorFoliage.R)-float64(ColorFungi.R))),
			G: uint8(float64(ColorFungi.G) + t*(float64(ColorFoliage.G)-float64(ColorFungi.G))),
			B: uint8(float64(ColorFungi.B) + t*(float64(ColorFoliage.B)-float64(ColorFungi.B))),
			A: 255,
		}
	}

	ratioSlider := &components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: color.White,
		FillColor: lerpFoodColor(initFoliageRatio),
		Min:       0, Max: 1,
		Value: initFoliageRatio,
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Fo:%.0f%% Fu:%.0f%%", v*100, (1-v)*100)
		},
	}
	ratioSlider.OnChange = func(v float64) {
		ratioSlider.FillColor = lerpFoodColor(v)
		sim.SetMaxFoliage(int(math.Round(v * float64(foodTotal))))
		sim.SetMaxFungi(foodTotal - int(math.Round(v*float64(foodTotal))))
	}
	d.addSlider(ratioSlider)

	d.addFountainPanel(newFountainPanel(font, sw, trackOff, trackW, sliderH,
		"Foliage", ColorFoliage, p.Food.Foliage.Count, p.Food.Foliage.DriftSpeed,
		p.Food.Foliage.Radius, p.Food.Foliage.RandomFraction,
		sim.SetFoliageFountainCount, sim.SetFoliageDriftSpeed,
		sim.SetFoliageRadius, sim.SetFoliageRandomFraction,
	))

	d.addFountainPanel(newFountainPanel(font, sw, trackOff, trackW, sliderH,
		"Fungi", ColorFungi, p.Food.Fungi.Count, p.Food.Fungi.DriftSpeed,
		p.Food.Fungi.Radius, p.Food.Fungi.RandomFraction,
		sim.SetFungiFountainCount, sim.SetFungiDriftSpeed,
		sim.SetFungiRadius, sim.SetFungiRandomFraction,
	))

	return d
}

func newFountainPanel(
	font *textv2.GoXFace,
	sw, trackOff, trackW, sliderH float32,
	label string, labelColor color.Color,
	initCount int, initDrift, initRadius, initRandom float64,
	onCount func(int), onDrift, onRadius, onRandom func(float64),
) *components.FountainPanel {
	count := &components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: labelColor,
		Min: 0, Max: 50,
		Value: float64(initCount),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Founts: %d", int(math.Round(v)))
		},
		OnChange: func(v float64) { onCount(int(math.Round(v))) },
	}
	drift := &components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: labelColor,
		Min: 0, Max: 10,
		Value: initDrift,
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Drift: %.2f", v)
		},
		OnChange: onDrift,
	}
	radius := &components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: labelColor,
		Min: 50, Max: 500,
		Value: initRadius,
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Radius: %.0f", v)
		},
		OnChange: onRadius,
	}
	random := &components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: labelColor,
		Min: 0, Max: 1,
		Value: initRandom,
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Random: %.1f%%", v*100)
		},
		OnChange: onRandom,
	}
	return components.NewFountainPanel(label, labelColor, font, count, drift, radius, random)
}
