package ui

import (
	"biogo/v2/ui/components"
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	fdPanelW    = float32(300)
	fdPad       = float32(8)
	fdTitleH    = float32(20)
	fdSliderH   = float32(24)
	fdSliderGap = float32(6)
)

// FoodDropdown is a dropdown panel anchored below a menu-bar button that
// exposes sliders for the four food-spawning parameters.
type FoodDropdown struct {
	open    bool
	trigger *components.Button
	sliders []*components.Slider
}

func newFoodDropdown(font *textv2.GoXFace, trigger *components.Button, sim SimulationState) *FoodDropdown {
	p := sim.GetParams()
	sw := fdPanelW - fdPad*2
	trackOff := float32(155)
	trackW := sw - trackOff

	sliders := []*components.Slider{
		{
			W: sw, H: fdSliderH,
			TrackOffX: trackOff, TrackW: trackW,
			Font: font, LabelColor: color.White,
			Min: 0, Max: 200000,
			Value: float64(p.MaxFood),
			FormatFunc: func(v float64) string {
				return fmt.Sprintf("Max Food: %d", int(math.Round(v)))
			},
			OnChange: func(v float64) { sim.SetMaxFood(int(math.Round(v))) },
		},
		{
			W: sw, H: fdSliderH,
			TrackOffX: trackOff, TrackW: trackW,
			Font: font, LabelColor: color.White,
			Min: 0, Max: 1,
			Value: p.FoodRandomFraction,
			FormatFunc: func(v float64) string {
				return fmt.Sprintf("Random: %.1f%%", v*100)
			},
			OnChange: func(v float64) { sim.SetFoodRandomFraction(v) },
		},
		{
			W: sw, H: fdSliderH,
			TrackOffX: trackOff, TrackW: trackW,
			Font: font, LabelColor: color.White,
			Min: 0, Max: 50,
			Value: float64(p.FountainCount),
			FormatFunc: func(v float64) string {
				return fmt.Sprintf("Fountains: %d", int(math.Round(v)))
			},
			OnChange: func(v float64) { sim.SetFountainCount(int(math.Round(v))) },
		},
		{
			W: sw, H: fdSliderH,
			TrackOffX: trackOff, TrackW: trackW,
			Font: font, LabelColor: color.White,
			Min: 0, Max: 100,
			Value: p.FountainDriftSpeed,
			FormatFunc: func(v float64) string {
				return fmt.Sprintf("Drift: %.1f", v)
			},
			OnChange: func(v float64) { sim.SetFountainDriftSpeed(v) },
		},
		{
			W: sw, H: fdSliderH,
			TrackOffX: trackOff, TrackW: trackW,
			Font: font, LabelColor: color.White,
			Min: 50, Max: 3000,
			Value: p.FountainRadius,
			FormatFunc: func(v float64) string {
				return fmt.Sprintf("Radius: %.0f", v)
			},
			OnChange: func(v float64) { sim.SetFountainRadius(v) },
		},
	}

	return &FoodDropdown{trigger: trigger, sliders: sliders}
}

func (d *FoodDropdown) Toggle() { d.open = !d.open }

func (d *FoodDropdown) AnySliderDragging() bool {
	for _, s := range d.sliders {
		if s.Dragging {
			return true
		}
	}
	return false
}

// panelBounds returns the screen-space rectangle of the dropdown panel.
func (d *FoodDropdown) panelBounds() (x, y, w, h float32) {
	bx, _, _, _ := d.trigger.Bounds()
	n := float32(len(d.sliders))
	ph := fdPad*2 + fdTitleH + fdSliderGap + n*fdSliderH + (n-1)*fdSliderGap
	return bx, menuBarH, fdPanelW, ph
}

// HandleClick routes a mouse-down into the panel. Returns true if consumed.
func (d *FoodDropdown) HandleClick(mx, my int) bool {
	if !d.open {
		return false
	}
	px, py, pw, ph := d.panelBounds()
	if float32(mx) < px || float32(mx) >= px+pw || float32(my) < py || float32(my) >= py+ph {
		return false
	}
	for _, s := range d.sliders {
		if s.InBounds(mx, my) {
			s.Dragging = true
			s.UpdateValue(mx)
		}
	}
	return true
}

func (d *FoodDropdown) HandleDrag(mx int) {
	if !d.open {
		return
	}
	for _, s := range d.sliders {
		if s.Dragging {
			s.UpdateValue(mx)
		}
	}
}

func (d *FoodDropdown) HandleRelease() {
	for _, s := range d.sliders {
		s.Dragging = false
	}
}

func (d *FoodDropdown) Draw(screen *ebiten.Image, font *textv2.GoXFace) {
	if !d.open {
		return
	}
	px, py, pw, ph := d.panelBounds()
	vector.FillRect(screen, px, py, pw, ph, color.RGBA{12, 14, 28, 235}, false)
	vector.StrokeRect(screen, px, py, pw, ph, 1, color.RGBA{90, 90, 150, 255}, false)

	if font != nil {
		m := font.Metrics()
		textH := m.HLineGap + m.HAscent + m.HDescent
		ty := float64(py+fdPad) + (float64(fdTitleH)-textH)/2
		op := &textv2.DrawOptions{}
		op.GeoM.Translate(float64(px+fdPad), ty)
		op.ColorScale.ScaleWithColor(color.RGBA{255, 220, 80, 255})
		textv2.Draw(screen, "Food Spawning", font, op)
	}

	sy := py + fdPad + fdTitleH + fdSliderGap
	for _, s := range d.sliders {
		s.Draw(screen, px+fdPad, sy)
		sy += fdSliderH + fdSliderGap
	}
}
