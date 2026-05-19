package ui

import (
	"biogo/v2/ui/components"
	"fmt"
	"image/color"
	"math"

	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
)

const spawnPanelW = float32(300)

func newSpawnDropdown(font *textv2.GoXFace, trigger *components.Button, sim SimulationState, onSpawnSaved func()) *Dropdown {
	p := sim.GetParams()
	sw := spawnPanelW - ddPad*2
	trackOff := float32(155)
	trackW := sw - trackOff
	sliderH := float32(24)

	d := newDropdown(font, trigger, "Cluster Spawn", ColorSpawnTitle, spawnPanelW)

	spawnSavedBtn := &components.Button{
		W: sw, H: 26,
		Label:      "Spawn Saved Genome",
		Color:      components.ColorDefault,
		LabelColor: color.White,
		Font:       font,
	}
	spawnSavedBtn.OnClick = onSpawnSaved
	d.addButton(spawnSavedBtn)

	enableBtn := &components.Button{
		W: sw, H: 26,
		Font:       font,
		LabelColor: color.White,
	}
	setEnableBtnState := func() {
		if p.Spawn.ClusterEnabled {
			enableBtn.Label = "Cluster Spawn: Enabled"
			enableBtn.Color = components.ColorButtonGreen
		} else {
			enableBtn.Label = "Cluster Spawn: Disabled"
			enableBtn.Color = components.ColorDefault
		}
	}
	setEnableBtnState()
	enableBtn.OnClick = func() {
		sim.SetClusterEnabled(!p.Spawn.ClusterEnabled)
		setEnableBtnState()
	}
	d.addButton(enableBtn)

	d.addSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: color.White,
		Min: 100, Max: 10000,
		Value: float64(p.Spawn.ClusterInterval),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Interval: %d ticks", int(math.Round(v)))
		},
		OnChange: func(v float64) { sim.SetClusterInterval(int(math.Round(v))) },
	})

	d.addSlider(&components.Slider{
		W: sw, H: sliderH,
		TrackOffX: trackOff, TrackW: trackW,
		Font: font, LabelColor: color.White,
		Min: 1, Max: 50,
		Value: float64(p.Spawn.ClusterSize),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Cluster Size: %d", int(math.Round(v)))
		},
		OnChange: func(v float64) { sim.SetClusterSize(int(math.Round(v))) },
	})

	return d
}
