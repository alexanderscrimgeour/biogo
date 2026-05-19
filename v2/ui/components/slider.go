package components

import (
	"biogo/v2/ui/colors"
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Slider is a horizontal range-selection control.
// TrackOffX is the horizontal offset from the slider's left edge to where the track begins,
// allowing a label to occupy the left portion.
// It stores its last drawn position for hit and drag detection.
type Slider struct {
	W, H       float32
	TrackOffX  float32
	TrackW     float32
	Label      string
	Color      color.RGBA
	LabelColor color.Color
	FillColor  color.RGBA // overrides default track fill when A > 0
	Min, Max   float64
	Value      float64
	Dragging   bool
	Font       *textv2.GoXFace
	OnChange   func(float64)
	FormatFunc func(float64) string // optional: overrides default label formatting
	lastX      float32
	lastY      float32
}

// Draw renders the slider at (x, y) and returns (W, H).
func (s *Slider) Draw(screen *ebiten.Image, x, y float32) (float32, float32) {
	s.lastX, s.lastY = x, y
	trackX := x + s.TrackOffX
	trackY := y + (s.H-6)/2

	vector.FillRect(screen, x, y, s.W, s.H, colors.ColorSliderBG, false)
	vector.FillRect(screen, trackX, trackY, s.TrackW, 6, colors.ColorTrackBG, false)

	t := (s.Value - s.Min) / (s.Max - s.Min)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	fillClr := color.RGBA(colors.ColorTrackFill)
	if s.FillColor.A > 0 {
		fillClr = s.FillColor
	}
	vector.FillRect(screen, trackX, trackY, s.TrackW*float32(t), 6, fillClr, false)

	if s.Font != nil {
		var lbl string
		if s.FormatFunc != nil {
			lbl = s.FormatFunc(s.Value)
		} else if s.Label != "" {
			lbl = fmt.Sprintf("%s: %.4f", s.Label, s.Value)
		} else {
			lbl = fmt.Sprintf("%.4f", s.Value)
		}
		m := s.Font.Metrics()
		textH := m.HLineGap + m.HAscent + m.HDescent
		ty := float64(y) + (float64(s.H)-textH)/2
		op := &textv2.DrawOptions{}
		op.GeoM.Translate(float64(x)+5, ty)
		lc := s.LabelColor
		if lc == nil {
			lc = color.White
		}
		op.ColorScale.ScaleWithColor(lc)
		textv2.Draw(screen, lbl, s.Font, op)
	}
	return s.W, s.H
}

// Size returns preferred dimensions for layout.
func (s *Slider) Size() (float32, float32) {
	return s.W, s.H
}

// InBounds reports whether (mx, my) falls within the last drawn slider area.
func (s *Slider) InBounds(mx, my int) bool {
	return float32(mx) >= s.lastX && float32(mx) < s.lastX+s.W &&
		float32(my) >= s.lastY && float32(my) < s.lastY+s.H
}

// UpdateValue sets Value from a mouse X position relative to the track.
func (s *Slider) UpdateValue(mx int) {
	trackX := s.lastX + s.TrackOffX
	t := float64(float32(mx)-trackX) / float64(s.TrackW)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	s.Value = s.Min + t*(s.Max-s.Min)
	if s.OnChange != nil {
		s.OnChange(s.Value)
	}
}
