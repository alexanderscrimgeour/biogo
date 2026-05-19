package ui

import (
	"biogo/v2/ui/colors"
	"biogo/v2/ui/components"
	"fmt"
	"image/color"
	"math"

	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
)

const foodPanelW = float32(300)

func newFoodDropdown(font *textv2.GoXFace, trigger *components.Button, sim SimulationState) *components.Dropdown {
	p := sim.GetParams()
	sw := foodPanelW - ddPad*2
	trackOff := float32(155)
	trackW := sw - trackOff
	sliderH := float32(24)

	d := components.NewDropdown(font, trigger, "Food Spawning", colors.ColorLabelPrimary, foodPanelW)

	// Three linked proportion sliders — always sum to 100%.
	d.AddProportionSliders(components.NewProportionSliders(
		sw, trackOff, trackW, sliderH, font,
		[3]string{"Foliage", "Fungi", "Meat"},
		[3]color.Color{colors.ColorFoliage, colors.ColorFungi, colors.ColorMeat},
		[3]float64{p.Food.FoliageProportion, p.Food.FungiProportion, p.Food.MeatProportion},
		func(fo, fu, me float64) {
			sim.SetFoliageProportion(fo)
			sim.SetFungiProportion(fu)
			sim.SetMeatProportion(me)
		},
	))

	d.AddFountainPanel(newFountainPanel(font, sw, trackOff, trackW, sliderH,
		"Foliage", colors.ColorFoliage, p.Food.Foliage.Count, p.Food.Foliage.StationaryCount,
		p.Food.Foliage.DriftSpeed, p.Food.Foliage.Radius, p.Food.Foliage.RandomFraction,
		sim.SetFoliageFountainCount, sim.SetFoliageStationaryCount,
		sim.SetFoliageDriftSpeed, sim.SetFoliageRadius, sim.SetFoliageRandomFraction,
	))

	d.AddFountainPanel(newFountainPanel(font, sw, trackOff, trackW, sliderH,
		"Fungi", colors.ColorFungi, p.Food.Fungi.Count, p.Food.Fungi.StationaryCount,
		p.Food.Fungi.DriftSpeed, p.Food.Fungi.Radius, p.Food.Fungi.RandomFraction,
		sim.SetFungiFountainCount, sim.SetFungiStationaryCount,
		sim.SetFungiDriftSpeed, sim.SetFungiRadius, sim.SetFungiRandomFraction,
	))

	d.AddFountainPanel(newFountainPanel(font, sw, trackOff, trackW, sliderH,
		"Meat", colors.ColorMeat, p.Food.Meat.Count, p.Food.Meat.StationaryCount,
		p.Food.Meat.DriftSpeed, p.Food.Meat.Radius, p.Food.Meat.RandomFraction,
		sim.SetMeatFountainCount, sim.SetMeatStationaryCount,
		sim.SetMeatDriftSpeed, sim.SetMeatRadius, sim.SetMeatRandomFraction,
	))

	return d
}

func newFountainPanel(
	font *textv2.GoXFace,
	sw, trackOff, trackW, sliderH float32,
	label string, labelColor color.Color,
	initCount, initStationary int, initDrift, initRadius, initRandom float64,
	onCount, onStationary func(int), onDrift, onRadius, onRandom func(float64),
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
	stationary := &components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: labelColor,
		Min: 0, Max: 50,
		Value: float64(initStationary),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Fixed: %d", int(math.Round(v)))
		},
		OnChange: func(v float64) { onStationary(int(math.Round(v))) },
	}
	return components.NewFountainPanel(label, labelColor, font, count, drift, radius, random, stationary)
}
