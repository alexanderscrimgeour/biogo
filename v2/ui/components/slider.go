package components

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Slider struct {
	X, Y, W, H     int
	TrackX, TrackW int
	Label          string
	Color          color.RGBA
	LabelColor     color.Gray16
	Min, Max       float64
	Value          float64
	Dragging       bool
}

func (s *Slider) Draw(screen *ebiten.Image, clr color.Color) {
	// 1. Draw the background panel
	vector.DrawFilledRect(screen, float32(s.X), float32(s.Y), float32(s.W), float32(s.H), color.RGBA{30, 30, 50, 220}, false)

	// 2. Draw the track
	vector.DrawFilledRect(screen, float32(s.TrackX), float32(s.Y+9), float32(s.TrackW), 6, color.RGBA{60, 60, 80, 255}, false)

	// 3. Draw the progress fill
	t := (s.Value - s.Min) / (s.Max - s.Min)
	vector.DrawFilledRect(screen, float32(s.TrackX), float32(s.Y+9), float32(s.TrackW)*float32(t), 6, color.RGBA{80, 140, 210, 255}, false)
}

func (s *Slider) InBounds(mx, my int) bool {
	return mx >= s.X && mx < s.X+s.W && my >= s.Y && my < s.Y+s.H
}

func (s *Slider) UpdateValue(mx int) {
	t := float64(mx-s.TrackX) / float64(s.TrackW)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	s.Value = s.Min + t*(s.Max-s.Min)
}
