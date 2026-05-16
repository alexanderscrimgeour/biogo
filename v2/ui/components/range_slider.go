package components

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const rsThumbR = float32(5)

// RangeSlider is a dual-handle horizontal slider that enforces Lo < Hi.
// Lo (left thumb, blue) and Hi (right thumb, orange) can be dragged independently.
type RangeSlider struct {
	W, H       float32
	TrackOffX  float32
	TrackW     float32
	LabelColor color.Color
	Font       *textv2.GoXFace
	Min, Max   float64
	Lo, Hi     float64
	FormatFunc func(lo, hi float64) string
	OnChange   func(lo, hi float64)

	draggingLo   bool
	draggingHi   bool
	lastX, lastY float32
}

func (s *RangeSlider) Size() (float32, float32) { return s.W, s.H }

func (s *RangeSlider) Draw(screen *ebiten.Image, x, y float32) (float32, float32) {
	s.lastX, s.lastY = x, y
	trackX := x + s.TrackOffX
	trackY := y + (s.H-6)/2
	thumbCY := y + s.H/2

	vector.FillRect(screen, x, y, s.W, s.H, color.RGBA{30, 30, 50, 220}, false)
	vector.FillRect(screen, trackX, trackY, s.TrackW, 6, color.RGBA{60, 60, 80, 255}, false)

	loT := float32(rsClamp01((s.Lo - s.Min) / (s.Max - s.Min)))
	hiT := float32(rsClamp01((s.Hi - s.Min) / (s.Max - s.Min)))
	vector.FillRect(screen, trackX+loT*s.TrackW, trackY, (hiT-loT)*s.TrackW, 6, color.RGBA{80, 140, 210, 255}, false)

	vector.DrawFilledCircle(screen, trackX+loT*s.TrackW, thumbCY, rsThumbR, color.RGBA{80, 160, 255, 255}, false)
	vector.DrawFilledCircle(screen, trackX+hiT*s.TrackW, thumbCY, rsThumbR, color.RGBA{255, 160, 60, 255}, false)

	if s.Font != nil && s.FormatFunc != nil {
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
		textv2.Draw(screen, s.FormatFunc(s.Lo, s.Hi), s.Font, op)
	}
	return s.W, s.H
}

// HandleMouseDown tries to grab a thumb near (mx, my). Returns true if a thumb was hit.
func (s *RangeSlider) HandleMouseDown(mx, my int) bool {
	thumbCY := s.lastY + s.H/2
	trackX := s.lastX + s.TrackOffX
	loT := float32(rsClamp01((s.Lo - s.Min) / (s.Max - s.Min)))
	hiT := float32(rsClamp01((s.Hi - s.Min) / (s.Max - s.Min)))
	lx := trackX + loT*s.TrackW
	hx := trackX + hiT*s.TrackW

	if rsInCircle(float32(mx), float32(my), lx, thumbCY, rsThumbR+3) {
		s.draggingLo = true
		return true
	}
	if rsInCircle(float32(mx), float32(my), hx, thumbCY, rsThumbR+3) {
		s.draggingHi = true
		return true
	}
	return false
}

func (s *RangeSlider) HandleDrag(mx int) {
	if !s.draggingLo && !s.draggingHi {
		return
	}
	trackX := s.lastX + s.TrackOffX
	t := rsClamp01(float64(float32(mx)-trackX) / float64(s.TrackW))
	v := s.Min + t*(s.Max-s.Min)
	const minGap = 0.05
	if s.draggingLo {
		if v > s.Hi-minGap {
			v = s.Hi - minGap
		}
		s.Lo = v
	} else {
		if v < s.Lo+minGap {
			v = s.Lo + minGap
		}
		s.Hi = v
	}
	if s.OnChange != nil {
		s.OnChange(s.Lo, s.Hi)
	}
}

func (s *RangeSlider) Release()     { s.draggingLo = false; s.draggingHi = false }
func (s *RangeSlider) IsDragging() bool { return s.draggingLo || s.draggingHi }

func rsClamp01(t float64) float64 {
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}

func rsInCircle(px, py, cx, cy, r float32) bool {
	dx, dy := px-cx, py-cy
	return dx*dx+dy*dy <= r*r
}
