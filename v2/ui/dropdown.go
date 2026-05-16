package ui

import (
	"biogo/v2/ui/components"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	ddPad      = float32(8)
	ddTitleH   = float32(20)
	ddSliderGap = float32(6)
)

type ddItem struct {
	h       float32
	draw    func(screen *ebiten.Image, x, y float32)
	onDown  func(mx, my int)
	onDrag  func(mx int)
	release func()
	dragging func() bool
}

// Dropdown is a panel anchored below a menu-bar button containing sliders and range sliders.
type Dropdown struct {
	open       bool
	trigger    *components.Button
	title      string
	titleColor color.Color
	panelW     float32
	font       *textv2.GoXFace
	items      []ddItem
}

func newDropdown(font *textv2.GoXFace, trigger *components.Button, title string, titleColor color.Color, panelW float32) *Dropdown {
	return &Dropdown{
		trigger:    trigger,
		title:      title,
		titleColor: titleColor,
		panelW:     panelW,
		font:       font,
	}
}

func (d *Dropdown) addSlider(s *components.Slider) {
	d.items = append(d.items, ddItem{
		h:    s.H,
		draw: func(screen *ebiten.Image, x, y float32) { s.Draw(screen, x, y) },
		onDown: func(mx, my int) {
			if s.InBounds(mx, my) {
				s.Dragging = true
				s.UpdateValue(mx)
			}
		},
		onDrag:   func(mx int) { if s.Dragging { s.UpdateValue(mx) } },
		release:  func() { s.Dragging = false },
		dragging: func() bool { return s.Dragging },
	})
}

func (d *Dropdown) addRangeSlider(s *components.RangeSlider) {
	d.items = append(d.items, ddItem{
		h:        s.H,
		draw:     func(screen *ebiten.Image, x, y float32) { s.Draw(screen, x, y) },
		onDown:   func(mx, my int) { s.HandleMouseDown(mx, my) },
		onDrag:   func(mx int) { s.HandleDrag(mx) },
		release:  s.Release,
		dragging: s.IsDragging,
	})
}

func (d *Dropdown) Toggle() { d.open = !d.open }

func (d *Dropdown) AnyDragging() bool {
	for _, item := range d.items {
		if item.dragging() {
			return true
		}
	}
	return false
}

func (d *Dropdown) panelBounds() (x, y, w, h float32) {
	bx, _, _, _ := d.trigger.Bounds()
	ph := ddPad*2 + ddTitleH
	for _, item := range d.items {
		ph += ddSliderGap + item.h
	}
	return bx, menuBarH, d.panelW, ph
}

// HandleClick routes a mouse-down into the panel. Returns true if consumed.
func (d *Dropdown) HandleClick(mx, my int) bool {
	if !d.open {
		return false
	}
	px, py, pw, ph := d.panelBounds()
	if float32(mx) < px || float32(mx) >= px+pw || float32(my) < py || float32(my) >= py+ph {
		return false
	}
	for _, item := range d.items {
		item.onDown(mx, my)
	}
	return true
}

func (d *Dropdown) HandleDrag(mx int) {
	if !d.open {
		return
	}
	for _, item := range d.items {
		item.onDrag(mx)
	}
}

func (d *Dropdown) HandleRelease() {
	for _, item := range d.items {
		item.release()
	}
}

func (d *Dropdown) Draw(screen *ebiten.Image) {
	if !d.open {
		return
	}
	px, py, pw, ph := d.panelBounds()
	vector.FillRect(screen, px, py, pw, ph, color.RGBA{12, 14, 28, 235}, false)
	vector.StrokeRect(screen, px, py, pw, ph, 1, color.RGBA{90, 90, 150, 255}, false)

	if d.font != nil {
		m := d.font.Metrics()
		textH := m.HLineGap + m.HAscent + m.HDescent
		ty := float64(py+ddPad) + (float64(ddTitleH)-textH)/2
		op := &textv2.DrawOptions{}
		op.GeoM.Translate(float64(px+ddPad), ty)
		tc := d.titleColor
		if tc == nil {
			tc = color.RGBA{255, 220, 80, 255}
		}
		op.ColorScale.ScaleWithColor(tc)
		textv2.Draw(screen, d.title, d.font, op)
	}

	sy := py + ddPad + ddTitleH
	for _, item := range d.items {
		sy += ddSliderGap
		item.draw(screen, px+ddPad, sy)
		sy += item.h
	}
}
