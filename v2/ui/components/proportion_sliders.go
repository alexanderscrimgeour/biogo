package components

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
)

const psGap = float32(2)

// ProportionSliders renders three vertically-stacked sliders whose values
// always sum to 1. Adjusting one slider rescales the other two proportionally.
type ProportionSliders struct {
	sliders [3]*Slider
	vals    [3]float64 // current proportions, always summing to 1
	H       float32
	W       float32
}

// NewProportionSliders creates three linked proportion sliders.
// initVals must have length 3 and should sum to 1; labels and labelColors are
// applied in the same order. onChange is called after every user adjustment
// with the updated (foliage, fungi, meat) proportions.
func NewProportionSliders(
	sw, trackOff, trackW, sliderH float32,
	font *textv2.GoXFace,
	labels [3]string,
	labelColors [3]color.Color,
	initVals [3]float64,
	onChange func(v0, v1, v2 float64),
) *ProportionSliders {
	ps := &ProportionSliders{vals: initVals}

	for i := range ps.sliders {
		idx := i
		ps.sliders[i] = &Slider{
			W: sw, H: sliderH,
			TrackOffX: trackOff, TrackW: trackW,
			Font: font, LabelColor: labelColors[i],
			Min:   0,
			Max:   1,
			Value: initVals[i],
			FormatFunc: func(v float64) string {
				return fmt.Sprintf("%s: %.0f%%", labels[idx], v*100)
			},
		}
	}

	for i := range ps.sliders {
		idx := i
		ps.sliders[i].OnChange = func(newVal float64) {
			old := ps.vals[idx]
			ps.vals[idx] = newVal
			delta := newVal - old
			// Rescale the other two proportionally to absorb delta.
			j, k := (idx+1)%3, (idx+2)%3
			otherSum := ps.vals[j] + ps.vals[k]
			if otherSum > 1e-9 {
				scale := (otherSum - delta) / otherSum
				if scale < 0 {
					scale = 0
				}
				ps.vals[j] *= scale
				ps.vals[k] *= scale
			} else {
				// Both others were zero — split freed space evenly.
				freed := 1.0 - newVal
				if freed < 0 {
					freed = 0
				}
				ps.vals[j] = freed / 2
				ps.vals[k] = freed / 2
			}
			// Clamp and renormalise to guard against float drift.
			total := ps.vals[0] + ps.vals[1] + ps.vals[2]
			if total > 1e-9 {
				for n := range ps.vals {
					if ps.vals[n] < 0 {
						ps.vals[n] = 0
					}
					ps.vals[n] /= total
				}
			}
			// Sync slider display values (not via OnChange).
			for n, s := range ps.sliders {
				if n != idx {
					s.Value = ps.vals[n]
				}
			}
			onChange(ps.vals[0], ps.vals[1], ps.vals[2])
		}
	}

	h := float32(0)
	for i, s := range ps.sliders {
		if i > 0 {
			h += psGap
		}
		h += s.H
	}
	ps.H = h
	ps.W = sw
	return ps
}

func (ps *ProportionSliders) Draw(screen *ebiten.Image, x, y float32) (float32, float32) {
	cy := y
	for i, s := range ps.sliders {
		if i > 0 {
			cy += psGap
		}
		s.Draw(screen, x, cy)
		cy += s.H
	}
	return ps.W, ps.H
}

func (ps *ProportionSliders) Size() (float32, float32) { return ps.W, ps.H }

func (ps *ProportionSliders) HandleDown(mx, my int) {
	for _, s := range ps.sliders {
		if s.InBounds(mx, my) {
			s.Dragging = true
			s.UpdateValue(mx)
		}
	}
}

func (ps *ProportionSliders) HandleDrag(mx int) {
	for _, s := range ps.sliders {
		if s.Dragging {
			s.UpdateValue(mx)
		}
	}
}

func (ps *ProportionSliders) Release() {
	for _, s := range ps.sliders {
		s.Dragging = false
	}
}

func (ps *ProportionSliders) IsDragging() bool {
	for _, s := range ps.sliders {
		if s.Dragging {
			return true
		}
	}
	return false
}
